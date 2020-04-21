package core

// Settings contains all configurable properties of a ping session
type Settings struct {
	TTL          int
	MaxCount     int
	Interval     int
	Timeout      int
	Deadline     int
	IsPrivileged bool
	Verbose      bool
	PrettyPrint  bool
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
