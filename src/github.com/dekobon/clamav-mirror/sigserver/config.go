package sigserver

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

import (
	"github.com/dekobon/clamav-mirror/utils"
)

import (
	"github.com/dekobon/clamav-mirror/sigupdate"
	"github.com/pborman/getopt"
)

type Config struct {
	UpdateConfig         sigupdate.Config
	Port                 uint16
	UpdateHourlyInterval uint16
}

// Function that parses the CLI options passed to the application.
func parseCliFlags(appVersionInfo utils.AppVersionInfo) Config {
	versionPart := getopt.BoolLong("version", 'V',
		"Display the version and exit")
	listenPortPart := getopt.Uint16Long("port", 'p',
		80, "Port to serve signatures on")
	updateHourlyIntervalPart := getopt.Uint16Long("houry-update-interval", 'h',
		4, "Number of hours to wait between signature updates")

	getopt.Parse()

	if *versionPart {
		fmt.Println("sigserver")
		fmt.Println("")
		fmt.Printf("Version        : %v\n", appVersionInfo.AppVersion)
		fmt.Printf("Git Commit Hash: %v\n", appVersionInfo.GitCommitHash)
		fmt.Printf("UTC Build Time : %v\n", appVersionInfo.UTCBuildTime)
		fmt.Print("License        : MPLv2\n")

		os.Exit(0)
	}

	updateConfig := sigupdate.ParseCliFlags(appVersionInfo)

	return Config{}
}
