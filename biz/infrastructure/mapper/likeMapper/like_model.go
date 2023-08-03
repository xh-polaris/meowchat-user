package likeMapper

import (
	"context"
	"github.com/google/wire"
	"github.com/zeromicro/go-zero/core/stores/monc"
	"go.mongodb.org/mongo-driver/bson"
	"meowchat-user/biz/infrastructure/config"
	"meowchat-user/biz/infrastructure/data/db"
)

const LikeCollectionName = "like"

var _ LikeModel = (*CustomLikeModel)(nil)

type (
	// LikeModel is an interface to be customized, add more methods here,
	// and implement the added methods in CustomLikeModel.
	LikeModel interface {
		likeModel
		GetUserLike(ctx context.Context, userId string, targetId string, targetType int64) error
		GetUserLikes(ctx context.Context, userId string, targetType int64) ([]*db.Like, error)
		GetTargetLikes(ctx context.Context, targetId string, targetType int64) ([]*db.Like, error)
		GetId(ctx context.Context, userId string, targetId string, targetType int64) (string, error)
	}

	CustomLikeModel struct {
		*defaultLikeModel
	}
)

var LikeSet = wire.NewSet(
	NewLikeModel,
)

func (m *CustomLikeModel) GetUserLikes(ctx context.Context, userId string, targetType int64) ([]*db.Like, error) {
	data := make([]*db.Like, 0)
	err := m.conn.Find(ctx, &data, bson.M{"userId": userId, "targetType": targetType})
	if err != nil {
		return nil, err
	} else {
		return data, nil
	}
}

func (m *CustomLikeModel) GetId(ctx context.Context, userId string, targetId string, targetType int64) (id string, err error) {
	like := db.Like{}
	err = m.conn.FindOneNoCache(ctx, &like, bson.M{"userId": userId, "targetId": targetId, "targetType": targetType})
	id = like.ID.Hex()
	return
}

func (m *CustomLikeModel) GetTargetLikes(ctx context.Context, targetId string, targetType int64) ([]*db.Like, error) {
	data := make([]*db.Like, 0)
	err := m.conn.Find(ctx, &data, bson.M{"targetId": targetId, "targetType": targetType})
	if err != nil {
		return nil, err
	} else {
		return data, nil
	}
}

func (m *CustomLikeModel) GetUserLike(ctx context.Context, userId string, targetId string, targetType int64) (err error) {
	like := db.Like{}
	err = m.conn.FindOneNoCache(ctx, &like, bson.M{"userId": userId, "targetId": targetId, "targetType": targetType})
	return
}

// NewLikeModel returns a model for the mongo.
func NewLikeModel(config *config.Config) LikeModel {
	conn := monc.MustNewModel(config.Mongo.URL, config.Mongo.LikeDB, LikeCollectionName, config.CacheConf)
	return &CustomLikeModel{
		defaultLikeModel: newDefaultLikeModel(conn),
	}
}
