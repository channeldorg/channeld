package channeld

import (
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var logger *zap.Logger

var msgReceived = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "messages_in",
		Help: "Received messages",
	},
	[]string{"connType" /*, "channel", "msgType"*/},
)

var msgSent = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "messages_out",
		Help: "Sent messages",
	},
	[]string{"connType" /*, "channel", "msgType"*/},
)
var packetReceived = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "packets_in",
		Help: "Received packets",
	},
	[]string{"connType"},
)

var packetSent = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "packets_out",
		Help: "Sent packets",
	},
	[]string{"connType"},
)

var bytesReceived = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "bytes_in",
		Help: "Received bytes",
	},
	[]string{"connType"},
)

var bytesSent = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "bytes_out",
		Help: "Sent bytes",
	},
	[]string{"connType"},
)

var connectionNum = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "connection_num",
		Help: "Number of connections",
	},
	[]string{"type"},
)

var channelNum = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "channel_num",
		Help: "Number of channels",
	},
	[]string{"type"},
)

func InitLogsAndMetrics() {
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
	logger, _ = cfg.Build()
	defer logger.Sync()

	prometheus.MustRegister(msgReceived)
	prometheus.MustRegister(msgSent)
	prometheus.MustRegister(packetReceived)
	prometheus.MustRegister(packetSent)
	prometheus.MustRegister(bytesReceived)
	prometheus.MustRegister(bytesSent)
	prometheus.MustRegister(connectionNum)
	prometheus.MustRegister(channelNum)
}
