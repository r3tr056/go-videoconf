package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/r3tr056/go-videoconf/users-service/controllers"
	"github.com/r3tr056/go-videoconf/users-service/database"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

func main() {
	// Structured JSON logs to stdout (12-factor; collectors scrape the stream).
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	// Fail fast on missing secrets: no default JWT secret is ever used.
	if os.Getenv("JWT_SECRET") == "" {
		log.Fatal("JWT_SECRET must be set (see .env.example)")
	}

	// Initialize database
	err := database.Database.Init()
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer database.Database.Close()

	// Create Gin router
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(metricsMiddleware("users-service"))

	// CORS middleware
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Health check endpoint
	// /health is shallow (process alive); /ready is deep (dependencies reachable).
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "users-service",
		})
	})
	router.GET("/ready", func(c *gin.Context) {
		pingCtx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		if err := database.Database.Client.Ping(pingCtx, readpref.Primary()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// User routes
	userController := &controllers.User{}
	router.POST("/auth", userController.Authenticate)
	router.GET("/users", userController.GetUsers)
	router.GET("/users/:id", userController.GetUser)
	router.POST("/users", userController.CreateUser)
	router.PUT("/users/:id", userController.UpdateUser)
	router.DELETE("/users/:id", userController.DeleteUser)

	// Start server
	port := getenv("PORT", "8081")
	slog.Info("Users service starting", "port", port)
	router.Run(":" + port)
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	return value
}
