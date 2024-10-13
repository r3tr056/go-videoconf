

package interfaces

import "go.mongodb.org/mongo-driver/bson/primitive"

type HexID struct {
	ID primitive.ObjectID `bson:"_id"`
}