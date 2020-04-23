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
	SessionStarted()        // SessionStarted is supposed to be called when the parent session has started
	SessionEnded()          // SessionEnded is supposed to be called when the parent session has ended
	EchoRequested()         // EchoRequested is supposed to be called when a new echo request is sent
	EchoReplied(rtt uint64) // EchoReplied is supposed to be called when a new echo reply has been received
	EchoTimedOut()          // EchoTimedOut is supposed to be called when an echo request timed out
	EchoTTLExpired()        // EchoTTLExpired is supposed to be called when an Time Exceeded ICMP message is received
	EchoRequestError()      // EchoRequestError is supposed to be called when an echo request returns an error

	GetStartTime() (time.Time, bool) // GetStartTime returns the start time and whether it has been initialized
	GetEndTime() (time.Time, bool)   // GetEndTime returns the end time and whether it has been initialized

	GetTotalSent() uint32       // GetTotalSent returns the total number of sent echo requests
	GetTotalRecv() uint32       // GetTotalRecv returns the total number of received echo replies
	GetTotalTimedOut() uint32   // GetTotalTimedOut returns the total number of timed out echo requests
	GetTotalTTLExpired() uint32 // GetTotalTTLExpired returns the total number of echo requests with ttl expired
	GetTotalErrors() uint32     // GetTotalErrors returns the total number of echo requests that returned an error
	GetTotalPending() uint32    // GetTotalPending returns the total number of pending echo requests
	GetPktLoss() float64        // GetPktLoss returns the packet loss rate

	GetRTTMax() uint64  // GetRTTMax returns the max RTT among the ones received via EchoReplied(rtt uint64)
	GetRTTMin() uint64  // GetRTTMin returns the min RTT among the ones received via EchoReplied(rtt uint64)
	GetRTTAvg() uint64  // GetRTTAvg returns the average among the RTTs received via EchoReplied(rtt uint64)
	GetRTTMDev() uint64 // GetRTTMDev returns the mdev among the ones received via EchoReplied(rtt uint64)
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

	// tiotalError is the total amount of echo requests that returned an error.
	totalError uint32

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

// SessionStarted is supposed to be called when the parent session has started
func (s *statistics) SessionStarted() {
	s.timeMutex.Lock()
	defer s.timeMutex.Unlock()

	s.stTime = time.Now()
	s.started = true
}

// SessionEnded is supposed to be called when the parent session has ended
func (s *statistics) SessionEnded() {
	s.timeMutex.Lock()
	defer s.timeMutex.Unlock()

	s.endTime = time.Now()
	s.ended = true
}

// EchoRequested is supposed to be called when a new echo request is sent
func (s *statistics) EchoRequested() {
	atomic.AddUint32(&s.totalSent, 1)
}

// EchoReplied is supposed to be called when a new echo reply has been received
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

// EchoTimedOut is supposed to be called when an echo request timed out
func (s *statistics) EchoTimedOut() {
	atomic.AddUint32(&s.totalTimedOut, 1)
}

// EchoTTLExpired is supposed to be called when an Time Exceeded ICMP message is received
func (s *statistics) EchoTTLExpired() {
	atomic.AddUint32(&s.totalTTLExpired, 1)
}

// // EchoRequestError is supposed to be called when an echo request returns an error
func (s *statistics) EchoRequestError() {
	atomic.AddUint32(&s.totalError, 1)
}

// GetStartTime returns the start time and whether it has been initialized
func (s *statistics) GetStartTime() (time.Time, bool) {
	s.timeMutex.RLock()
	defer s.timeMutex.RUnlock()

	return s.stTime, s.started
}

// GetEndTime returns the end time and whether it has been initialized
func (s *statistics) GetEndTime() (time.Time, bool) {
	s.timeMutex.RLock()
	defer s.timeMutex.RUnlock()

	return s.endTime, s.ended
}

// GetTotalSent returns the total number of sent echo requests
func (s *statistics) GetTotalSent() uint32 {
	return atomic.LoadUint32(&s.totalSent)
}

// GetTotalRecv returns the total number of received echo replies
func (s *statistics) GetTotalRecv() uint32 {
	return atomic.LoadUint32(&s.TotalRecv)
}

// GetTotalTimedOut returns the total number of timed out echo requests
func (s *statistics) GetTotalTimedOut() uint32 {
	return atomic.LoadUint32(&s.totalTimedOut)
}

// GetTotalTTLExpired returns the total number of echo requests with ttl expired
func (s *statistics) GetTotalTTLExpired() uint32 {
	return atomic.LoadUint32(&s.totalTTLExpired)
}

// GetTotalErrors returns the total number of echo requests that returned an error
func (s *statistics) GetTotalErrors() uint32 {
	return atomic.LoadUint32(&s.totalError)
}

// GetTotalPending returns the total number of pending echo requests
func (s *statistics) GetTotalPending() uint32 {
	return s.GetTotalSent() - s.GetTotalRecv() - s.GetTotalTimedOut() - s.GetTotalTTLExpired() - s.GetTotalErrors()
}

// GetPktLoss returns the packet loss rate
func (s *statistics) GetPktLoss() float64 {
	if s.GetTotalSent() == 0 {
		return 0
	}

	return float64(1) - (float64(s.GetTotalRecv()) / float64(s.GetTotalSent()))
}

// GetRTTMax returns the max RTT among the ones received via EchoReplied(rtt uint64)
func (s *statistics) GetRTTMax() uint64 {
	s.rttsMutex.RLock()
	defer s.rttsMutex.RUnlock()

	return s.rttsMax
}

// GetRTTMin returns the min RTT among the ones received via EchoReplied(rtt uint64)
func (s *statistics) GetRTTMin() uint64 {
	s.rttsMutex.RLock()
	defer s.rttsMutex.RUnlock()

	return min(s.rttsMax, s.rttsMin)
}

// GetRTTAvg returns the average among the RTTs received via EchoReplied(rtt uint64)
func (s *statistics) GetRTTAvg() uint64 {
	s.rttsMutex.RLock()
	defer s.rttsMutex.RUnlock()

	if len(s.rtts) == 0 {
		return 0
	}

	return s.rttsSum / uint64(len(s.rtts))
}

// GetRTTMDev returns the mdev among the ones received via EchoReplied(rtt uint64)
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
		totalError:      0,
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
