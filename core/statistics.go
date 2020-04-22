package core

import (
	"math"
	"time"

	"golang.org/x/net/icmp"
)

// Statistics aggregate stats about a session
type Statistics struct {

	// TotalSent is the total amount of echo requests sent in this session.
	TotalSent int

	// TotalRecv is the total amount of matching echo replies received in the appropriate time in this session.
	TotalRecv int

	// PktLoss represents the percentage of packages lost in the session
	PktLoss float64

	// RTTs contains the round-trip times of all successful replies of this session.
	RTTs []int64

	// RTTsMin contains the smallest encountered rtt
	RTTsMin int64

	// RTTsMax contains the largest encountered rtt
	RTTsMax int64

	// RTTsAvg contains the largest encountered rtt
	RTTsAvg int64

	// RTTsMDev contains the largest encountered rtt
	RTTsMDev int64

	// StTime contains the start time of the session
	StTime time.Time

	// EndTime contains the start time of the session
	EndTime time.Time
}

// NewStatistics creates and initializes a Statistics struct.
func NewStatistics() *Statistics {
	return &Statistics{
		RTTs:      []int64{},
		TotalSent: 0,
		TotalRecv: 0,
	}
}

// initStatsCb is a callback to be used when a session starts, initializing the start time.
func initStatsCb(s *Session, msg *icmp.Message) {
	s.Stats.StTime = time.Now()
}

// finishStatsCb is a callback to be used when a session ends, calculating all useful stats.
func finishStatsCb(s *Session) {
	s.Stats.EndTime = time.Now()

	rttsCnt := int64(len(s.Stats.RTTs))

	if rttsCnt == 0 {
		return
	}

	rttsSum := int64(0)
	rttsSqSum := int64(0)
	s.Stats.RTTsMax = s.Stats.RTTs[0]
	s.Stats.RTTsMin = s.Stats.RTTs[0]
	for _, rtt := range s.Stats.RTTs {
		rttsSum += rtt
		rttsSqSum += rtt * rtt

		if rtt > s.Stats.RTTsMax {
			s.Stats.RTTsMax = rtt
		}
		if rtt < s.Stats.RTTsMin {
			s.Stats.RTTsMin = rtt
		}
	}

	s.Stats.RTTsAvg = rttsSum / rttsCnt

	// https://sourceforge.net/p/iputils/code/ci/HEAD/tree/ping_common.c#l1043
	sqrd := float64((rttsSqSum / rttsCnt) - s.Stats.RTTsAvg*s.Stats.RTTsAvg)
	s.Stats.RTTsMDev = int64(math.Sqrt(sqrd))
}
