package main

import (
	"errors"
	"fmt"
	"github.com/hashicorp/errwrap"
	"github.com/pborman/getopt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
)

var logger *log.Logger
var logFatal *log.Logger

func init() {
	logger = log.New(os.Stdout, "", log.LstdFlags)
	logFatal = log.New(os.Stderr, "", log.LstdFlags)
}

func main() {
	verboseMode, dataFilePath := parseCliFlags()

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

	updateFile(dataFilePath)
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
			msg := fmt.Sprintf("Error parsing absolute path for: %v. {{err}}", localPath)
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

	_, err := os.Stat(filePath)

	if os.IsNotExist(err) {
		exists = false
	}

	return
}

func updateFile(file string) {

}
