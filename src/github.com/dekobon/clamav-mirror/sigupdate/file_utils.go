package main

import (
	"golang.org/x/sys/unix"
	"os"
)

// Function that determines if a given path exists.
func exists(filePath string) (exists bool) {
	exists = true

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		exists = false
	}

	return
}

// Function that determines if a given directory can be written to.
func isWritable(directory string) (writable bool) {
	return unix.Access(directory, unix.W_OK) == nil
}
