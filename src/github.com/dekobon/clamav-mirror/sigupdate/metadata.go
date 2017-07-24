package sigupdate

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
)

import (
	"github.com/hashicorp/errwrap"
)

import (
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
		msg := fmt.Sprintf("Error parsing build time [%v]. {{err}}",
			metadata["build time"])
		return info, errwrap.Wrapf(msg, err)
	}

	version, err := strconv.ParseInt(metadata["version"], 10, 64)

	if err != nil {
		msg := fmt.Sprintf("Error converting [%v] to 64-bit integer. {{err}}",
			metadata["version"])
		return info, errwrap.Wrapf(msg, err)
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
		return nil, errwrap.Wrapf("Error instantiating sigtool command. {{err}}", err)
	}

	defer stdout.Close()

	if err := cmd.Start(); err != nil {
		return nil, errwrap.Wrapf("Error running sigtool. {{err}}", err)
	}

	metadata, err := parseMetadata(stdout)

	if err := cmd.Wait(); err != nil {
		return nil, errwrap.Wrapf("Error waiting for sigtool STDOUT to flush", err)
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
		return nil, errwrap.Wrapf("Error parsing sigtool input", err)
	}

	return entries, nil
}
