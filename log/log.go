package log

import (
	"fmt"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var logger *zap.Logger

func init() {
	encConfig := zap.NewDevelopmentEncoderConfig()
	encConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	encConfig.EncodeCaller = nil
	encConfig.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.UTC().Format(time.StampMicro))
	}

	encoder := zapcore.NewConsoleEncoder(encConfig)

	stdout, closer, err := zap.Open("stdout")
	if err != nil {
		closer()
		panic(fmt.Errorf("failed to initialize logger: %w", err))
	}

	stderr, closer, err := zap.Open("stderr")
	if err != nil {
		closer()
		panic(fmt.Errorf("failed to initialize logger: %w", err))
	}

	level := zap.NewAtomicLevelAt(zapcore.InfoLevel)
	if os.Getenv("DEBUG") != "" {
		level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	}

	ws := zapcore.NewMultiWriteSyncer(stdout)
	core := zapcore.NewCore(encoder, ws, level)
	logger = zap.New(core, zap.ErrorOutput(stderr))
}

// S returns a *[zap.SugaredLogger].
func S() *zap.SugaredLogger {
	return logger.Sugar()
}

// L returns a *[zap.Logger].
func L() *zap.Logger {
	return logger
}
