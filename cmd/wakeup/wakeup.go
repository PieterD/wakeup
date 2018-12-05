package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/PieterD/wakeup"
)

func main() {
	var (
		ipStr   string
		hwStr   string
		udpPort int
	)
	flag.StringVar(&ipStr, "ipaddr", "", "IP Address")
	flag.StringVar(&hwStr, "hwaddr", "", "Hardware address")
	flag.IntVar(&udpPort, "port", 7, "UDP Port to send to")
	flag.Parse()
	if ipStr == "" || hwStr == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	if err := wakeup.Send(ipStr, udpPort, hwStr); err != nil {
		fmt.Fprintf(os.Stderr, "failed: %v\n", err)
		os.Exit(1)
	}
}
