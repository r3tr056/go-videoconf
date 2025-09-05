package main

import (
	"fmt"

	"github.com/r3tr056/go-videoconf/signalling-server/cmd"
	"github.com/r3tr056/go-videoconf/signalling-server/config"
	"github.com/sirupsen/logrus"
)

func main() {
	cfg := config.LoadConfig()
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2025-02-05 15:04:05",
	})

	service, err := cmd.NewSignallingService(cfg)
	if err != nil {
		logrus.Fatalf("Failed to initialize signaling service: %v", err)
	}

	if err := service.Run(); err != nil {
		logrus.Fatalf("Error running signaling service: %v, exiting...", fmt.Errorf("service run error: %w", err))
	}
}
