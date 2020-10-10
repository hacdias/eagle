package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewLogger(c *Config) *zap.SugaredLogger {
	encConfig := zap.NewDevelopmentEncoderConfig()
	encConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	encConfig.EncodeCaller = nil
	encConfig.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.UTC().Format(time.StampMicro))
	}

	encoder := zapcore.NewConsoleEncoder(encConfig)

	sout, closer, err := zap.Open("stdout")
	if err != nil {
		closer()
		panic(fmt.Errorf("failed to initialize logger: %w", err))
	}

	serr, closer, err := zap.Open("stderr")
	if err != nil {
		closer()
		panic(fmt.Errorf("failed to initialize logger: %w", err))
	}

	level := zap.NewAtomicLevelAt(zapcore.InfoLevel)

	debug := false
	val, ok := os.LookupEnv("DEBUG")
	if ok && strings.EqualFold(val, "true") {
		debug = true
	}

	if c.Development || debug {
		level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	}

	stdout := sout
	stderr := serr

	ws := zapcore.NewMultiWriteSyncer(stdout)
	core := zapcore.NewCore(encoder, ws, level)
	logger := zap.New(core, zap.ErrorOutput(stderr))

	return logger.Sugar()
}
