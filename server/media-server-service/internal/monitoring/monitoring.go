package monitoring

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	activeMediaSessions = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "media_server_active_sessions_total",
		Help: "Total number of active media streaming sessions.",
	})
	mediaSessionsStarted = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "media_server_sessions_started_total",
		Help: "Total number of media streaming sessions started.",
	})
	mediaSessionsStopped = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "media_server_sessions_stopped_total",
		Help: "Total number of media streaming sessions stopped.",
	})
	mediaBytesSent = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "media_server_bytes_sent_total",
		Help: "Total bytes of media data sent.",
	}, []string{"media_type"}) // e.g., "audio", "video"
	mediaBytesReceived = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "media_server_bytes_received_total",
		Help: "Total bytes of media data received.",
	}, []string{"media_type"})
)

func init() {
	prometheus.MustRegister(activeMediaSessions)
	prometheus.MustRegister(mediaSessionsStarted)
	prometheus.MustRegister(mediaSessionsStopped)
	prometheus.MustRegister(mediaBytesSent)
	prometheus.MustRegister(mediaBytesReceived)
}

func MetricsHandler() http.Handler {
	return promhttp.Handler()
}

func IncActiveMediaSessions() {
	activeMediaSessions.Inc()
}

func DecActiveMediaSessions() {
	activeMediaSessions.Dec()
}

func SetActiveMediaSessions(count float64) {
	activeMediaSessions.Set(count)
}

func IncMediaSessionsStarted() {
	mediaSessionsStarted.Inc()
}

func IncMediaSessionsStopped() {
	mediaSessionsStopped.Inc()
}

func IncMediaBytesSent(mediaType string, bytes float64) {
	mediaBytesSent.With(prometheus.Labels{"media_type": mediaType}).Add(bytes)
}

func IncMediaBytesReceived(mediaType string, bytes float64) {
	mediaBytesReceived.With(prometheus.Labels{"media_type": mediaType}).Add(bytes)
}
