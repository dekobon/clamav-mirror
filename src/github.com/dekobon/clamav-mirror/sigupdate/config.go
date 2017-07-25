package sigupdate

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

import (
	"github.com/pborman/getopt"
)

import (
	"github.com/dekobon/clamav-mirror/utils"
	"net/url"
)

// Config is a data structure that encapsulates the configuration parameters
// used to run the application.
type Config struct {
	Verbose           bool
	DataFilePath      string
	DiffThreshold     uint16
	DownloadMirrorURL *url.URL
	DnsDbInfoDomain   string
}

// AppVersionInfo is a data structure that represents the version information
// we want to display to users.
type AppVersionInfo struct {
	AppVersion    string
	GitCommitHash string
	UTCBuildTime  string
}

// ParseCliFlags parses the CLI options passed to the application.
func ParseCliFlags(appVersionInfo AppVersionInfo) Config {
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
	dnsDbInfoDomainPart := getopt.StringLong("clamav-dns-db-info-domain", 'i',
		"current.cvd.clamav.net", "DNS domain to verify the virus database "+
			"version via TXT record")

	getopt.Parse()

	if *versionPart {
		fmt.Println("sigupdate")
		fmt.Println("")
		fmt.Printf("Version        : %v\n", appVersionInfo.AppVersion)
		fmt.Printf("Git Commit Hash: %v\n", appVersionInfo.GitCommitHash)
		fmt.Printf("UTC Build Time : %v\n", appVersionInfo.UTCBuildTime)
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

	downloadMirrorURL, err := url.Parse(*downloadMirrorPart)

	if err != nil {
		log.Fatalf("Error parsing URL [%v]: %v", *downloadMirrorPart, err)
	}

	return Config{
		Verbose:           *verbosePart,
		DataFilePath:      dataFileAbsPath,
		DiffThreshold:     *diffThresholdPart,
		DownloadMirrorURL: downloadMirrorURL,
		DnsDbInfoDomain:   *dnsDbInfoDomainPart,
	}
}
