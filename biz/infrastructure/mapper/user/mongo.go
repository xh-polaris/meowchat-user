package user

import (
	"context"
	"time"

	"github.com/zeromicro/go-zero/core/stores/monc"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/xh-polaris/meowchat-user/biz/infrastructure/config"
	"github.com/xh-polaris/meowchat-user/biz/infrastructure/consts"
)

const (
	prefixUserCacheKey = "cache:user:"
	CollectionName     = "user"
)

type (
	// IMongoMapper is an interface to be customized, add more methods here,
	// and implement the added methods in MongoMapper.
	IMongoMapper interface {
		Insert(ctx context.Context, data *User) error
		FindOne(ctx context.Context, id string) (*User, error)
		Update(ctx context.Context, data *User) error
		Delete(ctx context.Context, id string) error
		UpsertUser(ctx context.Context, data *User) error
	}

	MongoMapper struct {
		conn *monc.Model
	}

	User struct {
		ID        primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
		AvatarUrl string             `bson:"avatarUrl,omitempty" json:"avatar_url,omitempty"`
		Nickname  string             `bson:"nickname,omitempty" json:"nickname,omitempty"`
		Motto     string             `bson:"motto,omitempty" json:"motto,omitempty"`
		UpdateAt  time.Time          `bson:"updateAt,omitempty" json:"updateAt,omitempty"`
		CreateAt  time.Time          `bson:"createAt,omitempty" json:"createAt,omitempty"`
		// 仅ES查询时使用
		Score_ float64 `bson:"_score,omitempty" json:"_score,omitempty"`
	}
)

func NewMongoMapper(config *config.Config) IMongoMapper {
	conn := monc.MustNewModel(config.Mongo.URL, config.Mongo.DB, CollectionName, config.CacheConf)
	return &MongoMapper{
		conn: conn,
	}
}

func (m *MongoMapper) UpsertUser(ctx context.Context, data *User) error {
	key := prefixUserCacheKey + data.ID.Hex()

	filter := bson.M{
		consts.ID: data.ID,
	}

	set := bson.M{
		consts.UpdateAt: time.Now(),
	}
	if data.Nickname != "" {
		set[consts.Nickname] = data.Nickname
	}
	if data.AvatarUrl != "" {
		set[consts.AvatarUrl] = data.AvatarUrl
	}
	if data.Motto != "" {
		set[consts.Motto] = data.Motto
	}

	update := bson.M{
		"$set": set,
		"$setOnInsert": bson.M{
			consts.ID:       data.ID,
			consts.CreateAt: time.Now(),
		},
	}

	option := options.UpdateOptions{}
	option.SetUpsert(true)

	_, err := m.conn.UpdateOne(ctx, key, filter, update, &option)
	return err
}

func (m *MongoMapper) Insert(ctx context.Context, data *User) error {
	if data.ID.IsZero() {
		data.ID = primitive.NewObjectID()
		data.CreateAt = time.Now()
		data.UpdateAt = time.Now()
	}

	key := prefixUserCacheKey + data.ID.Hex()
	_, err := m.conn.InsertOne(ctx, key, data)
	return err
}

func (m *MongoMapper) FindOne(ctx context.Context, id string) (*User, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, consts.ErrInvalidObjectId
	}

	var data User
	key := prefixUserCacheKey + id
	err = m.conn.FindOne(ctx, key, &data, bson.M{consts.ID: oid})
	switch err {
	case nil:
		return &data, nil
	case monc.ErrNotFound:
		return nil, consts.ErrNotFound
	default:
		return nil, err
	}
}

func (m *MongoMapper) Update(ctx context.Context, data *User) error {
	data.UpdateAt = time.Now()
	key := prefixUserCacheKey + data.ID.Hex()
	_, err := m.conn.UpdateOne(ctx, key, bson.M{consts.ID: data.ID}, bson.M{"$set": data})
	return err
}

func (m *MongoMapper) Delete(ctx context.Context, id string) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return consts.ErrInvalidObjectId
	}
	key := prefixUserCacheKey + id
	_, err = m.conn.DeleteOne(ctx, key, bson.M{consts.ID: oid})
	return err
}
