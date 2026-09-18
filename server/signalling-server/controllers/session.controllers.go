package controllers

import (
	"net/http"

	"github.com/r3tr056/go-videoconf/signalling-server/interfaces"
	"github.com/r3tr056/go-videoconf/signalling-server/utils"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func CreateSession(ctx *gin.Context) {
	db := ctx.MustGet("db").(*mongo.Client)
	collection := db.Database("vidchat").Collection("sessions")

	var session interfaces.Session
	if err := ctx.ShouldBindJSON(&session); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	session.Password = utils.HashPassword(session.Password)

	result, err := collection.InsertOne(ctx, session)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create session."})
		return
	}
	insertedID, ok := result.InsertedID.(primitive.ObjectID)
	if !ok {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create session."})
		return
	}

	url, err := CreateSocket(session, ctx, insertedID.Hex())
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create session."})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"socket": url})
}
