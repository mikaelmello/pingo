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
	ttlExceeded               = 0
	icmpProtocol              = 1
	icmpv6Protocol            = 58
	dataLength                = 16
	icmpPrivilegedNetwork     = "ip4:icmp"
	icmpv6PrivilegedNetwork   = "ip6:ipv6-icmp"
	icmpUnprivilegedNetwork   = "udp4"
	icmpv6UnprivilegedNetwork = "udp6"
)

// requestEcho sends an ECHO_REQUEST to the address defined in the Session receiving as a parameter
// the open connection with the target host.
func (s *Session) requestEcho(conn *icmp.PacketConn) error {

	bigID := uint64ToBytes(s.bigID)     // ensure same source
	tstp := unixNanoToBytes(time.Now()) // calculate rtt
	data := append(bigID, tstp...)

	body := &icmp.Echo{
		ID:   s.id,
		Seq:  s.lastSequence + 1, // verify pair of request-replies
		Data: data,
	}

	msg := &icmp.Message{
		Type: s.getICMPType(),
		Code: echoCode,
		Body: body,
	}

	bytesmsg, err := msg.Marshal(nil)

	if err != nil {
		return fmt.Errorf("could not marshal ICMP message with Echo body: %w", err)
	}

	_, err = conn.WriteTo(bytesmsg, s.addr)

	// request failing or not, we must update these values
	s.totalSent++
	s.lastSequence++

	if err != nil {
		return fmt.Errorf("error while sending ECHO_REQUEST: %w", err)
	}

	return nil
}

// pollICMP constantly polls the connection to receive and process any replies.
func (s *Session) pollICMP(wg *sync.WaitGroup, conn *icmp.PacketConn, recv chan<- *rawPacket) {
	defer wg.Done()

	for {
		select {
		case <-s.isFinished:
			// session has finished, end
			return
		default:
			buffer := make([]byte, 1024)
			if err := conn.SetReadDeadline(time.Now().Add(time.Second * 1)); err != nil {
				fmt.Printf("Fatal error here")

				// signal main loop and return
				s.isFinished <- true
				return
			}

			length, ttl, err := s.readFrom(conn, buffer)
			if err != nil {
				if neterr, ok := err.(*net.OpError); ok {
					if neterr.Timeout() {
						continue
					} else {
						// signal main loop and return
						s.isFinished <- true
						return
					}
				}
			}

			// sends the packet to the session so it can be checked and processed
			recv <- &rawPacket{content: buffer, length: length, ttl: ttl}
		}
	}
}

// readFrom is a wrapper meant to read bytes from the connection stream and gather relevant info such as the ttl.
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

// checkRawPacket returns whether the packet matches all requirements to be considered a successful reply.
// It also modifies the Session state by updating it with info from the packet if it is considered a successful reply.
func (s *Session) checkRawPacket(raw *rawPacket) (bool, error) {
	receivedTstp := time.Now()

	m, err := icmp.ParseMessage(s.getProtocol(), raw.content)
	if err != nil {
		return false, fmt.Errorf("error parsing ICMP message: %s", err.Error())
	}

	isEchoReply := m.Code == echoCode && (m.Type == ipv4.ICMPTypeEchoReply || m.Type == ipv6.ICMPTypeEchoReply)
	isTimeExceeded := m.Code == ttlExceeded &&
		(m.Type == ipv4.ICMPTypeTimeExceeded || m.Type == ipv6.ICMPTypeTimeExceeded)

	if !isEchoReply && !isTimeExceeded {
		// Not an echo reply or time exceeded, ignore it
		return false, nil
	}

	// cast body as icmp.Echo
	switch body := m.Body.(type) {
	case *icmp.TimeExceeded:
		println("Time exceeded hmm")
		fmt.Printf("%0b", body.Data)
		return false, nil
	case *icmp.Echo:

		if s.settings.IsPrivileged {
			// Check if reply from same ID
			if body.ID != s.id {
				return false, nil
			}
		}

		if len(body.Data) < dataLength {
			return false, fmt.Errorf("missing data, %d bytes out of %d", len(body.Data), dataLength)
		}

		// retrieve the info we serialized
		bigID := bytesToUint64(body.Data[:8])
		tstp := bytesToUnixNano(body.Data[8:])
		// checks if the body seq matches the seq of the last echo request
		if body.Seq != s.lastSequence {
			return false, nil
		}

		// checks if our unique identifier also matches
		if bigID != s.bigID {
			return false, nil
		}

		rtt := receivedTstp.Sub(tstp).Nanoseconds()
		if rtt > s.maxRtt {
			s.maxRtt = rtt
		}
		s.rtts = append(s.rtts, rtt) // stats purposes

		s.totalRecv++

		return true, nil
	default:
		return false, fmt.Errorf("invalid body type: '%T'", body)
	}
}

// getICMPType returns the appropriate type to be used in the ICMP request of this session.
func (s *Session) getICMPType() icmp.Type {
	if s.isIPv4 {
		return ipv4.ICMPTypeEcho
	}

	return ipv6.ICMPTypeEchoRequest
}

// getNetwork returns the appropriate ICMP network value of the session.
func (s *Session) getNetwork() string {
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

// getProtocol returns the appropriate ICMP protocol value of the session.
func (s *Session) getProtocol() int {
	if s.isIPv4 {
		return icmpProtocol
	}

	return icmpv6Protocol
}

// getConnection returns a connection made to the session's address.
func (s *Session) getConnection() (*icmp.PacketConn, error) {
	conn, err := icmp.ListenPacket(s.getNetwork(), "")

	if err != nil {
		return nil, fmt.Errorf("could not listen to ICMP packets, error: %s", err.Error())
	}

	if s.isIPv4 {
		if err := conn.IPv4PacketConn().SetTTL(s.settings.TTL); err != nil {
			return nil, fmt.Errorf("could not set TTL in connection, error: %s", err.Error())
		}
		if err := conn.IPv4PacketConn().SetControlMessage(ipv4.FlagTTL, true); err != nil {
			return nil, fmt.Errorf("could not set control message in connection, error: %s", err.Error())
		}
	} else {
		if err := conn.IPv6PacketConn().SetHopLimit(s.settings.TTL); err != nil {
			return nil, fmt.Errorf("could not set control message in connection, error: %s", err.Error())
		}
		if err := conn.IPv6PacketConn().SetControlMessage(ipv6.FlagHopLimit, true); err != nil {
			return nil, fmt.Errorf("could not set control message in connection, error: %s", err.Error())
		}
	}

	return conn, nil
}
