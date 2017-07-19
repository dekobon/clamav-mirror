package main

import (
	"log"
	"os"
	"net"
	"path/filepath"
	"fmt"
	"strings"
);

func main() {
	logger := log.New(os.Stdout, "", log.LstdFlags);

	sigtoolPath := findSigtoolPath();
	logger.Printf("ClamAV executable sigtool found at path: %v", sigtoolPath);

	var mirrorDomain string = "current.cvd.clamav.net";
	var mirrorTxtRecord string = pullTxtRecord(mirrorDomain);
	logger.Printf("TXT record for [%v]: %v", mirrorDomain, mirrorTxtRecord);

	clamav, mainv, dailyv, x, y, z, safebrowsingv, bytecodev := parseTxtRecord(mirrorTxtRecord);
	logger.Printf("TXT record values parsed: " +
		"[clamav=%v,mainv=%v,dailyv=%v,x=%v,y=%v,z=%v,safebrowsingv=%v,bytecodev=%v",
		clamav, mainv, dailyv, x, y, z, safebrowsingv, bytecodev);
}

func pullTxtRecord(mirrorDomain string) (string) {
	mirrorTxtRecords, err := net.LookupTXT(mirrorDomain);

	if (err != nil) {
		msg := fmt.Sprintf("Unable to resolve TXT record for %v", mirrorDomain);
		log.Fatal(msg, err);
		os.Exit(1);
	}

	if (len(mirrorTxtRecords) < 1) {
		msg := fmt.Sprintf("No TXT records returned for %v", mirrorDomain);
		log.Fatal(msg);
		os.Exit(1);
	}

	return mirrorTxtRecords[0];
}

func parseTxtRecord(mirrorTxtRecord string) (string, string, string, string, string, string, string, string) {
	s := strings.SplitN(mirrorTxtRecord, ":", 8);
	return s[0], s[1], s[2], s[3], s[4], s[5], s[6], s[7];
}

func updateFile(file string) {

}

func findSigtoolPath() (path string) {
	var execName string = "sigtool";
	var separator string = string(os.PathSeparator);
	var envPathSeparator string = string(os.PathListSeparator);
	var envPath string = os.Getenv("PATH");

	var localPath string = "." + separator + execName;
	if (exists(localPath)) {
		execPath, err := filepath.Abs(localPath);

		if (err != nil) {
			msg := fmt.Sprintf("Error parsing absolute path for: %v", localPath);
			log.Fatal(msg, err);
			os.Exit(1);
		}

		return execPath;
	}

	for _, pathElement := range strings.Split(envPath, envPathSeparator) {
		var execPath string = pathElement + separator + execName;
		if (exists(execPath)) {
			return execPath;
		}
	}

	log.Fatal("The ClamAV executable sigtool was not found in the " +
		"current directory nor in the system path");
	os.Exit(1);

	return;
}

func exists(filePath string) (exists bool) {
	exists = true

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		exists = false
	}

	return;
}