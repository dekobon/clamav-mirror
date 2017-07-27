package sigupdate

import (
	"strings"
	"testing"
)

func TestValidDomainPullTxtRecord(t *testing.T) {
	domain := "current.cvd.clamav.net"

	record, err := pullTxtRecord(domain)

	if err != nil {
		t.Errorf("Error pulling default TXT record. Check your network "+
			"connection. Error: %v", err)
	}

	if len(record) < 15 {
		t.Errorf("Invalid TXT record recieved: %v", record)
	}
}

func TestInvalidDomainPullTxtRecord(t *testing.T) {
	domain := "this-shouldnt-be-a-valid-domain.some-crazy-tld"

	_, err := pullTxtRecord(domain)

	if !strings.HasPrefix(err.Error(), "Unable to resolve TXT record "+
		"for [this-shouldnt-be-a-valid-domain.some-crazy-tld]") {
		t.Errorf("Expected error was not thrown. Actual error: %v", err)
	}
}
