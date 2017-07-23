package utils

import "time"

const CLAMAV_TIME_LAYOUT = "02 Jan 2006 15:04 -0700"

func ParseClamAVTimeStamp(timeString string) (time.Time, error) {
	return time.Parse(CLAMAV_TIME_LAYOUT, timeString)
}
