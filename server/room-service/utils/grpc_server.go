package utils

import (
	"github.com/r3tr056/go-videoconf/room-service/config"
	"github.com/r3tr056/go-videoconf/room-service/internal/handler"
)

type GRPCServer struct {
	config       *config.Config
	grpcHandlers *handler.GRPCHandlers
}
