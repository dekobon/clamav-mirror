package sigserver

import (
	"github.com/dekobon/clamav-mirror/utils"
)

import (
	"github.com/dekobon/clamav-mirror/sigupdate"
	"github.com/pborman/getopt"
	"os"
	"strconv"
	"log"
)

// Config is a data structure that encapsulates the configuration parameters
// used to run the sigserver application.
type Config struct {
	UpdateConfig         sigupdate.Config
	Port                 uint16
	UpdateHourlyInterval uint16
}

var defaultConfig = Config{
	Port: 80,
	UpdateHourlyInterval: 4,
}
// ParseConfig parses environment variables and command line options for
// the runtime configuration.
func ParseConfig(appVersionInfo utils.AppVersionInfo) Config {
	envConfig := ParseEnvVars(defaultConfig)
	return ParseCliFlags(appVersionInfo, envConfig)
}

// ParseEnvVars parses environment variables for runtime configuration.
func ParseEnvVars(defaults Config) Config {
	config := Config{}

	if port, present := os.LookupEnv("SIGSERVER_PORT"); present {
		i, err := strconv.ParseInt(port, 10, 16)

		if err != nil {
			log.Fatal("Error parsing SIGSERVER_PORT environment variable")
		}

		config.Port = uint16(i)
	} else {
		config.Port = defaults.Port
	}

	if updateInterval, present := os.LookupEnv("UPDATE_HOURLY_INTERVAL"); present {
		i, err := strconv.ParseInt(updateInterval, 10, 16)

		if err != nil {
			log.Fatal("Error parsing UPDATE_HOURLY_INTERVAL environment variable")
		}

		config.Port = uint16(i)
	} else {
		config.UpdateHourlyInterval = defaults.UpdateHourlyInterval
	}

	return config
}

// ParseCliFlags parses the CLI options passed to the application.
func ParseCliFlags(appVersionInfo utils.AppVersionInfo, defaults Config) Config {
	listenPortPart := getopt.Uint16Long("port", 'p',
		defaults.Port, "Port to serve signatures on")
	updateHourlyIntervalPart := getopt.Uint16Long("houry-update-interval", 'h',
		defaults.UpdateHourlyInterval, "Number of hours to wait between signature updates")

	updateConfig := sigupdate.ParseConfig(appVersionInfo)

	return Config{
		UpdateConfig:         updateConfig,
		Port:                 *listenPortPart,
		UpdateHourlyInterval: *updateHourlyIntervalPart,
	}
}
