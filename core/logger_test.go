package core

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestNewLogger(t *testing.T) {
	for i := uint32(0); i <= 6; i++ {
		logger := NewLogger(i)
		assert.Equal(t, logrus.Level(i), logger.GetLevel())
	}
	logger := NewLogger(8)
	assert.Equal(t, logrus.Level(8), logger.GetLevel())
}
