package core

import (
	"math"
	"math/rand"
	"net"
	"sync"
	"time"
)

// Bundle is an aggregation of ping executions
type Bundle struct {
	ttl            int
	count          int
	interval       time.Duration
	timeout        time.Duration
	deadline       time.Duration
	deadlineActive bool

	id              int
	bigID           uint64
	currentSequence int
	totalSent       int
	totalReceived   int
	maxRtt          int64
	rtts            []int64
	address         net.Addr
	isIPv4          bool
	finished        chan bool
}

// NewBundle creates a new Bundle
func NewBundle(addr string) (*Bundle, error) {
	ipaddr, err := net.ResolveIPAddr("ip", addr)
	if err != nil {
		return nil, err
	}

	ipv4 := isIPv4(ipaddr.IP)

	r := rand.New(rand.NewSource(time.Now().UTC().UnixNano()))
	return &Bundle{
		ttl:            64,
		count:          -1,
		interval:       time.Second * 1,
		timeout:        time.Second * 10,
		deadline:       -1,
		deadlineActive: false,

		id:              r.Intn(math.MaxUint16),
		bigID:           r.Uint64(),
		currentSequence: 0,
		totalSent:       0,
		totalReceived:   0,
		maxRtt:          0,
		address:         ipaddr,
		isIPv4:          ipv4,
	}, nil
}

// Start starts the sequence of pings
func (b *Bundle) Start() error {
	conn, err := b.GetConnection()
	if err != nil {
		return err
	}
	defer conn.Close()

	deadline := time.NewTimer(b.deadline)
	defer deadline.Stop()

	timeout := time.NewTimer(b.timeout)
	defer timeout.Stop()

	interval := time.NewTimer(b.interval)
	defer interval.Stop()

	rawPackets := make(chan *rawPacket, 5)
	defer close(rawPackets)

	var wg sync.WaitGroup
	wg.Add(1)
	go b.pollICMP(&wg, conn, rawPackets)

	for {
		select {
		case <-deadline.C:
			if !b.deadlineActive {
				continue
			}
			return nil
		case <-timeout.C:
			interval.Reset(0)
			continue
		case <-interval.C:
			err = b.requestEcho(conn)
			if err != nil {
				interval.Reset(b.interval)
				continue
			}

			// get max rtt * 2
			duration := time.Duration(2 * b.maxRtt)
			timeout.Reset(duration)
		case raw := <-rawPackets:
			match, err := b.checkRawPacket(raw)

			if err != nil || !match {
				continue
			}

			timeout.Stop()
			interval.Reset(b.interval)
		}
	}
}
