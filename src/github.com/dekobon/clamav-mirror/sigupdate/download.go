package sigupdate

import (
	"net"
	"context"
	"container/list"
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

type Download struct {
	Filename string
	LocalFilePath string
	oldSignatureInfo SignatureInfo
}

// resolveMirrorIp resolves all IPs associated with a domain.
func resolveMirrorIp(domain string) ([]net.IPAddr, error) {
	c := context.Background()
	addresses, err := net.DefaultResolver.LookupIPAddr(c, domain)

	if err != nil {
		msg := fmt.Sprintf("Error resolving domain [%v]", domain)
		return []net.IPAddr{}, errors.WrapPrefix(err, msg, 1)
	}

	return addresses, nil
}

func buildDownloadURL(downloadMirrorURL *url.URL, host net.IPAddr,
	filename string) *url.URL {
	hostIp := host.IP.String()

	downloadURL := url.URL{
		Host: hostIp,
		ForceQuery: downloadMirrorURL.ForceQuery,
		Fragment: downloadMirrorURL.Fragment,
		Opaque: downloadMirrorURL.Opaque,
		Path: downloadMirrorURL.Path + "/" + filename,
		RawPath: downloadMirrorURL.RawPath + "/" + filename,
		RawQuery: downloadMirrorURL.RawQuery,
		Scheme: downloadMirrorURL.Scheme,
		User: downloadMirrorURL.User,
	}

	return &downloadURL
}

func downloadFilesWithRetry(downloads *list.List, downloadMirrorURL *url.URL) error {
	addresses, err := resolveMirrorIp(downloadMirrorURL.Host)

	if err != nil {
		msg := fmt.Sprintf("Unable to resolve host [%v]", downloadMirrorURL.Host)
		return errors.WrapPrefix(err, msg, 1)
	}

	// We randomize the list of address so that we are not always
	// hitting the same mirrors.
	utils.Shuffle(addresses)
	mirrorIndex := 0
	mirrorCount := len(addresses)

	for e := downloads.Front(); e != nil; e = e.Next() {
		retry:

		d, ok := e.Value.(Download); if !ok {
			return errors.Errorf("Incorrect type. Expecting Download. " +
				"Actually: %v", e.Value)
		}

		if (d == Download{}) {
			continue
		}

		downloadURL := buildDownloadURL(downloadMirrorURL, addresses[mirrorIndex],
			d.Filename)

		statusCode, err := downloadFile(d, downloadURL)

		/* Many times different mirrors will have different .cdiff files
		 * available. We want to retry with a different mirror in case the
		 * file can't be found. Alternatively, in the case of 500 errors, we
		 * also want to try another mirror. */
		if (statusCode == http.StatusNotFound && mirrorIndex < mirrorCount) ||
			statusCode > 499{

			mirrorIndex++
			goto retry
		}

		if err != nil {
			return err
		}
	}

	return nil
}

func downloadWithRetry(download Download, downloadMirrorURL *url.URL) (int, error){
	addresses, err := resolveMirrorIp(downloadMirrorURL.Host)

	if err != nil {
		msg := fmt.Sprintf("Unable to resolve host [%v]", downloadMirrorURL.Host)
		return -1, errors.WrapPrefix(err, msg, 1)
	}

	// We randomize the list of address so that we are not always
	// hitting the same mirrors.
	utils.Shuffle(addresses)
	mirrorIndex := 0
	mirrorCount := len(addresses)

	retry:
	downloadURL := buildDownloadURL(downloadMirrorURL, addresses[mirrorIndex],
		download.Filename)
	statusCode, err := downloadFile(download, downloadURL)

	if err != nil && mirrorIndex < mirrorCount {
		mirrorIndex++
		goto retry
	} else if err != nil {
		return statusCode, err
	}

	return statusCode, err
}

func downloadFile(download Download, downloadURL *url.URL) (int, error) {
	logger.Printf("Attempting to download: %v", downloadURL.String())

	statusCode, err := executeHttpRequest(download.Filename,
		download.LocalFilePath, downloadURL, download.oldSignatureInfo)

	if verboseMode {
		logger.Printf("Status code: %v", statusCode)
	}

	return statusCode, err
}

// Function that downloads a file from the mirror URL and moves it into the
// data directory if it was successfully downloaded.
func executeHttpRequest(filename string, localFilePath string,
	downloadURL *url.URL, oldSignatureInfo SignatureInfo) (int, error) {

	unknownStatus := -1

	output, err := ioutil.TempFile(os.TempDir(), filename+"-")

	if verboseMode {
		logger.Printf("Downloading to temporary file: [%v]", output.Name())
	}

	if err != nil {
		msg := fmt.Sprintf("Unable to create file [%v]", output.Name())
		return unknownStatus, errors.WrapPrefix(err, msg, 1)
	}

	defer output.Close()

	request, err := http.NewRequest("GET", downloadURL.String(), nil)

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
