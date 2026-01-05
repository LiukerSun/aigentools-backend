package logger

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInitLogger(t *testing.T) {
	// Cleanup
	defer os.Remove("test.log")

	cfg := &Config{
		Level:      "DEBUG",
		Filename:   "test.log",
		MaxSize:    1,
		MaxBackups: 1,
		MaxAge:     1,
		Compress:   false,
	}

	err := InitLogger(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, Log)

	Log.Info("Test log message")
	Sync()

	// Verify file exists
	_, err = os.Stat("test.log")
	assert.NoError(t, err)
}

func TestInitLoggerInvalidLevel(t *testing.T) {
	cfg := &Config{
		Level:    "INVALID",
		Filename: "test_invalid.log",
	}

	err := InitLogger(cfg)
	assert.Error(t, err)
}
