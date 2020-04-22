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

type roundTrip struct {
	ttl  int             // time-to-live, receiving only
	seq  int             // seq of reply, successful or not
	len  int             // len of reply
	src  net.IP          // src of reply
	time time.Duration   // rtt, successful-only
	res  RoundTripResult // result
}
