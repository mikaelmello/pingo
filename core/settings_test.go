package core

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultSettingsValidate(t *testing.T) {
	settings := DefaultSettings()
	assert.NoError(t, settings.validate())
}

func TestSettingsNegativeTTL(t *testing.T) {
	settings := DefaultSettings()
	settings.TTL = -1
	assert.Error(t, settings.validate())
}

func TestSettingsZeroTTL(t *testing.T) {
	settings := DefaultSettings()
	settings.TTL = 0
	assert.Error(t, settings.validate())
}

func TestSettingsPositiveTTL(t *testing.T) {
	settings := DefaultSettings()
	settings.TTL = 1
	assert.NoError(t, settings.validate())
}

func TestSettingsNegativeCount(t *testing.T) {
	settings := DefaultSettings()
	settings.MaxCount = -1
	settings.IsMaxCountDefault = false
	assert.Error(t, settings.validate())
}

func TestSettingsZeroCount(t *testing.T) {
	settings := DefaultSettings()
	settings.MaxCount = 0
	settings.IsMaxCountDefault = false
	assert.Error(t, settings.validate())
}

func TestSettingsPositiveCount(t *testing.T) {
	settings := DefaultSettings()
	settings.MaxCount = 5
	settings.IsMaxCountDefault = false
	assert.NoError(t, settings.validate())
}

func TestSettingsNegativeDeadline(t *testing.T) {
	settings := DefaultSettings()
	settings.Deadline = -1
	settings.IsDeadlineDefault = false
	assert.Error(t, settings.validate())
}

func TestSettingsZeroDeadline(t *testing.T) {
	settings := DefaultSettings()
	settings.Deadline = 0
	settings.IsDeadlineDefault = false
	assert.Error(t, settings.validate())
}

func TestSettingsPositiveDeadline(t *testing.T) {
	settings := DefaultSettings()
	settings.Deadline = 5
	settings.IsDeadlineDefault = false
	assert.NoError(t, settings.validate())
}

func TestSettingsNegativeTimeout(t *testing.T) {
	settings := DefaultSettings()
	settings.Timeout = -1
	assert.Error(t, settings.validate())
}

func TestSettingsZeroTimeout(t *testing.T) {
	settings := DefaultSettings()
	settings.Timeout = 0
	assert.Error(t, settings.validate())
}

func TestSettingsPositiveTimeout(t *testing.T) {
	settings := DefaultSettings()
	settings.Timeout = 1
	assert.NoError(t, settings.validate())
}

func TestSettingsNegativeInterval(t *testing.T) {
	settings := DefaultSettings()
	settings.Interval = -1
	assert.Error(t, settings.validate())
}

func TestSettingsZeroInterval(t *testing.T) {
	settings := DefaultSettings()
	settings.Interval = 0
	assert.Error(t, settings.validate())
}

func TestSettingsLargeInterval(t *testing.T) {
	settings := DefaultSettings()
	settings.Interval = float64(time.Hour*24*365*10) / float64(time.Second)
	assert.Error(t, settings.validate())
}

func TestSettingsFloodPrivileged(t *testing.T) {
	settings := DefaultSettings()
	settings.Flood = true
	settings.IsPrivileged = true
	assert.NoError(t, settings.validate())
}

func TestSettingsFloodUnprivileged(t *testing.T) {
	settings := DefaultSettings()
	settings.Flood = true
	settings.IsPrivileged = false
	assert.Error(t, settings.validate())
}

func TestSettingsZeroIntervalPrivileged(t *testing.T) {
	settings := DefaultSettings()
	settings.Interval = 0
	settings.IsPrivileged = true
	assert.Error(t, settings.validate())
}

func TestSettingsMinIntervalPrivileged(t *testing.T) {
	settings := DefaultSettings()
	settings.Interval = 0.01
	settings.IsPrivileged = true
	assert.NoError(t, settings.validate())
}

func TestSettings200msInterval(t *testing.T) {
	settings := DefaultSettings()
	settings.Interval = 0.2
	assert.Error(t, settings.validate())
}

func TestSettings200msIntervalPrivileged(t *testing.T) {
	settings := DefaultSettings()
	settings.Interval = 0.2
	settings.IsPrivileged = true
	assert.NoError(t, settings.validate())
}

func TestSettingsPositiveIntegerInterval(t *testing.T) {
	settings := DefaultSettings()
	settings.Interval = 1
	assert.NoError(t, settings.validate())
}
