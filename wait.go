package wakeup

import (
	"bytes"
	"net"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/tomb.v2"
)

func Wait(ifaceName string, udpPort int) (net.IP, error) {
	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find interface '%s'", ifaceName)
	}
	addrs, err := iface.Addrs()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get addresses for interface '%s'", ifaceName)
	}
	if len(addrs) == 0 {
		return nil, errors.Errorf("interface '%s' has no addresses", ifaceName)
	}
	hw := iface.HardwareAddr
	if len(hw) != 6 {
		return nil, errors.Errorf("unknown hardware address format for interface '%s'", ifaceName)
	}
	var ifaceMac [6]byte
	copy(ifaceMac[:], hw)
	t := tomb.Tomb{}
	results := make(chan net.IP, len(addrs))
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
			ip, err := wait(udpAddr, ifaceMac)
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
		if ip == nil {
			return nil, errors.Errorf("no result in response")
		}
		return ip, nil
	}
	if err != nil {
		return nil, errors.Wrapf(err, "failed to wait for packet")
	}
	return nil, errors.Errorf("invalid response from wait process")
}

var errPacketFound = errors.New("packet found")

func wait(addr *net.UDPAddr, expectedMac [6]byte) (net.IP, error) {
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to listen for UDP")
	}
	buf := make([]byte, 108)
	for {
		n, _, _, cAddr, err := conn.ReadMsgUDP(buf, nil)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read UDP message")
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
			return nil, errors.Errorf("Received packet with wrong MAC address %s, expected %s", net.HardwareAddr(mac[:]), net.HardwareAddr(expectedMac[:]))
		}
		return cAddr.IP, nil
	}
}
