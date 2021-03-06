package core

import (
	"math"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/net/icmp"
)

// TestNewSession verifies that the variables are correctly initialized
func TestNewSession(t *testing.T) {
	s, err := NewSession("localhost", DefaultSettings())
	assert.NoError(t, err)
	assert.NotNil(t, s)

	assert.Equal(t, 0, s.lastSeq)
	assert.GreaterOrEqual(t, math.MaxUint16, s.id)
	assert.Len(t, s.onStart, 1, "new session does not start with one st handler")
	assert.Len(t, s.onFinish, 1, "new session does not start with one end handler")
	assert.Empty(t, s.onRecv, "new session does not start with empty rt handlers")

	assert.False(t, s.isStarted)
	assert.False(t, s.isFinished)
}

// TestSessionRequestStop verifies that a stop call correctly stops a started session
func TestSessionStop(t *testing.T) {
	s, err := NewSession("localhost", DefaultSettings())
	assert.NoError(t, err)
	assert.NotNil(t, s)

	c1 := make(chan error, 1)

	go func() {
		err := s.Run()
		c1 <- err
	}()

	s.RequestStop()

	// Listen on our channel AND a timeout channel - which ever happens first.
	select {
	case err := <-c1:
		assert.NoError(t, err)
		assert.True(t, s.isStarted)
		assert.True(t, s.isFinished)
	case <-time.After(1 * time.Second):
		t.Error("Stop did not stop the session in time")
	}
}

// TestSessionAddr verifies if the getter is correct
func TestSessionAddr(t *testing.T) {
	s, err := NewSession("localhost", DefaultSettings())
	assert.NoError(t, err)
	assert.NotNil(t, s)

	assert.Equal(t, s.addr, s.Address())
}

// TestSessionCNAME verifies if the getter is correct
func TestSessionCNAME(t *testing.T) {
	s, err := NewSession("localhost", DefaultSettings())
	assert.NoError(t, err)
	assert.NotNil(t, s)

	assert.Equal(t, s.cname, s.CNAME())
}

// TestSessionAddOnRecv verifies that a function is correctly added to the list
func TestSessionAddOnRecv(t *testing.T) {
	s, err := NewSession("localhost", DefaultSettings())
	assert.NoError(t, err)
	assert.NotNil(t, s)

	h := func(*Session, *RoundTrip) {}
	prevlen := len(s.onRecv)

	s.AddOnRecv(h)
	assert.Equal(t, prevlen+1, len(s.onRecv))
}

// TestSessionAddOnStart verifies that a function is correctly added to the list
func TestSessionAddOnStart(t *testing.T) {
	s, err := NewSession("localhost", DefaultSettings())
	assert.NoError(t, err)
	assert.NotNil(t, s)

	h := func(*Session, *icmp.Message) {}
	prevlen := len(s.onStart)

	s.AddOnStart(h)
	assert.Equal(t, prevlen+1, len(s.onStart))
}

// TestSessionAddOnEnd verifies that a function is correctly added to the list
func TestSessionAddOnEnd(t *testing.T) {
	s, err := NewSession("localhost", DefaultSettings())
	assert.NoError(t, err)
	assert.NotNil(t, s)

	h := func(*Session) {}
	prevlen := len(s.onFinish)

	s.AddOnFinish(h)
	assert.Equal(t, prevlen+1, len(s.onFinish))
}

// TODO(how): Implement this test when we refactor the code to use interfaces allowing us to mock
func TestSessionResolve(t *testing.T) {
	s, err := NewSession("localhost", DefaultSettings())
	assert.NoError(t, err)
	assert.NotNil(t, s)

	// s.resolve()
}

// TestSessionInitTimers verifies that all three timers are properly initialized
func TestSessionInitTimers(t *testing.T) {
	s, err := NewSession("localhost", DefaultSettings())
	assert.NoError(t, err)
	assert.NotNil(t, s)

	deadline, interval := s.initTimers()
	assert.NotNil(t, deadline)
	assert.NotNil(t, interval)
}

// TestSessionHandleDeadlineTimer1 verifies the proper behavior
// of the handler when the deadline is not active
func TestSessionHandleDeadlineTimer1(t *testing.T) {
	s, err := NewSession("localhost", DefaultSettings())
	assert.NoError(t, err)
	assert.NotNil(t, s)

	assert.False(t, s.isDeadlineActive())

	s.handleDeadlineTimer()
	assert.Empty(t, s.finishReqs)
}

// TestSessionHandleDeadlineTimer2 verifies the proper behavior
// of the handler when the deadline is active
func TestSessionHandleDeadlineTimer2(t *testing.T) {
	settings := DefaultSettings()
	settings.Deadline = 1
	settings.IsDeadlineDefault = false

	s, err := NewSession("localhost", settings)
	assert.NoError(t, err)
	assert.NotNil(t, s)

	assert.True(t, s.isDeadlineActive())

	s.handleDeadlineTimer()
	assert.NotEmpty(t, s.finishReqs)
}

// TODO(how): Implement this test when we refactor the code to use interfaces allowing us to mock
func TestSessionHandleIntervalTimer(t *testing.T) {
	s, err := NewSession("localhost", DefaultSettings())
	assert.NoError(t, err)
	assert.NotNil(t, s)

	// s.handleIntervalTimer()
}

// TestSessionHandleRawPacket1 verifies the proper behavior
// of the handler when we have not reached the request limit
// and received a proper echo reply
func TestSessionHandleRawPacket1(t *testing.T) {
	s, err := NewSession("localhost", DefaultSettings())
	assert.NoError(t, err)
	assert.NotNil(t, s)

	pkt, err := buildEchoReply(s.id, s.lastSeq, s.bigID, s.isIPv4)
	assert.NoError(t, err)

	ch := s.rMap.GetOrCreate(uint16(s.lastSeq))

	s.handleRawPacket(pkt)
	assert.Empty(t, s.finishReqs)
	assert.NotEmpty(t, ch)
}

// TestSessionHandleRawPacket2 verifies the proper behavior
// of the handler when we have not reached the request limit
// and received a ttl exceed
func TestSessionHandleRawPacket2(t *testing.T) {
	s, err := NewSession("localhost", DefaultSettings())
	assert.NoError(t, err)
	assert.NotNil(t, s)

	pkt, err := buildTimeExceeded(uint16(s.id), uint16(s.lastSeq), s.isIPv4)
	assert.NoError(t, err)

	ch := s.rMap.GetOrCreate(uint16(s.lastSeq))

	s.handleRawPacket(pkt)
	assert.NotEmpty(t, ch)
	assert.Empty(t, s.finishReqs)
}

// TestSessionHandleFinishRequest verifies the proper behavior
// of the handler when we have reached the request limit
// and received a proper reply
func TestSessionHandleFinishRequest(t *testing.T) {
	s, err := NewSession("localhost", DefaultSettings())
	assert.NoError(t, err)
	assert.NotNil(t, s)

	ch := make(chan bool, 1)
	eh := func(s *Session) {
		ch <- true
	}

	s.AddOnFinish(eh)
	var wg sync.WaitGroup
	s.handleFinishRequest(nil, &wg)

	assert.NotEmpty(t, ch)
	assert.NotEmpty(t, s.finished)
	assert.NotEmpty(t, s.finishReqs)
	assert.True(t, s.isFinished)
}

// TestSessionGetDeadlineDuration if the getter for deadline duration is correct
func TestSessionGetDeadlineDuration(t *testing.T) {
	s, err := NewSession("localhost", DefaultSettings())
	assert.NoError(t, err)
	assert.NotNil(t, s)

	expected := time.Second * time.Duration(s.settings.Deadline)
	assert.Equal(t, expected, s.getDeadlineDuration())
}

// TestSessionGetDeadlineDuration2 if the getter for a custom deadline duration is correct
func TestSessionGetDeadlineDuration2(t *testing.T) {
	settings := DefaultSettings()
	settings.Deadline = 5
	settings.IsDeadlineDefault = false
	s, err := NewSession("localhost", settings)
	assert.NoError(t, err)
	assert.NotNil(t, s)

	assert.Equal(t, 5, s.settings.Deadline)
	expected := time.Second * time.Duration(s.settings.Deadline)
	assert.Equal(t, expected, s.getDeadlineDuration())
}

// TestSessionGetIntervalDuration if the getter for interval duration is correct
func TestSessionGetIntervalDuration(t *testing.T) {
	s, err := NewSession("localhost", DefaultSettings())
	assert.NoError(t, err)
	assert.NotNil(t, s)

	expected := time.Duration(float64(time.Second) * s.settings.Interval)
	assert.Equal(t, expected, s.getIntervalDuration())
}

// TestSessionGetIntervalDuration2 if the getter for a custom interval duration is correct
func TestSessionGetIntervalDuration2(t *testing.T) {
	settings := DefaultSettings()
	settings.Interval = 5
	s, err := NewSession("localhost", settings)
	assert.NoError(t, err)
	assert.NotNil(t, s)

	assert.Equal(t, float64(5), s.settings.Interval)
	expected := time.Duration(float64(time.Second) * s.settings.Interval)
	assert.Equal(t, expected, s.getIntervalDuration())
}

// TestSessionIsDeadlineActive1 checks whether it returns correctly with an active deadline
func TestSessionIsDeadlineActive1(t *testing.T) {
	settings := DefaultSettings()
	settings.Deadline = 5
	settings.IsDeadlineDefault = false
	s, err := NewSession("localhost", settings)
	assert.NoError(t, err)
	assert.NotNil(t, s)

	assert.True(t, s.isDeadlineActive())
}

// TestSessionIsDeadlineActive2 checks whether it returns correctly without an active deadline
func TestSessionIsDeadlineActive2(t *testing.T) {
	settings := DefaultSettings()
	settings.Deadline = -1
	settings.IsDeadlineDefault = true
	s, err := NewSession("localhost", settings)
	assert.NoError(t, err)
	assert.NotNil(t, s)

	assert.False(t, s.isDeadlineActive())
}

// TestSessionIsMaxCountActive1 checks whether it returns correctly with an active MaxCount
func TestSessionIsMaxCountActive1(t *testing.T) {
	settings := DefaultSettings()
	settings.MaxCount = 5
	settings.IsMaxCountDefault = false
	s, err := NewSession("localhost", settings)
	assert.NoError(t, err)
	assert.NotNil(t, s)

	assert.True(t, s.isMaxCountActive())
}

// TestSessionIsMaxCountActive2 checks whether it returns correctly without an active MaxCount
func TestSessionIsMaxCountActive2(t *testing.T) {
	settings := DefaultSettings()
	settings.MaxCount = -1
	settings.IsMaxCountDefault = true
	s, err := NewSession("localhost", settings)
	assert.NoError(t, err)
	assert.NotNil(t, s)

	assert.False(t, s.isMaxCountActive())
}

// TestSessionReachedRequestLimit1 verifies if the getter
// returns correctly when there is no max count
func TestSessionReachedRequestLimit1(t *testing.T) {
	settings := DefaultSettings()
	settings.MaxCount = -1
	settings.IsMaxCountDefault = true
	s, err := NewSession("localhost", settings)
	assert.NoError(t, err)
	assert.NotNil(t, s)

	for i := 0; i < 5; i++ {
		s.Stats.EchoRequested()
	}

	assert.False(t, s.reachedRequestLimit())
}

// TestSessionReachedRequestLimit2 verifies if the getter
// returns correctly when there is max count but have not
// reached limit
func TestSessionReachedRequestLimit2(t *testing.T) {
	settings := DefaultSettings()
	settings.MaxCount = 55
	settings.IsMaxCountDefault = false
	s, err := NewSession("localhost", settings)
	assert.NoError(t, err)
	assert.NotNil(t, s)

	for i := 0; i < 5; i++ {
		s.Stats.EchoRequested()
	}

	assert.False(t, s.reachedRequestLimit())
}

// TestSessionReachedRequestLimit3 verifies if the getter
// returns correctly when there is max count and have
// reached limit
func TestSessionReachedRequestLimit3(t *testing.T) {
	settings := DefaultSettings()
	settings.MaxCount = 5
	settings.IsMaxCountDefault = false
	s, err := NewSession("localhost", settings)
	assert.NoError(t, err)
	assert.NotNil(t, s)

	for i := 0; i < 5; i++ {
		s.Stats.EchoRequested()
	}

	assert.True(t, s.reachedRequestLimit())
}

// TestSessionProcessRoundTrip1 verifies that the function
// correctly processes a roundtrip when it is a successful reply
func TestSessionProcessRoundTrip1(t *testing.T) {
	s, err := NewSession("localhost", DefaultSettings())
	assert.NoError(t, err)
	assert.NotNil(t, s)

	ch := make(chan bool, 1)
	rth := func(s *Session, rt *RoundTrip) {
		assert.Equal(t, Replied, rt.Res)
		ch <- true
	}
	s.AddOnRecv(rth)

	prevlen := s.Stats.GetTotalRecv()

	rt := buildRoundTrip(Replied)
	s.processRoundTrip(rt)

	assert.Equal(t, prevlen+1, s.Stats.GetTotalRecv())
}

// TestSessionProcessRoundTrip2 verifies that the function
// correctly processes a roundtrip when it is a time out
func TestSessionProcessRoundTrip2(t *testing.T) {
	s, err := NewSession("localhost", DefaultSettings())
	assert.NoError(t, err)
	assert.NotNil(t, s)

	ch := make(chan bool, 1)
	rth := func(s *Session, rt *RoundTrip) {
		assert.Equal(t, TimedOut, rt.Res)
		ch <- true
	}
	s.AddOnRecv(rth)

	prevlen := s.Stats.GetTotalRecv()
	prevtout := s.Stats.GetTotalTimedOut()

	rt := buildRoundTrip(TimedOut)
	s.processRoundTrip(rt)

	assert.Equal(t, prevlen, s.Stats.GetTotalRecv())
	assert.Equal(t, prevtout, s.Stats.GetTotalTimedOut())
}

// TestSessionProcessRoundTrip3 verifies that the function
// correctly processes a roundtrip when it is a TTL expired
func TestSessionProcessRoundTrip3(t *testing.T) {
	s, err := NewSession("localhost", DefaultSettings())
	assert.NoError(t, err)
	assert.NotNil(t, s)

	ch := make(chan bool, 1)
	rth := func(s *Session, rt *RoundTrip) {
		assert.Equal(t, TTLExpired, rt.Res)
		ch <- true
	}
	s.AddOnRecv(rth)

	prevlen := s.Stats.GetTotalRecv()
	prevttl := s.Stats.GetTotalTTLExpired()

	rt := buildRoundTrip(TTLExpired)
	s.processRoundTrip(rt)

	assert.Equal(t, prevlen, s.Stats.GetTotalRecv())
	assert.Equal(t, prevttl, s.Stats.GetTotalTTLExpired())
}
