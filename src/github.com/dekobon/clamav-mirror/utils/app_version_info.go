package utils

// AppVersionInfo is a data structure that represents the version information
// we want to display to users.
type AppVersionInfo struct {
	AppVersion    string
	GitCommitHash string
	UTCBuildTime  string
}
