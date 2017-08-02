package sigupdate

import (
	"testing"
)

func TestWithMultipleARecordsResolveMirrorIP(t *testing.T) {
	domainWithMultipleARecords := "db.us.clamav.net"

	addresses, err := resolveMirrorIp(domainWithMultipleARecords)

	if err != nil {
		t.Errorf("Check your network connection. Resolve error: %v", err)
	}

	if len(addresses) < 2 {
		t.Error("Expected multiple IP addresses and only got 1 or less")
	}
}

func TestWithIPAddressResolveMirrorIP(t *testing.T)  {
	ip := "127.0.0.1"

	addresses, err := resolveMirrorIp(ip)

	if err != nil {
		t.Error(err)
	}

	if len(addresses) != 1 {
		t.Error("Only a single result should be present")
	}
}