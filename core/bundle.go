package core

import (
	"math"
	"math/rand"
	"net"
	"time"
)

// Bundle is an aggregation of ping executions
type Bundle struct {
	id int

	bigID uint64

	currentSequence int

	totalSent int

	address net.Addr

	isIPv4 bool
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
		id:              r.Intn(math.MaxUint16),
		bigID:           r.Uint64(),
		currentSequence: 0,
		totalSent:       0,
		address:         ipaddr,
		isIPv4:          ipv4,
	}, nil
}
