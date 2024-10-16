package utils

import (
	"errors"
	"time"

	jwt_lib "github.com/dgrijalva/jwt-go"
	"github.com/r3tr056/go-videoconf/users-service/common"
	"gopkg.in/mgo.v2/bson"
)

type StdClaims struct {
	Name string `json:"name"`
	Role string `json:"role"`
	jwt_lib.StandardClaims
}

type Utils struct {
}

func (u *Utils) GenerateJWT(name string, role string) (string, error) {
	claims := StdClaims{
		name,
		role,
		jwt_lib.StandardClaims{
			ExpiresAt: time.Now().Add(time.Hour * 1).Unix(),
			Issuer:    common.Issuer,
		},
	}

	token := jwt_lib.NewWithClaims(jwt_lib.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(common.JwtSecretPassword))

	return tokenString, err
}

func (u *Utils) ValidateObjectId(id string) error {
	if !bson.IsObjectIdHex(id) {
		return errors.New("error object id not hex")
	}
	return nil
}
