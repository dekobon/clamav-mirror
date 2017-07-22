package sigserver

import (
	"fmt"
	"log"
	"net/http"
)

import (
	"github.com/hashicorp/errwrap"
	//"github.com/robfig/cron"
)

import (
//"github.com/dekobon/clamav-mirror/sigupdate"
)

var githash = "unknown"
var buildstamp = "unknown"
var appversion = "unknown"

var logger *log.Logger
var logFatal *log.Logger

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hi there, I love %s!", r.URL.Path[1:])
}

// RunUpdaterAndServer is the functional entry point to the application. This
// function starts the HTTP server and the periodic task executor.
func RunUpdaterAndServer(verboseMode bool, dataFilePath string, downloadMirrorURL string,
	diffCountThreshold uint16, port uint16, refreshHourInterval uint16) error {

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
	logger.Printf("Starting ClamAV signature mirror HTTP server on port [%v]", port)
	http.HandleFunc("/", handler)
	http.ListenAndServe(":"+string(port), nil)

	return nil
}

func scheduleUpdates(verboseMode bool, dataFilePath string, downloadMirrorURL string,
	diffCountThreshold uint16, refreshHourInterval uint16) error {

	//cronSchedule := fmt.Sprintf("0 */%v * * *", refreshHourInterval)
	//
	//c := cron.New()
	//c.AddFunc(cronSchedule)

	return nil
}
