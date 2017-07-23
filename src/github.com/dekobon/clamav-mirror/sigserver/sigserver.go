package sigserver

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

import (
	"github.com/hashicorp/errwrap"
	"github.com/robfig/cron"
)

import (
	"github.com/dekobon/clamav-mirror/sigupdate"
	"github.com/dekobon/clamav-mirror/utils"
	"io"
	"path/filepath"
)

var githash = "unknown"
var buildstamp = "unknown"
var appversion = "unknown"

var logger *log.Logger
var logFatal *log.Logger
var dataDirectory string

func init() {
	logger = log.New(os.Stdout, "", log.LstdFlags)
	logFatal = log.New(os.Stderr, "", log.LstdFlags|log.Lshortfile)
}

// RunUpdaterAndServer is the functional entry point to the application. This
// function starts the HTTP server and the periodic task executor.
func RunUpdaterAndServer(verboseMode bool, dataFilePath string, downloadMirrorURL string,
	diffCountThreshold uint16, port uint16, refreshHourInterval uint16) error {

	dataDirectory = dataFilePath

	{
		err := scheduleUpdates(verboseMode, dataFilePath, downloadMirrorURL,
			diffCountThreshold, refreshHourInterval)

		if err != nil {
			return errwrap.Wrapf("Error scheduling periodic updates. {{err}}", err)
		}
	}

	{
		err := runServer(port)

		if err != nil {
			return errwrap.Wrapf("Error running HTTP server. {{err}}", err)
		}
	}

	return nil
}

func runServer(port uint16) error {
	listenAddr := ":" + strconv.Itoa(int(port))
	logger.Printf("Starting ClamAV signature mirror HTTP server on port [%v]",
		listenAddr)
	http.HandleFunc("/", handler)
	http.ListenAndServe(listenAddr, nil)

	return nil
}

func scheduleUpdates(verboseMode bool, dataFilePath string, downloadMirrorURL string,
	diffCountThreshold uint16, refreshHourInterval uint16) error {

	cronSchedule := fmt.Sprintf("0 */%v * * *", refreshHourInterval)

	run := func() {
		err := sigupdate.RunSignatureUpdate(verboseMode, dataFilePath,
			downloadMirrorURL, diffCountThreshold)

		if err != nil {
			logger.Println(err)
		}
	}

	// Update once before scheduling
	run()

	c := cron.New()
	c.AddFunc(cronSchedule, run)
	c.Start()

	return nil
}

func validFileRequested(path string, file string) bool {
	dir := filepath.Dir(path)
	validFileExtension := (strings.HasSuffix(file, ".cvd") || strings.HasSuffix(file, ".cdiff")) &&
		!strings.Contains(file, "..")
	validDir := dir == "/"
	// CVD or cdiff filenames should never be very big
	validFilename := len(file) < 48 && len(file) > 4

	return validDir && validFileExtension && validFilename
}

func handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Server", "ClamAV Mirror")

	// Outright reject large paths before doing any processing
	if len(r.URL.Path) > 128 {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	path := filepath.Clean(r.URL.Path)
	file := filepath.Base(path)

	// If a non-signature file extension was requested, we just 404
	if !validFileRequested(path, file) {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	dataFilePath := dataDirectory + string(filepath.Separator) + file
	fileExists := utils.Exists(dataFilePath)

	if !fileExists {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	stat, err := os.Stat(dataFilePath)

	if err != nil {
		logger.Printf("Error running stat on file [%v]. %v", dataFilePath, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Last-Modified", stat.ModTime().UTC().Format(http.TimeFormat))
	w.Header().Set("Content-Type", "application/octet-stream")

	if r.Method == "GET" {
		dataFileReader, err := os.Open(dataFilePath)
		defer dataFileReader.Close()

		if err != nil {
			logger.Printf("Error reading [%v] from disk. %v",
				dataFilePath, err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		io.Copy(w, dataFileReader)
	} else if r.Method == "HEAD" {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
