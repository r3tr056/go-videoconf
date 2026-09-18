package dao

import (
	"context"

	"github.com/r3tr056/go-videoconf/users-service/database"
	"github.com/r3tr056/go-videoconf/users-service/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type User struct {
	utils *utils.Utils
}

func (u *User) GetAll() ([]database.UserModel, error) {
	ctx := context.Background()

	cursor, err := database.Database.Users().Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var users []database.UserModel
	err = cursor.All(ctx, &users)
	return users, err
}

func (u *User) FindByName(name string) (database.UserModel, error) {
	var user database.UserModel
	err := database.Database.Users().FindOne(context.Background(), bson.M{"name": name}).Decode(&user)
	return user, err
}

func (u *User) GetByID(id string) (database.UserModel, error) {
	if u.utils == nil {
		u.utils = &utils.Utils{}
	}

	err := u.utils.ValidateObjectId(id)
	if err != nil {
		return database.UserModel{}, err
	}

	objectID, _ := primitive.ObjectIDFromHex(id)

	var user database.UserModel
	err = database.Database.Users().FindOne(context.Background(), bson.M{"_id": objectID}).Decode(&user)
	return user, err
}

func (u *User) DeleteByID(id string) error {
	if u.utils == nil {
		u.utils = &utils.Utils{}
	}

	err := u.utils.ValidateObjectId(id)
	if err != nil {
		return err
	}

	objectID, _ := primitive.ObjectIDFromHex(id)

	result, err := database.Database.Users().DeleteOne(context.Background(), bson.M{"_id": objectID})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}
