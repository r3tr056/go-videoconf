package controllers

import (
	"github.com/gin-gonic/gin"
)

type User struct {
}

func (u *User) Authenticate(ctx *gin.Context) {
	username := ctx.PostForm("user")
	password := ctx.PostForm("password")

	var err error
	_, err = u.
}
