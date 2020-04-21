package core

// Settings contains all configurable properties of a ping session
type Settings struct {
	// IP Time to Live
	TTL int
	// Max amount of ECHO_REQUEST packets sent before exiting.
	MaxCount int

	// Interval between a receival and the next send of an ECHO_REQUEST
	Interval int

	// Time to wait for a response, in seconds. The option affects only timeout in absence of any responses, otherwise ping waits for two RTTs.
	Timeout int

	// Specify a timeout, in seconds, before ping exits regardless of how many packets have been sent or received.
	Deadline int

	// Whether to run as a privileged user (using TCP) or unprivileged (using UDP)
	IsPrivileged bool

	// Whether to log verbose output
	Verbose bool

	// Whether to format the output in a modern way
	PrettyPrint bool
}

// DefaultSettings returns the default settings for a ping session, change as you wish
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
