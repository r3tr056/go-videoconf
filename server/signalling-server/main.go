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

	"github.com/hashicorp/consul/api"
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
		log.Fatal("Error handling websocket connection.")
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
				err := client.Send(message)
				if err != nil {
					client.Socket.Close()
					delete(clients, user)
				}
			}
			delete(clients, message.UserID)
		default:
			for user, client := range clients {
				err := client.Send(message)
				if err != nil {
					delete(clients, user)
				}
			}
		}
	}
}

func main() {
	file, err := os.OpenFile("info.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}

	defer file.Close()
	log.SetOutput(file)

	config := cors.DefaultConfig()
	config.AllowOrigins = []string{getenv("HOST_URL", "localhost")}

	router := gin.Default()
	router.Use(cors.Default())

	credential := options.Credential{
		Username: "root",
		Password: "rootpassword",
	}
	clientOptions := options.Client().ApplyURI("mongodb://" + getenv("DB_URL", "localhost") + ":" + getenv("DB_PORT", "27017")).SetAuth(credential)
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		log.Fatal(err)
	}

	// Consul Client
	consulConfig := api.DefaultConfig()
	consulConfig.Address = "http://localhost:8500"
	consulClient, err := api.NewClient(consulConfig)
	if err != nil {
		log.Fatal("Error creating Consul client:", err)
	}

	registration := &api.AgentServiceRegistration{
		ID:      "signalling-service",
		Name:    "signalling-service",
		Address: "127.0.0.1",
		Port:    8080,
		Check: &api.AgentServiceCheck{
			HTTP:     "http://127.0.0.1:8080/health",
			Interval: "10s",
		},
	}

	err = consulClient.Agent().ServiceRegister(registration)
	if err != nil {
		log.Fatal("Error registering service with Consul: ", err)
	}

	err = client.Ping(context.TODO(), nil)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("MongoDB connection ok...")

	// middleware - intercept requests to use our db controller
	router.Use(func(context *gin.Context) {
		context.Set("db", client)
		context.Next()
	})

	router.POST("/session", controllers.CreateSession)
	router.GET("/connect", controllers.GetSession)
	router.POST("/connect/:url", controllers.ConnectSession)
	router.GET("/health", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{
			"message": "Service is Healthy",
		})
	})

	router.GET("/ws/:socket", func(c *gin.Context) {
		socket := c.Param("socket")
		wshandler(c.Writer, c.Request, socket)
	})

	router.Run(":" + getenv("PORT", "8080"))
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	return value
}
