package wakeup

import (
	"net"

	"github.com/pkg/errors"
)

func Send(ipStr string, udpPort int, hwStr string) error {
	ipAddr := net.ParseIP(ipStr)
	if ipAddr == nil {
		return errors.Errorf("failed to parse IP address '%s'", ipStr)
	}
	hwAddr, err := net.ParseMAC(hwStr)
	if err != nil {
		return errors.Wrapf(err, "failed to parse hardware address '%s'", hwStr)
	}
	udpAddr := &net.UDPAddr{IP: ipAddr, Port: udpPort}
	udp, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		return errors.Wrapf(err, "failed to dial UDP '%s'", udpAddr)
	}
	packet := genPacket(hwAddr)
	n, _, err := udp.WriteMsgUDP(packet, nil, nil)
	if err != nil {
		return errors.Wrapf(err, "failed to send UDP '%s'", udpAddr)
	}
	if n != len(packet) {
		return errors.Errorf("invalid number of bytes written %d", n)
	}
	return nil
}

func genPacket(hwAddr net.HardwareAddr) []byte {
	var b []byte
	b = append(b, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF)
	for i := 0; i < 16; i++ {
		b = append(b, hwAddr...)
	}
	b = append(b, 0, 0, 0, 0, 0, 0)
	return b
}
