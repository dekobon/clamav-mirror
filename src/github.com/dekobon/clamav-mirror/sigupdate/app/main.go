package main

import (
	"log"
)

import (
	"github.com/dekobon/clamav-mirror/sigupdate"
	"github.com/dekobon/clamav-mirror/utils"
)

import (
	"github.com/go-errors/errors"
)

var githash = "unknown"
var buildstamp = "unknown"
var appversion = "unknown"

// Main entry point to the downloader application. This will allow you to run
// the downloader as a stand-alone binary.
func main() {
	appVersionInfo := utils.AppVersionInfo{
		AppVersion:    appversion,
		GitCommitHash: githash,
		UTCBuildTime:  buildstamp,
	}

	config := sigupdate.ParseConfig(appVersionInfo)
	err := sigupdate.RunSignatureUpdate(config)

	if err != nil {
		log.Fatal(err.(*errors.Error).ErrorStack())
	}
}
