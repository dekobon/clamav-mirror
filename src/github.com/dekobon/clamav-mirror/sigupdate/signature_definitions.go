package sigupdate

import (
	"time"
)

// SignatureVersions is for storing the parsed results of the signature versions
// published in ClamAV's TXT record.
type SignatureVersions struct {
	MainVersion         int64
	DailyVersion        int64
	SafeBrowsingVersion int64
	ByteCodeVersion     int64
}

// Signature is for storing the definition of a single signature type.
type Signature struct {
	Name    string
	Version int64
}

// SignatureInfo is for storing a Signature's metadata
type SignatureInfo struct {
	File      string
	BuildTime time.Time
	Version   int64
	MD5       string
}
