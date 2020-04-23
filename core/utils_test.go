package core

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestIsIPv4TrueOnIPv4(t *testing.T) {
	ipv4 := net.IPv4(8, 8, 8, 8)
	assert.True(t, isIPv4(ipv4))
}

func TestIsIPv4TrueOnIPv6ThatCanBeTransformed(t *testing.T) {
	ipv4 := net.IP{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xff, 0xff, 192, 168, 0, 1}
	assert.True(t, isIPv4(ipv4))
}

func TestIsIPv4FalseOnIPv6(t *testing.T) {
	// 2606:4700::6811:af55
	ipv4 := net.IP{0x26, 0x06, 0x47, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x68, 0x11, 0xaf, 0x55}
	assert.False(t, isIPv4(ipv4))
}

func TestClearTimerStopsTimer(t *testing.T) {
	timer := time.NewTimer(time.Second)
	clearTimer(timer)

	assert.False(t, timer.Stop())
}
