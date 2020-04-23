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
	Stats Statistics

	settings *Settings

	// id is the session id used in the echo body.
	id int

	// bigID is a larger id used in the payload of the echo body, meant to verify if an echo reply matches a session
	// request with better accuracy.
	bigID uint64

	// lastSeq is the sequence number of the last sent echo request.
	lastSeq int

	// lastSeqMutex is the mutex to make requests
	reqMutex sync.Mutex

	// iaddr contains the original input address
	iaddr string

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

	// statusMutex is responsible synchronizing reads and writes of isStarted and isFinished status
	statusMutex sync.Mutex

	// rMap contains the channels for each seq
	rMap ReplyMap

	// reqW is responsible for synchronizing the hanging requests
	reqW sync.WaitGroup

	// onStart is a list of callback functions called when the session starts.
	// The function parameters are the session and a sample first echo request.
	onStart []func(*Session, *icmp.Message)

	// onSend is a list of callback functions called when an echo request is sent.
	// The function parameters are the session and the message.
	onSend []func(*Session)

	// onRecv is a list of callback functions called when a round trip happens.
	// The function parameters are the session.
	onRecv []func(*Session, *RoundTrip)

	// onFinish is a list of callback functions called when the session ends.
	// The function parameters are the session.
	onFinish []func(*Session)
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

	r := rand.New(rand.NewSource(time.Now().UTC().UnixNano()))

	session := &Session{
		Stats:      NewStatistics(),
		lastSeq:    0,
		finishReqs: make(chan error, 1),
		finished:   make(chan bool, 1),
		id:         r.Intn(math.MaxUint16),
		bigID:      r.Uint64(),
		rMap:       newReplyMap(),
		settings:   settings,
		iaddr:      address,
		logger:     logger,
		isStarted:  false,
		isFinished: false,
	}

	session.AddOnStart(initStatsCb)
	session.AddOnFinish(finishStatsCb)

	logger.Infof("Created session with id %d, bigID %d, iaddr %s",
		session.id, session.bigID, session.iaddr)

	return session, nil
}

// Run executes the sequence of pings
func (s *Session) Run() error {
	if s.IsFinished() {
		return fmt.Errorf("this session has already finished")
	}
	if s.IsStarted() {
		return fmt.Errorf("this session has already started")
	}

	defer close(s.finishReqs)
	s.setIsStarted(true)

	if !s.settings.IsPrivileged {
		s.logger.Warnf("You are running as non-privileged, meaning that it is not possible to receive TimeExceeded ICMP"+
			" messages. Echo requests that exceed the configured TTL of %d will be treated as timed out", s.settings.TTL)
	}

	err := s.resolve()
	if err != nil {
		return err
	}

	s.logger.Info("Calling start callbacks")
	for _, f := range s.onStart {
		f(s, s.buildEchoRequest(0))
	}

	conn, err := s.getConnection()
	if err != nil {
		return err
	}
	defer conn.Close()

	// channel that will stream all incoming ICMP packets
	s.logger.Debug("Creating channel of incoming raw packets")
	rawPackets := make(chan *rawPacket, 5)
	defer close(rawPackets)

	// start receiving incoming ICMP packets using a controlgroup to properly exit later
	s.logger.Info("Calling goroutine to poll for incoming raw packets")
	var wg sync.WaitGroup
	wg.Add(1)
	go s.pollConnection(&wg, conn, rawPackets)

	deadline, interval := s.initTimers()
	defer deadline.Stop()
	defer interval.Stop()

	go s.handleIntervalTimer(conn, interval)

	for {
		select {
		case <-deadline.C:
			s.handleDeadlineTimer()
		case <-interval.C:
			go s.handleIntervalTimer(conn, interval)
		case raw := <-rawPackets:
			s.handleRawPacket(raw)
		case err := <-s.finishReqs:
			return s.handleFinishRequest(err, &wg)
		}
	}
}

// RequestStop requests the stop the execution of the session
func (s *Session) RequestStop() {
	if s.IsFinished() {
		return
	}

	s.logger.Info("Requesting to end session")
	s.finishReqs <- nil
}

// IsStarted returns whether this session is started
func (s *Session) IsStarted() bool {
	s.statusMutex.Lock()
	defer s.statusMutex.Unlock()

	return s.isStarted
}

// IsFinished returns whether this session is finished
func (s *Session) IsFinished() bool {
	s.statusMutex.Lock()
	defer s.statusMutex.Unlock()

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

// AddOnStart adds a handler function that will be called when the session starts
func (s *Session) AddOnStart(handler func(*Session, *icmp.Message)) {
	s.onStart = append(s.onStart, handler)
}

// AddOnSend adds a handler function that will be called after an echo request is replied or expires
func (s *Session) AddOnSend(handler func(*Session)) {
	s.onSend = append(s.onSend, handler)
}

// AddOnRecv adds a handler function that will be called after an echo request is replied or expires
func (s *Session) AddOnRecv(handler func(*Session, *RoundTrip)) {
	s.onRecv = append(s.onRecv, handler)
}

// AddOnFinish adds a handler function that will be called when the session ends
func (s *Session) AddOnFinish(handler func(*Session)) {
	s.onFinish = append(s.onFinish, handler)
}

// resolve resolves the input address setting the session ip address and cname
func (s *Session) resolve() error {
	s.logger.Infof("Resolving address %s", s.iaddr)

	ipaddr, err := net.ResolveIPAddr("ip", s.iaddr)
	if err != nil {
		return fmt.Errorf("error while resolving address %s: %w", s.iaddr, err)
	}

	cname, err := net.LookupCNAME(s.iaddr)
	if err != nil {
		return fmt.Errorf("error while looking up cname of address %s: %w", s.iaddr, err)
	}

	s.logger.Infof("Address %s resolved to IP Address %s", s.iaddr, ipaddr.String())

	var resAddr net.Addr = ipaddr
	if !s.settings.IsPrivileged {
		// The provided dst must be net.UDPAddr when conn is a non-privileged
		// datagram-oriented ICMP endpoint.
		s.logger.Infof("Running as non-privileged, setting address to UDP")
		resAddr = &net.UDPAddr{IP: ipaddr.IP, Zone: ipaddr.Zone}
	}

	ipv4 := isIPv4(ipaddr.IP)
	s.logger.Infof("Resolved IP address is IPv4: %t", ipv4)

	s.cname = cname
	s.isIPv4 = ipv4
	s.addr = resAddr
	return nil
}

// initTimers initializes all timers used to manage the session flow
func (s *Session) initTimers() (deadline *time.Timer, interval *time.Ticker) {
	// timer responsible for shutting down the execution, if enabled
	s.logger.Debugf("Initializing deadline timer to duration %s", s.getDeadlineDuration())
	deadline = time.NewTimer(s.getDeadlineDuration())

	// timer responsible for handling the interval between two requests
	s.logger.Debugf("Initializing interval ticker to duration %s", s.getIntervalDuration())
	interval = time.NewTicker(s.getIntervalDuration())

	return deadline, interval
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

// handleIntervalTimer is responsible for handling when the interval timer is triggered, sending a new echo request.
func (s *Session) handleIntervalTimer(conn *icmp.PacketConn, interval *time.Ticker) {
	s.logger.Trace("Interval ticker has been triggered")

	if s.reachedRequestLimit() {
		s.logger.Trace("Not firing more requests as we have reached the set count")
		interval.Stop()

		s.reqW.Wait()

		s.logger.Info("Requesting to finish the session")
		s.finishReqs <- nil
		return
	}

	s.reqMutex.Lock()

	selectedSeq := s.lastSeq + 1
	s.Stats.EchoRequested()
	s.lastSeq = (s.lastSeq + 1) & 0xffff

	err := s.sendEchoRequest(conn, selectedSeq)
	s.logger.Infof("Incrementing number of packages sent and of last sequence to %d and %d respectively",
		s.Stats.GetTotalSent(), s.lastSeq)

	s.reqMutex.Unlock()

	for _, f := range s.onSend {
		f(s)
	}

	if err != nil {
		s.Stats.EchoRequestError()
		s.logger.Errorf("Could not send echo request: %s", err)
		return
	}

	ch := s.rMap.GetOrCreate(uint16(selectedSeq))
	defer s.rMap.Erase(uint16(selectedSeq))

	s.reqW.Add(1)
	defer s.reqW.Done()

	timeout := s.getTimeoutDuration()
	select {
	case rt := <-ch:
		if rt == nil {
			s.Stats.EchoRequestError()
			break
		}

		go s.processRoundTrip(rt)
	case <-time.After(timeout):
		rt := buildTimedOutRT(selectedSeq, timeout)
		go s.processRoundTrip(rt)
	}
}

// handleRawPacket is responsible for properly handling an incoming raw packet from our connection.
func (s *Session) handleRawPacket(raw *rawPacket) {

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

	ch, ok := s.rMap.Get(uint16(rt.Seq))
	if !ok {
		s.logger.Info("Received raw packet from seq that has already timed out")
		return
	}

	ch <- rt
}

// handleFinishRequest handles where we should finish the session.
func (s *Session) handleFinishRequest(err error, wg *sync.WaitGroup) error {
	if err != nil {
		return err
	}

	s.logger.Info("Finish request received")

	s.finishReqs <- nil // forwarding to polling if it did not come from there
	wg.Wait()           // waiting for polling to return

	s.finished <- true // sending to stop, if it came from there
	s.setIsFinished(true)

	s.logger.Info("Calling ending callbacks")
	for _, f := range s.onFinish {
		f(s)
	}

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
	if s.Stats.GetTotalRecv() > 0 {
		return time.Duration(2 * s.Stats.GetRTTMax())
	}
	return time.Second * time.Duration(s.settings.Timeout)
}

// setIsFinished updates the isFinished property to val
func (s *Session) setIsFinished(val bool) {
	s.statusMutex.Lock()
	defer s.statusMutex.Unlock()

	s.isFinished = val
}

// setIsStarted updates the isStarted property to val
func (s *Session) setIsStarted(val bool) {
	s.statusMutex.Lock()
	defer s.statusMutex.Unlock()

	s.isStarted = val
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
	return s.isMaxCountActive() && int64(s.Stats.GetTotalSent()) >= int64(s.settings.MaxCount)
}

// processRoundTrip calls all handlers for a round trip.
func (s *Session) processRoundTrip(rt *RoundTrip) {
	if s.IsFinished() {
		return
	}

	if rt.Res == Replied {
		rtt := rt.Time.Nanoseconds()
		s.Stats.EchoReplied(uint64(rtt))
	}

	s.logger.Info("Calling all handlers for latest round trip")
	for _, f := range s.onRecv {
		f(s, rt)
	}
}
