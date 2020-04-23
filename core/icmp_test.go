package core

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

// TODO(how): Implement this test when we refactor the code to use interfaces allowing us to mock
func TestSessionSendEchoRequest(t *testing.T) {
	s, err := NewSession("localhost", DefaultSettings())
	assert.NoError(t, err)
	assert.NotNil(t, s)

	// s.sendEchoRequest()
}

// TestSessionBuildEchoRequest verifies if the echo requests
// build contain correct data
func TestSessionBuildEchoRequest(t *testing.T) {
	s, err := NewSession("localhost", DefaultSettings())
	assert.NoError(t, err)
	assert.NotNil(t, s)

	msg := s.buildEchoRequest(s.lastSeq)

	assert.Equal(t, s.getICMPTypeEcho(), msg.Type)
	assert.Equal(t, echoCode, msg.Code)

	switch body := msg.Body.(type) {
	case *icmp.Echo:
		assert.Equal(t, s.id, body.ID)
		assert.Equal(t, s.lastSeq+1, body.Seq)

		// retrieve the info we serialized
		bigID := bytesToUint64(body.Data[:8])
		tstp := bytesToUnixNano(body.Data[8:])
		assert.Equal(t, s.bigID, bigID)
		assert.NotEqual(t, time.Time{}, tstp)
		assert.True(t, time.Now().After(tstp))
	default:
		assert.Fail(t, "body of echo request message is not of echo type")
	}
}

// TODO(how): Implement this test when we refactor the code to use interfaces allowing us to mock
func TestSessionPollConnection(t *testing.T) {
	s, err := NewSession("localhost", DefaultSettings())
	assert.NoError(t, err)
	assert.NotNil(t, s)

	// s.pollConnection()
}

// TODO(how): Implement this test when we refactor the code to use interfaces allowing us to mock
func TestSessionReadFrom(t *testing.T) {
	s, err := NewSession("localhost", DefaultSettings())
	assert.NoError(t, err)
	assert.NotNil(t, s)

	// s.readFrom()
}

// TestSessionPreProcessRawPacket1 verifies if an echo reply
// packet is properly recognized and parsed using all fields,
// including id in privileged mode
func TestSessionPreProcessRawPacket1(t *testing.T) {
	s, err := NewSession("localhost", DefaultSettings())
	assert.NoError(t, err)
	assert.NotNil(t, s)
	s.settings.IsPrivileged = true

	pkt, err := buildEchoReply(s.id, s.lastSeq, s.bigID, s.isIPv4)
	assert.NoError(t, err)

	rt, err := s.preProcessRawPacket(pkt)
	assert.NoError(t, err)

	assert.Equal(t, rt.Time, rt.Time)
	assert.Equal(t, pkt.cm.Src, rt.Src)
	assert.Equal(t, pkt.cm.TTL, rt.TTL)
	assert.Equal(t, pkt.length, rt.Len)
	assert.Equal(t, s.lastSeq, rt.Seq)
	assert.Equal(t, Replied, rt.Res)
}

// TestSessionPreProcessRawPacket2 verifies if an ICMP Time
// Exceeded (TTL) packet is properly recognized and parsed
func TestSessionPreProcessRawPacket2(t *testing.T) {
	s, err := NewSession("localhost", DefaultSettings())
	assert.NoError(t, err)
	assert.NotNil(t, s)

	id := uint16(s.id)
	seq := uint16(s.lastSeq)
	pkt, err := buildTimeExceeded(id, seq, s.isIPv4)
	assert.NoError(t, err)

	rt, err := s.preProcessRawPacket(pkt)
	assert.NoError(t, err)

	assert.Equal(t, rt.Time, rt.Time)
	assert.Equal(t, pkt.cm.Src, rt.Src)
	assert.Equal(t, pkt.cm.TTL, rt.TTL)
	assert.Equal(t, pkt.length, rt.Len)
	assert.Equal(t, s.lastSeq, rt.Seq)
	assert.Equal(t, TTLExpired, rt.Res)
}

// TestSessionPreProcessRawPacket3 verifies if a broken
// echo returns err
func TestSessionPreProcessRawPacket3(t *testing.T) {
	s, err := NewSession("localhost", DefaultSettings())
	assert.NoError(t, err)
	assert.NotNil(t, s)

	pkt, err := buildBrokenEchoReply(s.isIPv4)
	assert.NoError(t, err)

	rt, err := s.preProcessRawPacket(pkt)
	assert.Error(t, err)
	assert.Nil(t, rt)
}

// TestSessionPreProcessRawPacket4 verifies if an ICMP
// packet of a type that does not matter is ignored
func TestSessionPreProcessRawPacket4(t *testing.T) {
	s, err := NewSession("localhost", DefaultSettings())
	assert.NoError(t, err)
	assert.NotNil(t, s)

	pkt, err := buildParameterProblem(s.isIPv4)
	assert.NoError(t, err)

	rt, err := s.preProcessRawPacket(pkt)
	assert.NoError(t, err)
	assert.Nil(t, rt)
}

// TestSessionPreProcessRawPacket5 verifies if an echo reply
// packet with wrong bigID is ignored
func TestSessionPreProcessRawPacket5(t *testing.T) {
	s, err := NewSession("localhost", DefaultSettings())
	assert.NoError(t, err)
	assert.NotNil(t, s)

	pkt, err := buildEchoReply(s.id, s.lastSeq, s.bigID+1, s.isIPv4)
	assert.NoError(t, err)

	rt, err := s.preProcessRawPacket(pkt)
	assert.NoError(t, err)
	assert.Nil(t, rt)
}

// TestSessionPreProcessRawPacket6 verifies if an echo reply
// packet with wrong id is ignored
func TestSessionPreProcessRawPacket6(t *testing.T) {
	s, err := NewSession("localhost", DefaultSettings())
	assert.NoError(t, err)
	assert.NotNil(t, s)
	s.settings.IsPrivileged = true

	pkt, err := buildEchoReply(s.id+1, s.lastSeq, s.bigID, s.isIPv4)
	assert.NoError(t, err)

	rt, err := s.preProcessRawPacket(pkt)
	assert.NoError(t, err)
	assert.Nil(t, rt)
}

// TestSessionPreProcessRawPacket7 verifies if an ICMP Time
// Exceeded (TTL) packet with wrong id is ignored
func TestSessionPreProcessRawPacket7(t *testing.T) {
	s, err := NewSession("localhost", DefaultSettings())
	assert.NoError(t, err)
	assert.NotNil(t, s)
	// no need to set privileged mode since if this packet
	// is received, privileged mode is assumed

	id := uint16(s.id)
	seq := uint16(s.lastSeq)
	pkt, err := buildTimeExceeded(id+1, seq, s.isIPv4)
	assert.NoError(t, err)

	rt, err := s.preProcessRawPacket(pkt)
	assert.NoError(t, err)
	assert.Nil(t, rt)
}

// TestSessionGetICMPTypeEchoIPv4 tests whether session.getICMPType()
// returns the correct ICMP Protocol when the resolved IP is v4
func TestSessionGetICMPTypeEchoIPv4(t *testing.T) {
	s, err := NewSession("localhost", DefaultSettings())
	assert.NoError(t, err)
	assert.NotNil(t, s)

	s.isIPv4 = true
	assert.Equal(t, ipv4.ICMPTypeEcho, s.getICMPTypeEcho())
}

// TestSessionGetICMPTypeEchoIPv6 tests whether session.getICMPType()
// returns the correct ICMP Protocol when the resolved IP is v6
func TestSessionGetICMPTypeEchoIPv6(t *testing.T) {
	s, err := NewSession("localhost", DefaultSettings())
	assert.NoError(t, err)
	assert.NotNil(t, s)

	s.isIPv4 = false
	assert.Equal(t, ipv6.ICMPTypeEchoRequest, s.getICMPTypeEcho())
}

// TestSessionGetNetworkIPv4Privileged tests whether session.getNetwork()
// returns the correct network when the resolved IP is v4 and we are running
// under privileged mode
func TestSessionGetNetworkIPv4Privileged(t *testing.T) {
	s, err := NewSession("localhost", DefaultSettings())
	assert.NoError(t, err)
	assert.NotNil(t, s)

	s.isIPv4 = true
	s.settings.IsPrivileged = true
	assert.Equal(t, icmpPrivilegedNetwork, s.getNetwork())
}

// TestSessionGetNetworkIPv4Unprivileged tests whether session.getNetwork()
// returns the correct network when the resolved IP is v4 and we are not
// running under privileged mode
func TestSessionGetNetworkIPv4Unprivileged(t *testing.T) {
	s, err := NewSession("localhost", DefaultSettings())
	assert.NoError(t, err)
	assert.NotNil(t, s)

	s.isIPv4 = true
	s.settings.IsPrivileged = false
	assert.Equal(t, icmpUnprivilegedNetwork, s.getNetwork())
}

// TestSessionGetNetworkIPv6Privileged tests whether session.getNetwork()
// returns the correct network when the resolved IP is v6 and we are running
// under privileged mode
func TestSessionGetNetworkIPv6Privileged(t *testing.T) {
	s, err := NewSession("localhost", DefaultSettings())
	assert.NoError(t, err)
	assert.NotNil(t, s)

	s.isIPv4 = false
	s.settings.IsPrivileged = true
	assert.Equal(t, icmpv6PrivilegedNetwork, s.getNetwork())
}

// TestSessionGetNetworkIPv6Unprivileged tests whether session.getNetwork()
// returns the correct network when the resolved IP is v6 and we are not
// running under privileged mode
func TestSessionGetNetworkIPv6Unprivileged(t *testing.T) {
	s, err := NewSession("localhost", DefaultSettings())
	assert.NoError(t, err)
	assert.NotNil(t, s)

	s.isIPv4 = false
	s.settings.IsPrivileged = false
	assert.Equal(t, icmpv6UnprivilegedNetwork, s.getNetwork())
}

// TestSessionGetProtocolIPv4 tests whether session.getProtocol()
// returns the correct ICMP Protocol when the resolved IP is v4
func TestSessionGetProtocolIPv4(t *testing.T) {
	s, err := NewSession("localhost", DefaultSettings())
	assert.NoError(t, err)
	assert.NotNil(t, s)

	s.isIPv4 = true
	assert.Equal(t, icmpProtocol, s.getProtocol())
}

// TestSessionGetProtocolIPv6 tests whether session.getProtocol()
// returns the correct ICMP Protocol when the resolved IP is v6
func TestSessionGetProtocolIPv6(t *testing.T) {
	s, err := NewSession("localhost", DefaultSettings())
	assert.NoError(t, err)
	assert.NotNil(t, s)

	s.isIPv4 = false
	assert.Equal(t, icmpv6Protocol, s.getProtocol())
}

// buildEchoReply builds a stub echo reply
func buildEchoReply(id int, seq int, bigID uint64, isIPv4 bool) (pkt *rawPacket, err error) {
	now := time.Now()
	bigIDb := uint64ToBytes(bigID) // ensure same source
	tstp := unixNanoToBytes(now)   // calculate rtt
	data := append(bigIDb, tstp...)
	body := &icmp.Echo{
		ID:   id,
		Seq:  seq,
		Data: data,
	}

	var tp icmp.Type = ipv4.ICMPTypeEchoReply
	if !isIPv4 {
		tp = ipv6.ICMPTypeEchoReply
	}

	msg := &icmp.Message{
		Type: tp,
		Code: echoCode,
		Body: body,
	}

	bytes, err := msg.Marshal(nil)
	if err != nil {
		return nil, err
	}

	ttl := 5
	src := net.IPv4(127, 0, 0, 1)
	return &rawPacket{
		content: bytes,
		length:  len(bytes),
		cm: &controlMessage{
			TTL: ttl,
			Src: src,
		},
	}, nil
}

// buildTimeExceeded builds a stub icmp time exceeded (ttl)
func buildTimeExceeded(id uint16, seq uint16, isIPv4 bool) (*rawPacket, error) {
	padlen := 24
	if !isIPv4 {
		padlen = 44
	}
	pad := make([]byte, padlen)
	idb := uint16ToBytes(id)   // ensure same source
	seqb := uint16ToBytes(seq) // calculate rtt
	data := append(append(pad, idb...), seqb...)
	body := &icmp.TimeExceeded{
		Data: data,
	}

	var tp icmp.Type = ipv4.ICMPTypeTimeExceeded
	if !isIPv4 {
		tp = ipv6.ICMPTypeTimeExceeded
	}

	msg := &icmp.Message{
		Type: tp,
		Code: 0,
		Body: body,
	}
	bytes, err := msg.Marshal(nil)
	if err != nil {
		return nil, err
	}

	ttl := 5
	ip := net.IPv4(127, 0, 0, 1)
	return &rawPacket{
		content: bytes,
		length:  len(bytes),
		cm: &controlMessage{
			TTL: ttl,
			Src: ip,
		},
	}, nil
}

// build builds a stub echo reply that is broken
// and will cause the parser to fail
func buildParameterProblem(isIPv4 bool) (pkt *rawPacket, err error) {
	var tp icmp.Type = ipv4.ICMPTypeParameterProblem
	if !isIPv4 {
		tp = ipv6.ICMPTypeParameterProblem
	}

	body := &icmp.ParamProb{
		Pointer: 0,
		Data:    []byte{0xff, 0xff, 0xff, 0xff},
	}
	msg := &icmp.Message{
		Type: tp,
		Code: echoCode,
		Body: body,
	}

	bytes, err := msg.Marshal(nil)
	if err != nil {
		return nil, err
	}

	ttl := 5
	src := net.IPv4(127, 0, 0, 1)
	return &rawPacket{
		content: bytes,
		length:  len(bytes),
		cm: &controlMessage{
			TTL: ttl,
			Src: src,
		},
	}, nil
}

// buildBrokenEchoReply builds a stub echo reply that is broken
// and will cause the parser to fail
func buildBrokenEchoReply(isIPv4 bool) (pkt *rawPacket, err error) {
	var tp icmp.Type = ipv4.ICMPTypeEchoReply
	if !isIPv4 {
		tp = ipv6.ICMPTypeEchoReply
	}

	body := &icmp.RawBody{
		Data: []byte{0xff, 0xff, 0xff},
	}
	msg := &icmp.Message{
		Type: tp,
		Code: echoCode,
		Body: body,
	}

	bytes, err := msg.Marshal(nil)
	if err != nil {
		return nil, err
	}

	ttl := 5
	src := net.IPv4(127, 0, 0, 1)
	return &rawPacket{
		content: bytes,
		length:  len(bytes),
		cm: &controlMessage{
			TTL: ttl,
			Src: src,
		},
	}, nil
}
