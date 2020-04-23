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

// sendEchoRequest sends an echo request to the address defined in the Session receiving as a parameter
// the open connection with the target host.
func (s *Session) sendEchoRequest(conn *icmp.PacketConn) (*icmp.Message, error) {
	s.logger.Infof("Making a new echo request to address %s", s.addr.String())

	msg := s.buildEchoRequest()
	bytesmsg, err := msg.Marshal(nil)
	if err != nil {
		return msg, fmt.Errorf("could not marshal ICMP message with Echo body: %w", err)
	}

	s.logger.Infof("Writing ICMP message %x to address %s", bytesmsg, s.addr.String())
	_, err = conn.WriteTo(bytesmsg, s.addr)

	// request failing or not, we must update these values
	s.Stats.EchoRequested()
	s.lastSequence = (s.lastSequence + 1) & 0xffff
	s.logger.Infof("Incrementing number of packages sent and of last sequence to %d and %d respectively",
		s.Stats.GetTotalSent(), s.lastSequence)

	if err != nil {
		return msg, fmt.Errorf("error while sending echo request: %w", err)
	}

	return msg, nil
}

// Builds the next ICMP package, does not modify session's state.
func (s *Session) buildEchoRequest() *icmp.Message {
	s.logger.Tracef("Building new echo request")

	now := time.Now()
	bigID := uint64ToBytes(s.bigID) // ensure same source
	tstp := unixNanoToBytes(now)    // calculate rtt
	data := append(bigID, tstp...)

	body := &icmp.Echo{
		ID:   s.id,
		Seq:  s.lastSequence + 1, // verify pair of request-replies
		Data: data,
	}

	s.logger.Tracef("Body id %d, seq %d, bigID %d, tstp %s", s.id, s.lastSequence+1, s.bigID, now)
	s.logger.Tracef("Body data %x", data)

	msg := &icmp.Message{
		Type: s.getICMPTypeEcho(),
		Code: echoCode,
		Body: body,
	}
	s.logger.Tracef("ICMP message with type %s and code %d", s.getICMPTypeEcho(), echoCode)

	return msg
}

// pollConnection constantly polls the connection to receive and process any replies.
func (s *Session) pollConnection(wg *sync.WaitGroup, conn *icmp.PacketConn, recv chan<- *rawPacket) {
	defer wg.Done()

	// here we are sure that we will never consume a finishReqs produced by us, as we always return
	// after producing
	for {
		select {
		case <-s.finishReqs:
			s.logger.Info("Received request to finish, ending and forwarding")
			return
		default:
			buffer := make([]byte, 256)

			maxwait := time.Millisecond * 200

			s.logger.Tracef("Setting read deadline to %s", maxwait)
			if err := conn.SetReadDeadline(time.Now().Add(maxwait)); err != nil {
				s.finishReqs <- fmt.Errorf("error while setting read deadline, finishing polling and session: %w", err)
				return
			}

			s.logger.Trace("Reading from connection")
			length, cm, err := s.readFrom(conn, buffer)
			if err != nil {
				if neterr, ok := err.(*net.OpError); ok {
					if neterr.Timeout() {
						s.logger.Trace("Read deadline has expired, trying again")
						continue
					} else {
						// request to finish
						s.finishReqs <- fmt.Errorf("error while reading from connection, finishing polling and session: %s", err)
						return
					}
				}
			}

			// sends the packet to the session so it can be checked and processed
			s.logger.Infof("Sending raw packet %x with ttl %d to main session loop", buffer[:length], cm.TTL)
			recv <- &rawPacket{content: buffer, length: length, cm: cm}
		}
	}
}

// readFrom is a wrapper meant to read bytes from the connection stream and gather relevant info such as the ttl.
func (s *Session) readFrom(conn *icmp.PacketConn, bytes []byte) (int, *controlMessage, error) {
	var length int
	var cm *controlMessage
	var err error
	if s.isIPv4 {
		var cmv4 *ipv4.ControlMessage
		length, cmv4, _, err = conn.IPv4PacketConn().ReadFrom(bytes)
		if cmv4 != nil {
			cm = &controlMessage{
				TTL: cmv4.TTL,
				Src: cmv4.Src,
				Dst: cmv4.Dst,
			}
		}
	} else {
		var cmv6 *ipv6.ControlMessage
		length, cmv6, _, err = conn.IPv6PacketConn().ReadFrom(bytes)
		if cmv6 != nil {
			cm = &controlMessage{
				TTL: cmv6.HopLimit,
				Src: cmv6.Src,
				Dst: cmv6.Dst,
			}
		}
	}

	return length, cm, err
}

// checkRawPacket returns whether the packet matches all requirements to be considered a successful reply.
// It also modifies the Session state by updating it with info from the packet if it is considered a successful reply.
func (s *Session) preProcessRawPacket(raw *rawPacket) (*RoundTrip, error) {
	receivedTstp := time.Now()

	s.logger.Infof("Parsing raw packet %x as an ICMP message using protocol %d",
		raw.content[:raw.length], s.getProtocol())
	m, err := icmp.ParseMessage(s.getProtocol(), raw.content)
	if err != nil {
		return nil, fmt.Errorf("error parsing ICMP message: %s", err.Error())
	}

	isEchoReply := m.Code == echoCode && (m.Type == ipv4.ICMPTypeEchoReply || m.Type == ipv6.ICMPTypeEchoReply)
	isTimeExceeded := m.Code == ttlExceeded &&
		(m.Type == ipv4.ICMPTypeTimeExceeded || m.Type == ipv6.ICMPTypeTimeExceeded)

	if !isEchoReply && !isTimeExceeded {
		// Not an echo reply or time exceeded, ignore it
		s.logger.Debugf("Received message that is not an echo reply or time exceeded, code %d and type %d", m.Code, m.Type)
		return nil, nil
	}

	// cast body as icmp.Echo
	switch body := m.Body.(type) {
	case *icmp.TimeExceeded:
		s.logger.Info("Received a TimeExceeded message")

		var origdgram []byte
		if s.isIPv4 {
			if len(body.Data) < 28 {
				return nil, fmt.Errorf("received TimeExceeded does not minimum length that we need."+
					" %d bytes received of min %d", 28, len(body.Data))
			}
			origdgram = body.Data[20:28]
		} else {
			if len(body.Data) < 48 {
				return nil, fmt.Errorf("received TimeExceeded does not minimum length that we need."+
					" %d bytes received of min %d", 48, len(body.Data))
			}
			origdgram = body.Data[40:48]
		}

		echoBody := &icmp.Echo{
			ID:  int(bytesToUint16(origdgram[4:6])),
			Seq: int(bytesToUint16(origdgram[6:])),
		}

		// Check if TLE came from same ID
		if echoBody.ID != s.id {
			s.logger.Debugf("TimeExceeded message does not match session, parsed id differs. Expected: %d."+
				" Actual: %d", s.id, echoBody.ID)
			return nil, nil
		}
		s.logger.Debugf("TimeExceeded message echoBody ID matches session ID.")

		rt := &RoundTrip{
			TTL:  raw.cm.TTL,
			Src:  raw.cm.Src,
			Len:  raw.length,
			Seq:  echoBody.Seq,
			Res:  TTLExpired,
			Time: time.Duration(0),
		}

		return rt, nil
	case *icmp.Echo:
		s.logger.Debug("Received an echo reply message")

		if s.settings.IsPrivileged {
			// Check if reply from same ID
			if body.ID != s.id {
				s.logger.Debugf("Echo reply body ID does not match session ID. Expected: %d. Actual: %d.", s.id, body.ID)
				return nil, nil
			}
			s.logger.Debugf("Echo reply body ID matches session ID. Expected: %d.", s.id)
		}

		if len(body.Data) < dataLength {
			return nil, fmt.Errorf("missing data, %d bytes received of min %d", len(body.Data), dataLength)
		}

		// retrieve the info we serialized
		bigID := bytesToUint64(body.Data[:8])
		tstp := bytesToUnixNano(body.Data[8:])

		// checks if our unique identifier also matches
		if bigID != s.bigID {
			s.logger.Debugf("Echo reply body data big ID does not match session big ID. Expected: %d. Actual: %d.",
				s.bigID, bigID)
			return nil, nil
		}
		s.logger.Debugf("Echo reply body data bigID matches session big ID. Expected: %d.", s.bigID)

		rttduration := receivedTstp.Sub(tstp)

		rt := &RoundTrip{
			TTL:  raw.cm.TTL,
			Src:  raw.cm.Src,
			Len:  raw.length,
			Seq:  body.Seq,
			Res:  Replied,
			Time: rttduration,
		}

		return rt, nil
	default:
		return nil, fmt.Errorf("invalid body type: '%T'", body)
	}
}

// getICMPType returns the appropriate type to be used in the ICMP request of this session.
func (s *Session) getICMPTypeEcho() icmp.Type {
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
	s.logger.Infof("Starting to listen to packets in network %s", s.getNetwork())
	conn, err := icmp.ListenPacket(s.getNetwork(), "")
	if err != nil {
		return nil, fmt.Errorf("could not listen to ICMP packets, error: %s", err.Error())
	}
	s.logger.Debug("Connection successfully created")

	if s.isIPv4 {
		s.logger.Info("Setting TTL and control message to receive TTL")
		if err := conn.IPv4PacketConn().SetTTL(s.settings.TTL); err != nil {
			return nil, fmt.Errorf("could not set TTL in connection, error: %s", err.Error())
		}
		if err := conn.IPv4PacketConn().SetControlMessage(ipv4.FlagTTL, true); err != nil {
			return nil, fmt.Errorf("could not set control message in connection, error: %s", err.Error())
		}
	} else {
		s.logger.Info("Setting TTL and control message to receive TTL")
		if err := conn.IPv6PacketConn().SetHopLimit(s.settings.TTL); err != nil {
			return nil, fmt.Errorf("could not set control message in connection, error: %s", err.Error())
		}
		if err := conn.IPv6PacketConn().SetControlMessage(ipv6.FlagHopLimit, true); err != nil {
			return nil, fmt.Errorf("could not set control message in connection, error: %s", err.Error())
		}
	}

	s.logger.Debug("Connection to listen to packets successfully created and configured")

	return conn, nil
}
