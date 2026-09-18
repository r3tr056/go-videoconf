package interfaces

type Session struct {
	Host     string `bson:"host" json:"host"`
	Title    string `bson:"title" json:"title"`
	Password string `bson:"password" json:"password"`
}
