package core

import (
	"math"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/net/icmp"
)

// Statistics provides several functions to update and retrieve stats about a session
type Statistics interface {
	SessionStarted()
	SessionEnded()
	EchoRequested()
	EchoReplied(rtt uint64)
	EchoTimedOut()
	EchoTTLExpired()

	GetStartTime() (time.Time, bool)
	GetEndTime() (time.Time, bool)

	GetTotalSent() uint32
	GetTotalRecv() uint32
	GetTotalTimedOut() uint32
	GetTotalTTLExpired() uint32
	GetTotalPending() uint32
	GetPktLoss() float64

	GetRTTMax() uint64
	GetRTTMin() uint64
	GetRTTAvg() uint64
	GetRTTMDev() uint64
}

// statistics aggregate stats about a session
type statistics struct {

	// totalSent is the total amount of echo requests sent in this session.
	totalSent uint32

	// TotalRecv is the total amount of echo replies received in the appropriate time in this session.
	TotalRecv uint32

	// totalTimedOut is the total amount of echo requests that timed out before receiving a reply.
	totalTimedOut uint32

	// totalTTLExpired is the total amount of echo requests that had their TTL expired before reaching the target.
	totalTTLExpired uint32

	// rttsMutex controls the append of a rtt into the array
	rttsMutex sync.RWMutex

	// rtts contains the round-trip times of all successful replies of this session.
	rtts []uint64

	// rttsMin contains the smallest encountered rtt
	rttsMin uint64

	// rttsMax contains the largest encountered rtt
	rttsMax uint64

	// RTTsAvg contains the largest encountered rtt
	rttsSum uint64

	// RTTsMDev contains the largest encountered rtt
	rttsSqSum uint64

	// timeMutex controls updates to the times
	timeMutex sync.RWMutex

	// stTime contains the start time of the session
	stTime time.Time

	// started indicates whether the stTime has been initialized
	started bool

	// endTime contains the start time of the session
	endTime time.Time

	// ended indicates whether the endTime has been initialized
	ended bool
}

func (s *statistics) SessionStarted() {
	s.timeMutex.Lock()
	defer s.timeMutex.Unlock()

	s.stTime = time.Now()
	s.started = true
}

func (s *statistics) SessionEnded() {
	s.timeMutex.Lock()
	defer s.timeMutex.Unlock()

	s.endTime = time.Now()
	s.ended = true
}

func (s *statistics) EchoRequested() {
	atomic.AddUint32(&s.totalSent, 1)
}

func (s *statistics) EchoReplied(rtt uint64) {
	atomic.AddUint32(&s.TotalRecv, 1)

	s.rttsMutex.Lock()
	defer s.rttsMutex.Unlock()

	s.rtts = append(s.rtts, rtt)
	s.rttsMax = max(s.rttsMax, rtt)
	s.rttsMin = min(s.rttsMin, rtt)
	s.rttsSum += rtt
	s.rttsSqSum += rtt * rtt
}

func (s *statistics) EchoTimedOut() {
	atomic.AddUint32(&s.totalTimedOut, 1)
}

func (s *statistics) EchoTTLExpired() {
	atomic.AddUint32(&s.totalTTLExpired, 1)
}

func (s *statistics) GetStartTime() (time.Time, bool) {
	s.timeMutex.RLock()
	defer s.timeMutex.RUnlock()

	return s.stTime, s.started
}

func (s *statistics) GetEndTime() (time.Time, bool) {
	s.timeMutex.RLock()
	defer s.timeMutex.RUnlock()

	return s.endTime, s.ended
}

func (s *statistics) GetTotalSent() uint32 {
	return atomic.LoadUint32(&s.totalSent)
}

func (s *statistics) GetTotalRecv() uint32 {
	return atomic.LoadUint32(&s.TotalRecv)
}

func (s *statistics) GetTotalTimedOut() uint32 {
	return atomic.LoadUint32(&s.totalTimedOut)
}

func (s *statistics) GetTotalTTLExpired() uint32 {
	return atomic.LoadUint32(&s.totalTTLExpired)
}

func (s *statistics) GetTotalPending() uint32 {
	return s.GetTotalSent() - s.GetTotalRecv() - s.GetTotalTimedOut() - s.GetTotalTTLExpired()
}

func (s *statistics) GetPktLoss() float64 {
	if s.GetTotalSent() == 0 {
		return 0
	}

	return float64(1) - (float64(s.GetTotalRecv()) / float64(s.GetTotalSent()))
}

func (s *statistics) GetRTTMax() uint64 {
	s.rttsMutex.RLock()
	defer s.rttsMutex.RUnlock()

	return s.rttsMax
}

func (s *statistics) GetRTTMin() uint64 {
	s.rttsMutex.RLock()
	defer s.rttsMutex.RUnlock()

	return min(s.rttsMax, s.rttsMin)
}

func (s *statistics) GetRTTAvg() uint64 {
	s.rttsMutex.RLock()
	defer s.rttsMutex.RUnlock()

	if len(s.rtts) == 0 {
		return 0
	}

	return s.rttsSum / uint64(len(s.rtts))
}

func (s *statistics) GetRTTMDev() uint64 {
	s.rttsMutex.RLock()
	defer s.rttsMutex.RUnlock()

	if len(s.rtts) == 0 {
		return 0
	}

	sqrd := float64((s.rttsSqSum/uint64(len(s.rtts)) - s.GetRTTAvg()*s.GetRTTAvg()))
	return uint64(math.Sqrt(sqrd))
}

// NewStatistics creates and initializes a Statistics struct.
func NewStatistics() Statistics {
	return &statistics{
		rtts:            []uint64{},
		totalSent:       0,
		TotalRecv:       0,
		totalTimedOut:   0,
		totalTTLExpired: 0,
		rttsMax:         0,
		rttsMin:         math.MaxInt64,
		rttsSum:         0,
		rttsSqSum:       0,
		started:         false,
		ended:           false,
	}
}

// initStatsCb is a callback to be used when a session starts, initializing the start time.
func initStatsCb(s *Session, msg *icmp.Message) {
	s.Stats.SessionStarted()
}

// finishStatsCb is a callback to be used when a session ends, calculating all useful stats.
func finishStatsCb(s *Session) {
	s.Stats.SessionEnded()
}
