package sigupdate

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
)

import (
	"github.com/go-errors/errors"
)

import (
	"bytes"
	"github.com/dekobon/clamav-mirror/utils"
)

// Function that uses the ClamAV sigtool executable to extract metadata
// from a signature definition file.
func readSignatureInfo(localFilePath string) (SignatureInfo, error) {
	info := SignatureInfo{}
	metadata, err := readMetadataFromSigtool(localFilePath)

	if err != nil {
		return info, err
	}

	buildTime, err := utils.ParseClamAVTimeStamp(metadata["build time"])

	if err != nil {
		msg := fmt.Sprintf("Error parsing build time [%v]",
			metadata["build time"])
		return info, errors.WrapPrefix(err, msg, 1)
	}

	version, err := strconv.ParseInt(metadata["version"], 10, 64)

	if err != nil {
		msg := fmt.Sprintf("Error converting [%v] to 64-bit integer",
			metadata["version"])
		return info, errors.WrapPrefix(err, msg, 1)
	}

	info.File = metadata["file"]
	info.BuildTime = buildTime.UTC()
	info.Version = version
	info.MD5 = metadata["md5"]

	return info, nil
}

func readMetadataFromSigtool(localFilePath string) (map[string]string, error) {
	cmd := exec.Command(sigtoolPath, "-i", localFilePath)
	stdout, err := cmd.StdoutPipe()

	if err != nil {
		return nil, errors.WrapPrefix(err, "Error configuring STDOUT for sigtool", 1)
	}

	stderr, err := cmd.StderrPipe()

	if err != nil {
		return nil, errors.WrapPrefix(err, "Error configuring STDERR for sigtool", 1)
	}

	defer stdout.Close()

	if err := cmd.Start(); err != nil {
		return nil, errors.WrapPrefix(err, "Error running sigtool", 1)
	}

	metadata, err := parseMetadata(stdout)
	stderrBuf := new(bytes.Buffer)
	stderrBuf.ReadFrom(stderr)

	if err := cmd.Wait(); err != nil {
		return nil, errors.WrapPrefix(err, stderrBuf.String(), 1)
	}

	return metadata, nil
}

func parseMetadata(reader io.Reader) (map[string]string, error) {
	delim := ": "
	scanner := bufio.NewScanner(reader)
	verified := false
	entries := make(map[string]string)

	for scanner.Scan() {
		line := scanner.Text()

		// Record the verification value if present
		if strings.HasPrefix(line, "Verification OK") {
			verified = true
			continue
		}

		parts := strings.SplitN(line, delim, 2)

		if len(parts) < 2 {
			continue
		}

		entries[strings.ToLower(parts[0])] = strings.TrimSpace(parts[1])
	}

	if !verified {
		return nil, errors.New("The file was not reported as verified")
	}

	if err := scanner.Err(); err != nil {
		return nil, errors.WrapPrefix(err, "Error parsing sigtool input", 1)
	}

	return entries, nil
}
