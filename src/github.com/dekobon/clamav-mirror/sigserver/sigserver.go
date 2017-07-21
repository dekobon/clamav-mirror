package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

import (
	"github.com/pborman/getopt"
)

import (
	"github.com/dekobon/clamav-mirror/utils"
)

var githash = "unknown"
var buildstamp = "unknown"
var appversion = "unknown"

var logger *log.Logger
var logFatal *log.Logger

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hi there, I love %s!", r.URL.Path[1:])
}

func main() {
	http.HandleFunc("/", handler)
	http.ListenAndServe(":8080", nil)
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
		fmt.Printf("License        : MPLv2\n")
		fmt.Printf("Git Commit Hash: %v\n", githash)
		fmt.Printf("UTC Build Time : %v\n", buildstamp)

		os.Exit(0)
	}

	if !utils.Exists(*dataFilePart) {
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

	if !utils.IsWritable(dataFileAbsPath) {
		msg := fmt.Sprintf("Data file path doesn't have write access for "+
			"current user at path: %v", dataFileAbsPath)
		logFatal.Fatal(msg)
	}

	return *verbosePart, dataFileAbsPath, *downloadMirrorPart, *diffThresholdPart
}
