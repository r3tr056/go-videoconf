package utils

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/r3tr056/go-videoconf/users-service/common"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type StdClaims struct {
	Name string `json:"name"`
	Role string `json:"role"`
	jwt.RegisteredClaims
}

type Utils struct {
}

func (u *Utils) GenerateJWT(name string, role string) (string, error) {
	claims := StdClaims{
		Name: name,
		Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 1)),
			Issuer:    common.Issuer,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(common.JwtSecretPassword))

	return tokenString, err
}

func (u *Utils) ValidateObjectId(id string) error {
	if _, err := primitive.ObjectIDFromHex(id); err != nil {
		return errors.New("error object id not hex")
	}
	return nil
}
