package like

import (
	"github.com/xh-polaris/meowchat-user/biz/infrastructure/consts"
	"go.mongodb.org/mongo-driver/bson"
)

type FilterOptions struct {
	OnlyTargetId   *string
	OnlyTargetType *int32
}

type MongoFilter struct {
	m bson.M
	*FilterOptions
}

func makeMongoFilter(options *FilterOptions) bson.M {
	return (&MongoFilter{
		m:             bson.M{},
		FilterOptions: options,
	}).toBson()
}

func (f *MongoFilter) toBson() bson.M {
	f.CheckOnlyTargetId()
	f.CheckOnlyTargetType()
	return f.m
}

func (f *MongoFilter) CheckOnlyTargetId() {
	if f.OnlyTargetId != nil {
		f.m[consts.TargetId] = *f.OnlyTargetId
	}
}

func (f *MongoFilter) CheckOnlyTargetType() {
	if f.OnlyTargetType != nil {
		f.m[consts.TargetType] = *f.OnlyTargetType
	}
}

//
//type EsFilter struct {
//	q []types.Query
//	*FilterOptions
//}
//
//func makeEsFilter(opts *FilterOptions) []types.Query {
//	return (&EsFilter{
//		q:             make([]types.Query, 0),
//		FilterOptions: opts,
//	}).toQuery()
//}
//
//func (f *EsFilter) toQuery() []types.Query {
//	f.checkOnlyTargetId()
//	f.checkOnlyTargetType()
//	return f.q
//}
//
//func (f *EsFilter) checkOnlyTargetId() {
//	if f.OnlyTargetId != nil {
//		f.q = append(f.q, types.Query{
//			Term: map[string]types.TermQuery{
//				consts.TargetId: {Value: *f.OnlyTargetId},
//			},
//		})
//	}
//}
//
//func (f *EsFilter) checkOnlyTargetType() {
//	if f.OnlyTargetType != nil {
//		f.q = append(f.q, types.Query{
//			Term: map[string]types.TermQuery{
//				consts.TargetType: {Value: *f.OnlyTargetType},
//			},
//		})
//	}
//}
