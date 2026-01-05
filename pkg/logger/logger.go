package logger

import (
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	Log *zap.Logger
)

type Config struct {
	Level      string `mapstructure:"level"`
	Filename   string `mapstructure:"filename"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"`
	Compress   bool   `mapstructure:"compress"`
}

// InitLogger initializes the global logger
func InitLogger(cfg *Config) error {
	writeSyncer := getLogWriter(cfg)
	encoder := getEncoder()

	var l = new(zapcore.Level)
	err := l.UnmarshalText([]byte(cfg.Level))
	if err != nil {
		return err
	}

	core := zapcore.NewCore(encoder, writeSyncer, l)

	Log = zap.New(core, zap.AddCaller())
	zap.ReplaceGlobals(Log)

	return nil
}

func getEncoder() zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	return zapcore.NewJSONEncoder(encoderConfig)
}

func getLogWriter(cfg *Config) zapcore.WriteSyncer {
	lumberJackLogger := &lumberjack.Logger{
		Filename:   cfg.Filename,
		MaxSize:    cfg.MaxSize,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAge,
		Compress:   cfg.Compress,
	}

	// Create a buffered write syncer for better performance (buffering)
	// This writes to the underlying writer (lumberjack)
	// buffer size: 256 kB (default is 256 kB if not specified in some implementations, but here we can specify or use default wrapper)
	// zap doesn't export BufferedWriteSyncer directly in older versions, but let's check.
	// Actually zapcore.BufferedWriteSyncer is available.

	// Console output
	consoleSyncer := zapcore.AddSync(os.Stdout)

	// File output with buffering
	fileSyncer := zapcore.AddSync(lumberJackLogger)
	bufferedFileSyncer := &zapcore.BufferedWriteSyncer{
		WS:            fileSyncer,
		Size:          256 * 1024, // 256KB buffer
		FlushInterval: 5 * time.Second,
	}

	return zapcore.NewMultiWriteSyncer(consoleSyncer, bufferedFileSyncer)
}

// Sync flushes any buffered log entries
func Sync() {
	if Log != nil {
		Log.Sync()
	}
}
