package core

import (
	"net"
	"time"
)

// RoundTripResult is the end ressult of a round trip
type RoundTripResult int

const (
	// Replied is the result of when an echo request is successfully replied
	Replied RoundTripResult = iota
	// TTLExpired is the result of when an echo request exceeds the TTL
	TTLExpired
	// TimedOut is the result of when an echo request does not receive a reply in an expected time
	TimedOut
)

// RoundTrip represents an echo request and its counterpart reply (or absence of it)
type RoundTrip struct {
	TTL  int             // time-to-live, receiving only
	Seq  int             // seq of reply, successful or not
	Len  int             // len of reply
	Src  net.IP          // src address
	Time time.Duration   // rtt, successful-only
	Res  RoundTripResult // result
}
