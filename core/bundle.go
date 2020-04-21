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
	ttl              int
	count            int
	interval         time.Duration
	timeout          time.Duration
	deadline         time.Duration
	isDeadlineActive bool
	isPrivileged     bool

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

// NewBundle creates a new Bundle
func NewBundle(addr string) (*Bundle, error) {
	ipaddr, err := net.ResolveIPAddr("ip", addr)
	if err != nil {
		return nil, err
	}

	ipv4 := isIPv4(ipaddr.IP)

	r := rand.New(rand.NewSource(time.Now().UTC().UnixNano()))
	return &Bundle{
		ttl:              64,
		count:            -1,
		interval:         time.Second * 1,
		timeout:          time.Second * 10,
		deadline:         -1,
		isDeadlineActive: false,
		isPrivileged:     false,

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

	interval := time.NewTimer(0)
	defer interval.Stop()

	rawPackets := make(chan *rawPacket, 5)
	defer close(rawPackets)

	var wg sync.WaitGroup
	wg.Add(1)
	go b.pollICMP(&wg, conn, rawPackets)

	for {
		select {
		case <-deadline.C:
			if !b.isDeadlineActive {
				continue
			}
			b.isFinished <- true
			wg.Wait()
			return nil

		case <-timeout.C:
			println("Oops timeout")
			interval.Reset(0)
			continue

		case <-interval.C:
			if b.count > 0 && b.totalSent >= b.count {
				println(time.Now().String(), "Reached max of count")
				clearTimer(interval)
				clearTimer(timeout)
				continue
			}

			// get max rtt * 2
			var duration time.Duration
			if len(b.rtts) > 0 {
				println(time.Now().String(), "Waiting for 2*maxRtt")
				duration = time.Duration(2 * b.maxRtt)
			} else {
				println(time.Now().String(), "Waiting for btimeout")
				duration = time.Duration(b.timeout)
			}
			timeout.Reset(duration)

			println(time.Now().String(), "Sending echo", b.address.String())
			err = b.requestEcho(conn)
			if err != nil {
				println(time.Now().String(), "Echo failed %s", err.Error())
				interval.Reset(b.interval)
				clearTimer(timeout)
				continue
			}

		case raw := <-rawPackets:
			println(time.Now().String(), "Received ICMP")
			match, err := b.checkRawPacket(raw)

			if err != nil || !match {
				if err != nil {
					println(time.Now().String(), "Not matchh or err %s", err.Error())
				} else {
					println(time.Now().String(), "Not match")
				}
				continue
			}

			clearTimer(timeout)
			interval.Reset(b.interval)

		case <-b.isFinished:
			wg.Wait()
			return nil
		}
	}
}
