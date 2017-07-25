package main

import (
	"log"
)

import (
	"github.com/go-errors/errors"
)

import (
	"github.com/dekobon/clamav-mirror/sigserver"
	"github.com/dekobon/clamav-mirror/utils"
	"github.com/dekobon/clamav-mirror/sigupdate"
)

var githash = "unknown"
var buildstamp = "unknown"
var appversion = "unknown"

// Main entry point to the server application. This will allow you to run
// the server as a stand-alone binary.
func main() {
	appVersionInfo := utils.AppVersionInfo{
		AppVersion:    appversion,
		GitCommitHash: githash,
		UTCBuildTime:  buildstamp,
	}

	cliFlags := sigserver.ParseCliFlags(appVersionInfo)

	err := sigserver.RunUpdaterAndServer(parseCliFlags())

	if err != nil {
		log.Fatal(err.(*errors.Error).ErrorStack())
	}
}
