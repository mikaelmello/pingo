package core

import (
	"fmt"
	"math"
	"math/rand"
	"net"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/net/icmp"
)

// Session is an aggregation of ping executions
type Session struct {
	settings *Settings

	// id is the session id used in the echo body.
	id int

	// bigID is a larger id used in the payload of the echo body, meant to verify if an echo reply matches a session
	// request with better accuracy.
	bigID uint64

	// lastSequence is the sequence number of the last sent echo request.
	lastSequence int

	// totalSent is the total amount of echo requests sent in this session.
	totalSent int

	// totalRecv is the total amount of matching echo replies received in the appropriate time in this session.
	totalRecv int

	// maxRtt is the largest round-trip time among all successful replies of this session.
	maxRtt int64

	// rtts contains the round-trip times of all successful replies of this session.
	rtts []int64

	// addr contains the net.Addr of the target host
	addr net.Addr

	// isIPv4 contains whether the stored address is IPv4 or not (IPv6)
	isIPv4 bool

	// logger is an instance of logrus used to log activities related to this session
	logger *log.Logger

	// finishRequest is the channel that will signal a request to end the session run.
	finishReqs chan bool

	// finished is the channel that will signal the end of the session run.
	finished chan bool

	// rtHandlers are the callback functions called when a round trip happens.
	// The function parameters are the session.
	rtHandlers []func(*RoundTrip)

	// stHandlers are the callback functions called when the session starts.
	// The function parameters are the session and a sample first echo request.
	stHandlers []func(*Session, *icmp.Message)

	// endHandlers are the callback functions called when the session ends.
	// The function parameters are the session.
	endHandlers []func(*Session)
}

// NewSession creates a new Session
func NewSession(address string, settings *Settings) (*Session, error) {
	logger := NewLogger(settings.LoggingLevel)

	logger.Debug("Validating settings")

	err := settings.validate()
	if err != nil {
		return nil, fmt.Errorf("invalid settings: %w", err)
	}

	logger.Debug("Settings configured correctly")

	logger.Infof("Resolving address %s", address)

	ipaddr, err := net.ResolveIPAddr("ip", address)
	if err != nil {
		return nil, fmt.Errorf("error while resolving address %s: %w", address, err)
	}

	logger.Infof("Address %s resolved to IP Address %s", address, ipaddr.String())

	var resAddr net.Addr = ipaddr
	if !settings.IsPrivileged {
		// The provided dst must be net.UDPAddr when conn is a non-privileged
		// datagram-oriented ICMP endpoint.
		logger.Infof("Running as non-privileged, setting address to UDP")
		resAddr = &net.UDPAddr{IP: ipaddr.IP, Zone: ipaddr.Zone}
	}

	ipv4 := isIPv4(ipaddr.IP)
	logger.Infof("Resolved IP address is IPv4: %t", ipv4)

	r := rand.New(rand.NewSource(time.Now().UTC().UnixNano()))

	session := &Session{
		lastSequence: 0,
		totalSent:    0,
		totalRecv:    0,
		maxRtt:       0,
		finishReqs:   make(chan bool, 1),
		finished:     make(chan bool, 1),
		id:           r.Intn(math.MaxUint16),
		bigID:        r.Uint64(),
		isIPv4:       ipv4,
		addr:         resAddr,
		settings:     settings,
		logger:       logger,
	}

	logger.Infof("Created session with id %d, bigID %d, ipv4 %t, addr %s",
		session.id, session.bigID, session.isIPv4, session.addr.String())

	return session, nil
}

// Start starts the sequence of pings
func (s *Session) Start() error {
	defer close(s.finishReqs)

	if !s.settings.IsPrivileged {
		s.logger.Warnf("You are running as non-privileged, meaning that it is not possible to receive TimeExceeded ICMP"+
			" requests. Requests that exceed the configured TTL of %d will be treated as timed out", s.settings.TTL)
	}

	s.logger.Info("Calling start callbacks")
	for _, f := range s.stHandlers {
		f(s, s.buildEchoRequest())
	}

	conn, err := s.getConnection()
	if err != nil {
		return err
	}
	defer conn.Close()

	// timer responsible for shutting down the execution, if enabled
	s.logger.Debugf("Initializing deadline timer to duration %s", s.getDeadlineDuration())
	deadline := time.NewTimer(s.getDeadlineDuration())
	defer deadline.Stop()

	// timer responsible for timing out requests and requiring new ones
	s.logger.Debugf("Initializing timeout timer to duration %s", s.getTimeoutDuration())
	timeout := time.NewTimer(s.getTimeoutDuration())
	defer timeout.Stop()

	// timer responsible for handling the interval between two requests
	s.logger.Debugf("Initializing interval timer to duration %s", time.Duration(0))
	interval := time.NewTimer(0)
	defer interval.Stop()

	// channel that will stream all incoming ICMP packets
	s.logger.Debug("Creating chhannel of incoming raw packets")
	rawPackets := make(chan *rawPacket, 5)
	defer close(rawPackets)

	// start receiving incoming ICMP packets using a controlgroup to properly exit later
	s.logger.Info("Calling goroutine to poll for incoming raw packets")
	var wg sync.WaitGroup
	wg.Add(1)
	go s.pollConnection(&wg, conn, rawPackets)

	for {
		select {
		case <-deadline.C:
			s.logger.Info("Deadline timer has fired")

			if !s.isDeadlineActive() {
				s.logger.Info("Ignoring deadline timer because the deadline config has not been activated")
				continue
			}

			// deadline is active and triggered, let's end everything
			s.logger.Info("Requesting to finish the session")
			s.finishReqs <- true
		case <-timeout.C:
			s.logger.Info("Timeout timer has fired")

			rt := s.buildTimedOutRT()
			s.processRoundTrip(rt)

			if s.reachedRequestLimit() {
				s.logger.Info("Not firing more requests as we have reached the set count")

				s.logger.Info("Requesting to finish the session")
				s.finishReqs <- true
				continue
			}

			s.logger.Infof("Resetting interval timer to trigger a new request in %s", s.getIntervalDuration())

			interval.Reset(s.getIntervalDuration())
			continue

		case <-interval.C:
			s.logger.Info("Interval timer has fired")

			s.logger.Infof("Resetting timeout timer to account for a timeout of the reply for the next request",
				s.getTimeoutDuration())

			timeout.Reset(s.getTimeoutDuration())

			err = s.sendEchoRequest(conn)
			if err != nil {
				s.logger.Errorf("Could not send echo request: %w", err)

				// this request already failed, clearing timer and resetting interval
				s.logger.Infof("Stopping timeout timer and resetting interval timer to trigger a new request in %s",
					s.getIntervalDuration())

				interval.Reset(s.getIntervalDuration())
				clearTimer(timeout)
				continue
			}

		case raw := <-rawPackets:
			s.logger.Tracef("Raw packet received: %x", raw.content[:raw.length])

			// checks whether this ICMP is the reply of the last request and process it
			rt, err := s.preProcessRawPacket(raw)

			if err != nil {
				s.logger.Errorf("Could not parse raw packet: %w", err)
				continue
			}

			if rt == nil {
				s.logger.Info("Received raw packet was not a match")
				continue
			}

			s.processRoundTrip(rt)

			// checks if we have to stop somewhere and if we are already there
			if s.reachedRequestLimit() {
				s.logger.Info("Not firing more requests as we have reached the set count")

				s.logger.Info("Requesting to finish the session")
				s.finishReqs <- true
			}

			// it is a match, clearing timeout and resetting interval for next request
			s.logger.Infof("Stopping timeout timer and resetting interval timer to trigger a new request in %s",
				s.getIntervalDuration())

			clearTimer(timeout)
			interval.Reset(s.getIntervalDuration())

		case <-s.finishReqs:
			s.logger.Info("Finish request received")

			s.finishReqs <- true // forwarding to polling if it did not come from there
			wg.Wait()            // waiting for polling to return

			s.logger.Info("Calling ending callbacks")
			for _, f := range s.endHandlers {
				f(s)
			}

			s.finished <- true // sending to stop, if it came from there
			s.logger.Info("Session ended")
			return nil
		}
	}
}

// Stop finishes the execution of the session
func (s *Session) Stop() {
	s.logger.Info("Requesting to end session")
	s.finishReqs <- true
	<-s.finished
}

// AddRtHandler adds a handler function that will be called after an echo request is replied or expires
func (s *Session) AddRtHandler(handler func(*RoundTrip)) {
	s.rtHandlers = append(s.rtHandlers, handler)
}

// AddStHandler adds a handler function that will be called when the session starts
func (s *Session) AddStHandler(handler func(*Session, *icmp.Message)) {
	s.stHandlers = append(s.stHandlers, handler)
}

// AddEndHandler adds a handler function that will be called when the session ends
func (s *Session) AddEndHandler(handler func(*Session)) {
	s.endHandlers = append(s.endHandlers, handler)
}

// Returns the deadline setting parsed as a duration in seconds
func (s *Session) getDeadlineDuration() time.Duration {
	return time.Second * time.Duration(s.settings.Deadline)
}

// Returns the interval setting parsed as a duration in seconds
func (s *Session) getIntervalDuration() time.Duration {
	return time.Duration(float64(time.Second) * s.settings.Interval)
}

// Returns the appropriate value for the next timeout parsed as a duration in seconds
func (s *Session) getTimeoutDuration() time.Duration {

	// if we already have successful pings, our timeout is now 2 times
	// the longest registered rtt, as the original ping does
	// otherwise, we use the standard timeout
	if len(s.rtts) > 0 {
		return time.Duration(2 * s.maxRtt)
	}
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

// reachedRequestLimit whethher we ahave reached the request limit of this session
func (s *Session) reachedRequestLimit() bool {
	// checks if we have to stop somewhere and if we are already there
	return s.isMaxCountActive() && s.totalSent >= s.settings.MaxCount
}

// buildTimedOutRT builds a round trip object containing data relevant to a timed out request
func (s *Session) buildTimedOutRT() *RoundTrip {
	return &RoundTrip{
		TTL:  0,
		Src:  nil,
		Time: s.getTimeoutDuration(),
		Len:  0,
		Seq:  s.lastSequence,
		Res:  TimedOut,
	}
}

// processRoundTrip calls all handlers for a round trip
func (s *Session) processRoundTrip(rt *RoundTrip) {
	for _, f := range s.rtHandlers {
		f(rt)
	}
}
