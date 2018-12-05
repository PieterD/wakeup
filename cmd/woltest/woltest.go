package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"

	"github.com/pkg/errors"

	"github.com/PieterD/wakeup"
)

func main() {
	var (
		ifaceName string
		udpPort   int
		list      bool
	)
	flag.StringVar(&ifaceName, "iface", "", "Interface to listen to")
	flag.BoolVar(&list, "list", false, "List the available interfaces")
	flag.IntVar(&udpPort, "port", 7, "UDP Port to listen to")
	flag.Parse()

	if list {
		err := listInterfaces()
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed: %+v\n", err)
			os.Exit(1)
		}
		return
	}

	if ifaceName == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	ip, err := wakeup.Wait(context.Background(), ifaceName, udpPort)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed: %+v\n", err)
		os.Exit(1)
	}
	fmt.Printf("WOL packet received from %s\n", ip)

}

func listInterfaces() error {
	ifaces, err := net.Interfaces()
	if err != nil {
		return errors.Wrapf(err, "failed to list interfaces")
	}
	longestName := 0
	for _, iface := range ifaces {
		if len(iface.Name) > longestName {
			longestName = len(iface.Name)
		}
	}
	for _, iface := range ifaces {
		fmt.Printf("%-*s", longestName, iface.Name)
		if iface.HardwareAddr != nil {
			fmt.Printf(" [%s]", iface.HardwareAddr)
		}
		fmt.Printf("\n")
		addrs, err := iface.Addrs()
		if err != nil {
			return errors.Wrapf(err, "failed to fetch addresses for interface '%s'", iface.Name)
		}
		for _, addr := range addrs {
			fmt.Printf("  %s: %s\n", addr.Network(), addr)
		}
		fmt.Printf("\n")
	}
	return nil
}
