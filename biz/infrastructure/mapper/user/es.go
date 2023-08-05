package user

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/xh-polaris/meowchat-user/biz/infrastructure/config"
	"github.com/xh-polaris/meowchat-user/biz/infrastructure/consts"
	"github.com/xh-polaris/paginator-go"
	"github.com/xh-polaris/paginator-go/esp"
	"log"
	"net/http"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/typedapi/core/search"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
	"github.com/mitchellh/mapstructure"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type (
	IEsMapper interface {
		SearchUser(ctx context.Context, name string, popts *paginator.PaginationOptions, sorter any) ([]*User, int64, error)
	}

	EsMapper struct {
		es        *elasticsearch.TypedClient
		indexName string
	}
)

func NewEsMapper(config *config.Config) IEsMapper {
	esClient, err := elasticsearch.NewTypedClient(elasticsearch.Config{
		Addresses: config.Elasticsearch.Addresses,
		Username:  config.Elasticsearch.Username,
		Password:  config.Elasticsearch.Password,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	return &EsMapper{
		es:        esClient,
		indexName: fmt.Sprintf("%s.%s-alias", config.Mongo.DB, CollectionName),
	}
}

func (m *EsMapper) SearchUser(ctx context.Context, name string, popts *paginator.PaginationOptions, sorter any) ([]*User, int64, error) {
	p := esp.NewEsPaginator(paginator.NewRawStore(sorter), popts)
	s, sa, err := p.MakeSortOptions(ctx)
	if err != nil {
		return nil, 0, err
	}
	res, err := m.es.Search().From(int(*popts.Offset)).Size(int(*popts.Limit)).Index(m.indexName).Request(&search.Request{
		Query: &types.Query{
			Bool: &types.BoolQuery{
				Must: []types.Query{
					{
						Match: map[string]types.MatchQuery{
							consts.Nickname: {
								Query: name,
							},
						},
					},
				},
			},
		},
		Sort:        s,
		SearchAfter: sa,
	}).Do(ctx)
	if err != nil {
		return nil, 0, err
	}

	hits := res.Hits.Hits
	total := res.Hits.Total.Value
	datas := make([]*User, 0, len(hits))
	for i := range hits {
		hit := hits[i]
		data := &User{}
		var source map[string]any
		err = json.Unmarshal(hit.Source_, &source)
		if err != nil {
			return nil, 0, err
		}
		if source[consts.CreateAt], err = time.Parse("2006-01-02T15:04:05Z07:00", source[consts.CreateAt].(string)); err != nil {
			return nil, 0, err
		}
		if source[consts.UpdateAt], err = time.Parse("2006-01-02T15:04:05Z07:00", source[consts.UpdateAt].(string)); err != nil {
			return nil, 0, err
		}
		err = mapstructure.Decode(source, data)
		if err != nil {
			return nil, 0, err
		}
		oid := hit.Id_
		data.ID, err = primitive.ObjectIDFromHex(oid)
		if err != nil {
			return nil, 0, err
		}
		data.Score_ = float64(hit.Score_)
		datas = append(datas, data)
	}
	// 如果是反向查询，反转数据
	if *popts.Backward {
		for i := 0; i < len(datas)/2; i++ {
			datas[i], datas[len(datas)-i-1] = datas[len(datas)-i-1], datas[i]
		}
	}
	if len(datas) > 0 {
		err = p.StoreSorter(ctx, datas[0], datas[len(datas)-1])
		if err != nil {
			return nil, 0, err
		}
	}
	return datas, total, nil
}
