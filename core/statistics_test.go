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

	assert.Empty(t, stats.RTTs)
	assert.Zero(t, stats.TotalSent)
	assert.Zero(t, stats.TotalRecv)
	assert.Equal(t, time.Time{}, stats.StTime)
	assert.Equal(t, time.Time{}, stats.EndTime)
}

// TestInitStatsCb tests if the callback used in the start of a session correctly set fields
func TestInitStatsCb(t *testing.T) {
	s, err := NewSession("localhost", DefaultSettings())
	assert.NoError(t, err)
	assert.NotNil(t, s)

	msg := s.buildEchoRequest()

	now := time.Now()
	initStatsCb(s, msg)
	st := s.Stats.StTime
	assert.True(t, st.After(now))
}

// TestInitStatsCb tests if the callback used in the start of a session correctly set fields
func TestFinishInitStatsCb(t *testing.T) {
	r := rand.New(rand.NewSource(time.Now().UTC().UnixNano()))
	s, err := NewSession("localhost", DefaultSettings())
	assert.NoError(t, err)
	assert.NotNil(t, s)

	s.Stats.TotalRecv = 100
	s.Stats.TotalSent = 400
	loss := 1 - float64(s.Stats.TotalRecv)/float64(s.Stats.TotalSent)

	cnt := 100
	sum := int64(0)
	sqsum := int64(0)
	mx := int64(0)
	mn := int64(math.MaxUint32)
	for i := 0; i < cnt; i++ {
		val := int64(r.Uint32())
		sum += val
		sqsum += val * val
		mx = max(mx, val)
		mn = min(mn, val)
		s.Stats.RTTs = append(s.Stats.RTTs, val)
	}

	avg := sum / int64(cnt)
	sqrd := float64((sqsum / int64(cnt)) - avg*avg)
	mdev := int64(math.Sqrt(sqrd))
	now := time.Now()

	finishStatsCb(s)

	assert.True(t, s.Stats.EndTime.After(now))
	assert.Equal(t, mn, s.Stats.RTTsMin)
	assert.Equal(t, mx, s.Stats.RTTsMax)
	assert.Equal(t, avg, s.Stats.RTTsAvg)
	assert.Equal(t, mdev, s.Stats.RTTsMDev)
	assert.Equal(t, loss, s.Stats.PktLoss)
}
