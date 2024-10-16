package database

import (
	"log"
	"time"

	"github.com/r3tr056/go-videoconf/users-service/common"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var (
	Database MongoDB
)

type MongoDB struct {
	MgDBSession  *mgo.Session
	DatabaseName string
}

func (db *MongoDB) Init() error {
	db.DatabaseName = common.MgDBName

	dialInfo := &mgo.DialInfo{
		Addrs:    []string{common.MgAddress},
		Timeout:  60 * time.Second,
		Database: db.DatabaseName,
		Username: common.MgUsername,
		Password: common.MgPassword,
	}

	var err error
	db.MgDBSession, err = mgo.DialWithInfo(dialInfo)

	if err != nil {
		log.Print("Can't connect to mongo, go error:", err)
		return err
	}

	return db.initData()
}

func (db *MongoDB) initData() error {
	var err error
	var count int

	sessionCopy := db.MgDBSession.Copy()
	defer sessionCopy.Close()

	collection := sessionCopy.DB(db.DatabaseName).C(common.UsersCol)
	count, err = collection.Find(bson.M{}).Count()

	if count < 1 {
		user := UserModel{bson.NewObjectId(), "admin", "admin"}
		err = collection.Insert(&user)
	}

	return err
}

func (db *MongoDB) Close() {
	if db.MgDBSession != nil {
		db.MgDBSession.Close()
	}
}
