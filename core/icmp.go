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

func (s *Session) requestEcho(conn *icmp.PacketConn) error {

	bigID := uint64ToBytes(s.bigID)     // ensure same source
	tstp := unixNanoToBytes(time.Now()) // calculate rtt
	data := append(bigID, tstp...)

	body := &icmp.Echo{
		ID:   s.id,
		Seq:  s.currentSequence,
		Data: data,
	}

	msg := &icmp.Message{
		Type: s.GetICMPType(),
		Code: echoCode,
		Body: body,
	}

	msgBytes, err := msg.Marshal(nil)

	if err != nil {
		return err
	}

	var address net.Addr = s.address
	if !s.settings.IsPrivileged {
		address = &net.UDPAddr{IP: s.address.IP, Zone: s.address.Zone}
	}
	_, err = conn.WriteTo(msgBytes, address)
	s.totalSent++
	s.currentSequence++

	return err
}

func (s *Session) pollICMP(
	wg *sync.WaitGroup,
	conn *icmp.PacketConn,
	recv chan<- *rawPacket,
) {
	defer wg.Done()
	for {
		select {
		case <-s.isFinished:
			return
		default:
			buffer := make([]byte, 1024)
			conn.SetReadDeadline(time.Now().Add(time.Second * 1))
			length, ttl, err := s.readFrom(conn, buffer)
			if err != nil {
				if neterr, ok := err.(*net.OpError); ok {
					if neterr.Timeout() {
						continue
					} else {
						close(s.isFinished)
						return
					}
				}
			}

			recv <- &rawPacket{content: buffer, length: length, ttl: ttl}
		}
	}
}

func (s *Session) readFrom(conn *icmp.PacketConn, bytes []byte) (int, int, error) {
	var length int
	var ttl int
	var err error
	if s.isIPv4 {
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

func (s *Session) checkRawPacket(raw *rawPacket) (bool, error) {
	receivedTstp := time.Now()

	m, err := icmp.ParseMessage(s.GetProtocol(), raw.content)
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

		if (body.Seq + 1) != s.currentSequence {
			return false, nil
		}

		if bigID != s.bigID {
			return false, nil
		}

		rtt := receivedTstp.Sub(tstp).Nanoseconds()
		if rtt > s.maxRtt {
			s.maxRtt = rtt
		}
		s.rtts = append(s.rtts, rtt)
		s.totalReceived++
		return true, nil
	default:
		return false, fmt.Errorf("Invalid body type: '%T'", body)
	}
}

// GetICMPType returns the appropriate type to be used in the ICMP request of this session
func (s *Session) GetICMPType() icmp.Type {
	if s.isIPv4 {
		return ipv4.ICMPTypeEcho
	}

	return ipv6.ICMPTypeEchoRequest
}

// GetNetwork returns the appropriate ICMP network value of the session
func (s *Session) GetNetwork() string {
	if s.isIPv4 && s.settings.IsPrivileged {
		return icmpPrivilegedNetwork
	}
	if s.isIPv4 && !s.settings.IsPrivileged {
		return icmpUnprivilegedNetwork
	}
	if !s.isIPv4 && s.settings.IsPrivileged {
		return icmpv6PrivilegedNetwork
	}

	return icmpv6UnprivilegedNetwork
}

// GetProtocol returns the appropriate ICMP protocol value of the session
func (s *Session) GetProtocol() int {
	if s.isIPv4 {
		return icmpProtocol
	}

	return icmpv6Protocol
}

// GetConnection returns a connection made to the session's address
func (s *Session) GetConnection() (*icmp.PacketConn, error) {
	conn, err := icmp.ListenPacket(s.GetNetwork(), "")

	if err != nil {
		return nil, fmt.Errorf("Could not listen to ICMP packets, error: %s", err.Error())
	}

	if s.isIPv4 {
		if err := conn.IPv4PacketConn().SetTTL(s.settings.TTL); err != nil {
			return nil, fmt.Errorf("Could not set TTL in connection, error: %s", err.Error())
		}
		if err := conn.IPv4PacketConn().SetControlMessage(ipv4.FlagTTL, true); err != nil {
			return nil, fmt.Errorf("Could not set control message in connection, error: %s", err.Error())
		}
	} else {
		if err := conn.IPv6PacketConn().SetHopLimit(s.settings.TTL); err != nil {
			return nil, fmt.Errorf("Could not set control message in connection, error: %s", err.Error())
		}
		if err := conn.IPv6PacketConn().SetControlMessage(ipv6.FlagHopLimit, true); err != nil {
			return nil, fmt.Errorf("Could not set control message in connection, error: %s", err.Error())
		}
	}

	return conn, nil
}
