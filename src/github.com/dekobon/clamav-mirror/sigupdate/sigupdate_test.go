package sigupdate

import (
	"strings"
	"testing"
)

func TestTypicalRecordParseTxtRecord(t *testing.T) {
	record := "0.99.2:58:23602:1501176540:1:63:46223:307"
	versions, err := parseTxtRecord(record)

	if err != nil {
		t.Errorf("Failed to parse: %v\n%v", record, err)
	}

	if versions.ClamAVVersion != "0.99.2" {
		t.Errorf("Didn't parse ClamAV version correctly. Expected 0.99.2."+
			"Actually: %v", versions.ClamAVVersion)
	}

	if versions.ByteCodeVersion != 307 {
		t.Errorf("Didn't parse byte code version correctly. Expected 307."+
			"Actually: %v", versions.ByteCodeVersion)
	}

	if versions.DailyVersion != 23602 {
		t.Errorf("Didn't parse daily version correctly. Expected 23602."+
			"Actually: %v", versions.DailyVersion)
	}

	if versions.MainVersion != 58 {
		t.Errorf("Didn't parse main version correctly. Expected 58."+
			"Actually: %v", versions.MainVersion)
	}

	if versions.SafeBrowsingVersion != 46223 {
		t.Errorf("Didn't parse safe browsing version correctly. Expected 46223."+
			"Actually: %v", versions.SafeBrowsingVersion)
	}
}

func TestMinimalRecordParseTxtRecord(t *testing.T) {
	record := "0.0.0:1:2:3:4:5:6:7"
	versions, err := parseTxtRecord(record)

	if err != nil {
		t.Errorf("Failed to parse: %v\n%v", record, err)
	}

	if versions.ClamAVVersion != "0.0.0" {
		t.Errorf("Didn't parse ClamAV version correctly. Expected 0.0.0."+
			"Actually: %v", versions.ClamAVVersion)
	}

	if versions.ByteCodeVersion != 7 {
		t.Errorf("Didn't parse byte code version correctly. Expected 7."+
			"Actually: %v", versions.ByteCodeVersion)
	}

	if versions.DailyVersion != 2 {
		t.Errorf("Didn't parse daily version correctly. Expected 2."+
			"Actually: %v", versions.DailyVersion)
	}

	if versions.MainVersion != 1 {
		t.Errorf("Didn't parse main version correctly. Expected 1."+
			"Actually: %v", versions.MainVersion)
	}

	if versions.SafeBrowsingVersion != 6 {
		t.Errorf("Didn't parse safe browsing version correctly. Expected 6."+
			"Actually: %v", versions.SafeBrowsingVersion)
	}
}

func TestEmptyRecordParseTxtRecord(t *testing.T) {
	record := ""

	_, err := parseTxtRecord(record)

	if !strings.HasPrefix(err.Error(), "Invalid TXT record - records "+
		"must have at least 16 characters.") {
		t.Error("Expected string length error not thrown")
	}
}

func TestNoDelimitersRecordParseTxtRecord(t *testing.T) {
	record := "1234567890123456789"

	_, err := parseTxtRecord(record)

	if !strings.HasPrefix(err.Error(), "Invalid TXT record - Invalid number of "+
		"delimiters characters") {
		t.Error("Expected delimiter count error not thrown")
	}
}

func TestTooFewDelimitersRecordParseTxtRecord(t *testing.T) {
	record := "0.0.0:12:345:6789:0123456789"

	_, err := parseTxtRecord(record)

	if !strings.HasPrefix(err.Error(), "Invalid TXT record - Invalid number of "+
		"delimiters characters") {
		t.Error("Expected delimiter count error not thrown")
	}
}

func TestWrongCharactersMainVerRecordParseTxtRecord(t *testing.T) {
	record := "0.0.0:AAA:1:1:1:1:1:1"

	_, err := parseTxtRecord(record)

	if !strings.HasPrefix(err.Error(), "Error parsing main version") {
		t.Errorf("Expected parse error [main] not thrown")
	}
}

func TestWrongCharactersDailyVerRecordParseTxtRecord(t *testing.T) {
	record := "0.0.0:1:BBBB:1:1:1:1:1"

	_, err := parseTxtRecord(record)

	if !strings.HasPrefix(err.Error(), "Error parsing daily version") {
		t.Errorf("Expected parse error [daily] not thrown")
	}
}

func TestWrongCharactersBytecodeVerRecordParseTxtRecord(t *testing.T) {
	record := "0.0.0:1:1:1:1:1:1:CCC"

	_, err := parseTxtRecord(record)

	if !strings.HasPrefix(err.Error(), "Error parsing bytecode version") {
		t.Errorf("Expected parse error [bytecode] not thrown")
	}
}
