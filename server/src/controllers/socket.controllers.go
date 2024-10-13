package controllers

import (
	"crypto/sha1"
	"encoding/hex"
	"videoconf.com/videoconf/src/interfaces"
	"videoconf.com/videoconf/src/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func ConnectSession(ctx *gin.Context) {
	db := ctx.MustGet("db").(*mongo.Client);
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
		"title": session.Title,
		"socket": socket.SocketURL,
	})
}

func GetSession(ctx *gin.Context) {
	db :=ctx.MustGet("db").(*mongo.Client)
	collection := db.Database("vidchat").Collection("sockets")

	id := ctx.Request.URL.Query()["url"][0]
	result := collection.FindOne(ctx, bson.M{"hashedUrl": id})

	if result.Err() != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Socket connection not found."})
		return
	}

	ctx.Status(http.StatusOK)
}

func CreateSocket(session interfaces.Session, ctx *gin.Context, id string) string {
	db := ctx.MustGet("db").(*mongo.Client)
	collection := db.Database("vidchat").Collection("sockets")

	var socket interfaces.Socket
	hashURL := hashSession(session.Host + session.Title)
	socketURL := hashSession(session.Host + session.Password)
	socket.SessionID = id
	socket.HashedURL = hashURL
	socket.SocketURL = socketURL

	collection.InsertOne(ctx, socket)

	return hashURL
}

func hashSession(str string) string {
	hash := sha1.New()
	hash.Write([]byte(str))
	return hex.EncodeToString(hash.Sum(nil))
}