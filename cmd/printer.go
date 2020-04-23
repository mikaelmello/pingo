package cmd

import (
	"fmt"
	"time"

	"github.com/mikaelmello/pingo/core"
	"golang.org/x/net/icmp"
)

func printOnStart(s *core.Session, msg *icmp.Message) {
	msgbytes, err := msg.Marshal(nil)
	if err != nil {
		fmt.Printf("PING %s (%s)\n", s.Address(), s.CNAME())
		return
	}

	fmt.Printf("PING %s (%s) %d bytes of data\n", s.CNAME(), s.Address(), len(msgbytes))
}

func printOnRoundTrip(s *core.Session, rt *core.RoundTrip) {
	switch rt.Res {
	case core.Replied:
		fmt.Printf("%d bytes from %s: icmp_seq=%d ttl=%d time=%s\n",
			rt.Len, rt.Src, rt.Seq, rt.TTL, rt.Time.Truncate(time.Microsecond))
	case core.TimedOut:
		fmt.Printf("icmp_seq=%d time=%s timeout expired\n", rt.Seq, rt.Time)
	case core.TTLExpired:
		fmt.Printf("From %s: icmp_seq=%d time to live exceeded\n", rt.Src, rt.Seq)
	}
}

func printOnEnd(s *core.Session) {
	println()
	// cname, err := net.LookupCNAME(address)

	stTime, ok := s.Stats.GetStartTime()
	if !ok {
		println("Stats were not initialized")
	}

	endTime, ok := s.Stats.GetEndTime()
	if !ok {
		println("Stats were not initialized")
	}

	totalTime := endTime.Sub(stTime).Truncate(time.Millisecond)
	rttMin := float64(s.Stats.GetRTTMin()) / float64(time.Millisecond)
	rttMax := float64(s.Stats.GetRTTMax()) / float64(time.Millisecond)
	rttAvg := float64(s.Stats.GetRTTAvg()) / float64(time.Millisecond)
	rttMDev := float64(s.Stats.GetRTTMDev()) / float64(time.Millisecond)

	fmt.Printf("--- %s ping statistics ---\n", s.CNAME())
	fmt.Printf("%d packets transmitted, %d received, %.0f%% packet loss, time %s\n",
		s.Stats.GetTotalSent(), s.Stats.GetTotalRecv(), s.Stats.GetPktLoss()*100, totalTime)
	fmt.Printf("rtt min/avg/max/mdev = %.3f/%.3f/%.3f/%.3f ms\n", rttMin, rttAvg, rttMax, rttMDev)
}
