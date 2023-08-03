package db

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	AvatarUrl string             `bson:"avatarUrl,omitempty" json:"avatar_url,omitempty"`
	Nickname  string             `bson:"nickname,omitempty" json:"nickname,omitempty"`
	Motto     string             `bson:"motto,omitempty" json:"motto,omitempty"`
	UpdateAt  time.Time          `bson:"updateAt,omitempty" json:"updateAt,omitempty"`
	CreateAt  time.Time          `bson:"createAt,omitempty" json:"createAt,omitempty"`
	// 仅ES查询时使用
	Score_ float64 `bson:"_score,omitempty" json:"_score,omitempty"`
}

const (
	ID        = "_id"
	AvatarUrl = "avatarUrl"
	Nickname  = "nickname"
	Motto     = "motto"
	UpdateAt  = "updateAt"
	CreateAt  = "createAt"
)

const (
	IdSorter = iota
	ScoreSorter
)
