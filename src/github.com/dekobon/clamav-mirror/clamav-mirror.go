package main

import (
	"log"
	"os"
	"net"
	"fmt"
);

func main() {
	var mirrorDomain string = "current.cvd.clamav.net";
	mirrorTxtRecord, err := net.LookupTXT(mirrorDomain);

	if (err != nil) {
		msg := fmt.Sprintf("Unable to resolve TXT record for %v", mirrorDomain);
		log.Fatal(msg, err);
		os.Exit(1);
	}

	fmt.Printf("%v", mirrorTxtRecord);
}
