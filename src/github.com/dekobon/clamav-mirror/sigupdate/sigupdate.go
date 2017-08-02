package sigupdate

import (
	"container/list"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

import (
	"github.com/go-errors/errors"
)

import (
	"github.com/dekobon/clamav-mirror/utils"
)

var logger *log.Logger
var logError *log.Logger
var sigtoolPath string
var verboseMode bool

func init() {
	logger = log.New(os.Stdout, "", log.LstdFlags)
	logError = log.New(os.Stderr, "", log.LstdFlags|log.Lshortfile)
}

// RunSignatureUpdate is the functional entry point to the application.
// Use this method to invoke the downloader from external code.
func RunSignatureUpdate(config Config) error {
	logger.Println("Updating ClamAV signatures")

	verboseMode = config.Verbose

	if verboseMode {
		logger.Printf("Data file directory: %v", config.DataFilePath)
	}

	sigtoolParsedPath, err := findSigtoolPath(os.Getenv("PATH"))

	if err != nil {
		return err
	}

	sigtoolPath = sigtoolParsedPath

	if verboseMode {
		logger.Printf("ClamAV executable sigtool found at path: %v", sigtoolPath)
	}

	versionTxtRecord, err := pullTxtRecord(config.DNSDbInfoDomain)

	if err != nil {
		return err
	}

	if verboseMode {
		logger.Printf("TXT record for [%v]: %v", config.DNSDbInfoDomain, versionTxtRecord)
	}

	versions, err := parseTxtRecord(versionTxtRecord)

	if err != nil {
		return err
	}

	if verboseMode {
		logger.Printf("TXT record values parsed: %v", versions)
	}

	var signaturesToUpdate = [3]Signature{
		{Name: "main", Version: versions.MainVersion},
		{Name: "daily", Version: versions.DailyVersion},
		{Name: "bytecode", Version: versions.ByteCodeVersion},
	}

	for _, signature := range signaturesToUpdate {
		err = updateFile(config.DataFilePath, signature,
			config.DownloadMirrorURL, config.DiffThreshold)

		if err != nil {
			return err
		}
	}

	return nil
}

// Function that gets retrieves the value of the DNS TXT record published by
// ClamAV.
func pullTxtRecord(dnsDbInfoDomain string) (string, error) {
	mirrorTxtRecords, err := net.LookupTXT(dnsDbInfoDomain)

	if err != nil {
		msg := fmt.Sprintf("Unable to resolve TXT record for [%v]", dnsDbInfoDomain)
		return "", errors.WrapPrefix(err, msg, 1)
	}

	if len(mirrorTxtRecords) < 1 {
		msg := fmt.Sprintf("No TXT records returned for [%v]. {{err}}", dnsDbInfoDomain)
		return "", errors.WrapPrefix(err, msg, 1)
	}

	return mirrorTxtRecords[0], nil
}

// Function that parses the DNS TXT record published by ClamAV for the latest
// signature versions.
func parseTxtRecord(mirrorTxtRecord string) (SignatureVersions, error) {
	var versions SignatureVersions

	if len(mirrorTxtRecord) < 15 {
		return versions, errors.Errorf("Invalid TXT record - records must "+
			"have at least 16 characters. Actual: [%v]", mirrorTxtRecord)
	}

	delimCount := strings.Count(mirrorTxtRecord, ":")

	if delimCount < 7 || delimCount > 7 {
		return versions, errors.Errorf("Invalid TXT record - Invalid number "+
			"of delimiters characters [:] in record. Total delimiters: [%v]", delimCount)
	}

	s := strings.SplitN(mirrorTxtRecord, ":", 8)

	clamavVersion := s[0]

	mainv, err := strconv.ParseUint(s[1], 10, 64)

	if err != nil {
		return versions, errors.WrapPrefix(err, "Error parsing main version", 1)
	}

	daily, err := strconv.ParseUint(s[2], 10, 64)

	if err != nil {
		return versions, errors.WrapPrefix(err, "Error parsing daily version:", 1)
	}

	safebrowsingv, err := strconv.ParseUint(s[6], 10, 64)

	if err != nil {
		return versions, errors.WrapPrefix(err, "Error parsing safe browsing version", 1)
	}

	bytecodev, err := strconv.ParseUint(s[7], 10, 64)

	if err != nil {
		return versions, errors.WrapPrefix(err, "Error parsing bytecode version", 1)
	}

	versions = SignatureVersions{
		ClamAVVersion:       clamavVersion,
		MainVersion:         mainv,
		DailyVersion:        daily,
		SafeBrowsingVersion: safebrowsingv,
		ByteCodeVersion:     bytecodev,
	}

	return versions, nil
}

// Function that finds the path to the sigtool utility on the local system.
func findSigtoolPath(envPath string) (string, error) {
	execName := "sigtool"
	separator := string(os.PathSeparator)
	envPathSeparator := string(os.PathListSeparator)
	localPath := "." + separator + execName

	if utils.Exists(localPath) {
		execPath, err := filepath.Abs(localPath)

		if err != nil {
			logError.Printf("Error parsing absolute path for [%v]", localPath)
		} else {
			return execPath, nil
		}
	}

	for _, pathElement := range strings.Split(envPath, envPathSeparator) {
		execPath := pathElement + separator + execName

		if utils.Exists(execPath) {
			return execPath, nil
		}
	}

	err := errors.New("The ClamAV executable sigtool was not found in the " +
		"current directory nor in the system path.")

	return "", err
}

// Function that updates the data files for a given signature by either
// downloading the datafile or downloading diffs.
func updateFile(dataFilePath string,
	signature Signature, downloadMirrorURL *url.URL, diffCountThreshold uint16) error {
	filePrefix := signature.Name
	currentVersion := signature.Version
	separator := string(filepath.Separator)

	filename := filePrefix + ".cvd"
	localFilePath := dataFilePath + separator + filename

	downloadNewBaseSignature, err := existsAndIsAccessible(localFilePath)

	if err != nil {
		return err
	}

	signatureInfo, err := readSignatureInfo(localFilePath)

	if !downloadNewBaseSignature && err != nil {
		logger.Printf("There was a problem with extracting the metadata from the signature file [%v]. "+
			"The file will be downloaded again. Original Error: %v", localFilePath, err)
		downloadNewBaseSignature = true
	} else {
		if verboseMode && signatureInfo != (SignatureInfo{}) {
			logger.Printf("%v metadata: [File=%v,BuildTime=%v,Version=%v,MD5=%v]",
				filename, signatureInfo.File, signatureInfo.BuildTime, signatureInfo.Version,
				signatureInfo.MD5)
		}
	}

	oldVersion := signatureInfo.Version

	if !downloadNewBaseSignature {
		downloads := list.New()

		/* Attempt to download a diff for each version until we reach the current
		 * version. */
		for count := oldVersion + 1; count <= currentVersion; count++ {
			diffFilename := filePrefix + "-" + strconv.FormatUint(count, 10) + ".cdiff"
			localDiffFilePath := dataFilePath + separator + diffFilename

			// Don't bother downloading a diff if it already exists
			if utils.Exists(localDiffFilePath) {
				if verboseMode {
					logger.Printf("Local copy of [%v] already exists, not downloading",
						localDiffFilePath)
				}
				continue
			}

			downloads.PushBack(Download{
				Filename:         diffFilename,
				LocalFilePath:    localDiffFilePath,
				oldSignatureInfo: signatureInfo,
			})
		}

		err := downloadFilesWithRetry(downloads, downloadMirrorURL)

		/* Give up attempting to download incremental diffs if we can't find a
		 * diff file corresponding to the version needed. We just go download
		 * the main signature file again if we hit this case. */
		if err != nil {
			logger.Printf("There was a problem downloading diffs [%v]-[%v]. "+
				"The file original file [%v] will be downloaded again. Last Error: %v",
				oldVersion+1, currentVersion, filename, err)
			downloadNewBaseSignature = true
		}
	}

	/* If we have too many diffs, we go ahead and download the whole signatures
	 * after we have the diffs so that our base signature files stay relatively
	 * current. */
	if !downloadNewBaseSignature && (currentVersion-oldVersion > uint64(diffCountThreshold)) {
		logger.Printf("Original signature has deviated beyond threshold from diffs, "+
			"so we are downloading the file [%v] again", filename)

		downloadNewBaseSignature = true
	}

	if downloadNewBaseSignature {
		download := Download{
			Filename:         filename,
			LocalFilePath:    localFilePath,
			oldSignatureInfo: signatureInfo,
		}

		_, err := downloadWithRetry(download, downloadMirrorURL)

		if err != nil {
			return err
		}

		return nil
	}

	return nil
}

// Function that checks to see if the specified file already exists. This function
// will error if the file path is not readable or not writable.
func existsAndIsAccessible(localFilePath string) (bool, error) {
	if utils.Exists(localFilePath) {
		if verboseMode {
			logger.Printf("Local copy of [%v] already exists - "+
				"initiating diff based update", localFilePath)
		}

		if !utils.IsReadable(localFilePath) {
			return false, errors.Errorf("Unable to read file [%v]", localFilePath)
		}

		if !utils.IsWritable(localFilePath) {
			return false, errors.Errorf("Unable to write to file [%v]", localFilePath)
		}

		return false, nil
	}

	logger.Printf("Local copy of [%v] does not exist - initiating download.",
		localFilePath)
	// Download the signatures for the first time if they don't exist
	return true, nil
}
