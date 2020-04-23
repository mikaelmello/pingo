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
	// Stats contain the overall statistics of the session
	Stats *Statistics

	settings *Settings

	// id is the session id used in the echo body.
	id int

	// bigID is a larger id used in the payload of the echo body, meant to verify if an echo reply matches a session
	// request with better accuracy.
	bigID uint64

	// lastSequence is the sequence number of the last sent echo request.
	lastSequence int

	// addr contains the net.Addr of the target host
	addr net.Addr

	// string contains the cname (if applicable) of the target host
	cname string

	// isIPv4 contains whether the stored address is IPv4 or not (IPv6)
	isIPv4 bool

	// logger is an instance of logrus used to log activities related to this session
	logger *log.Logger

	// finishRequest is the channel that will signal a request to end the session run.
	finishReqs chan error

	// finished is the channel that will signal the end of the session run.
	finished chan bool

	// isFinished contains whether the session has been finished
	isStarted bool

	// isFinished contains whether the session has been finished
	isFinished bool

	// rtHandlers are the callback functions called when a round trip happens.
	// The function parameters are the session.
	rtHandlers []func(*Session, *RoundTrip)

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

	cname, err := net.LookupCNAME(address)
	if err != nil {
		return nil, fmt.Errorf("error while looking up cname of address %s: %w", address, err)
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
		Stats:        NewStatistics(),
		lastSequence: 0,
		finishReqs:   make(chan error, 1),
		finished:     make(chan bool, 1),
		id:           r.Intn(math.MaxUint16),
		bigID:        r.Uint64(),
		isIPv4:       ipv4,
		addr:         resAddr,
		cname:        cname,
		settings:     settings,
		logger:       logger,
		isStarted:    false,
		isFinished:   false,
	}

	session.AddStHandler(initStatsCb)
	session.AddEndHandler(finishStatsCb)

	logger.Infof("Created session with id %d, bigID %d, ipv4 %t, addr %s",
		session.id, session.bigID, session.isIPv4, session.addr.String())

	return session, nil
}

// Run executes the sequence of pings
func (s *Session) Run() error {
	if s.isFinished {
		return fmt.Errorf("This session has already finished")
	}
	if s.isStarted {
		return fmt.Errorf("This session has already started")
	}
	defer close(s.finishReqs)
	s.isStarted = true

	if !s.settings.IsPrivileged {
		s.logger.Warnf("You are running as non-privileged, meaning that it is not possible to receive TimeExceeded ICMP"+
			" messages. Echo requests that exceed the configured TTL of %d will be treated as timed out", s.settings.TTL)
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

	deadline, timeout, interval := s.initTimers()
	defer deadline.Stop()
	defer timeout.Stop()
	defer interval.Stop()

	// channel that will stream all incoming ICMP packets
	s.logger.Debug("Creating channel of incoming raw packets")
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
			s.handleDeadlineTimer()
		case <-timeout.C:
			s.handleTimeoutTimer(interval)
		case <-interval.C:
			s.handleIntervalTimer(conn, interval, timeout)
		case raw := <-rawPackets:
			s.handleRawPacket(raw, interval, timeout)
		case err := <-s.finishReqs:
			return s.handleFinishRequest(err, &wg)
		}
	}
}

// RequestStop requests the stop the execution of the session
func (s *Session) RequestStop() {
	if s.isFinished {
		return
	}

	s.logger.Info("Requesting to end session")
	s.finishReqs <- nil
}

// IsStarted returns whether this session is started
func (s *Session) IsStarted() bool {
	return s.isStarted
}

// IsFinished returns whether this session is finished
func (s *Session) IsFinished() bool {
	return s.isFinished
}

// Address is the resolved address of the target hostname in this session
func (s *Session) Address() net.Addr {
	return s.addr
}

// CNAME is the CNAME of the input address
func (s *Session) CNAME() string {
	return s.cname
}

// AddRtHandler adds a handler function that will be called after an echo request is replied or expires
func (s *Session) AddRtHandler(handler func(*Session, *RoundTrip)) {
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

// initTimers initializes all timers used to manage the session flow
func (s *Session) initTimers() (deadline *time.Timer, timeout *time.Timer, interval *time.Timer) {
	// timer responsible for shutting down the execution, if enabled
	s.logger.Debugf("Initializing deadline timer to duration %s", s.getDeadlineDuration())
	deadline = time.NewTimer(s.getDeadlineDuration())

	// timer responsible for timing out requests and requiring new ones
	s.logger.Debugf("Initializing timeout timer to duration %s", s.getTimeoutDuration())
	timeout = time.NewTimer(s.getTimeoutDuration())

	// timer responsible for handling the interval between two requests
	s.logger.Debugf("Initializing interval timer to duration %s", time.Duration(0))
	interval = time.NewTimer(0)

	return deadline, timeout, interval
}

// handleTimeoutTimer is responsible for handling when the deadline timer is triggered, checking if the deadline
// option is active and whether we should terminate the session.
func (s *Session) handleDeadlineTimer() {
	s.logger.Info("Deadline timer has fired")

	if !s.isDeadlineActive() {
		s.logger.Info("Ignoring deadline timer because the deadline config has not been activated")
		return
	}

	// deadline is active and triggered, let's end everything
	s.logger.Info("Requesting to finish the session")
	s.finishReqs <- nil
}

// handleTimeoutTimer is responsible for handling when the timeout timer is triggered, timing out the latest request
// and resetting the interval timer.
func (s *Session) handleTimeoutTimer(interval *time.Timer) {

	s.logger.Info("Timeout timer has fired")

	rt := s.buildTimedOutRT()
	s.processRoundTrip(rt)

	if s.reachedRequestLimit() {
		s.logger.Info("Not firing more requests as we have reached the set count")

		s.logger.Info("Requesting to finish the session")
		s.finishReqs <- nil
		return
	}

	s.logger.Infof("Resetting interval timer to trigger a new request in %s", s.getIntervalDuration())

	interval.Reset(s.getIntervalDuration())
}

// handleIntervalTimer is responsible for handling when the interval timer is triggered, sending a new echo request.
func (s *Session) handleIntervalTimer(conn *icmp.PacketConn, interval *time.Timer, timeout *time.Timer) {
	s.logger.Info("Interval timer has fired")

	s.logger.Infof("Resetting timeout timer to account for a timeout (%s) of the reply for the next request",
		s.getTimeoutDuration())

	timeout.Reset(s.getTimeoutDuration())

	err := s.sendEchoRequest(conn)
	if err != nil {
		s.logger.Errorf("Could not send echo request: %s", err)

		// this request already failed, clearing timer and resetting interval
		s.logger.Infof("Stopping timeout timer and resetting interval timer to trigger a new request in %s",
			s.getIntervalDuration())

		interval.Reset(s.getIntervalDuration())
		clearTimer(timeout)
	}
}

// handleRawPacket is responsible for properly handling an incoming raw packet from our connection.
func (s *Session) handleRawPacket(raw *rawPacket, interval *time.Timer, timeout *time.Timer) {

	s.logger.Tracef("Raw packet received: %x", raw.content[:raw.length])

	// checks whether this ICMP is the reply of the last request and process it
	rt, err := s.preProcessRawPacket(raw)

	if err != nil {
		s.logger.Errorf("Could not parse raw packet: %s", err)
		return
	}

	if rt == nil {
		s.logger.Info("Received raw packet was not a match")
		return
	}

	// it is a match, clearing timeout and resetting interval for next request
	s.logger.Infof("Stopping timeout timer and resetting interval timer to trigger a new request in %s",
		s.getIntervalDuration())

	clearTimer(timeout)
	interval.Reset(s.getIntervalDuration())

	s.processRoundTrip(rt)

	// checks if we have to stop somewhere and if we are already there
	if s.reachedRequestLimit() {
		s.logger.Info("Not firing more requests as we have reached the set count")

		s.logger.Info("Requesting to finish the session")
		s.finishReqs <- nil
	}
}

// handleFinishRequest handles where we should finish the session.
func (s *Session) handleFinishRequest(err error, wg *sync.WaitGroup) error {
	if err != nil {
		return err
	}

	s.logger.Info("Finish request received")

	s.finishReqs <- nil // forwarding to polling if it did not come from there
	wg.Wait()           // waiting for polling to return

	s.logger.Info("Calling ending callbacks")
	for _, f := range s.endHandlers {
		f(s)
	}

	s.finished <- true // sending to stop, if it came from there
	s.isFinished = true
	s.logger.Info("Session ended")
	return nil
}

// Returns the deadline setting parsed as a duration in seconds.
func (s *Session) getDeadlineDuration() time.Duration {
	return time.Second * time.Duration(s.settings.Deadline)
}

// Returns the interval setting parsed as a duration in seconds.
func (s *Session) getIntervalDuration() time.Duration {
	return time.Duration(float64(time.Second) * s.settings.Interval)
}

// Returns the appropriate value for the next timeout parsed as a duration in seconds.
func (s *Session) getTimeoutDuration() time.Duration {

	// if we already have successful pings, our timeout is now 2 times
	// the longest registered rtt, as the original ping does
	// otherwise, we use the standard timeout
	if len(s.Stats.RTTs) > 0 {
		return time.Duration(2 * s.Stats.RTTsMax)
	}
	return time.Second * time.Duration(s.settings.Timeout)
}

// Returns whether the deadline setting is active.
func (s *Session) isDeadlineActive() bool {
	return s.settings.Deadline > 0
}

// Returns whether we should stop sending requests some time.
func (s *Session) isMaxCountActive() bool {
	return s.settings.MaxCount > 0
}

// reachedRequestLimit whether we ahave reached the request limit of this session.
func (s *Session) reachedRequestLimit() bool {
	// checks if we have to stop somewhere and if we are already there
	return s.isMaxCountActive() && s.Stats.TotalSent >= s.settings.MaxCount
}

// buildTimedOutRT builds a round trip object containing data relevant to a timed out request.
func (s *Session) buildTimedOutRT() *RoundTrip {
	return &RoundTrip{
		TTL:  0,
		Time: s.getTimeoutDuration(),
		Len:  0,
		Seq:  s.lastSequence,
		Src:  nil,
		Res:  TimedOut,
	}
}

// processRoundTrip calls all handlers for a round trip.
func (s *Session) processRoundTrip(rt *RoundTrip) {

	if rt.Res == Replied {
		rtt := rt.Time.Nanoseconds()
		s.Stats.RTTsMax = max(s.Stats.RTTsMax, rtt)
		s.Stats.RTTs = append(s.Stats.RTTs, rtt) // stats purposes
		s.Stats.TotalRecv++
	}

	s.logger.Info("Calling all handlers for latest round trip")
	for _, f := range s.rtHandlers {
		f(s, rt)
	}
}
