package core

import (
	"net"
	"syscall"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

const (
	echoCode = 0
)

func (b *Bundle) requestEcho(conn *icmp.PacketConn) error {

	bigID := int64ToBytes(b.bigID)      // ensure same source
	tstp := unixNanoToBytes(time.Now()) // calculate rtt
	data := append(bigID, tstp...)

	body := &icmp.Echo{
		ID:   b.id,
		Seq:  b.currentSequence,
		Data: data,
	}

	msg := &icmp.Message{
		Type: b.GetICMPType(),
		Code: echoCode,
		Body: body,
	}

	msgBytes, err := msg.Marshal(nil)

	if err != nil {
		return err
	}

	for {
		_, err := conn.WriteTo(msgBytes, b.address)

		if err == nil {
			break
		}

		neterr, ok := err.(*net.OpError)

		if ok && neterr.Err == syscall.ENOBUFS {
			continue
		}

		break
	}

	b.totalSent++
	b.currentSequence++

	return nil
}

// GetICMPType returns the appropriate type to be used in the ICMP request of this bundle
func (b *Bundle) GetICMPType() icmp.Type {
	if b.isIPv4 {
		return ipv4.ICMPTypeEcho
	}

	return ipv6.ICMPTypeEchoRequest
}
