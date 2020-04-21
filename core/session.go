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

	id            int
	bigID         uint64
	lastSequence  int
	totalSent     int
	totalReceived int
	maxRtt        int64
	rtts          []int64
	address       *net.IPAddr
	isIPv4        bool
	isFinished    chan bool
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
		lastSequence:  0,
		totalSent:     0,
		totalReceived: 0,
		maxRtt:        0,
		id:            r.Intn(math.MaxUint16),
		bigID:         r.Uint64(),
		isIPv4:        ipv4,
		address:       ipaddr,
		settings:      settings,
	}, nil
}

// Start starts the sequence of pings
func (s *Session) Start() error {
	defer close(s.isFinished)

	conn, err := s.GetConnection()
	if err != nil {
		return err
	}
	defer conn.Close()

	// timer responsible for shutting down the execution, if enabled
	deadline := time.NewTimer(s.getDeadlineDuration())
	defer deadline.Stop()

	// timer responsible for timing out requests and requiring new ones
	timeout := time.NewTimer(s.getTimeoutDuration())
	defer timeout.Stop()

	// timer responsible for handling the interval between two requests
	interval := time.NewTimer(0)
	defer interval.Stop()

	// channel that will stream all incoming ICMP packets
	rawPackets := make(chan *rawPacket, 5)
	defer close(rawPackets)

	// start receiving incoming ICMP packets using a controlgroup to properly exit later
	var wg sync.WaitGroup
	wg.Add(1)
	go s.pollICMP(&wg, conn, rawPackets)

	for {
		select {
		case <-deadline.C:
			if !s.isDeadlineActive() {
				continue
			}

			// deadline is active and triggered, let's end everything
			s.isFinished <- true
			wg.Wait()
			return nil

		case <-timeout.C:
			// timeout, onto the next request
			println("Oops timeout")
			interval.Reset(s.getIntervalDuration())
			continue

		case <-interval.C:
			// checks if we have to stop somewhere and if we are already there
			if s.isMaxCountActive() && s.totalSent >= s.settings.MaxCount {
				println(time.Now().String(), "Reached max of count")
				clearTimer(interval)
				clearTimer(timeout)
				continue
			}

			// if we already have successful pings, our timeout is now 2 times
			// the longest registered rtt, as the original ping does
			// otherwise, we use the standard timeout
			var duration time.Duration
			if len(s.rtts) > 0 {
				duration = time.Duration(2 * s.maxRtt)
			} else {
				duration = time.Duration(s.getTimeoutDuration())
			}
			timeout.Reset(duration)

			println(time.Now().String(), "Sending echo", s.address.String())
			err = s.requestEcho(conn)
			if err != nil {
				println(time.Now().String(), "Echo failed %s", err.Error())
				// this request already failed, clearing timer and resetting interval
				interval.Reset(s.getIntervalDuration())
				clearTimer(timeout)
				continue
			}

		case raw := <-rawPackets:
			println(time.Now().String(), "Received ICMP")
			// checks whether this ICMP is the reply of the last request and process it
			match, err := s.checkRawPacket(raw)

			if err != nil || !match {
				if err != nil {
					println(time.Now().String(), "Not matchh or err %s", err.Error())
				} else {
					println(time.Now().String(), "Not match")
				}
				continue
			}

			// it is a match, clearing timeout and resetting interval for next request
			clearTimer(timeout)
			interval.Reset(s.getIntervalDuration())

		case <-s.isFinished:
			// we have been stopped
			wg.Wait()
			return nil
		}
	}
}

// Stop finishes the execution of the session
func (s *Session) Stop() {
	s.isFinished <- true
}

// Returns the deadline setting parsed as a duration in seconds
func (s *Session) getDeadlineDuration() time.Duration {
	return time.Second * time.Duration(s.settings.Deadline)
}

// Returns the interval setting parsed as a duration in seconds
func (s *Session) getIntervalDuration() time.Duration {
	return time.Second * time.Duration(s.settings.Interval)
}

// Returns the timeout setting parsed as a duration in seconds
func (s *Session) getTimeoutDuration() time.Duration {
	return time.Second * time.Duration(s.settings.Timeout)
}

// Returns whether the deadline setting is active
func (s *Session) isDeadlineActive() bool {
	return s.settings.Deadline > 0
}

// Returns whether we should stop sending requests some time
func (s *Session) isMaxCountActive() bool {
	return s.settings.MaxCount > 0
}
