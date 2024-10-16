package database

import (
	"errors"

	"gopkg.in/mgo.v2/bson"
)

// user model
type UserModel struct {
	ID       bson.ObjectId `bson:"_id" json:"id"`
	Name     string        `bson:"name" json:"name" example:"ankur"`
	Password string        `bson:"password" json:"password" example:"test123"`
}

// add user information
type AddUser struct {
	Name     string `json:"name" example:"User Name"`
	Password string `json:"password" example:"User Password"`
}

func (a AddUser) Validate() error {
	switch {
	case len(a.Name) == 0:
		return errors.New("name is empty")
	case len(a.Password) == 0:
		return errors.New("password is empty")
	default:
		return nil
	}
}
