package logger

import (
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func New(level string) (*zap.Logger, error) {
	logLevel, err := zap.ParseAtomicLevel(level)
	if err != nil {
		logLevel = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	}

	logConf := zap.Config{
		Level:            logLevel,
		Development:      false,
		Encoding:         "json",
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
		EncoderConfig: zapcore.EncoderConfig{
			MessageKey:   "message",
			LevelKey:     "level",
			TimeKey:      "time",
			NameKey:      "logger_name",
			CallerKey:    "caller",
			FunctionKey:  "function",
			EncodeLevel:  zapcore.CapitalLevelEncoder,
			EncodeTime:   zapcore.ISO8601TimeEncoder,
			EncodeCaller: zapcore.ShortCallerEncoder,
		},
	}

	logger, err := logConf.Build()
	if err != nil {
		return nil, fmt.Errorf("can't configure logger: %w", err)
	}
	if logger == nil {
		return nil, fmt.Errorf("logger is nil")
	}

	return logger, nil
}
