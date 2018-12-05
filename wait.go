package wakeup

import (
	"bytes"
	"context"
	"gopkg.in/tomb.v2"
	"net"
	"strings"

	"github.com/pkg/errors"
)

func Wait(ctx context.Context, ifaceName string, udpPort int) (string, error) {
	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return "", errors.Wrapf(err, "failed to find interface '%s'", ifaceName)
	}
	addrs, err := iface.Addrs()
	if err != nil {
		return "", errors.Wrapf(err, "failed to get addresses for interface '%s'", ifaceName)
	}
	if len(addrs) == 0 {
		return "", errors.Errorf("interface '%s' has no addresses", ifaceName)
	}
	hw := iface.HardwareAddr
	if len(hw) != 6 {
		return "", errors.Errorf("unknown hardware address format for interface '%s'", ifaceName)
	}
	var ifaceMac [6]byte
	copy(ifaceMac[:], hw)
	t, ctx := tomb.WithContext(ctx)
	results := make(chan string, len(addrs))
	for i := range addrs {
		addr := addrs[i]
		if !strings.Contains(addr.Network(), "ip") {
			continue
		}
		ip, _, err := net.ParseCIDR(addr.String())
		if err != nil {
			continue
		}
		udpAddr := &net.UDPAddr{IP: ip, Port: udpPort}
		t.Go(func() error {
			ip, err := wait(ctx, udpAddr, ifaceMac)
			if err == nil {
				results <- ip
				return errPacketFound
			}
			return err
		})
	}
	err = t.Wait()
	close(results)
	if err == errPacketFound {
		ip := <-results
		if ip == "" {
			return "", errors.Errorf("no result in response")
		}
		return ip, nil
	}
	if err != nil {
		return "", errors.Wrapf(err, "failed to wait for packet")
	}
	return "", errors.Errorf("invalid response from wait process")
}

var errPacketFound = errors.New("packet found")

func wait(ctx context.Context, addr *net.UDPAddr, expectedMac [6]byte) (string, error) {
	listener := &net.ListenConfig{}
	conn, err := listener.ListenPacket(ctx, "udp", addr.String())
	if err != nil {
		return "", errors.Wrapf(err, "failed to listen for UDP")
	}
	go func() {
		<-ctx.Done()
		conn.Close()
	}()
	buf := make([]byte, 108)
	for {
		n, cAddr, err := conn.ReadFrom(buf)
		if err != nil {
			return "", errors.Wrapf(err, "failed to read UDP message")
		}
		if n < 102 {
			continue
		}
		body := buf[:n]
		if !bytes.HasPrefix(body, []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}) {
			continue
		}
		body = buf[6:]
		var mac [6]byte
		copy(mac[:], body)
		bad := false
		for i := 0; i < 16; i++ {
			var cMac [6]byte
			copy(cMac[:], body)
			body = body[6:]
			if cMac != mac {
				bad = true
				break
			}
		}
		if bad {
			continue
		}
		if mac != expectedMac {
			return "", errors.Errorf("Received packet with wrong MAC address %s, expected %s", net.HardwareAddr(mac[:]), net.HardwareAddr(expectedMac[:]))
		}
		return cAddr.String(), nil
	}
}
