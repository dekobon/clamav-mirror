package sigupdate

import (
	"fmt"
	"log"
	"net"
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
func RunSignatureUpdate(verboseModeEnabled bool, dataFilePath string, downloadMirrorURL string,
	diffCountThreshold uint16) error {
	logger.Println("Updating ClamAV signatures")

	verboseMode = verboseModeEnabled

	if verboseMode {
		logger.Printf("Data file directory: %v", dataFilePath)
	}

	sigtoolParsedPath, err := findSigtoolPath()

	if err != nil {
		return err
	}

	sigtoolPath = sigtoolParsedPath

	if verboseMode {
		logger.Printf("ClamAV executable sigtool found at path: %v", sigtoolPath)
	}

	mirrorDomain := "current.cvd.clamav.net"
	mirrorTxtRecord, err := pullTxtRecord(mirrorDomain)

	if err != nil {
		return err
	}

	if verboseMode {
		logger.Printf("TXT record for [%v]: %v", mirrorDomain, mirrorTxtRecord)
	}

	versions, err := parseTxtRecord(mirrorTxtRecord)

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
		err = updateFile(dataFilePath, signature,
			downloadMirrorURL, diffCountThreshold)

		if err != nil {
			return err
		}
	}

	return nil
}

// Function that gets retrieves the value of the DNS TXT record published by
// ClamAV.
func pullTxtRecord(mirrorDomain string) (string, error) {
	mirrorTxtRecords, err := net.LookupTXT(mirrorDomain)

	if err != nil {
		msg := fmt.Sprintf("Unable to resolve TXT record for [%v]", mirrorDomain)
		return "", errors.WrapPrefix(err, msg, 1)
	}

	if len(mirrorTxtRecords) < 1 {
		msg := fmt.Sprintf("No TXT records returned for [%v]. {{err}}", mirrorDomain)
		return "", errors.WrapPrefix(err, msg, 1)
	}

	return mirrorTxtRecords[0], nil
}

// Function that parses the DNS TXT record published by ClamAV for the latest
// signature versions.
func parseTxtRecord(mirrorTxtRecord string) (SignatureVersions, error) {
	var versions SignatureVersions

	s := strings.SplitN(mirrorTxtRecord, ":", 8)

	mainv, err := strconv.ParseInt(s[1], 10, 64)

	if err != nil {
		return versions, errors.WrapPrefix(err, "Error parsing main version", 1)
	}

	daily, err := strconv.ParseInt(s[2], 10, 64)

	if err != nil {
		return versions, errors.WrapPrefix(err, "Error parsing daily version:", 1)
	}

	safebrowsingv, err := strconv.ParseInt(s[6], 10, 64)

	if err != nil {
		return versions, errors.WrapPrefix(err, "Error parsing safe browsing version", 1)
	}

	bytecodev, err := strconv.ParseInt(s[7], 10, 64)

	if err != nil {
		return versions, errors.WrapPrefix(err, "Error parsing bytecode version", 1)
	}

	versions = SignatureVersions{
		MainVersion:         mainv,
		DailyVersion:        daily,
		SafeBrowsingVersion: safebrowsingv,
		ByteCodeVersion:     bytecodev,
	}

	return versions, nil
}

// Function that finds the path to the sigtool utility on the local system.
func findSigtoolPath() (string, error) {
	execName := "sigtool"
	separator := string(os.PathSeparator)
	envPathSeparator := string(os.PathListSeparator)
	envPath := os.Getenv("PATH")
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
	signature Signature, downloadMirrorURL string, diffCountThreshold uint16) error {
	filePrefix := signature.Name
	currentVersion := signature.Version
	separator := string(filepath.Separator)

	filename := filePrefix + ".cvd"
	localFilePath := dataFilePath + separator + filename

	var downloadNewBaseSignature bool

	if utils.Exists(localFilePath) {
		if verboseMode {
			logger.Printf("Local copy of [%v] already exists - "+
				"initiating diff based update", localFilePath)
		}

		if !utils.IsReadable(localFilePath) {
			return errors.Errorf("Unable to read file [%v]", localFilePath)
		}

		if !utils.IsWritable(localFilePath) {
			return errors.Errorf("Unable to write to file [%v]", localFilePath)
		}

		downloadNewBaseSignature = false
	} else {
		logger.Printf("Local copy of [%v] does not exist - initiating download.",
			localFilePath)
		// Download the signatures for the first time if they don't exist
		downloadNewBaseSignature = true
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
		/* Attempt to download a diff for each version until we reach the current
		 * version. */
		for count := oldVersion + 1; count <= currentVersion; count++ {
			diffFilename := filePrefix + "-" + strconv.FormatInt(count, 10) + ".cdiff"
			localDiffFilePath := dataFilePath + separator + diffFilename

			// Don't bother downloading a diff if it already exists
			if utils.Exists(localDiffFilePath) {
				if verboseMode {
					logger.Printf("Local copy of [%v] already exists, not downloading",
						localDiffFilePath)
				}
				continue
			}

			_, err := downloadFile(diffFilename, localDiffFilePath, downloadMirrorURL,
				SignatureInfo{})

			/* Give up attempting to download incremental diffs if we can't find a
			 * diff file corresponding to the version needed. We just go download
			 * the main signature file again if we hit this case. */
			if err != nil {
				logger.Printf("There was a problem downloading diff [%v] of file [%v]. "+
					"The file original file [%v] will be downloaded again. Original Error: %v",
					count, diffFilename, filename, err)
				downloadNewBaseSignature = true
				break
			}

		}
	}

	/* If we have too many diffs, we go ahead and download the whole signatures
	 * after we have the diffs so that our base signature files stay relatively
	 * current. */
	if !downloadNewBaseSignature && (currentVersion-oldVersion > int64(diffCountThreshold)) {
		logger.Printf("Original signature has deviated beyond threshold from diffs, "+
			"so we are downloading the file [%v] again", filename)

		downloadNewBaseSignature = true
	}

	if downloadNewBaseSignature {
		_, err := downloadFile(filename, localFilePath, downloadMirrorURL,
			signatureInfo)

		if err != nil {
			return err
		}

		return nil
	}

	return nil
}
