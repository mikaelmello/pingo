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

func TestMax(t *testing.T) {
	assert.Equal(t, uint64(5), max(5, 4))
	assert.Equal(t, uint64(5), max(5, 5))
	assert.Equal(t, uint64(6), max(5, 6))
	assert.Equal(t, uint64(2232), max(2232, 333))
	assert.Equal(t, uint64(4125421412), max(4125421412, 0))
}

func TestMin(t *testing.T) {
	assert.Equal(t, uint64(4), min(5, 4))
	assert.Equal(t, uint64(5), min(5, 5))
	assert.Equal(t, uint64(5), min(5, 6))
	assert.Equal(t, uint64(333), min(2232, 333))
	assert.Equal(t, uint64(0), min(4125421412, 0))
}
