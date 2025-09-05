package server

import (
	"crypto/tls"
	"net/http"
	"strconv"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/r3tr056/go-videoconf/room-service/config"
	"github.com/r3tr056/go-videoconf/room-service/internal/auth"
	"github.com/r3tr056/go-videoconf/room-service/internal/handler"
	"github.com/r3tr056/go-videoconf/room-service/internal/monitoring"
	"github.com/sirupsen/logrus"
)

type RoomHTTPService struct {
	config       *config.Config
	httpHandlers *handler.RoomHTTPHandler
	authConfig   *auth.AuthConfig
}

func NewHTTPServer(cfg *config.Config, httpHandlers *handler.RoomHTTPHandler, authConfig *auth.AuthConfig) *RoomHTTPService {
	return &RoomHTTPService{
		config:       cfg,
		httpHandlers: httpHandlers,
		authConfig:   authConfig,
	}
}

func (s *RoomHTTPService) Run() error {
	mux := http.NewServeMux()

	jwtMiddleware := auth.JWTMiddleware(s.authConfig)

	// room endpoints
	mux.HandleFunc("/rooms", jwtMiddleware(s.handleRooms))
	mux.HandleFunc("/rooms/", jwtMiddleware(s.handleRoomsByID))

	// participant endpoints
	mux.HandleFunc("/rooms/", jwtMiddleware(s.handleRoomParticipants))
	mux.HandleFunc("/participants/", jwtMiddleware(s.handleParticipantByID))

	mux.HandleFunc("/health", monitoring.HealthCheckHandler)
	mux.Handle("/metrics", promhttp.Handler())

	server := &http.Server{
		Addr:    ":" + strconv.Itoa(s.config.ServicePort),
		Handler: mux,
		TLSConfig: &tls.Config{
			MinVersion:               tls.VersionTLS12,
			PreferServerCipherSuites: true,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
				tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			},
		},
	}

	logrus.Infof("HTTP server listening on port %d", s.config.ServicePort)
	if err := server.ListenAndServeTLS(s.config.ServiceTLSCert, s.config.ServiceTLSKey); err != nil && err != http.ErrServerClosed {
		logrus.Fatalf("HTTP server ListenAndServe error: %v", err)
		return err
	}

	return nil
}

func (s *RoomHTTPService) handleRooms(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		s.httpHandlers.CreateRoomHandler(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *RoomHTTPService) handleRoomsByID(w http.ResponseWriter, r *http.Request) {
	roomID := ""
	path := r.URL.Path
	if len(path) > len("/rooms/") {
		roomID = path[len("/rooms/"):]
		if roomID == "" || roomID[len(roomID)-1:] == "/" || containsSlash(roomID) {
			http.Error(w, "Invalid Room ID", http.StatusBadRequest)
			return
		}
	} else {
		http.Error(w, "Room ID is required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.httpHandlers.GetRoomHandler(w, r, roomID)
	case http.MethodPut:
		s.httpHandlers.UpdateRoomHandler(w, r, roomID)
	case http.MethodDelete:
		s.httpHandlers.DeleteRoomHandler(w, r, roomID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *RoomHTTPService) handleRoomParticipants(w http.ResponseWriter, r *http.Request) {
	roomID := ""
	path := r.URL.Path
	if len(path) > len("/rooms/") {
		roomPath := path[len("/rooms/"):]
		parts := splitPath(roomPath)
		if len(parts) >= 2 && parts[1] == "participants" {
			roomID = parts[0]
		} else {
			http.Error(w, "Invalid path for participants", http.StatusBadRequest)
			return
		}
		if roomID == "" || roomID[len(roomID)-1:] == "/" || containsSlash(roomID) {
			http.Error(w, "Invalid Room ID in participant path", http.StatusBadRequest)
			return
		}
	} else {
		http.Error(w, "Room ID is required for participants", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodPost:
		s.httpHandlers.AddParticipantHandler(w, r, roomID)
	case http.MethodGet:
		s.httpHandlers.ListParticipantsHandler(w, r, roomID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *RoomHTTPService) handleParticipantByID(w http.ResponseWriter, r *http.Request) {
	participantID := ""
	path := r.URL.Path
	if len(path) > len("/participants/") {
		participantID = path[len("/participants/"):]
		if participantID == "" || participantID[len(participantID)-1:] == "/" || containsSlash(participantID) {
			http.Error(w, "Invalid Participant ID", http.StatusBadRequest)
			return
		}
	} else {
		http.Error(w, "Participant ID is required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodDelete:
		s.httpHandlers.RemoveParticipantHandler(w, r, participantID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// Helper function to check if string contains a slash, for basic path segment validation
func containsSlash(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == '/' {
			return true
		}
	}
	return false
}

// Helper function to split path into segments
func splitPath(path string) []string {
	if path[0] == '/' {
		path = path[1:]
	}
	if path[len(path)-1] == '/' {
		path = path[:len(path)-1]
	}
	parts := []string{}
	currentPart := ""
	for i := 0; i < len(path); i++ {
		if path[i] == '/' {
			parts = append(parts, currentPart)
			currentPart = ""
		} else {
			currentPart += string(path[i])
		}
	}
	if currentPart != "" {
		parts = append(parts, currentPart)
	}
	return parts
}
