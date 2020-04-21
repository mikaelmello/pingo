package core

import (
	"fmt"
	"net"
	"syscall"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

const (
	echoCode       = 0
	icmpProtocol   = 1
	icmpv6Protocol = 58
	icmpNetwork    = "ip4:icmp"
	icmpv6Network  = "ip6:ipv6-icmp"
)

func (b *Bundle) requestEcho(conn *icmp.PacketConn) error {

	bigID := uint64ToBytes(b.bigID)     // ensure same source
	tstp := unixNanoToBytes(time.Now()) // calculate rtt
	data := append(bigID, tstp...)

	body := &icmp.Echo{
		ID:   b.id,
		Seq:  b.currentSequence,
		Data: data,
	}

	msg := &icmp.Message{
		Type: b.GetICMPType(),
		Code: echoCode,
		Body: body,
	}

	msgBytes, err := msg.Marshal(nil)

	if err != nil {
		return err
	}

	for {
		_, err := conn.WriteTo(msgBytes, b.address)

		if err == nil {
			break
		}

		neterr, ok := err.(*net.OpError)

		if ok && neterr.Err == syscall.ENOBUFS {
			continue
		}

		break
	}

	b.totalSent++
	b.currentSequence++

	return nil
}

// GetICMPType returns the appropriate type to be used in the ICMP request of this bundle
func (b *Bundle) GetICMPType() icmp.Type {
	if b.isIPv4 {
		return ipv4.ICMPTypeEcho
	}

	return ipv6.ICMPTypeEchoRequest
}

// GetNetwork returns the appropriate ICMP network value of the bundle
func (b *Bundle) GetNetwork() string {
	if b.isIPv4 {
		return icmpNetwork
	}

	return icmpv6Network
}

// GetProtocol returns the appropriate ICMP protocol value of the bundle
func (b *Bundle) GetProtocol() int {
	if b.isIPv4 {
		return icmpProtocol
	}

	return icmpv6Protocol
}

// GetConnection returns a connection made to the bundle's address
func (b *Bundle) GetConnection() (*icmp.PacketConn, error) {
	conn, err := icmp.ListenPacket(b.GetNetwork(), "")

	if err != nil {
		return nil, fmt.Errorf("Could not listen to ICMP packets, error: %s", err.Error())
	}

	if b.isIPv4 {
		if err := conn.IPv4PacketConn().SetTTL(b.ttl); err != nil {
			return nil, fmt.Errorf("Could not set TTL in connection, error: %s", err.Error())
		}
		if err := conn.IPv4PacketConn().SetControlMessage(ipv4.FlagTTL, true); err != nil {
			return nil, fmt.Errorf("Could not set control message in connection, error: %s", err.Error())
		}
	} else {
		if err := conn.IPv6PacketConn().SetHopLimit(b.ttl); err != nil {
			return nil, fmt.Errorf("Could not set control message in connection, error: %s", err.Error())
		}
		if err := conn.IPv6PacketConn().SetControlMessage(ipv6.FlagHopLimit, true); err != nil {
			return nil, fmt.Errorf("Could not set control message in connection, error: %s", err.Error())
		}
	}

	return conn, nil
}
