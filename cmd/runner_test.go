package cmd

import (
	"syscall"
	"testing"
	"time"

	"github.com/mikaelmello/pingo/core"
	"github.com/stretchr/testify/assert"
)

// TestNewRunner tests if a runner is properly initialized
func TestNewRunner(t *testing.T) {
	r, err := newRunner("localhost", core.DefaultSettings())
	assert.NoError(t, err)

	// TODO(checkadd): Mock add handler calls to check if we are actually adding the printer handlers
	assert.NotNil(t, r.session)
	assert.Empty(t, r.endch)
	assert.Empty(t, r.sigch)
}

// TestRequestStopWaitStops tests if when a runner is stopped, the session has really finished
func TestRequestStopWaitStops(t *testing.T) {
	r, err := newRunner("localhost", core.DefaultSettings())
	assert.NoError(t, err)

	r.Start()
	r.RequestStop()

	ch := make(chan error, 1)
	go func() {
		ch <- r.Wait()
	}()

	select {
	case err := <-ch:
		assert.NoError(t, err)
		assert.True(t, r.session.IsStarted())
		assert.True(t, r.session.IsFinished())
	case <-time.After(time.Second):
		assert.Fail(t, "Requesting stop of session did not stop session")
	}
}

// TestSigTermHandling tests if the sigterm signal really stops the run
func TestSigTermHandling(t *testing.T) {
	r, err := newRunner("localhost", core.DefaultSettings())
	assert.NoError(t, err)

	r.Start()

	ch := make(chan error)
	go func() {
		ch <- r.Wait()
	}()

	assert.Empty(t, ch)
	r.sigch <- syscall.SIGTERM

	select {
	case err := <-ch:
		assert.NoError(t, err)
	case <-time.After(time.Second):
		assert.Fail(t, "Sigterm did not end run on time")
	}
}
