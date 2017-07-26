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

	cliFlags := sigserver.ParseConfig(appVersionInfo)

	err := sigserver.RunUpdaterAndServer(cliFlags)

	if err != nil {
		log.Fatal(err.(*errors.Error).ErrorStack())
	}
}
