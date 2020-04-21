package core

import (
	"net"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

// Bundle is an aggregation of ping executions
type Bundle struct {
	id int

	bigID int64

	currentSequence int

	totalSent int

	address net.Addr

	isIPv4 bool
}

// GetICMPType returns the appropriate type to be used in the ICMP request of this bundle
func (b *Bundle) GetICMPType() icmp.Type {
	if b.isIPv4 {
		return ipv4.ICMPTypeEcho
	}

	return ipv6.ICMPTypeEchoRequest
}
