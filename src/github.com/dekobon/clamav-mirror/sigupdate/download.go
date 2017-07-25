package sigupdate

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

import (
	"github.com/dekobon/clamav-mirror/utils"
)

import (
	"github.com/go-errors/errors"
)

// Function that downloads a file from the mirror URL and moves it into the
// data directory if it was successfully downloaded.
func downloadFile(filename string, localFilePath string,
	downloadMirrorURL *url.URL, oldSignatureInfo SignatureInfo) (int, error) {

	unknownStatus := -1
	downloadURL := downloadMirrorURL.String() + "/" + filename

	output, err := ioutil.TempFile(os.TempDir(), filename+"-")

	if verboseMode {
		logger.Printf("Downloading to temporary file: [%v]", output.Name())
	}

	if err != nil {
		msg := fmt.Sprintf("Unable to create file [%v]", output.Name())
		return unknownStatus, errors.WrapPrefix(err, msg, 1)
	}

	defer output.Close()

	request, err := http.NewRequest("GET", downloadURL, nil)

	if err != nil {
		msg := fmt.Sprintf("Unable to create request for: [GEt %v]", downloadURL)
		return unknownStatus, errors.WrapPrefix(err, msg, 1)
	}

	request.Header.Add("User-Agent", "github.com/dekobon/clamav-mirror")

	/* For .cvd files, the only authoritative way know what is newer is
	 * to use sigtool. */
	if oldSignatureInfo != (SignatureInfo{}) {
		request.Header.Add("If-Modified-Since", oldSignatureInfo.BuildTime.Format(http.TimeFormat))
		/* For all non-cvd files, skip downloading the file if our local copy is
		 * newer than the remote copy. */
	} else if utils.Exists(localFilePath) {
		stat, err := os.Stat(localFilePath)

		if err == nil {
			localModTime := stat.ModTime().UTC().Truncate(time.Second).Format(http.TimeFormat)
			request.Header.Add("If-Modified-Since", localModTime)
		} else {
			logger.Printf("Unable to stat local file [%v]. %v", localFilePath, err)
		}
	}

	response, err := http.DefaultClient.Do(request)

	if err != nil {
		msg := fmt.Sprintf("Unable to retrieve file from [%v]", downloadURL)
		return unknownStatus, errors.WrapPrefix(err, msg, 1)
	}

	if response.StatusCode == http.StatusNotModified {
		logger.Printf("Not downloading [%v] because local copy is newer or the same as remote",
			filename)
		return response.StatusCode, nil
	}

	if response.StatusCode != http.StatusOK {
		msg := fmt.Sprintf("Unable to download file: [%v]", response.Status)
		return response.StatusCode, errors.New(msg)
	}

	defer response.Body.Close()

	n, err := io.Copy(output, response.Body)

	if err != nil {
		msg := fmt.Sprintf("Error copying data from URL [%v] to local file [%v]",
			downloadURL, localFilePath)
		return response.StatusCode, errors.WrapPrefix(err, msg, 1)
	}

	var newSignatureInfo SignatureInfo

	if strings.HasSuffix(filename, ".cvd") && oldSignatureInfo != (SignatureInfo{}) {
		info, err := readSignatureInfo(output.Name())

		// If there is a problem with the new file, we don't overwrite the original
		if err != nil {
			return unknownStatus, err
		}

		newSignatureInfo = info
	}

	if isItOkToOverwrite(filename, oldSignatureInfo, newSignatureInfo) {
		/* Change the last modified time so that we have a record that corresponds to the
		 * server's timestamps. */

		var lastModified time.Time

		if newSignatureInfo == (SignatureInfo{}) {
			modified, err := http.ParseTime(response.Header.Get("Last-Modified"))

			if err != nil {
				logger.Printf("Error parsing last-modified header [%v] for file: %v",
					response.Header.Get("Last-Modified"), downloadURL)
				modified = time.Now()
			}

			lastModified = modified
		} else {
			lastModified = newSignatureInfo.BuildTime
		}

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

// Function that checks to see if we can overwrite a file with a newly downloaded file
func isItOkToOverwrite(filename string, oldSignatureInfo SignatureInfo, newSignatureInfo SignatureInfo) bool {
	if !strings.HasSuffix(filename, ".cvd") || oldSignatureInfo == (SignatureInfo{}) {
		return true
	}

	isNewer := newSignatureInfo.Version > oldSignatureInfo.Version

	if verboseMode {
		logger.Printf("Current file [%v] version [%v]. New file version [%v]. "+
			"Will overwrite: %v",
			filename, newSignatureInfo.Version, newSignatureInfo, isNewer)
	}

	return isNewer
}
