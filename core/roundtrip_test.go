package core

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestRTBuildTimedOut tests whether the TimedOut RT is properly built
func TestRTBuildTimedOut(t *testing.T) {
	s, err := NewSession("localhost", DefaultSettings())
	assert.NoError(t, err)
	assert.NotNil(t, s)

	rt := buildTimedOutRT(s.lastSeq, s.getTimeoutDuration())

	assert.Equal(t, TimedOut, rt.Res)
	assert.Equal(t, s.lastSeq, rt.Seq)
	assert.Equal(t, s.getTimeoutDuration(), rt.Time)
	assert.Equal(t, 0, rt.Len)
	assert.Equal(t, 0, rt.TTL)
	assert.Nil(t, rt.Src)
}

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
