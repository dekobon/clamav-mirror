package main

import (
	"os"
	"golang.org/x/sys/unix"
)

func exists(filePath string) (exists bool) {
	exists = true

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		exists = false
	}

	return
}

func isWritable(directory string) (writable bool) {
	return unix.Access(directory, unix.W_OK) == nil
}
