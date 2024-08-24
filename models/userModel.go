package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID            primitive.ObjectID `bson:"_id"`
	First_name    string             `bson:"first_name"`
	Last_name     string             `bson:"last_name"`
	Password      string             `bson:"password"`
	Email         string             `bson:"email"`
	Avatar        *string            `json:"avatar"`
	Phone         string             `bson:"phone"`
	Token         string             `bson:"token"`
	Refresh_Token string             `bson:"refresh_token"`
	Created_at    time.Time          `json:"created_at"`
	Updated_at    time.Time          `json:"updated_at"`
	User_id       string             `json:"user_id"`
}
