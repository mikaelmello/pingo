package core

import (
	log "github.com/sirupsen/logrus"
)

// NewLogger returns a new pre-configured logger
func NewLogger(level uint32) *log.Logger {
	logger := log.New()

	logger.SetFormatter(&log.TextFormatter{})

	// Only log the warning severity or above.
	logger.SetLevel(log.Level(level))

	return logger
}
