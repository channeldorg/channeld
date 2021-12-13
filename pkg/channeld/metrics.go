package channeld

import (
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

var logger *zap.Logger

var msgReceived = prometheus.NewCounter(
	prometheus.CounterOpts{
		Name: "messages_in",
		Help: "Received messages",
	},
	//[]string{"channel", "msgType"},
)

var msgSent = prometheus.NewCounter(
	prometheus.CounterOpts{
		Name: "messages_out",
		Help: "Sent messages",
	},
	//[]string{"channel", "msgType"},
)
var packetReceived = prometheus.NewCounter(
	prometheus.CounterOpts{
		Name: "packets_in",
		Help: "Received packets",
	},
)

var packetSent = prometheus.NewCounter(
	prometheus.CounterOpts{
		Name: "packets_out",
		Help: "Sent packets",
	},
)

var bytesReceived = prometheus.NewCounter(
	prometheus.CounterOpts{
		Name: "bytes_in",
		Help: "Received bytes",
	},
)

var bytesSent = prometheus.NewCounter(
	prometheus.CounterOpts{
		Name: "bytes_out",
		Help: "Sent bytes",
	},
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
	logger, _ = zap.NewDevelopment()
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
