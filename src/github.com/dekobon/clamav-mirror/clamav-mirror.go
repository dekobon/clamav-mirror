package main

import (
	"log"
	"os"
	"net"
	"fmt"
	"strings"
);

func main() {
	logger := log.New(os.Stdout, "", log.LstdFlags);
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