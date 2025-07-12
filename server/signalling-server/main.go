package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/r3tr056/go-videoconf/signalling-server/controllers"
	"github.com/r3tr056/go-videoconf/signalling-server/interfaces"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var sockets = make(map[string]map[string]*interfaces.Connection)

func wshandler(w http.ResponseWriter, r *http.Request, socket string) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Error handling websocket connection: %v", err)
		return
	}

	defer conn.Close()

	if sockets[socket] == nil {
		sockets[socket] = make(map[string]*interfaces.Connection)
	}

	clients := sockets[socket]

	var message interfaces.Message
	for {
		err = conn.ReadJSON(&message)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}

		if clients[message.UserID] == nil {
			connection := new(interfaces.Connection)
			connection.Socket = conn
			clients[message.UserID] = connection
		}

		switch message.Type {
		case "connect":
			message.Type = "session_joined"
			err := conn.WriteJSON(message)
			if err != nil {
				log.Printf("Websocket error: %s", err)
				delete(clients, message.UserID)
			}

		case "disconnect":
			for user, client := range clients {
				if user != message.UserID {
					err := client.Send(message)
					if err != nil {
						client.Socket.Close()
						delete(clients, user)
					}
				}
			}
			delete(clients, message.UserID)
		default:
			// Relay message to all other clients
			for user, client := range clients {
				if user != message.UserID {
					err := client.Send(message)
					if err != nil {
						delete(clients, user)
					}
				}
			}
		}
	}
}

func main() {
	// Set up logging
	file, err := os.OpenFile("info.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	log.SetOutput(file)

	// CORS configuration
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}

	router := gin.Default()
	router.Use(cors.New(config))

	// MongoDB connection
	credential := options.Credential{
		Username: getenv("DB_USERNAME", "root"),
		Password: getenv("DB_PASSWORD", "rootpassword"),
	}
	
	dbHost := getenv("DB_URL", "localhost")
	dbPort := getenv("DB_PORT", "27017")
	clientOptions := options.Client().ApplyURI("mongodb://" + dbHost + ":" + dbPort).SetAuth(credential)
	
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		log.Fatal("Failed to connect to MongoDB:", err)
	}

	err = client.Ping(context.TODO(), nil)
	if err != nil {
		log.Fatal("Failed to ping MongoDB:", err)
	}

	log.Println("MongoDB connection established successfully")

	// Middleware to inject database client
	router.Use(func(context *gin.Context) {
		context.Set("db", client)
		context.Next()
	})

	// Routes
	router.POST("/session", controllers.CreateSession)
	router.GET("/connect", controllers.GetSession)
	router.POST("/connect/:url", controllers.ConnectSession)
	
	router.GET("/health", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{
			"status":  "healthy",
			"service": "signalling-server",
		})
	})

	router.GET("/ws/:socket", func(c *gin.Context) {
		socket := c.Param("socket")
		wshandler(c.Writer, c.Request, socket)
	})

	port := getenv("PORT", "8080")
	log.Printf("Signalling server starting on port %s", port)
	router.Run(":" + port)
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	return value
}
