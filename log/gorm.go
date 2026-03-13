package log

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

const SlowThreshold = 100 * time.Millisecond

type GormLogger struct {
	logger *zap.SugaredLogger
	level  gormlogger.LogLevel
}

func NewGormLogger() *GormLogger {
	return &GormLogger{
		logger: S().Named("database"),
		level:  gormlogger.Warn,
	}
}

func (l *GormLogger) SetAsDefault() {
	gormlogger.Default = l
}

func (l *GormLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	newLogger := *l
	newLogger.level = level
	return &newLogger
}

func (l *GormLogger) Info(ctx context.Context, str string, args ...interface{}) {
	if l.level < gormlogger.Info {
		return
	}
	l.logger.Debugf(str, args...)
}

func (l *GormLogger) Warn(ctx context.Context, str string, args ...interface{}) {
	if l.level < gormlogger.Warn {
		return
	}
	l.logger.Warnf(str, args...)
}

func (l *GormLogger) Error(ctx context.Context, str string, args ...interface{}) {
	if l.level < gormlogger.Error {
		return
	}
	l.logger.Errorf(str, args...)
}

func (l *GormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.level <= 0 {
		return
	}
	elapsed := time.Since(begin)

	switch {
	case err != nil && l.level >= gormlogger.Error && !errors.Is(err, gorm.ErrRecordNotFound):
		sql, rows := fc()
		l.logger.Error("trace", zap.Error(err), zap.Duration("elapsed", elapsed), zap.Int64("rows", rows), zap.String("sql", sql))
	case elapsed > SlowThreshold && l.level >= gormlogger.Warn:
		sql, rows := fc()
		l.logger.Warn("trace", zap.Duration("elapsed", elapsed), zap.Int64("rows", rows), zap.String("sql", sql))
	case l.level >= gormlogger.Info:
		sql, rows := fc()
		l.logger.Debug("trace", zap.Duration("elapsed", elapsed), zap.Int64("rows", rows), zap.String("sql", sql))
	}
}
