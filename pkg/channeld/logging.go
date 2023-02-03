package channeld

import (
	"path/filepath"
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
var securityLogger *Logger

func RootLogger() *Logger {
	return rootLogger
}

const VerboseLevel LogLevel = -2
const VeryVerboseLevel LogLevel = -3
const TraceLevel LogLevel = -4

/*
func (l LogLevel) String() string {
	if l == TraceLevel {
		return "trace"
	} else {
		return zapcore.Level(l).String()
	}
}
*/

func (logger *Logger) Verbose(msg string, fields ...zap.Field) {
	if logger == nil {
		return
	}
	if ce := logger.Check(zapcore.Level(VerboseLevel), msg); ce != nil {
		ce.Write(fields...)
	}
}

func (logger *Logger) VeryVerbose(msg string, fields ...zap.Field) {
	if logger == nil {
		return
	}
	if ce := logger.Check(zapcore.Level(VeryVerboseLevel), msg); ce != nil {
		ce.Write(fields...)
	}
}

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

	cfg.OutputPaths = append(cfg.OutputPaths, filepath.Dir(GlobalSettings.LogFile.Value)+"/security.log")
	zapLogger, _ = cfg.Build()
	securityLogger = &Logger{zapLogger}

	zap.Hooks(func(e zapcore.Entry) error {
		if e.Level >= zapcore.WarnLevel {
			logNum.WithLabelValues(e.Level.String()).Inc()
		}
		return nil
	})

	defer rootLogger.Sync()
}
