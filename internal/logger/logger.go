package logger

import (
	"fmt"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func customLevelEncoder(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	if l == zapcore.ErrorLevel || l == zapcore.DPanicLevel || l == zapcore.PanicLevel || l == zapcore.FatalLevel {
		enc.AppendString(fmt.Sprintf("\x1b[31m%s\x1b[0m", l.CapitalString()))
	} else {
		enc.AppendString(l.CapitalString())
	}
}

func New() (*zap.Logger, error) {
	env := os.Getenv("ENV")

	var config zap.Config
	if env == "production" {
		config = zap.NewProductionConfig()
	} else {
		encoderConfig := zap.NewDevelopmentEncoderConfig()
		encoderConfig.EncodeLevel = customLevelEncoder
		encoderConfig.TimeKey = "timestamp"
		encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		encoderConfig.EncodeCaller = zapcore.ShortCallerEncoder

		config = zap.Config{
			Level:            zap.NewAtomicLevelAt(zap.DebugLevel),
			Development:      true,
			Encoding:         "console",
			EncoderConfig:    encoderConfig,
			OutputPaths:      []string{"stdout"},
			ErrorOutputPaths: []string{"stderr"},
		}
	}

	if env == "production" {
		config.EncoderConfig.TimeKey = "timestamp"
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	logger, err := config.Build(
		zap.AddCaller(),
		zap.AddCallerSkip(0),
	)
	if err != nil {
		return nil, err
	}

	return logger, nil
}

func NewNop() *zap.Logger {
	return zap.NewNop()
}
