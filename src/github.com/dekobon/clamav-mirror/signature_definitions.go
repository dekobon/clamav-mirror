package main

/*
 * Data structure for storing the parsed results of the signature versions
 * published in ClamAV's TXT record.
 */
type SignatureVersions struct {
	MainVersion         int64
	DailyVersion        int64
	SafeBrowsingVersion int64
	ByteCodeVersion     int64
}

/*
 * Data structure for storing the definition of a single signature type.
 */
type Signature struct {
	Name string
	Version int64
}