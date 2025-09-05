package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	server "github.com/r3tr056/go-videoconf/room-service/cmd"
	"github.com/r3tr056/go-videoconf/room-service/config"
	"github.com/r3tr056/go-videoconf/room-service/internal/handler"
	"github.com/r3tr056/go-videoconf/room-service/internal/room"
	servicediscovery "github.com/r3tr056/go-videoconf/room-service/internal/service_discovery"
	"github.com/r3tr056/go-videoconf/room-service/internal/storage"
	"github.com/sirupsen/logrus"
)

func main() {
	cfg := config.LoadConfig()
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})

	mongoStorage, err := storage.NewMongoDBStorage(cfg)
	if err != nil {
		logrus.Fatalf("Failed to initialize MongoDB storage: %v", err)
	}

	authConfig := cfg.GetJWTAuthConfig()
	roomManager := room.NewRoomManager(mongoStorage)
	httpHandlers := handler.NewRoomHandler(roomManager)
	httpServer := server.NewHTTPServer(cfg, httpHandlers, authConfig)

	consulRegistry, err := servicediscovery.NewConsulRegistry(
		cfg.ConsulAddress,
		cfg.ConsulServiceName,
		cfg.ServicePort,
		"",
		"/health",
	)
	if err != nil {
		logrus.Fatalf("Failed to initialize Consul registry: %v", err)
	}

	if err := consulRegistry.RegisterService(); err != nil {
		logrus.Fatalf("Failed to register service with Consul: %v", err)
	}
	defer func() {
		if err := consulRegistry.DeregisterService(); err != nil {
			logrus.Errorf("Failed to deregister service from Consul: %v", err)
		}
		if err := mongoStorage.Client.Disconnect(context.Background()); err != nil {
			logrus.Errorf("Error disconnecting from MongoDB: %v", err)
		} else {
			logrus.Info("Disconnected from MongoDB")
		}
	}()

	// start http server
	go func() {
		if err := httpServer.Run(); err != nil {
			logrus.Fatalf("Failed to start HTTP server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logrus.Info("Shutting down Room Management Service...")

	_, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	logrus.Info("Room Management Service gracefully stopped")
}
