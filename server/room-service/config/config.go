package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"github.com/r3tr056/go-videoconf/room-service/internal/auth"
	"github.com/sirupsen/logrus"
)

type Config struct {
	ServiceName             string
	ServicePort             int
	ConsulAddress           string
	ConsulServiceName       string
	MongoDBURI              string
	LogLevel                string
	ServiceTLSCert          string
	ServiceTLSKey           string
	JWTSigningKey           string
	JWTTokenDurationMinutes int
}

func LoadConfig() *Config {
	godotenv.Load()

	servicePort, err := strconv.Atoi(getEnvOrDefault("SERVICE_PORT", "8081"))
	if err != nil {
		logrus.Fatalf("Invalid SERVICE_PORT: %v", err)
	}

	jwtTokenDurationMinutes, err := strconv.Atoi(getEnvOrDefault("JWT_TOKEN_DURATION_MINUTES", "30"))
	if err != nil {
		logrus.Fatalf("Invalid JWT_TOKEN_DURATION_MINUTES: %v", err)
	}

	cfg := &Config{
		ServiceName:             getEnvOrDefault("SERVICE_NAME", "room-management-service"),
		ServicePort:             servicePort,
		ConsulAddress:           getEnvOrDefault("CONSUL_ADDRESS", "localhost:8500"),
		ConsulServiceName:       getEnvOrDefault("CONSUL_SERVICE_NAME", "room-management-service"),
		MongoDBURI:              getEnvOrDefault("MONGODB_URI", "mongodb://localhost:27017"),
		LogLevel:                getEnvOrDefault("LOG_LEVEL", "info"),
		ServiceTLSCert:          getEnvOrDefault("SERVICE_TLS_CERT", "cert.pem"),
		ServiceTLSKey:           getEnvOrDefault("SERVICE_TLS_KEY", "key.pem"),
		JWTSigningKey:           getEnvOrDefault("JWT_SIGNING_KEY", "your-jwt-secret-key-very-long-and-secure"),
		JWTTokenDurationMinutes: jwtTokenDurationMinutes,
	}

	level, err := logrus.ParseLevel(cfg.LogLevel)
	if err != nil {
		logrus.Warnf("Invalid log level '%s', defaulting to 'info'", cfg.LogLevel)
		level = logrus.InfoLevel
	}
	logrus.SetLevel(level)

	return cfg
}

func (c *Config) GetJWTAuthConfig() *auth.AuthConfig {
	return &auth.AuthConfig{
		JWTSigningKey: c.JWTSigningKey,
		TokenDuration: time.Duration(c.JWTTokenDurationMinutes) * time.Minute,
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
