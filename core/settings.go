package core

import (
	"fmt"
	"time"
)

// Settings contains all configurable properties of a ping session.
type Settings struct {
	// TTL is the set IP Time to Live
	TTL int

	// IsTTLSet contains whether the TTL setting is the default
	IsTTLDefault bool

	// MaxCount is the max amount of ECHO_REQUEST packets sent before exiting.
	MaxCount int

	// IsMaxCountDefault contains whether the MaxCount setting is the default
	IsMaxCountDefault bool

	// Interval is the interval in seconds between a receival and the next send of an ECHO_REQUEST.
	Interval float64

	// Timeout is the time in seconds to wait for a response.
	// The option affects only timeout in absence of any responses, otherwise ping waits for two RTTs.
	Timeout int

	// Deadline is the time in seconds before ping exits regardless of how many packets have been sent or received.
	Deadline int

	// IsDeadlineDefault contains whether the Deadline setting is the default
	IsDeadlineDefault bool

	// IsPrivileged defines if privileged (raw ICMP sockets) or unprivileged (datagram-oriented) mode is used.
	IsPrivileged bool

	// LoggingLevel defines the level of the session logger.
	LoggingLevel uint32

	// Flood defines whether we should treat as Flood
	Flood bool
}

// DefaultSettings returns the default settings for a ping session, change as you wish.
func DefaultSettings() *Settings {
	return &Settings{
		TTL:          64,
		IsTTLDefault: true,

		MaxCount:          -1,
		IsMaxCountDefault: true,

		Deadline:          -1,
		IsDeadlineDefault: true,

		Timeout:      10,
		Interval:     1,
		IsPrivileged: false,
		LoggingLevel: 0,
		Flood:        false,
	}
}

func (s *Settings) validate() error {
	if s.TTL <= 0 {
		return fmt.Errorf("TTL must be a positive integer")
	}

	if !s.IsMaxCountDefault && s.MaxCount <= 0 {
		return fmt.Errorf("count must be a positive integer")
	}

	if !s.IsDeadlineDefault && s.Deadline <= 0 {
		return fmt.Errorf("deadline must be a positive integer")
	}

	if s.Timeout <= 0 {
		return fmt.Errorf("timeout must be a positive integer")
	}

	if s.Flood && !s.IsPrivileged {
		return fmt.Errorf("non-privileged mode can not use flood option")
	}

	if s.Flood && s.IsPrivileged {
		s.Interval = 0.01
	}

	if s.Interval < 0 {
		return fmt.Errorf("interval must be non-negative")
	}

	if s.Interval < 0.01 {
		return fmt.Errorf("interval must be larger than or equal to 0.01s")
	}

	if (s.Interval * float64(time.Second)) >= float64(time.Hour*24*365*10) {
		return fmt.Errorf("interval must be smaller than 10 years, very arbitrary I know")
	}

	if s.Interval <= 0.2 && !s.IsPrivileged {
		return fmt.Errorf("minimal interval allowed for non-privileged mode is 0.2s")
	}

	return nil
}
