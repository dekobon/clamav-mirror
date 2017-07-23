package sigupdate

import (
	"io/ioutil"
	"os"
	"github.com/dekobon/clamav-mirror/utils"
	"fmt"
	"net/http"
	"errors"
	"time"
	"io"
	"strings"
)

import (
	"github.com/hashicorp/errwrap"
)


// Function that downloads a file from the mirror URL and moves it into the
// data directory if it was successfully downloaded.
func downloadFile(filename string, localFilePath string,
downloadMirrorURL string) (int, error) {

	unknownStatus := -1
	downloadURL := downloadMirrorURL + "/" + filename

	output, err := ioutil.TempFile(os.TempDir(), filename+"-")

	/* For all non-cvd files, skip downloading the file if our local copy is
	 * newer than the remote copy. For .cvd files, the only authoritative way
	 * to know what is newer is to use sigtool. */
	if utils.Exists(localFilePath) {
		newer, err := checkIfRemoteIsNewer(localFilePath, downloadURL)

		if err != nil {
			return unknownStatus, err
		}

		if !newer {
			logger.Printf("Not downloading [%v] because remote copy is "+
				"older than local copy", filename)
			return unknownStatus, nil
		}
	}

	if verboseMode {
		logger.Printf("Downloading to temporary file: [%v]", output.Name())
	}

	if err != nil {
		msg := fmt.Sprintf("Unable to create file: [%v]. {{err}}", output.Name())
		return unknownStatus, errwrap.Wrapf(msg, err)
	}

	defer output.Close()

	response, err := http.Get(downloadURL)

	if err != nil {
		msg := fmt.Sprintf("Unable to retrieve file from: [%v]. {{err}}", downloadURL)
		return unknownStatus, errwrap.Wrapf(msg, err)
	}

	if response.StatusCode != http.StatusOK {
		msg := fmt.Sprintf("Unable to download file: [%v]", response.Status)
		return response.StatusCode, errors.New(msg)
	}

	lastModified, err := http.ParseTime(response.Header.Get("Last-Modified"))

	if err != nil {
		logger.Printf("Error parsing last-modified header [%v] for file: %v",
			response.Header.Get("Last-Modified"), downloadURL)
		lastModified = time.Now()
	}

	defer response.Body.Close()

	n, err := io.Copy(output, response.Body)

	if err != nil {
		msg := fmt.Sprintf("Error copying data from URL [%v] to local file [%v]. {{err}}",
			downloadURL, localFilePath)
		return response.StatusCode, errwrap.Wrapf(msg, err)
	}

	if isItOkToOverwrite(filename, localFilePath, output.Name()) {
		/* Change the last modified time so that we have a record that corresponds to the
		 * server's timestamps. */
		os.Chtimes(output.Name(), lastModified, lastModified)
		os.Rename(output.Name(), localFilePath)

		logger.Printf("Download complete: %v --> %v [%v bytes]", downloadURL, localFilePath, n)
	} else {
		logger.Println("Downloaded file an older signature version than the current file")

		err := os.Remove(output.Name())

		if err != nil {
			logger.Printf("Unable to delete temporary file: %v", output.Name())
		}
	}

	return response.StatusCode, nil
}

// Function that checks to see if the remote file is newer than the locally stored
// file.
func checkIfRemoteIsNewer(localFilePath string, downloadURL string) (bool, error) {
	localFileStat, err := os.Stat(localFilePath)

	if err != nil {
		return true, errwrap.Wrapf("Unable to stat file. {{err}}", err)
	}

	localModTime := localFileStat.ModTime()
	response, err := http.Head(downloadURL)

	if err != nil {
		msg := fmt.Sprintf("Unable to complete HEAD request: [%v]. {{err}}", downloadURL)
		return true, errwrap.Wrapf(msg, err)
	}

	remoteModTime, err := http.ParseTime(response.Header.Get("Last-Modified"))

	if verboseMode {
		logger.Printf("Local file [%v] last-modified: %v", downloadURL, localModTime)
		logger.Printf("Remote file [%v] last-modified: %v", downloadURL, remoteModTime)
	}

	if err != nil {
		msg := fmt.Sprintf("Error parsing last-modified header [%v] for file [%v]. {{err}}",
			response.Header.Get("Last-Modified"), downloadURL)
		return true, errwrap.Wrapf(msg, err)
	}

	if localModTime.After(remoteModTime) {
		logger.Printf("Skipping download of [%v] because local copy is newer", downloadURL)
		return false, nil
	}

	return true, nil
}

// Function that checks to see if we can overwrite a file with a newly downloaded file
func isItOkToOverwrite(filename string, originalFilePath string, newFileTempPath string) bool {
	if !strings.HasSuffix(filename, ".cvd") {
		return true
	}

	oldVersion, err := findLocalVersion(originalFilePath)

	// If there is a problem with the original file, we just overwrite it
	if err != nil {
		return true
	}

	newVersion, err := findLocalVersion(newFileTempPath)

	// If there is a problem with the new file, we don't overwrite the original
	if err != nil {
		return false
	}

	isNewer := newVersion > oldVersion

	if verboseMode {
		logger.Printf("Current file [%v] version [%v]. New file version [%v]. "+
			"Will overwrite: %v",
			filename, oldVersion, newVersion, isNewer)
	}

	return isNewer
}
