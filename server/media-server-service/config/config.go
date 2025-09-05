package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

type Config struct {
	ServiceName         string
	ServicePort         int
	ConsulAddress       string
	ConsulServiceName   string
	LogLevel            string
	MediaServerID       string
	PingIntervalSeconds int
	SFUMode             string
}

func LoadConfig() *Config {
	godotenv.Load()

	servicePort, err := strconv.Atoi(getEnvOrDefault("SERVICE_PORT", "8083"))
	if err != nil {
		logrus.Fatalf("Invalid SERVICE_PORT: %v", err)
	}

	pingIntervalSeconds, err := strconv.Atoi(getEnvOrDefault("PING_INTERVAL_SECONDS", "30"))
	if err != nil {
		logrus.Fatalf("Invalid PING_INTERVAL_SECONDS: %v", err)
	}

	cfg := &Config{
		ServiceName:         getEnvOrDefault("SERVICE_NAME", "media-server-service"),
		ServicePort:         servicePort,
		ConsulAddress:       getEnvOrDefault("CONSUL_ADDRESS", "localhost:8500"),
		ConsulServiceName:   getEnvOrDefault("CONSUL_SERVICE_NAME", "media-server-service"),
		LogLevel:            getEnvOrDefault("LOG_LEVEL", "info"),
		MediaServerID:       getEnvOrDefault("MEDIA_SERVER_ID", generateMediaServerID()),
		PingIntervalSeconds: pingIntervalSeconds,
		SFUMode:             getEnvOrDefault("SFU_MODE", "sfu"),
	}

	level, err := logrus.ParseLevel(cfg.LogLevel)
	if err != nil {
		logrus.Warnf("Invalid log level '%s', defaulting to 'info'", cfg.LogLevel)
		level = logrus.InfoLevel
	}
	logrus.SetLevel(level)

	return cfg
}

func getEnvOrDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func (c *Config) GetPingInterval() time.Duration {
	return time.Duration(c.PingIntervalSeconds) * time.Second
}

// NOTE: generateMediaServerID generates a simple unique ID (can be improved for production)
func generateMediaServerID() string {
	hostname, _ := os.Hostname()
	return hostname + "-" + strconv.FormatInt(int64(os.Getpid()), 10)
}
