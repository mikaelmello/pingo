package cmd

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/mikaelmello/pingo/core"
)

// Runner is the struct that is responsible for running the program
type Runner struct {
	session *core.Session
	sigch   chan os.Signal
	endch   chan error
}

// newRunner creates a runner with the initialized values
func newRunner(addr string, settings *core.Settings) (*Runner, error) {
	session, err := core.NewSession(addr, settings)
	if err != nil {
		return nil, err
	}

	session.AddOnStart(printOnStart)
	session.AddOnRecv(printOnRoundTrip)
	session.AddOnFinish(printOnEnd)

	return &Runner{
		session: session,
		sigch:   make(chan os.Signal, 1),
		endch:   make(chan error, 1),
	}, nil
}

// Start starts the runner
func (r *Runner) Start() {
	r.handleSignals()

	go func() {
		err := r.session.Run()
		r.endch <- err
	}()
}

// RequestStop requests the stop of the session
func (r *Runner) RequestStop() {
	r.session.RequestStop()
}

// Wait blocks the caller until the runner finishes
func (r *Runner) Wait() error {
	return <-r.endch
}

// handleSignals registers
func (r *Runner) handleSignals() {
	signal.Notify(r.sigch, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-r.sigch
		r.RequestStop()
	}()
}
