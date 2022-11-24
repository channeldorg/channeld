package channeld

import (
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type LogLevel zapcore.Level

type Logger struct {
	*zap.Logger
}

var rootLogger *Logger //*zap.Logger

func RootLogger() *Logger {
	return rootLogger
}

const TraceLevel LogLevel = -2

/*
func (l LogLevel) String() string {
	if l == TraceLevel {
		return "trace"
	} else {
		return zapcore.Level(l).String()
	}
}
*/

func (logger *Logger) Trace(msg string, fields ...zap.Field) {
	if logger == nil {
		return
	}
	if ce := logger.Check(zapcore.Level(TraceLevel), msg); ce != nil {
		ce.Write(fields...)
	}
}

func InitLogs() {
	var cfg zap.Config
	if GlobalSettings.Development {
		cfg = zap.NewDevelopmentConfig()
	} else {
		cfg = zap.NewProductionConfig()
	}
	if GlobalSettings.LogLevel.HasValue {
		cfg.Level = zap.NewAtomicLevelAt(zapcore.Level(GlobalSettings.LogLevel.Value))
	}
	if GlobalSettings.LogFile.HasValue {
		cfg.OutputPaths = append(cfg.OutputPaths, strings.ReplaceAll(GlobalSettings.LogFile.Value, "{time}", time.Now().Format("20060102150405")))
	}
	zapLogger, _ := cfg.Build()
	rootLogger = &Logger{zapLogger}

	zap.Hooks(func(e zapcore.Entry) error {
		if e.Level >= zapcore.WarnLevel {
			logNum.WithLabelValues(e.Level.String()).Inc()
		}
		return nil
	})

	defer rootLogger.Sync()
}
