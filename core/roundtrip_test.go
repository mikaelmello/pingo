package core

import (
	"net"
	"time"
)

// buildRoundTrip returns a stub round trip with the desired result
func buildRoundTrip(res RoundTripResult) *RoundTrip {
	return &RoundTrip{
		TTL:  5,
		Seq:  0,
		Len:  24,
		Src:  net.IPv4(127, 0, 0, 1),
		Time: time.Millisecond,
		Res:  res,
	}
}
