package database

import (
	"gopkg.in/mgo.v2/bson"

	"github.com/r3tr056/go-videoconf/users-service/common"
	"github.com/r3tr056/go-videoconf/users-service/database"
	"github.com/r3tr056/go-videoconf/users-service/utils"
)

type User struct {
	utils *utils.Utils
}

func (u *User) GetAll() ([]database.UserModel, error) {
	sessionCopy := database.Database.MgDBSession.Copy()
	defer sessionCopy.Copy()

	collection := sessionCopy.DB(database.Database.DatabaseName).C(common.UsersCol)

	var users []database.UserModel
	err := collection.Find(bson.M{}).All(&users)
	return users, err
}

func (u *User) GetByID(id string) (database.UserModel, error) {
	var err error
	err = u.utils.ValidateObjectId(id)
	if err != nil {
		return database.UserModel{}, err
	}

	sessionCopy := database.Database.MgDBSession.Copy()
	defer sessionCopy.Close()

	collection := sessionCopy.DB(database.Database.DatabaseName).C(common.UsersCol)

	var user database.UserModel
	err = collection.Find(bson.ObjectIdHex(id)).One(&user)
	return user, err
}

func (u *User) DeleteByID(id string) error {
	var err error
	err = u.utils.ValidateObjectId(id)
	if err != nil {
		return err
	}

	sessionCopy := database.Database.MgDBSession.Copy()
	defer sessionCopy.Close()

	collection := sessionCopy.DB(database.Database.DatabaseName).C(common.UsersCol)
	err = collection.Remove(bson.M{"_id": bson.ObjectIdHex(id)})
	return err
}
