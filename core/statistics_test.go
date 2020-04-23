package core

import (
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestNewStatistics tests if a new statistics struct is properly initialized
func TestNewStatistics(t *testing.T) {
	stats := NewStatistics()

	assert.Zero(t, stats.GetPktLoss())
	assert.Zero(t, stats.GetRTTAvg())
	assert.Zero(t, stats.GetRTTMDev())
	assert.Zero(t, stats.GetRTTMax())
	assert.Zero(t, stats.GetRTTMin())
	assert.Zero(t, stats.GetTotalPending())
	assert.Zero(t, stats.GetTotalRecv())
	assert.Zero(t, stats.GetTotalSent())
	assert.Zero(t, stats.GetTotalTTLExpired())
	assert.Zero(t, stats.GetTotalTimedOut())
}

// TestInitStatsCb tests if the callback used in the start of a session correctly set fields
func TestInitStatsCb(t *testing.T) {
	s, err := NewSession("localhost", DefaultSettings())
	assert.NoError(t, err)
	assert.NotNil(t, s)

	msg := s.buildEchoRequest()

	now := time.Now()
	initStatsCb(s, msg)
	st, started := s.Stats.GetStartTime()
	assert.True(t, started)
	assert.True(t, st.After(now))
}

// TestInitStatsCb tests if the callback used in the start of a session correctly set fields
func TestFinishInitStatsCb(t *testing.T) {
	r := rand.New(rand.NewSource(time.Now().UTC().UnixNano()))
	s, err := NewSession("localhost", DefaultSettings())
	assert.NoError(t, err)
	assert.NotNil(t, s)

	for i := 0; i < 400; i++ {
		s.Stats.EchoRequested()
	}

	sum := uint64(0)
	sqsum := uint64(0)
	mx := uint64(0)
	mn := uint64(math.MaxUint32)
	for i := 0; i < 100; i++ {
		rtt := uint64(r.Uint32())
		sum += rtt
		sqsum += rtt * rtt
		mx = max(mx, rtt)
		mn = min(mn, rtt)
		s.Stats.EchoReplied(rtt)
	}

	loss := 1 - float64(100)/float64(400)

	avg := sum / uint64(100)
	sqrd := float64((sqsum / uint64(100)) - avg*avg)
	mdev := uint64(math.Sqrt(sqrd))
	now := time.Now()

	finishStatsCb(s)

	end, started := s.Stats.GetEndTime()
	assert.True(t, started)
	assert.True(t, end.After(now))
	assert.Equal(t, mn, s.Stats.GetRTTMin())
	assert.Equal(t, mx, s.Stats.GetRTTMax())
	assert.Equal(t, avg, s.Stats.GetRTTAvg())
	assert.Equal(t, mdev, s.Stats.GetRTTMDev())
	assert.Equal(t, loss, s.Stats.GetPktLoss())
}
