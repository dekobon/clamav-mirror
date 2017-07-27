package sigupdate

import (
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestInPathFindSigtoolPath(t *testing.T) {
	separator := string(os.PathSeparator)
	listSeparator := string(os.PathListSeparator)

	tempDir := os.TempDir()
	dir1 := tempDir + separator + "dir-" + strconv.Itoa(int(time.Now().Unix())) + "-1"
	if err := os.Mkdir(dir1, 0700); err != nil {
		t.Errorf("Couldn't make directory [%v]. %v", dir1, err)
	}

	dir2 := tempDir + separator + "dir-" + strconv.Itoa(int(time.Now().Unix())) + "-2"
	if err := os.Mkdir(dir2, 0700); err != nil {
		t.Errorf("Couldn't make directory [%v]. %v", dir2, err)
	}

	dir3 := tempDir + separator + "dir-" + strconv.Itoa(int(time.Now().Unix())) + "-3"
	if err := os.Mkdir(dir3, 0700); err != nil {
		t.Errorf("Couldn't make directory [%v]. %v", dir3, err)
	}

	defer os.RemoveAll(dir1)
	defer os.RemoveAll(dir2)
	defer os.RemoveAll(dir3)

	expectedSigtoolPath := dir2 + separator + "sigtool"

	os.OpenFile(expectedSigtoolPath, os.O_RDONLY|os.O_CREATE, 0666)

	pathString := dir1 + listSeparator + dir2 + listSeparator + dir3

	actualSigtoolPath, err := findSigtoolPath(pathString)

	if err != nil {
		t.Errorf("Error executing findSigtoolPath. %v", err)
	}

	if actualSigtoolPath != expectedSigtoolPath {
		t.Errorf("Expected sigtool path: %v\n"+
			"Actual sigtool path  : %v\n", expectedSigtoolPath, actualSigtoolPath)
	}
}

func TestNotInPathFindSigtoolPath(t *testing.T) {
	separator := string(os.PathSeparator)
	listSeparator := string(os.PathListSeparator)

	tempDir := os.TempDir()
	dir1 := tempDir + separator + "dir-" + strconv.Itoa(int(time.Now().Unix())) + "-1"
	if err := os.Mkdir(dir1, 0700); err != nil {
		t.Errorf("Couldn't make directory [%v]. %v", dir1, err)
	}

	dir2 := tempDir + separator + "dir-" + strconv.Itoa(int(time.Now().Unix())) + "-2"
	if err := os.Mkdir(dir2, 0700); err != nil {
		t.Errorf("Couldn't make directory [%v]. %v", dir2, err)
	}

	dir3 := tempDir + separator + "dir-" + strconv.Itoa(int(time.Now().Unix())) + "-3"
	if err := os.Mkdir(dir3, 0700); err != nil {
		t.Errorf("Couldn't make directory [%v]. %v", dir3, err)
	}

	defer os.RemoveAll(dir1)
	defer os.RemoveAll(dir2)
	defer os.RemoveAll(dir3)

	pathString := dir1 + listSeparator + dir2 + listSeparator + dir3

	_, err := findSigtoolPath(pathString)

	if !strings.HasPrefix(err.Error(), "The ClamAV executable sigtool was not found") {
		t.Errorf("Expected exception was not thrown")
	}
}
