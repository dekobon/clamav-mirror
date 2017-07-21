package utils

import (
	"golang.org/x/sys/unix"
	"os"
)

// Exists function that determines if a given path exists.
func Exists(filePath string) (exists bool) {
	exists = true

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		exists = false
	}

	return exists
}

// IsWritable function that determines if a given directory
// can be written to.
func IsWritable(directory string) (writable bool) {
	return unix.Access(directory, unix.W_OK) == nil
}
