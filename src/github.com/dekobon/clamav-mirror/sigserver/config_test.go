package sigserver

import (
	"os"
	"testing"
)

func TestParseEnvVars(t *testing.T) {
	defaults := Config{
		Port:                 9999,
		UpdateHourlyInterval: 9999,
	}

	os.Setenv("SIGSERVER_PORT", "8888")
	os.Setenv("UPDATE_HOURLY_INTERVAL", "9")

	actual := ParseEnvVars(defaults)

	if actual.Port != 8888 {
		t.Errorf("Expected port 8888 to be parsed from env var SIGSERVER_PORT. "+
			"Actual: %v", actual.Port)
	}

	if actual.UpdateHourlyInterval != 9 {
		t.Errorf("Expected value 9 to be parsed from env var UPDATE_HOURLY_INTERVAL. "+
			"Actual: %v", actual.UpdateHourlyInterval)
	}
}
