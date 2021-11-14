package channeld

import "github.com/prometheus/client_golang/prometheus"

var packetReceived = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "packets_in",
		Help: "Received packets",
	},
	[]string{"size", "channel", "msgType"},
)

func InitMetrics() {
	prometheus.MustRegister(packetReceived)
}
