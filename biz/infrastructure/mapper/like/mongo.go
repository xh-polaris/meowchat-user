package like

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

const prefixLikeCacheKey = "cache:like:"
const CollectionName = "like"

var _ IMongoMapper = (*MongoMapper)(nil)

type (
	IMongoMapper interface {
		Insert(ctx context.Context, data *Like) error
		FindOne(ctx context.Context, id string) (*Like, error)
		Update(ctx context.Context, data *Like) error
		Delete(ctx context.Context, id string) error
		GetUserLike(ctx context.Context, userId string, targetId string, targetType int64) error
		GetUserLikes(ctx context.Context, userId string, targetType int64) ([]*Like, error)
		GetTargetLikes(ctx context.Context, targetId string, targetType int64) ([]*Like, error)
		GetId(ctx context.Context, userId string, targetId string, targetType int64) (string, error)
	}

	MongoMapper struct {
		conn *monc.Model
	}

	Like struct {
		ID           primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
		UserId       string             `bson:"userId,omitempty" json:"userId,omitempty"`
		TargetId     string             `bson:"targetId,omitempty" json:"targetId,omitempty"`
		TargetType   int64              `bson:"targetType,omitempty" json:"targetType,omitempty"`
		AssociatedId string             `bson:"associatedId,omitempty" json:"associatedId,omitempty"`
		UpdateAt     time.Time          `bson:"updateAt,omitempty" json:"updateAt,omitempty"`
		CreateAt     time.Time          `bson:"createAt,omitempty" json:"createAt,omitempty"`
	}
)

func NewMongoModel(config *config.Config) IMongoMapper {
	conn := monc.MustNewModel(config.Mongo.URL, config.Mongo.DB, CollectionName, config.CacheConf)
	return &MongoMapper{
		conn: conn,
	}
}

func (m *MongoMapper) GetUserLikes(ctx context.Context, userId string, targetType int64) ([]*Like, error) {
	data := make([]*Like, 0)
	err := m.conn.Find(ctx, &data,
		bson.M{consts.UserId: userId, consts.TargetType: targetType},
		&options.FindOptions{Sort: bson.M{consts.ID: -1}},
	)
	if err != nil {
		return nil, err
	} else {
		return data, nil
	}
}

func (m *MongoMapper) GetId(ctx context.Context, userId string, targetId string, targetType int64) (id string, err error) {
	like := Like{}
	err = m.conn.FindOneNoCache(ctx, &like, bson.M{consts.UserId: userId, consts.TargetId: targetId, consts.TargetType: targetType})
	id = like.ID.Hex()
	return
}

func (m *MongoMapper) GetTargetLikes(ctx context.Context, targetId string, targetType int64) ([]*Like, error) {
	data := make([]*Like, 0)
	err := m.conn.Find(ctx, &data, bson.M{consts.TargetId: targetId, consts.TargetType: targetType})
	if err != nil {
		return nil, err
	} else {
		return data, nil
	}
}

func (m *MongoMapper) GetUserLike(ctx context.Context, userId string, targetId string, targetType int64) (err error) {
	like := Like{}
	err = m.conn.FindOneNoCache(ctx, &like, bson.M{consts.UserId: userId, consts.TargetId: targetId, consts.TargetType: targetType})
	return
}

func (m *MongoMapper) Insert(ctx context.Context, data *Like) error {
	if data.ID.IsZero() {
		data.ID = primitive.NewObjectID()
		data.CreateAt = time.Now()
		data.UpdateAt = time.Now()
	}

	key := prefixLikeCacheKey + data.ID.Hex()
	_, err := m.conn.InsertOne(ctx, key, data)
	return err
}

func (m *MongoMapper) FindOne(ctx context.Context, id string) (*Like, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, consts.ErrInvalidObjectId
	}

	var data Like
	key := prefixLikeCacheKey + id
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

func (m *MongoMapper) Update(ctx context.Context, data *Like) error {
	data.UpdateAt = time.Now()
	key := prefixLikeCacheKey + data.ID.Hex()
	_, err := m.conn.UpdateOne(ctx, key, bson.M{consts.ID: data.ID}, bson.M{"$set": data})
	return err
}

func (m *MongoMapper) Delete(ctx context.Context, id string) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return consts.ErrInvalidObjectId
	}
	key := prefixLikeCacheKey + id
	_, err = m.conn.DeleteOne(ctx, key, bson.M{consts.ID: oid})
	return err
}
