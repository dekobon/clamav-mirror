package sigupdate

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
)

import (
	"github.com/dekobon/clamav-mirror/utils"
)

import (
	"github.com/pborman/getopt"
)

// Config is a data structure that encapsulates the configuration parameters
// used to run the sigupdate application.
type Config struct {
	Verbose           bool
	DataFilePath      string
	DiffThreshold     uint16
	DownloadMirrorURL *url.URL
	DNSDbInfoDomain   string
}

var defaultConfig = Config{
	Verbose:           false,
	DataFilePath:      "/var/clamav/data",
	DiffThreshold:     100,
	DownloadMirrorURL: defaultMirrorURL(),
	DNSDbInfoDomain:   "current.cvd.clamav.net",
}

func defaultMirrorURL() *url.URL {
	defaultMirrorURL, _ := url.Parse("http://database.clamav.net")
	return defaultMirrorURL
}

// ParseConfig parses environment variables and command line options for
// the runtime configuration.
func ParseConfig(appVersionInfo utils.AppVersionInfo) Config {
	envConfig := ParseEnvVars(defaultConfig)
	return ParseCliFlags(appVersionInfo, envConfig)
}

// ParseEnvVars parses environment variables for runtime configuration.
func ParseEnvVars(defaults Config) Config {
	config := Config{}
	if verbose, present := os.LookupEnv("VERBOSE"); present {
		b, err := strconv.ParseBool(verbose)

		if err != nil {
			log.Fatal("Error parsing VERBOSE environment variable", err)
		}
		config.Verbose = b
	} else {
		config.Verbose = defaultConfig.Verbose
	}

	if dataFilePath, present := os.LookupEnv("DATA_FILE_PATH"); present {
		config.DataFilePath = dataFilePath
	} else {
		config.DataFilePath = defaultConfig.DataFilePath
	}

	if diffThreshold, present := os.LookupEnv("DIFF_THRESHOLD"); present {
		i, err := strconv.ParseInt(diffThreshold, 10, 16)

		if err != nil {
			log.Fatal("Error parsing DIFF_THRESHOLD environment variable")
		}

		config.DiffThreshold = uint16(i)
	} else {
		config.DiffThreshold = defaults.DiffThreshold
	}

	if downloadMirrorURL, present := os.LookupEnv("DOWNLOAD_MIRROR_URL"); present {
		u, err := url.Parse(downloadMirrorURL)

		if err != nil {
			log.Fatal("Error parsing DOWNLOAD_MIRROR_URL", err)
		}

		config.DownloadMirrorURL = u
	} else {
		config.DownloadMirrorURL = defaults.DownloadMirrorURL
	}

	if DNSDbDomain, present := os.LookupEnv("DNS_DB_DOMAIN"); present {
		config.DNSDbInfoDomain = DNSDbDomain
	} else {
		config.DNSDbInfoDomain = defaults.DNSDbInfoDomain
	}

	return config
}

// ParseCliFlags parses the CLI options passed to the application.
func ParseCliFlags(appVersionInfo utils.AppVersionInfo, defaults Config) Config {
	verbosePart := getopt.BoolLong("verbose", 'v',
		"Enable verbose mode with additional debugging information")
	versionPart := getopt.BoolLong("version", 'V',
		"Display the version and exit")
	dataFilePart := getopt.StringLong("data-file-path", 'd',
		defaults.DataFilePath, "Path to ClamAV data files")
	diffThresholdPart := getopt.Uint16Long("diff-count-threshold", 't',
		defaults.DiffThreshold,
		"Number of diffs to download until we redownload the signature files")
	downloadMirrorPart := getopt.StringLong("download-mirror-url", 'm',
		defaults.DownloadMirrorURL.String(),
		"URL to download signature updates from")
	dnsDbInfoDomainPart := getopt.StringLong("clamav-dns-db-info-domain", 'i',
		defaults.DNSDbInfoDomain,
		"DNS domain to verify the virus database "+
			"version via TXT record")

	getopt.Parse()

	if *versionPart {
		fmt.Println(os.Args[0])
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
		Verbose:           *verbosePart || defaults.Verbose,
		DataFilePath:      dataFileAbsPath,
		DiffThreshold:     *diffThresholdPart,
		DownloadMirrorURL: downloadMirrorURL,
		DNSDbInfoDomain:   *dnsDbInfoDomainPart,
	}
}
