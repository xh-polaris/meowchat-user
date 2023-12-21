package like

import (
	"context"
	"github.com/zeromicro/go-zero/core/mr"
	"sync"
	"time"

	"github.com/xh-polaris/gopkg/pagination"
	"github.com/xh-polaris/gopkg/pagination/mongop"
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
		FindMany(ctx context.Context, fopts *FilterOptions, popts *pagination.PaginationOptions, sorter mongop.MongoCursor) ([]*Like, error)
		Count(ctx context.Context, filter *FilterOptions) (int64, error)
		FindManyAndCount(ctx context.Context, fopts *FilterOptions, popts *pagination.PaginationOptions, sorter mongop.MongoCursor) ([]*Like, int64, error)
		GetUserLike(ctx context.Context, userId string, targetId string, targetType int64) error
		GetUserLikes(ctx context.Context, userId string, targetType int64, popts *pagination.PaginationOptions, sorter mongop.MongoCursor) ([]*Like, int64, error)
		FindUserLikes(ctx context.Context, userId string, targetType int64, popts *pagination.PaginationOptions, sorter mongop.MongoCursor) ([]*Like, error)
		CountUserLikes(ctx context.Context, userId string, targetType int64) (int64, error)
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

func (m *MongoMapper) GetUserLikes(ctx context.Context, userId string, targetType int64, popts *pagination.PaginationOptions, sorter mongop.MongoCursor) ([]*Like, int64, error) {
	var data []*Like
	var total int64
	wg := sync.WaitGroup{}
	wg.Add(2)
	c := make(chan error)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		defer wg.Done()
		var err error
		data, err = m.FindUserLikes(ctx, userId, targetType, popts, sorter)
		if err != nil {
			c <- err
			return
		}
	}()
	go func() {
		defer wg.Done()
		var err error
		total, err = m.CountUserLikes(ctx, userId, targetType)
		if err != nil {
			c <- err
			return
		}
	}()
	go func() {
		wg.Wait()
		defer close(c)
	}()
	if err := <-c; err != nil {
		return nil, 0, err
	}
	return data, total, nil
}

func (m *MongoMapper) FindUserLikes(ctx context.Context, userId string, targetType int64, popts *pagination.PaginationOptions, sorter mongop.MongoCursor) ([]*Like, error) {
	p := mongop.NewMongoPaginator(pagination.NewRawStore(sorter), popts)

	filter := bson.M{consts.UserId: userId, consts.TargetType: targetType}
	sort, err := p.MakeSortOptions(ctx, filter)
	if err != nil {
		return nil, err
	}

	var data []*Like
	if err = m.conn.Find(ctx, &data, filter, &options.FindOptions{
		Sort:  sort,
		Limit: popts.Limit,
		Skip:  popts.Offset,
	}); err != nil {
		return nil, err
	}

	// 如果是反向查询，反转数据
	if *popts.Backward {
		for i := 0; i < len(data)/2; i++ {
			data[i], data[len(data)-i-1] = data[len(data)-i-1], data[i]
		}
	}
	if len(data) > 0 {
		err = p.StoreCursor(ctx, data[0], data[len(data)-1])
		if err != nil {
			return nil, err
		}
	}
	return data, nil
}

func (m *MongoMapper) CountUserLikes(ctx context.Context, userId string, targetType int64) (int64, error) {
	f := bson.M{consts.UserId: userId, consts.TargetType: targetType}
	return m.conn.CountDocuments(ctx, f)
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

func (m *MongoMapper) FindMany(ctx context.Context, fopts *FilterOptions, popts *pagination.PaginationOptions, sorter mongop.MongoCursor) ([]*Like, error) {
	p := mongop.NewMongoPaginator(pagination.NewRawStore(sorter), popts)
	filter := makeMongoFilter(fopts)
	sort, err := p.MakeSortOptions(ctx, filter)
	if err != nil {
		return nil, err
	}
	var data []*Like
	if err = m.conn.Find(ctx, &data, filter, &options.FindOptions{
		Sort:  sort,
		Limit: popts.Limit,
		Skip:  popts.Offset,
	}); err != nil {
		return nil, err
	}

	// 如果是反向查询，反转数据
	if *popts.Backward {
		for i := 0; i < len(data)/2; i++ {
			data[i], data[len(data)-i-1] = data[len(data)-i-1], data[i]
		}
	}
	if len(data) > 0 {
		err = p.StoreCursor(ctx, data[0], data[len(data)-1])
		if err != nil {
			return nil, err
		}
	}
	return data, nil
}

func (m *MongoMapper) FindManyAndCount(ctx context.Context, fopts *FilterOptions, popts *pagination.PaginationOptions, sorter mongop.MongoCursor) ([]*Like, int64, error) {
	var data []*Like
	var total int64
	if err := mr.Finish(func() error {
		var err error
		data, err = m.FindMany(ctx, fopts, popts, sorter)
		return err
	}, func() error {
		var err error
		total, err = m.Count(ctx, fopts)
		return err
	}); err != nil {
		return nil, 0, err
	}
	return data, total, nil
}

func (m *MongoMapper) Count(ctx context.Context, filter *FilterOptions) (int64, error) {
	f := makeMongoFilter(filter)
	return m.conn.CountDocuments(ctx, f)
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
