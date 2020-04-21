package core

import (
	"encoding/binary"
	"net"
	"time"
)

func isIPv4(ip net.IP) bool {
	return ip.To4() != nil
}

func isIPv6(ip net.IP) bool {
	return !isIPv4(ip)
}

func unixNanoToBytes(t time.Time) []byte {
	return int64ToBytes(t.UnixNano())
}

func bytesToUnixNano(b []byte) time.Time {
	unixNano := bytesToInt64(b)
	return time.Unix(0, unixNano)
}

func int64ToBytes(i int64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(i))
	return b
}

func bytesToInt64(b []byte) int64 {
	return int64(binary.BigEndian.Uint64(b))
}

func uint64ToBytes(i uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(i))
	return b
}

func bytesToUint64(b []byte) uint64 {
	return binary.BigEndian.Uint64(b)
}
