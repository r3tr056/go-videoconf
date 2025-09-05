package monitoring

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	roomsCreated = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "room_management_rooms_created_total",
		Help: "Total number of rooms created.",
	})
	roomsDeleted = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "room_management_rooms_deleted_total",
		Help: "Total number of rooms deleted.",
	})
	participantsAdded = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "room_management_participants_added_total",
		Help: "Total number of participants added to rooms.",
	})
	participantsRemoved = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "room_management_participants_removed_total",
		Help: "Total number of participants removed from rooms.",
	})
	activeRoomsGauge     prometheus.GaugeFunc
	getActiveRoomCountFn func() int
)

func init() {
	prometheus.MustRegister(roomsCreated)
	prometheus.MustRegister(roomsDeleted)
	prometheus.MustRegister(participantsAdded)
	prometheus.MustRegister(participantsRemoved)

	activeRoomsGauge = prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "room_management_active_rooms",
		Help: "Current number of active rooms.",
	}, func() float64 {
		if getActiveRoomCountFn != nil {
			return float64(getActiveRoomCountFn())
		}
		return 0
	})
	prometheus.MustRegister(activeRoomsGauge)
}

func MetricsHandler() http.Handler {
	return promhttp.Handler()
}

func IncRoomsCreated() {
	roomsCreated.Inc()
}

func IncRoomsDeleted() {
	roomsDeleted.Inc()
}

func IncParticipantsAdded() {
	participantsAdded.Inc()
}

func IncParticipantsRemoved() {
	participantsRemoved.Inc()
}

// SetGetActiveRoomCountFn sets the function to retrieve the active room count.
func SetGetActiveRoomCountFn(fn func() int) {
	getActiveRoomCountFn = fn
}
