package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

import (
	"github.com/hashicorp/errwrap"
	"github.com/pborman/getopt"
)

var logger *log.Logger
var logFatal *log.Logger

func init() {
	logger = log.New(os.Stdout, "", log.LstdFlags)
	logFatal = log.New(os.Stderr, "", log.LstdFlags|log.Lshortfile)
}

/*
 * Main entry point to the downloader application. This will allow you to run
 * the downloader as a stand-alone binary.
 */
func main() {
	err := run(parseCliFlags())

	if err != nil {
		logFatal.Fatal(err)
	}
}

/*
 * Functional entry point to the application. Use this method to invoke the
 * downloader from external code.
 */
func run(verboseMode bool, dataFilePath string, downloadMirrorUrl string) error {
	if verboseMode {
		logger.Printf("Data file directory: %v", dataFilePath)
	}

	sigtoolPath, err := findSigtoolPath()

	if err != nil {
		return err
	}

	if verboseMode {
		logger.Printf("ClamAV executable sigtool found at path: %v", sigtoolPath)
	}

	var mirrorDomain string = "current.cvd.clamav.net"
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

	err = updateFile(verboseMode, dataFilePath, sigtoolPath, "main", versions.MainVersion,
		downloadMirrorUrl)

	if err != nil {
		return err
	}

	err = updateFile(verboseMode, dataFilePath, sigtoolPath, "daily", versions.DailyVersion,
		downloadMirrorUrl)

	if err != nil {
		return err
	}

	err = updateFile(verboseMode, dataFilePath, sigtoolPath, "bytecode", versions.ByteCodeVersion,
		downloadMirrorUrl)

	if err != nil {
		return err
	}

	return nil
}

func parseCliFlags() (bool, string, string) {
	verbosePart := getopt.BoolLong("verbose", 'v',
		"Enable verbose mode with additional debugging information")
	dataFilePart := getopt.StringLong("data-file-path", 'd',
		"/var/clamav/data", "Path to ClamAV data files")
	downloadMirrorPart := getopt.StringLong("download-mirror-url", 'm',
		"http://database.clamav.net", "URL to download signature updates from")

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

	return *verbosePart, dataFileAbsPath, *downloadMirrorPart
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

func updateFile(verboseMode bool, dataFilePath string, sigtoolPath string,
	filePrefix string, currentVersion int64, downloadMirrorUrl string) error {

	separator := string(filepath.Separator)

	filename := filePrefix + ".cvd"
	localFilePath := dataFilePath + separator + filename

	if !exists(localFilePath) {
		logger.Printf("Local copy of [%v] does not exist - initiating download.",
			localFilePath)
		_, err := downloadFile(verboseMode, filename, localFilePath, downloadMirrorUrl)

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
		_, err := downloadFile(verboseMode, filename, localFilePath, downloadMirrorUrl)

		if err != nil {
			return err
		} else {
			return nil
		}
	}

	if verboseMode {
		logger.Printf("%v current version: %v", filename, oldVersion)
	}

	for count := oldVersion + 1; count <= currentVersion; count++ {
		diffFilename := filePrefix + "-" + strconv.FormatInt(count, 10) + ".cdiff"
		localDiffFilePath := dataFilePath + separator + diffFilename

		// Don't bother downloading a diff if it already exists
		if exists(localDiffFilePath) {
			if verboseMode {
				logger.Printf("Local copy of [%v] already exists, not downloading",
					localDiffFilePath)
			}
			continue
		}

		_, err := downloadFile(verboseMode, diffFilename, localDiffFilePath, downloadMirrorUrl)

		/* Give up attempting to download incremental diffs if we can't find a
		 * diff file corresponding to the version needed. We just go download
		 * the main signature file again if we hit this case. */
		if err != nil {
			logger.Printf("There was a problem downloading diff [%v] of file [%v]. "+
				"The file original file [%v] will be downloaded again. Original Error: %v",
				count, diffFilename, filename, err)

			_, err := downloadFile(verboseMode, filename, localFilePath, downloadMirrorUrl)

			if err != nil {
				return err
			}
			break
		}
	}

	/* If we have too many diffs, we go ahead and redownload the whole signatures
	 * after we have the diffs so that our base signature files stay relatively
	 * current. */
	if currentVersion-oldVersion > 100 {
		logger.Printf("Original signature has deviated beyond threshold from diffs, "+
			"so we are downloading the file [%v] again", filename)

		_, err := downloadFile(verboseMode, filename, localFilePath, downloadMirrorUrl)

		if err != nil {
			return err
		}
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

func downloadFile(verboseMode bool, filename string, localFilePath string,
	downloadMirrorUrl string) (int, error) {

	unknownStatus := -1
	downloadUrl := downloadMirrorUrl + "/" + filename

	output, err := ioutil.TempFile(os.TempDir(), filename+"-")

	if verboseMode {
		logger.Printf("Downloading to temporary file: [%v]", output.Name())
	}

	if err != nil {
		msg := fmt.Sprintf("Unable to create file: [%v]. {{err}}", output.Name())
		return unknownStatus, errwrap.Wrapf(msg, err)
	}

	defer output.Close()

	response, err := http.Get(downloadUrl)

	if err != nil {
		msg := fmt.Sprintf("Unable to retrieve file from: [%v]. {{err}}", downloadUrl)
		return unknownStatus, errwrap.Wrapf(msg, err)
	}

	if response.StatusCode != http.StatusOK {
		msg := fmt.Sprintf("Unable to download file: [%v]", response.Status)
		return response.StatusCode, errors.New(msg)
	}

	defer response.Body.Close()

	n, err := io.Copy(output, response.Body)

	if err != nil {
		msg := fmt.Sprintf("Error copying data from URL [%v] to local file [%v]. {{err}}",
			downloadUrl, localFilePath)
		return response.StatusCode, errwrap.Wrapf(msg, err)
	}

	os.Rename(output.Name(), localFilePath)

	logger.Printf("Download complete: %v --> %v [%v bytes]", downloadUrl, localFilePath, n)

	return response.StatusCode, nil
}
