package core

import (
	"net"
)

// Raw packet read from the connection and used to pass information to the session.
type rawPacket struct {
	content []byte
	length  int
	cm      *controlMessage
}

// controlMessage contains relevant info from the incoming ICMP message
type controlMessage struct {
	TTL int    // time-to-live, receiving only
	Src net.IP // source address, specifying only
	Dst net.IP // destination address, receiving only
}
