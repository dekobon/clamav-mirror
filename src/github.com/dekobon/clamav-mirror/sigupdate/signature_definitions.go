package sigupdate

import (
	"time"
)

// SignatureVersions is for storing the parsed results of the signature versions
// published in ClamAV's TXT record.
type SignatureVersions struct {
	ClamAVVersion       string
	MainVersion         uint64
	DailyVersion        uint64
	SafeBrowsingVersion uint64
	ByteCodeVersion     uint64
}

// Signature is for storing the definition of a single signature type.
type Signature struct {
	Name    string
	Version uint64
}

// SignatureInfo is for storing a Signature's metadata
type SignatureInfo struct {
	File      string
	BuildTime time.Time
	Version   uint64
	MD5       string
}
