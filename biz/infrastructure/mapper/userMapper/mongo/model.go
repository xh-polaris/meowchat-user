package mongo

import (
	"context"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"meowchat-user/biz/infrastructure/config"
	"meowchat-user/biz/infrastructure/consts"
	"meowchat-user/biz/infrastructure/data/db"
	"time"

	"github.com/zeromicro/go-zero/core/stores/monc"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const prefixUserCacheKey = "cache:user:"
const (
	UserCollectionName = "user"
)

type (
	// UserMongoModel is an interface to be customized, add more methods here,
	// and implement the added methods in defaultUserModel.
	UserMongoModel interface {
		Insert(ctx context.Context, data *db.User) error
		FindOne(ctx context.Context, id string) (*db.User, error)
		Update(ctx context.Context, data *db.User) error
		Delete(ctx context.Context, id string) error
		UpsertUser(ctx context.Context, data *db.User) error
	}

	defaultUserModel struct {
		conn *monc.Model
	}
)

// NewUserModel returns a model for the mongo.
func NewUserModel(config *config.Config) UserMongoModel {
	conn := monc.MustNewModel(config.Mongo.URL, config.Mongo.UserDB, UserCollectionName, config.CacheConf)
	return &defaultUserModel{
		conn: conn,
	}
}

func (m *defaultUserModel) UpsertUser(ctx context.Context, data *db.User) error {
	key := prefixUserCacheKey + data.ID.Hex()

	filter := bson.M{
		db.ID: data.ID,
	}

	set := bson.M{
		db.UpdateAt: time.Now(),
	}
	if data.Nickname != "" {
		set[db.Nickname] = data.Nickname
	}
	if data.AvatarUrl != "" {
		set[db.AvatarUrl] = data.AvatarUrl
	}
	if data.Motto != "" {
		set[db.Motto] = data.Motto
	}

	update := bson.M{
		"$set": set,
		"$setOnInsert": bson.M{
			db.ID:       data.ID,
			db.CreateAt: time.Now(),
		},
	}

	option := options.UpdateOptions{}
	option.SetUpsert(true)

	_, err := m.conn.UpdateOne(ctx, key, filter, update, &option)
	return err
}

func (m *defaultUserModel) Insert(ctx context.Context, data *db.User) error {
	if data.ID.IsZero() {
		data.ID = primitive.NewObjectID()
		data.CreateAt = time.Now()
		data.UpdateAt = time.Now()
	}

	key := prefixUserCacheKey + data.ID.Hex()
	_, err := m.conn.InsertOne(ctx, key, data)
	return err
}

func (m *defaultUserModel) FindOne(ctx context.Context, id string) (*db.User, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, consts.ErrInvalidObjectId
	}

	var data db.User
	key := prefixUserCacheKey + id
	err = m.conn.FindOne(ctx, key, &data, bson.M{db.ID: oid})
	switch err {
	case nil:
		return &data, nil
	case monc.ErrNotFound:
		return nil, consts.ErrNotFound
	default:
		return nil, err
	}
}

func (m *defaultUserModel) Update(ctx context.Context, data *db.User) error {
	data.UpdateAt = time.Now()
	key := prefixUserCacheKey + data.ID.Hex()
	_, err := m.conn.UpdateOne(ctx, key, bson.M{db.ID: data.ID}, bson.M{"$set": data})
	return err
}

func (m *defaultUserModel) Delete(ctx context.Context, id string) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return consts.ErrInvalidObjectId
	}
	key := prefixUserCacheKey + id
	_, err = m.conn.DeleteOne(ctx, key, bson.M{db.ID: oid})
	return err
}
