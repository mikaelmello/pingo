package core

// Settings contains all configurable properties of a ping session.
type Settings struct {
	// TTL is the set IP Time to Live
	TTL int
	// MaxCount is the max amount of ECHO_REQUEST packets sent before exiting.
	MaxCount int

	// Interval is the interval in seconds between a receival and the next send of an ECHO_REQUEST.
	Interval int

	// Timeout is the time in seconds to wait for a response.
	// The option affects only timeout in absence of any responses, otherwise ping waits for two RTTs.
	Timeout int

	// Deadline is the time in seconds before ping exits regardless of how many packets have been sent or received.
	Deadline int

	// IsPrivileged defines if privileged (raw ICMP sockets) or unprivileged (datagram-oriented) mode is used.
	IsPrivileged bool

	// Verbose defines if verbose output is logged.
	Verbose bool

	// PrettyPrint defines if the output is formatted different from the normal ping.
	PrettyPrint bool
}

// DefaultSettings returns the default settings for a ping session, change as you wish.
func DefaultSettings() *Settings {
	return &Settings{
		TTL:          64,
		MaxCount:     -1,
		Interval:     1,
		Timeout:      10,
		Deadline:     -1,
		IsPrivileged: false,
		Verbose:      false,
		PrettyPrint:  false,
	}
}
