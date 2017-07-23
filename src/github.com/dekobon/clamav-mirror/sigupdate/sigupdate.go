package sigupdate

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"math"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

import (
	"github.com/hashicorp/errwrap"
)

import (
	"github.com/dekobon/clamav-mirror/utils"
)

var logger *log.Logger
var logFatal *log.Logger
var sigtoolPath string
var verboseMode bool

func init() {
	logger = log.New(os.Stdout, "", log.LstdFlags)
	logFatal = log.New(os.Stderr, "", log.LstdFlags|log.Lshortfile)
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
		msg := fmt.Sprintf("Unable to resolve TXT record for %v. {{err}}", mirrorDomain)
		return "", errwrap.Wrapf(msg, err)
	}

	if len(mirrorTxtRecords) < 1 {
		msg := fmt.Sprintf("No TXT records returned for %v. {{err}}", mirrorDomain)
		return "", errwrap.Wrapf(msg, err)
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
		return versions, errwrap.Wrapf("Error parsing main version. {{err}}", err)
	}

	daily, err := strconv.ParseInt(s[2], 10, 64)

	if err != nil {
		return versions, errwrap.Wrapf("Error parsing daily version. {{err}}", err)
	}

	safebrowsingv, err := strconv.ParseInt(s[6], 10, 64)

	if err != nil {
		return versions, errwrap.Wrapf("Error parsing safe browsing version. {{err}}", err)
	}

	bytecodev, err := strconv.ParseInt(s[7], 10, 64)

	if err != nil {
		return versions, errwrap.Wrapf("Error parsing bytecode version. {{err}}", err)
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
			logger.Printf("Error parsing absolute path for [%v]", localPath)
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

		downloadNewBaseSignature = false
	} else {
		logger.Printf("Local copy of [%v] does not exist - initiating download.",
			localFilePath)
		// Download the signatures for the first time if they don't exist
		downloadNewBaseSignature = true
	}

	oldVersion, err := findLocalVersion(localFilePath)

	if !downloadNewBaseSignature && (err != nil || oldVersion < 0) {
		logger.Printf("There was a problem with the version [%v] of file [%v]. "+
			"The file will be downloaded again. Original Error: %v", oldVersion, localFilePath, err)
		downloadNewBaseSignature = true
	} else {
		if verboseMode {
			logger.Printf("%v current version: %v", filename, oldVersion)
		}
	}

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

			_, err := downloadFile(diffFilename, localDiffFilePath, downloadMirrorURL)

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
		_, err := downloadFile(filename, localFilePath, downloadMirrorURL)

		if err != nil {
			return err
		}

		return nil
	}

	return nil
}

// Function that uses the ClamAV sigtool executable to extract the version number
// from a signature definition file.
func findLocalVersion(localFilePath string) (int64, error) {
	versionDelim := "Version:"
	errVersion := int64(-1)

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
	validated := false

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