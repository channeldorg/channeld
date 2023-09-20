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
var zapConfig zap.Config
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
	if rootLogger != nil {
		return
	}

	if GlobalSettings.Development {
		zapConfig = zap.NewDevelopmentConfig()
	} else {
		zapConfig = zap.NewProductionConfig()
	}
	if GlobalSettings.LogLevel.HasValue {
		zapConfig.Level = zap.NewAtomicLevelAt(zapcore.Level(GlobalSettings.LogLevel.Value))
	}
	if GlobalSettings.LogFile.HasValue {
		zapConfig.OutputPaths = append(zapConfig.OutputPaths, strings.ReplaceAll(GlobalSettings.LogFile.Value, "{time}", time.Now().Format("20060102150405")))
	}

	zapLogger, err := zapConfig.Build()
	if err != nil {
		panic(err)
	}
	rootLogger = &Logger{zapLogger}

	zapConfig.OutputPaths = append(zapConfig.OutputPaths, filepath.Dir(GlobalSettings.LogFile.Value)+"/security.log")
	zapLogger, _ = zapConfig.Build()
	securityLogger = &Logger{zapLogger}

	zap.Hooks(func(e zapcore.Entry) error {
		if e.Level >= zapcore.WarnLevel {
			logNum.WithLabelValues(e.Level.String()).Inc()
		}
		return nil
	})

	// defer rootLogger.Sync()
}

func SetLogLevel(level zapcore.Level) {
	// atomicLevel.SetLevel(level)
	zapConfig.Level.SetLevel(level)
}
