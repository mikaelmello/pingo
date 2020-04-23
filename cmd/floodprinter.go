package cmd

import (
	"sync"

	"github.com/mikaelmello/pingo/core"
	"golang.org/x/net/icmp"
)

var printMutex sync.Mutex

// registerFlood registers its callbacks to be called by the session
func registerFlood(s *core.Session) {
	s.AddOnStart(floodPrintOnStart)
	s.AddOnSend(floodPrintOnSend)
	s.AddOnRecv(floodPrintOnRoundTrip)
	s.AddOnFinish(floodPrintOnEnd)
}

func floodPrintOnStart(s *core.Session, msg *icmp.Message) {
	stdPrintOnStart(s, msg)
}

func floodPrintOnSend(s *core.Session) {
	printMutex.Lock()
	defer printMutex.Unlock()

	print(".")
}

func floodPrintOnRoundTrip(s *core.Session, rt *core.RoundTrip) {
	printMutex.Lock()
	defer printMutex.Unlock()

	if rt.Res == core.Replied {
		print("\b \b")
	}
}

func floodPrintOnEnd(s *core.Session) {
	println()
	stdPrintOnEnd(s)
}
