package database

import (
	"context"
	"log/slog"
	"time"

	"github.com/r3tr056/go-videoconf/users-service/common"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var (
	Database MongoDB
)

type MongoDB struct {
	Client       *mongo.Client
	DatabaseName string
}

func (db *MongoDB) Init() error {
	db.DatabaseName = common.MgDBName

	credential := options.Credential{
		Username:   common.MgUsername,
		Password:   common.MgPassword,
		AuthSource: "admin",
	}
	clientOptions := options.Client().
		ApplyURI("mongodb://" + common.MgAddress).
		SetAuth(credential)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var err error
	db.Client, err = mongo.Connect(ctx, clientOptions)
	if err != nil {
		slog.Error("Can't connect to mongo", "error", err)
		return err
	}

	if err = db.Client.Ping(ctx, readpref.Primary()); err != nil {
		slog.Error("Can't ping mongo", "error", err)
		return err
	}

	return db.initData()
}

// Users returns the users collection handle.
func (db *MongoDB) Users() *mongo.Collection {
	return db.Client.Database(db.DatabaseName).Collection(common.UsersCol)
}

func (db *MongoDB) initData() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	count, err := db.Users().CountDocuments(ctx, bson.M{})
	if err != nil {
		return err
	}

	if count < 1 {
		user := UserModel{ID: primitive.NewObjectID(), Name: "admin", Password: "admin"}
		if _, err = db.Users().InsertOne(ctx, &user); err != nil {
			return err
		}
	}

	// Indexes (idempotent): authentication looks users up by name.
	_, err = db.Users().Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "name", Value: 1}},
	})
	return err
}

func (db *MongoDB) Close() {
	if db.Client != nil {
		_ = db.Client.Disconnect(context.Background())
	}
}
