package controllers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/r3tr056/go-videoconf/users-service/dao"
	"github.com/r3tr056/go-videoconf/users-service/database"
	"github.com/r3tr056/go-videoconf/users-service/utils"
	"gopkg.in/mgo.v2/bson"
)

type User struct {
	utils   utils.Utils
	userDAO dao.User
}

func (u *User) Authenticate(ctx *gin.Context) {
	username := ctx.PostForm("user")
	password := ctx.PostForm("password")

	var err error
	_, err = u.userDAO.Login(username, password)

	if err == nil {
		var tokenString string
		tokenString, err = u.utils.GenerateJWT(username, "")
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, utils.HTTPError{Code: -1, Message: err.Error()})
			log.Printf("[ERROR]: ", err)
			return
		}

		token := database.Token{Token: tokenString}
		ctx.JSON(http.StatusOK, token)
	} else {
		ctx.JSON(http.StatusUnauthorized, utils.HTTPError{Code: -1, Message: err.Error()})
	}
}

func (u *User) AddUser(ctx *gin.Context) {
	var addUser database.AddUser
	if err := ctx.ShouldBindJSON(&addUser); err != nil {
		ctx.JSON(http.StatusInternalServerError, utils.HTTPError{Code: -1, Message: err.Error()})
		return
	}

	if err := addUser.Validate(); err != nil {
		ctx.JSON(http.StatusBadRequest, utils.HTTPError{Code: -1, Message: err.Error()})
		return
	}

	user := database.UserModel{bson.NewObjectId(), addUser.Name, addUser.Password}
	err := u.userDAO.Insert(user)
	if err == nil {
		ctx.JSON(http.StatusOK, bson.M{"Message": "Successfully"})
	} else {
		ctx.JSON(http.StatusInternalServerError)
	}
}
