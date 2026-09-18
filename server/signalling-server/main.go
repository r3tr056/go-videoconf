package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	"github.com/r3tr056/go-videoconf/signalling-server/controllers"
	"github.com/r3tr056/go-videoconf/signalling-server/interfaces"
)

const (
	// wsReadLimit caps a single inbound message (SDP offers are a few KB).
	wsReadLimit = 64 * 1024
	// wsPongWait bounds how long a dead peer can linger; wsPingInterval
	// keeps healthy peers refreshing their deadline.
	wsPongWait     = 60 * time.Second
	wsPingInterval = (wsPongWait * 9) / 10
	// maxMembersPerSocket caps mesh size: beyond this, P2P quality collapses.
	// Clients should stay at or below ~6; the server refuses past 8.
	maxMembersPerSocket = 8
	maxUserIDLength     = 128
)

// processRelay is the process-wide message router behind the MessageRouter
// seam. It is set once and never reassigned; handlers take it as a parameter
// so tests can inject isolated instances.
var processRelay MessageRouter = NewMeshRelay()

// parseOrigins splits WS_ALLOWED_ORIGINS ("https://a,https://b") into a list.
func parseOrigins(raw string) []string {
	var out []string
	for _, o := range strings.Split(raw, ",") {
		if o = strings.TrimSpace(o); o != "" {
			out = append(out, o)
		}
	}
	return out
}

func originAllowed(r *http.Request) bool {
	allowed := parseOrigins(os.Getenv("WS_ALLOWED_ORIGINS"))
	if len(allowed) == 0 {
		return true // dev default; main logs a warning at startup
	}
	origin := r.Header.Get("Origin")
	for _, o := range allowed {
		if o == origin {
			return true
		}
	}
	return false
}

var upgrader = websocket.Upgrader{
	CheckOrigin: originAllowed,
}

func validMessageType(t string) bool {
	switch t {
	case "connect", "disconnect", "offer", "answer", "ice-candidate":
		return true
	default:
		return false
	}
}

func wshandler(w http.ResponseWriter, r *http.Request, socket string, relay MessageRouter) {
	if socket == "" {
		http.Error(w, "missing socket", http.StatusBadRequest)
		return
	}
	if len(relay.Members(socket)) >= maxMembersPerSocket {
		http.Error(w, "room is full", http.StatusTooManyRequests)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("Error handling websocket connection", "error", err)
		return
	}
	defer conn.Close()

	conn.SetReadLimit(wsReadLimit)
	conn.SetReadDeadline(time.Now().Add(wsPongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(wsPongWait))
		return nil
	})

	limiter := newRateLimiter(100, 5*time.Second)

	var self *interfaces.Connection
	userID := ""
	// done stops the ping loop; pinging starts once the member registers.
	done := make(chan struct{})
	defer close(done)

	// On any exit, make sure a registered member is announced gone.
	defer func() {
		if userID != "" {
			relay.Route(socket, userID, interfaces.Message{Type: "disconnect", UserID: userID})
			relay.Remove(socket, userID)
			wsConnectionsCurrent.Dec()
		}
	}()

	for {
		var message interfaces.Message
		if err := conn.ReadJSON(&message); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				slog.Error("WebSocket read error", "error", err)
			}
			return
		}
		if !limiter.allow() {
			slog.Warn("rate limit exceeded", "socket", socket)
			return
		}
		if len(message.UserID) == 0 || len(message.UserID) > maxUserIDLength {
			slog.Warn("invalid user id", "socket", socket)
			return
		}
		if !validMessageType(message.Type) {
			continue // ignore unknown types for protocol forward-compat
		}
		wsMessagesTotal.WithLabelValues(message.Type).Inc()

		if userID == "" {
			// First valid message registers the member.
			userID = message.UserID
			self = relay.Add(socket, userID, conn)
			wsConnectionsCurrent.Inc()
			go pingLoop(self, done)
			// Presence, both directions: tell the room, and tell the
			// newcomer who is already here so late media can connect.
			relay.Route(socket, userID, interfaces.Message{Type: "connect", UserID: userID})
			for _, member := range relay.Members(socket) {
				if member != userID {
					_ = self.Send(interfaces.Message{Type: "connect", UserID: member})
				}
			}
		} else if message.UserID != userID {
			// One connection, one identity: spoofed sender ids are dropped.
			continue
		}

		switch message.Type {
		case "connect":
			// Ack preserved for older clients; presence already relayed.
			_ = self.Send(interfaces.Message{
				Type:   "session_joined",
				UserID: userID,
			})
		case "disconnect":
			relay.Route(socket, userID, message)
			relay.Remove(socket, userID)
			wsConnectionsCurrent.Dec()
			userID = ""
			return
		default:
			relay.Route(socket, userID, message)
		}
	}
}

func pingLoop(c *interfaces.Connection, done <-chan struct{}) {
	ticker := time.NewTicker(wsPingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			if err := c.Ping(); err != nil {
				return
			}
		}
	}
}

func main() {
	// Set up logging
	// Structured JSON logs to stdout (12-factor; collectors scrape the stream).
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	// CORS configuration: restrictive in production (WS_ALLOWED_ORIGINS),
	// permissive only for local development.
	config := cors.DefaultConfig()
	if origins := parseOrigins(os.Getenv("WS_ALLOWED_ORIGINS")); len(origins) > 0 {
		config.AllowOrigins = origins
	} else {
		config.AllowAllOrigins = true
		slog.Warn("WS_ALLOWED_ORIGINS unset; CORS and WebSocket origins allow all (dev mode)")
	}
	config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(cors.New(config))

	// MongoDB connection
	credential := options.Credential{
		Username: getenv("DB_USERNAME", "root"),
		Password: getenv("DB_PASSWORD", "rootpassword"),
	}

	dbHost := getenv("DB_HOST", "localhost")
	dbPort := getenv("DB_PORT", "27017")
	clientOptions := options.Client().ApplyURI("mongodb://" + dbHost + ":" + dbPort).SetAuth(credential)

	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		log.Fatal("Failed to connect to MongoDB:", err)
	}

	// Bound the startup ping so a missing DB fails fast with a clear log
	// instead of hanging the boot (and the container) indefinitely.
	pingCtx, pingCancel := context.WithTimeout(context.Background(), 10*time.Second)
	err = client.Ping(pingCtx, nil)
	pingCancel()
	if err != nil {
		log.Fatal("Failed to ping MongoDB:", err)
	}

	controllers.EnsureIndexes(client)

	slog.Info("MongoDB connection established successfully")

	// Middleware to inject database client
	router.Use(func(context *gin.Context) {
		context.Set("db", client)
		context.Next()
	})
	router.Use(metricsMiddleware("signalling-server"))

	// Routes
	router.POST("/session", controllers.CreateSession)
	router.GET("/connect", controllers.GetSession)
	router.POST("/connect/:url", controllers.ConnectSession)

	// /health is shallow (process alive); /ready is deep (dependencies reachable).
	router.GET("/health", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{
			"status":  "healthy",
			"service": "signalling-server",
		})
	})
	router.GET("/ready", func(ctx *gin.Context) {
		pingCtx, cancel := context.WithTimeout(ctx.Request.Context(), 2*time.Second)
		defer cancel()
		if err := client.Ping(pingCtx, readpref.Primary()); err != nil {
			ctx.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready"})
			return
		}
		ctx.JSON(200, gin.H{"status": "ready"})
	})
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	router.GET("/ws/:socket", func(c *gin.Context) {
		socket := c.Param("socket")
		wshandler(c.Writer, c.Request, socket, processRelay)
	})

	port := getenv("PORT", "8080")
	slog.Info("Signalling server starting", "port", port)
	router.Run(":" + port)
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	return value
}
