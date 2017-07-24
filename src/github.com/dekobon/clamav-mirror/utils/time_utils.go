package utils

import "time"

const clamavTimeLayout = "02 Jan 2006 15:04 -0700"

// ParseClamAVTimeStamp parses a ClamAV build time timstamp string and returns
// a time.Time instance representing the timestamp
func ParseClamAVTimeStamp(timeString string) (time.Time, error) {
	return time.Parse(clamavTimeLayout, timeString)
}
