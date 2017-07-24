package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

import (
	"github.com/go-errors/errors"
	"github.com/pborman/getopt"
)

import (
	"github.com/dekobon/clamav-mirror/sigupdate"
	"github.com/dekobon/clamav-mirror/utils"
)

var githash = "unknown"
var buildstamp = "unknown"
var appversion = "unknown"

// Main entry point to the downloader application. This will allow you to run
// the downloader as a stand-alone binary.
func main() {
	err := sigupdate.RunSignatureUpdate(parseCliFlags())

	if err != nil {
		log.Fatal(err.(*errors.Error).ErrorStack())
	}
}

// Function that parses the CLI options passed to the application.
func parseCliFlags() (bool, string, string, uint16) {
	verbosePart := getopt.BoolLong("verbose", 'v',
		"Enable verbose mode with additional debugging information")
	versionPart := getopt.BoolLong("version", 'V',
		"Display the version and exit")
	dataFilePart := getopt.StringLong("data-file-path", 'd',
		"/var/clamav/data", "Path to ClamAV data files")
	diffThresholdPart := getopt.Uint16Long("diff-count-threshold", 't',
		100, "Number of diffs to download until we redownload the signature files")
	downloadMirrorPart := getopt.StringLong("download-mirror-url", 'm',
		"http://database.clamav.net", "URL to download signature updates from")

	getopt.Parse()

	if *versionPart {
		fmt.Println("sigupdate")
		fmt.Println("")
		fmt.Printf("Version        : %v\n", appversion)
		fmt.Printf("Git Commit Hash: %v\n", githash)
		fmt.Printf("UTC Build Time : %v\n", buildstamp)
		fmt.Print("License        : MPLv2\n")

		os.Exit(0)
	}

	if !utils.Exists(*dataFilePart) {
		msg := fmt.Sprintf("Data file path doesn't exist or isn't accessible: %v",
			*dataFilePart)
		log.Fatal(msg)
	}

	dataFileAbsPath, err := filepath.Abs(*dataFilePart)

	if err != nil {
		msg := fmt.Sprintf("Unable to parse absolute path of data file path: %v",
			*dataFilePart)
		log.Fatal(msg)
	}

	if !utils.IsReadable(dataFileAbsPath) {
		msg := fmt.Sprintf("Data file path doesn't have read access for "+
			"current user at path: %v", dataFileAbsPath)
		log.Fatal(msg)
	}

	if !utils.IsWritable(dataFileAbsPath) {
		msg := fmt.Sprintf("Data file path doesn't have write access for "+
			"current user at path: %v", dataFileAbsPath)
		log.Fatal(msg)
	}

	return *verbosePart, dataFileAbsPath, *downloadMirrorPart, *diffThresholdPart
}
