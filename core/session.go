package core

import (
	"math"
	"math/rand"
	"net"
	"sync"
	"time"
)

// Session is an aggregation of ping executions
type Session struct {
	settings *Settings

	id              int
	bigID           uint64
	currentSequence int
	totalSent       int
	totalReceived   int
	maxRtt          int64
	rtts            []int64
	address         *net.IPAddr
	isIPv4          bool
	isFinished      chan bool
}

// NewSession creates a new Session
func NewSession(addr string, settings *Settings) (*Session, error) {
	ipaddr, err := net.ResolveIPAddr("ip", addr)
	if err != nil {
		return nil, err
	}

	ipv4 := isIPv4(ipaddr.IP)

	r := rand.New(rand.NewSource(time.Now().UTC().UnixNano()))
	return &Session{
		currentSequence: 0,
		totalSent:       0,
		totalReceived:   0,
		maxRtt:          0,
		id:              r.Intn(math.MaxUint16),
		bigID:           r.Uint64(),
		isIPv4:          ipv4,
		address:         ipaddr,
		settings:        settings,
	}, nil
}

// Start starts the sequence of pings
func (s *Session) Start() error {
	conn, err := s.GetConnection()
	if err != nil {
		return err
	}
	defer conn.Close()

	deadline := time.NewTimer(s.getDeadlineDuration())
	defer deadline.Stop()

	timeout := time.NewTimer(s.getTimeoutDuration())
	defer timeout.Stop()

	interval := time.NewTimer(0)
	defer interval.Stop()

	rawPackets := make(chan *rawPacket, 5)
	defer close(rawPackets)

	var wg sync.WaitGroup
	wg.Add(1)
	go s.pollICMP(&wg, conn, rawPackets)

	for {
		select {
		case <-deadline.C:
			if !s.isDeadlineActive() {
				continue
			}
			s.isFinished <- true
			wg.Wait()
			return nil

		case <-timeout.C:
			println("Oops timeout")
			interval.Reset(0)
			continue

		case <-interval.C:
			if s.settings.MaxCount > 0 && s.totalSent >= s.settings.MaxCount {
				println(time.Now().String(), "Reached max of count")
				clearTimer(interval)
				clearTimer(timeout)
				continue
			}

			// get max rtt * 2
			var duration time.Duration
			if len(s.rtts) > 0 {
				println(time.Now().String(), "Waiting for 2*maxRtt")
				duration = time.Duration(2 * s.maxRtt)
			} else {
				println(time.Now().String(), "Waiting for btimeout")
				duration = time.Duration(s.getTimeoutDuration())
			}
			timeout.Reset(duration)

			println(time.Now().String(), "Sending echo", s.address.String())
			err = s.requestEcho(conn)
			if err != nil {
				println(time.Now().String(), "Echo failed %s", err.Error())
				interval.Reset(s.getIntervalDuration())
				clearTimer(timeout)
				continue
			}

		case raw := <-rawPackets:
			println(time.Now().String(), "Received ICMP")
			match, err := s.checkRawPacket(raw)

			if err != nil || !match {
				if err != nil {
					println(time.Now().String(), "Not matchh or err %s", err.Error())
				} else {
					println(time.Now().String(), "Not match")
				}
				continue
			}

			clearTimer(timeout)
			interval.Reset(s.getIntervalDuration())

		case <-s.isFinished:
			wg.Wait()
			return nil
		}
	}
}

func (s *Session) getDeadlineDuration() time.Duration {
	return time.Second * time.Duration(s.settings.Deadline)
}

func (s *Session) getIntervalDuration() time.Duration {
	return time.Second * time.Duration(s.settings.Interval)
}

func (s *Session) getTimeoutDuration() time.Duration {
	return time.Second * time.Duration(s.settings.Timeout)
}

func (s *Session) isDeadlineActive() bool {
	return s.settings.Deadline > 0
}
