package main

import (
	"bufio"
	"errors"
	"fmt"
	"golang.org/x/sys/unix"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

import (
	"github.com/hashicorp/errwrap"
	"github.com/pborman/getopt"
	"math"
	"strconv"
)

var logger *log.Logger
var logFatal *log.Logger

func init() {
	logger = log.New(os.Stdout, "", log.LstdFlags)
	logFatal = log.New(os.Stderr, "", log.LstdFlags)
}

func main() {
	verboseMode, dataFilePath := parseCliFlags()

	if verboseMode {
		logger.Printf("Data file directory: %v", dataFilePath)
	}

	sigtoolPath, err := findSigtoolPath()

	if err != nil {
		logFatal.Fatal(err)
	}

	if verboseMode {
		logger.Printf("ClamAV executable sigtool found at path: %v", sigtoolPath)
	}

	var mirrorDomain string = "current.cvd.clamav.net"
	mirrorTxtRecord, err := pullTxtRecord(mirrorDomain)

	if err != nil {
		logFatal.Fatal(err)
	}

	if verboseMode {
		logger.Printf("TXT record for [%v]: %v", mirrorDomain, mirrorTxtRecord)
	}

	clamav, mainv, dailyv, x, y, z, safebrowsingv, bytecodev := parseTxtRecord(mirrorTxtRecord)

	if verboseMode {
		logger.Printf("TXT record values parsed: "+
			"[clamav=%v,mainv=%v,dailyv=%v,x=%v,y=%v,z=%v,safebrowsingv=%v,bytecodev=%v",
			clamav, mainv, dailyv, x, y, z, safebrowsingv, bytecodev)
	}

	updateFile(verboseMode, dataFilePath, sigtoolPath, "main", mainv)
	updateFile(verboseMode, dataFilePath, sigtoolPath, "daily", dailyv)
	updateFile(verboseMode, dataFilePath, sigtoolPath, "bytecode", bytecodev)
}

func parseCliFlags() (bool, string) {
	verbosePart := getopt.BoolLong("verbose", 'v',
		"Enable verbose mode with additional debugging information")
	dataFilePart := getopt.StringLong("data-file-path", 'd',
		"/var/clamav/data", "Path to ClamAV data files")
	getopt.Parse()

	if !exists(*dataFilePart) {
		msg := fmt.Sprintf("Data file path doesn't exist or isn't accessible: %v",
			*dataFilePart)
		logFatal.Fatal(msg)
	}

	dataFileAbsPath, err := filepath.Abs(*dataFilePart)

	if err != nil {
		msg := fmt.Sprintf("Unable to parse absolute path of data file path: %v",
			*dataFilePart)
		logFatal.Fatal(msg)
	}

	if !isWritable(dataFileAbsPath) {
		msg := fmt.Sprintf("Data file path doesn't have write access for "+
			"current user at path: %v", dataFileAbsPath)
		logFatal.Fatal(msg)
	}

	return *verbosePart, dataFileAbsPath
}

func pullTxtRecord(mirrorDomain string) (string, error) {
	mirrorTxtRecords, err := net.LookupTXT(mirrorDomain)

	if err != nil {
		msg := fmt.Sprintf("Unable to resolve TXT record for %v. {{err}}", mirrorDomain)
		return "", errwrap.Wrapf(msg, err)
	}

	if len(mirrorTxtRecords) < 1 {
		msg := fmt.Sprintf("No TXT records returned for %v. {{err}}", mirrorDomain)
		return "", errwrap.Wrapf(msg, err)
	}

	return mirrorTxtRecords[0], nil
}

func parseTxtRecord(mirrorTxtRecord string) (string, string, string, string, string, string, string, string) {
	s := strings.SplitN(mirrorTxtRecord, ":", 8)
	return s[0], s[1], s[2], s[3], s[4], s[5], s[6], s[7]
}

func findSigtoolPath() (string, error) {
	var execName string = "sigtool"
	var separator string = string(os.PathSeparator)
	var envPathSeparator string = string(os.PathListSeparator)
	var envPath string = os.Getenv("PATH")

	var localPath string = "." + separator + execName
	if exists(localPath) {
		execPath, err := filepath.Abs(localPath)

		if err != nil {
			msg := fmt.Sprintf("Error parsing absolute path for [%v]. {{err}}", localPath)
			return "", errwrap.Wrapf(msg, err)
		}

		return execPath, nil
	}

	for _, pathElement := range strings.Split(envPath, envPathSeparator) {
		var execPath string = pathElement + separator + execName
		if exists(execPath) {
			return execPath, nil
		}
	}

	err := errors.New("The ClamAV executable sigtool was not found in the " +
		"current directory nor in the system path.")

	return "", err
}

func exists(filePath string) (exists bool) {
	exists = true

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		exists = false
	}

	return
}

func isWritable(directory string) (writable bool) {
	return unix.Access(directory, unix.W_OK) == nil
}

func updateFile(verboseMode bool, dataFilePath string, sigtoolPath string,
	filePrefix string, currentVersion string) error {

	filename := filePrefix + ".cvd"
	localFilePath := dataFilePath + string(filepath.Separator) + filename

	if !exists(localFilePath) {
		logger.Printf("Local copy of [%v] does not exist - initiating download.",
			localFilePath)
		err := downloadFile(verboseMode, filename, localFilePath)

		if err != nil {
			return err
		} else {
			return nil
		}
	}

	if verboseMode {
		logger.Printf("Local copy of [%v] already exists - "+
			"initiating diff based update", localFilePath)
	}

	oldVersion, err := findLocalVersion(localFilePath, sigtoolPath)

	if err != nil || oldVersion < 0 {
		logger.Printf("There was a problem with the version [%v] of file [%v]. "+
			"The file will be downloaded again. Original Error: %v", oldVersion, localFilePath, err)
		err := downloadFile(verboseMode, filename, localFilePath)

		if err != nil {
			return err
		} else {
			return nil
		}
	}

	if verboseMode {
		logger.Printf("%v current version: %v", filename, oldVersion)
	}

	return nil
}

func findLocalVersion(localFilePath string, sigtoolPath string) (int64, error) {
	var versionDelim string = "Version:"
	var errVersion int64 = -1

	cmd := exec.Command(sigtoolPath, "-i", localFilePath)
	stdout, err := cmd.StdoutPipe()

	defer stdout.Close()

	if err != nil {
		return errVersion, errwrap.Wrapf("Error instantiating sigtool command. {{err}}", err)
	}

	if err := cmd.Start(); err != nil {
		return errVersion, errwrap.Wrapf("Error running sigtool. {{err}}", err)
	}

	scanner := bufio.NewScanner(stdout)
	var version int64 = math.MinInt64
	var validated bool = false

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, versionDelim) {
			s := strings.SplitAfter(line, versionDelim+" ")
			versionString := strings.TrimSpace(s[1])
			parsedVersion, err := strconv.ParseInt(versionString, 10, 64)

			if err != nil {
				msg := fmt.Sprintf("Error converting [%v] to 64-bit integer. {{err}}",
					versionString)
				return errVersion, errwrap.Wrapf(msg, err)
			}

			version = parsedVersion
		}

		if strings.HasPrefix(line, "Verification OK") {
			validated = true
		}
	}

	if !validated {
		return errVersion, errors.New("The file was not reported as validated")
	}

	if version == math.MinInt64 {
		return errVersion, errors.New("No version information was available for file")
	}

	if err := scanner.Err(); err != nil {
		return errVersion, errwrap.Wrapf("Error parsing sigtool STDOUT", err)
	}

	if err := cmd.Wait(); err != nil {
		return errVersion, errwrap.Wrapf("Error waiting for sigtool STDOUT to flush", err)
	}

	return version, nil
}

func downloadFile(verboseMode bool, filename string, localFilePath string) error {
	downloadMirror := "http://database.clamav.net"
	downloadUrl := downloadMirror + "/" + filename

	output, err := ioutil.TempFile(os.TempDir(), filename+"-")

	if verboseMode {
		logger.Printf("Downloading to temporary file: [%v]", output.Name())
	}

	if err != nil {
		msg := fmt.Sprintf("Unable to create file: [%v]. {{err}}", output.Name())
		return errwrap.Wrapf(msg, err)
	}

	defer output.Close()

	response, err := http.Get(downloadUrl)

	if err != nil {
		msg := fmt.Sprintf("Unable to retrieve file from: [%v]. {{err}}", downloadUrl)
		return errwrap.Wrapf(msg, err)
	}

	defer response.Body.Close()

	n, err := io.Copy(output, response.Body)

	if err != nil {
		msg := fmt.Sprintf("Error copying data from URL [%v] to local file [%v]. {{err}}",
			downloadUrl, localFilePath)
		return errwrap.Wrapf(msg, err)
	}

	os.Rename(output.Name(), localFilePath)

	logger.Printf("Download complete: %v --> %v [%v bytes]", downloadUrl, localFilePath, n)

	return nil
}
