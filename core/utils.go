package core

import (
	"encoding/binary"
	"net"
	"time"
)

// isIPv4 returns whether the ip is a valid IPv4 address.
func isIPv4(ip net.IP) bool {
	return ip.To4() != nil
}

// unixNanoToBytes converts the UnixNano representation of a time to an array of 8 bytes.
func unixNanoToBytes(t time.Time) []byte {
	return int64ToBytes(t.UnixNano())
}

// bytesToUnixNano converts an array of 8 bytes to an Int64 that will then be parsed as UnixNano.
func bytesToUnixNano(b []byte) time.Time {
	unixNano := bytesToInt64(b)
	return time.Unix(0, unixNano)
}

// int64ToBytes converts an int64 to an array of 8 bytes.
func int64ToBytes(i int64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(i))
	return b
}

// bytesToInt64 converts an array of bytes to int64.
func bytesToInt64(b []byte) int64 {
	return int64(binary.BigEndian.Uint64(b))
}

// uint16ToBytes converts an uint16 to an array of 2 bytes.
func uint16ToBytes(i uint16) []byte {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, i)
	return b
}

// bytesToUint16 converts an array of bytes to int16.
func bytesToUint16(b []byte) uint16 {
	return binary.BigEndian.Uint16(b)
}

// uint64ToBytes converts an uint64 to an array of 8 bytes.
func uint64ToBytes(i uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, i)
	return b
}

// bytesToUint64 converts an array of bytes to uint64.
func bytesToUint64(b []byte) uint64 {
	return binary.BigEndian.Uint64(b)
}

// clearTimer stops a timer and ensures that it is empty after the stop.
func clearTimer(t *time.Timer) {
	if !t.Stop() {
		<-t.C
	}
}

// max returns the max number between a and b
func max(a int64, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

// min returns the min number between a and b
func min(a int64, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
