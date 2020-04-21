package core

import (
	"fmt"
	"net"
	"sync"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

const (
	echoCode                  = 0
	icmpProtocol              = 1
	icmpv6Protocol            = 58
	dataLength                = 16
	icmpPrivilegedNetwork     = "ip4:icmp"
	icmpv6PrivilegedNetwork   = "ip6:ipv6-icmp"
	icmpUnprivilegedNetwork   = "udp4"
	icmpv6UnprivilegedNetwork = "udp6"
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

	var address net.Addr = b.address
	if !b.isPrivileged {
		address = &net.UDPAddr{IP: b.address.IP, Zone: b.address.Zone}
	}
	_, err = conn.WriteTo(msgBytes, address)
	b.totalSent++
	b.currentSequence++

	return err
}

func (b *Bundle) pollICMP(
	wg *sync.WaitGroup,
	conn *icmp.PacketConn,
	recv chan<- *rawPacket,
) {
	defer wg.Done()
	for {
		select {
		case <-b.isFinished:
			return
		default:
			buffer := make([]byte, 1024)
			conn.SetReadDeadline(time.Now().Add(time.Second * 1))
			length, ttl, err := b.readFrom(conn, buffer)
			if err != nil {
				if neterr, ok := err.(*net.OpError); ok {
					if neterr.Timeout() {
						continue
					} else {
						close(b.isFinished)
						return
					}
				}
			}

			recv <- &rawPacket{content: buffer, length: length, ttl: ttl}
		}
	}
}

func (b *Bundle) readFrom(conn *icmp.PacketConn, bytes []byte) (int, int, error) {
	var length int
	var ttl int
	var err error
	if b.isIPv4 {
		var cm *ipv4.ControlMessage
		length, cm, _, err = conn.IPv4PacketConn().ReadFrom(bytes)
		if cm != nil {
			ttl = cm.TTL
		}
	} else {
		var cm *ipv6.ControlMessage
		length, cm, _, err = conn.IPv6PacketConn().ReadFrom(bytes)
		if cm != nil {
			ttl = cm.HopLimit
		}
	}

	return length, ttl, err
}

func (b *Bundle) checkRawPacket(raw *rawPacket) (bool, error) {
	receivedTstp := time.Now()

	m, err := icmp.ParseMessage(b.GetProtocol(), raw.content)
	if err != nil {
		return false, fmt.Errorf("Error parsing ICMP message: %s", err.Error())
	}

	if m.Code != echoCode || (m.Type != ipv4.ICMPTypeEchoReply && m.Type != ipv6.ICMPTypeEchoReply) {
		// Not an echo reply, ignore it
		return false, nil
	}

	switch body := m.Body.(type) {
	case *icmp.Echo:
		// // If we are privileged, we can match icmp.ID
		// if p.network == "ip" {
		// 	// Check if reply from same ID
		// 	if pkt.ID != p.id {
		// 		return nil
		// 	}
		// }

		if len(body.Data) < dataLength {
			return false, fmt.Errorf("Missing data, %d bytes out of %d", len(body.Data), dataLength)
		}

		bigID := bytesToUint64(body.Data[:8])
		tstp := bytesToUnixNano(body.Data[8:])

		if (body.Seq + 1) != b.currentSequence {
			return false, nil
		}

		if bigID != b.bigID {
			return false, nil
		}

		rtt := receivedTstp.Sub(tstp).Nanoseconds()
		if rtt > b.maxRtt {
			b.maxRtt = rtt
		}
		b.rtts = append(b.rtts, rtt)
		b.totalReceived++
		return true, nil
	default:
		return false, fmt.Errorf("Invalid body type: '%T'", body)
	}
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
	if b.isIPv4 && b.isPrivileged {
		return icmpPrivilegedNetwork
	}
	if b.isIPv4 && !b.isPrivileged {
		return icmpUnprivilegedNetwork
	}
	if !b.isIPv4 && b.isPrivileged {
		return icmpv6PrivilegedNetwork
	}

	return icmpv6UnprivilegedNetwork
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
