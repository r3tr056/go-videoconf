package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/r3tr056/go-videoconf/users-service/dao"
	"github.com/r3tr056/go-videoconf/users-service/database"
	"github.com/r3tr056/go-videoconf/users-service/utils"
	"gopkg.in/mgo.v2/bson"
)

type User struct {
	userDao *dao.User
	utils   *utils.Utils
}

func (u *User) Authenticate(ctx *gin.Context) {
	var credentials struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := ctx.ShouldBindJSON(&credentials); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Find user by name
	users, err := u.userDao.GetAll()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	var foundUser *database.UserModel
	for _, user := range users {
		if user.Name == credentials.Username && user.Password == credentials.Password {
			foundUser = &user
			break
		}
	}

	if foundUser == nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Generate JWT token
	if u.utils == nil {
		u.utils = &utils.Utils{}
	}
	token, err := u.utils.GenerateJWT(foundUser.Name, "user")
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Token generation failed"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"id":   foundUser.ID.Hex(),
			"name": foundUser.Name,
		},
	})
}

func (u *User) GetUsers(ctx *gin.Context) {
	if u.userDao == nil {
		u.userDao = &dao.User{}
	}

	users, err := u.userDao.GetAll()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, users)
}

func (u *User) GetUser(ctx *gin.Context) {
	id := ctx.Param("id")
	
	if u.userDao == nil {
		u.userDao = &dao.User{}
	}

	user, err := u.userDao.GetByID(id)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	ctx.JSON(http.StatusOK, user)
}

func (u *User) CreateUser(ctx *gin.Context) {
	var newUser database.AddUser
	if err := ctx.ShouldBindJSON(&newUser); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := newUser.Validate(); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create user in database
	user := database.UserModel{
		ID:       bson.NewObjectId(),
		Name:     newUser.Name,
		Password: newUser.Password,
	}

	sessionCopy := database.Database.MgDBSession.Copy()
	defer sessionCopy.Close()

	collection := sessionCopy.DB(database.Database.DatabaseName).C("users")
	err := collection.Insert(&user)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"id":   user.ID.Hex(),
		"name": user.Name,
	})
}

func (u *User) UpdateUser(ctx *gin.Context) {
	id := ctx.Param("id")
	
	var updateUser database.AddUser
	if err := ctx.ShouldBindJSON(&updateUser); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := updateUser.Validate(); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if u.utils == nil {
		u.utils = &utils.Utils{}
	}

	if err := u.utils.ValidateObjectId(id); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	sessionCopy := database.Database.MgDBSession.Copy()
	defer sessionCopy.Close()

	collection := sessionCopy.DB(database.Database.DatabaseName).C("users")
	err := collection.Update(
		bson.M{"_id": bson.ObjectIdHex(id)},
		bson.M{"$set": bson.M{
			"name":     updateUser.Name,
			"password": updateUser.Password,
		}},
	)

	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "User updated successfully"})
}

func (u *User) DeleteUser(ctx *gin.Context) {
	id := ctx.Param("id")
	
	if u.userDao == nil {
		u.userDao = &dao.User{}
	}

	err := u.userDao.DeleteByID(id)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}
