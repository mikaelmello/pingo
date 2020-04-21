package core

import (
	"net"
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
