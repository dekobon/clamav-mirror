package sigserver

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

import (
	"github.com/go-errors/errors"
	"github.com/robfig/cron"
)

import (
	"github.com/dekobon/clamav-mirror/sigupdate"
	"github.com/dekobon/clamav-mirror/utils"
)

var logger *log.Logger
var logError *log.Logger
var dataDirectory string
var verboseMode bool

func init() {
	logger = log.New(os.Stdout, "", log.LstdFlags)
	logError = log.New(os.Stderr, "", log.LstdFlags|log.Lshortfile)
}

// RunUpdaterAndServer is the functional entry point to the application. This
// function starts the HTTP server and the periodic task executor.
func RunUpdaterAndServer(config Config) error {
	updateConfig := config.UpdateConfig
	dataDirectory = updateConfig.DataFilePath
	verboseMode = updateConfig.Verbose

	{
		err := scheduleUpdates(config)

		if err != nil {
			return errors.WrapPrefix(err, "Error scheduling periodic updates", 1).Err
		}
	}

	{
		err := runServer(config.Port)

		if err != nil {
			return errors.WrapPrefix(err, "Error starting HTTP server", 1).Err
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

func scheduleUpdates(config Config) error {

	cronSchedule := fmt.Sprintf("@every %dh", config.UpdateHourlyInterval)

	run := func() {
		err := sigupdate.RunSignatureUpdate(config.UpdateConfig)

		if err != nil {
			logError.Println(err)
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
	w.Header().Add("X-Robots-Tag", "noindex, nofollow, noarchive")

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
		logError.Printf("Error running stat on file [%v]. %v", dataFilePath, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	localModTime := stat.ModTime().UTC().Truncate(time.Second)

	w.Header().Set("Last-Modified", stat.ModTime().UTC().Truncate(time.Second).Format(http.TimeFormat))
	w.Header().Set("Content-Type", "application/octet-stream")

	if !(r.Method == "GET" || r.Method == "HEAD") {
		logger.Printf("[%v] {%v} %v DENIED", r.Method, r.RemoteAddr, r.URL)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	modifiedSinceString := r.Header.Get("If-Modified-Since")

	if len(modifiedSinceString) > 0 {
		modifiedSince, err := http.ParseTime(modifiedSinceString)

		if err != nil {
			logError.Printf("Couldn't parse time value [%v]. %v", r.Header.Get("If-Modified-Since"), err)
		}

		modifiedSince = modifiedSince.UTC()

		logger.Printf("Local modification time: %v. Remote modification time: %v (%v)",
			localModTime, modifiedSince, modifiedSinceString)

		if modifiedSince.After(localModTime) || modifiedSince.Equal(localModTime.UTC()) {
			if verboseMode {
				logger.Printf("[%v] {%v} %v --> %v (304 Not-Modified)", r.Method, r.RemoteAddr, r.URL, dataFilePath)
			}

			w.WriteHeader(http.StatusNotModified)
			return
		}
	}

	logger.Printf("[%v] {%v} %v --> %v", r.Method, r.RemoteAddr, r.URL, dataFilePath)

	if r.Method == "GET" {
		dataFileReader, err := os.Open(dataFilePath)
		defer dataFileReader.Close()

		if err != nil {
			logError.Printf("Error reading [%v] from disk. %v",
				dataFilePath, err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		io.Copy(w, dataFileReader)
	}
}
