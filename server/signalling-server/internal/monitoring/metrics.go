package monitoring

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	connectedClients = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "signalling_connected_clients_total",
		Help: "Total number of connected WebSocket clients.",
	})
	activeRooms = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "signalling_active_rooms_total",
		Help: "Total number of active signalling rooms.",
	})
	messagesSent = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "signalling_messages_sent_total",
		Help: "Total number of signalling messages sent.",
	}, []string{"type"})
	messagesRecvd = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "signalling_messages_received_total",
		Help: "Total number of signalling messages received.",
	}, []string{"type"})
)

func init() {
	prometheus.MustRegister(connectedClients)
	prometheus.MustRegister(activeRooms)
	prometheus.MustRegister(messagesSent)
	prometheus.MustRegister(messagesRecvd)
}

func MetricsHandler() http.Handler {
	return promhttp.Handler()
}

func IncConnectedClients() {
	connectedClients.Inc()
}

func DecConnectedClients() {
	connectedClients.Dec()
}

func SetActiveRooms(count int) {
	activeRooms.Set(float64(count))
}

func IncMessagesSent(msgType string) {
	messagesSent.With(prometheus.Labels{"type": msgType}).Inc()
}

func IncMessagesReceived(msgType string) {
	messagesRecvd.With(prometheus.Labels{"type": msgType}).Inc()
}
