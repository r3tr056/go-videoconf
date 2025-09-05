package cmd

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/r3tr056/go-videoconf/signalling-server/config"
	"github.com/r3tr056/go-videoconf/signalling-server/internal/monitoring"
	servicediscovery "github.com/r3tr056/go-videoconf/signalling-server/internal/service_discovery"
	"github.com/r3tr056/go-videoconf/signalling-server/internal/signalling"
	"github.com/r3tr056/go-videoconf/signalling-server/internal/storage"
	"github.com/sirupsen/logrus"
)

type SignallingService struct {
	config          *config.Config
	consulRegistry  *servicediscovery.ConsulServiceRegistery
	roomManager     *signalling.RoomManager
	userManager     *signalling.UserManager
	websocketServer *signalling.WebSocketServer
	messageHandler  *signalling.MessageHandler
	mongoStorage    *storage.MongoDBStorage
}

func NewSignallingService(cfg *config.Config) (*SignallingService, error) {
	mongoStorage, err := storage.NewMongoDBStorage(cfg)
	if err != nil {
		return nil, err
	}

	roomManager := signalling.NewRoomManager(mongoStorage)
	userManager := signalling.NewUserManager(cfg.GetPingInterval())
	wsServer := signalling.NewWebSocketServer(cfg, userManager, roomManager, nil, nil)

	msgHandler := signalling.NewMessageHandler(roomManager, userManager, wsServer)
	wsServer.MessageHandler = msgHandler

	consulRegistry, err := servicediscovery.NewConsulRegistry(
		cfg.ConsulAddress,
		cfg.ConsulServiceName,
		cfg.ServicePort,
		"", // Service Address - let Consul resolve this automatically
		"/health",
	)
	if err != nil {
		return nil, err
	}

	return &SignallingService{
		config:          cfg,
		consulRegistry:  consulRegistry,
		roomManager:     roomManager,
		userManager:     userManager,
		websocketServer: wsServer,
		messageHandler:  msgHandler,
		mongoStorage:    mongoStorage,
	}, nil
}

func (s *SignallingService) Run() error {
	if err := s.consulRegistry.RegisterService(); err != nil {
		return err
	}
	defer s.consulRegistry.DeregisterService()

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.websocketServer.HandleWebSocketConnection)
	mux.HandleFunc("/health", monitoring.HealthCheckHandler)
	mux.HandleFunc("/metrics", monitoring.MetricsHandler().ServeHTTP)

	httpServer := &http.Server{
		Addr:    ":" + strconv.Itoa(s.config.ServicePort),
		Handler: mux,
	}

	go func() {
		logrus.Infof("HTTP server listining on port %d", s.config.ServicePort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.Fatalf("HTTP server ListenAndServe error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logrus.Info("Shuttind down signalling service...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		logrus.Errorf("HTTP server shutdown error: %v", err)
		return err
	}

	if err := s.mongoStorage.Client.Disconnect(context.Background()); err != nil {
		logrus.Errorf("Error disconnecting from MongoDB: %v", err)
	} else {
		logrus.Info("Disconnected from MongoDB")
	}

	logrus.Info("Singalling service gracefully stopped")
	return nil
}
