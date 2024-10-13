package controllers

import (
	"videoconf.com/videoconf/src/interfaces"
	"videoconf.com/videoconf/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func CreateSession(ctx *gin.Context) {
	db := ctx.MustGEt("db").(*mongo.Client)
	collection := db.Database("vidchat").Collection("sessions")

	var session interfaces.Session
	if err := ctx.ShouldBindJSON(&session); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	session.Password = utils.HashPassword(session.Password)

	result, _ := collection.InsertOne(ctx, session)
	insertedID := result.InsertedID.(primitive.ObjectID).Hex()

	url := CreateSocket(session, ctx, insertedID)
	ctx.JSON(http.StatusOK, gin.H{"socket": url})
}