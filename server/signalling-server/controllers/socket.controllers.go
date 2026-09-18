package controllers

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"

	"github.com/r3tr056/go-videoconf/signalling-server/interfaces"
	"github.com/r3tr056/go-videoconf/signalling-server/utils"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func ConnectSession(ctx *gin.Context) {
	db := ctx.MustGet("db").(*mongo.Client)
	collection := db.Database("vidchat").Collection("sockets")

	url := ctx.Param("url")
	result := collection.FindOne(ctx, bson.M{"hashedUrl": url})

	var input interfaces.Session
	if err := ctx.ShouldBindJSON(&input); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if result.Err() != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Socket connection not found."})
		return
	}

	var socket interfaces.Socket
	result.Decode(&socket)

	collection = db.Database("vidchat").Collection("sessions")
	objectID, err := primitive.ObjectIDFromHex(socket.SessionID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Session not found."})
		return
	}

	result = collection.FindOne(ctx, bson.M{"_id": objectID})
	if result.Err() != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Session not found."})
		return
	}

	var session interfaces.Session
	result.Decode(&session)

	if !utils.ComparePasswords(session.Password, []byte(input.Password)) {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid password."})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"title":  session.Title,
		"socket": socket.SocketURL,
	})
}

func GetSession(ctx *gin.Context) {
	urls, ok := ctx.Request.URL.Query()["url"]
	if !ok || len(urls) == 0 || urls[0] == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Missing url query parameter."})
		return
	}

	db := ctx.MustGet("db").(*mongo.Client)
	collection := db.Database("vidchat").Collection("sockets")

	result := collection.FindOne(ctx, bson.M{"hashedUrl": urls[0]})

	if result.Err() != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Socket connection not found."})
		return
	}

	ctx.Status(http.StatusOK)
}

func CreateSocket(session interfaces.Session, ctx *gin.Context, id string) (string, error) {
	db := ctx.MustGet("db").(*mongo.Client)
	collection := db.Database("vidchat").Collection("sockets")

	var socket interfaces.Socket
	hashURL := hashSession(session.Host + session.Title)
	socketURL := hashSession(session.Host + session.Password)
	socket.SessionID = id
	socket.HashedURL = hashURL
	socket.SocketURL = socketURL

	if _, err := collection.InsertOne(ctx, socket); err != nil {
		return "", err
	}

	return hashURL, nil
}

func hashSession(str string) string {
	sum := sha256.Sum256([]byte(str))
	return hex.EncodeToString(sum[:])
}
