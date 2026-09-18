package database

import (
	"encoding/json"
	"errors"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// user model
type UserModel struct {
	ID       primitive.ObjectID `bson:"_id" json:"-"`
	Name     string             `bson:"name" json:"name" example:"ankur"`
	Password string             `bson:"password" json:"password" example:"test123"`
}

// MarshalJSON keeps the historic hex-string "id" shape instead of the
// driver's default {"$oid": ...} encoding.
func (u UserModel) MarshalJSON() ([]byte, error) {
	type wire struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Password string `json:"password"`
	}
	return json.Marshal(wire{ID: u.ID.Hex(), Name: u.Name, Password: u.Password})
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
