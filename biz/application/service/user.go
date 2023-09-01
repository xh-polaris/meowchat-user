package service

import (
	"context"
	"github.com/apache/rocketmq-client-go/v2"
	"github.com/xh-polaris/paginator-go/esp"
	"github.com/zeromicro/go-zero/core/jsonx"
	"net/url"

	"github.com/xh-polaris/meowchat-user/biz/infrastructure/consts"
	usermapper "github.com/xh-polaris/meowchat-user/biz/infrastructure/mapper/user"

	mqprimitive "github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/google/wire"
	"github.com/xh-polaris/paginator-go"
	"github.com/xh-polaris/service-idl-gen-go/kitex_gen/meowchat/user"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UserService interface {
	GetUser(ctx context.Context, req *user.GetUserReq) (res *user.GetUserResp, err error)
	GetUserDetail(ctx context.Context, req *user.GetUserDetailReq) (res *user.GetUserDetailResp, err error)
	UpdateUser(ctx context.Context, req *user.UpdateUserReq) (res *user.UpdateUserResp, err error)
	SearchUser(ctx context.Context, req *user.SearchUserReq) (res *user.SearchUserResp, err error)
}

type UserServiceImpl struct {
	UserMongoMapper usermapper.IMongoMapper
	UserEsMapper    usermapper.IEsMapper
	MqProducer      rocketmq.Producer
}

var UserSet = wire.NewSet(
	wire.Struct(new(UserServiceImpl), "*"),
	wire.Bind(new(UserService), new(*UserServiceImpl)),
)

func (s *UserServiceImpl) GetUser(ctx context.Context, req *user.GetUserReq) (res *user.GetUserResp, err error) {
	user1, err := s.UserMongoMapper.FindOne(ctx, req.UserId)
	if err != nil {
		return nil, err
	}

	return &user.GetUserResp{
		User: &user.UserPreview{
			Id:        user1.ID.Hex(),
			AvatarUrl: user1.AvatarUrl,
			Nickname:  user1.Nickname,
		},
	}, nil
}

func (s *UserServiceImpl) GetUserDetail(ctx context.Context, req *user.GetUserDetailReq) (res *user.GetUserDetailResp, err error) {
	user1, err := s.UserMongoMapper.FindOne(ctx, req.UserId)
	if err != nil {
		return nil, err
	}

	return &user.GetUserDetailResp{
		User: &user.UserDetail{
			Id:        user1.ID.Hex(),
			AvatarUrl: user1.AvatarUrl,
			Nickname:  user1.Nickname,
			Motto:     user1.Motto,
		},
	}, nil
}

func (s *UserServiceImpl) UpdateUser(ctx context.Context, req *user.UpdateUserReq) (res *user.UpdateUserResp, err error) {
	oid, err := primitive.ObjectIDFromHex(req.User.Id)
	if err != nil {
		return nil, consts.ErrInvalidObjectId
	}

	err = s.UserMongoMapper.UpsertUser(ctx, &usermapper.User{
		ID:        oid,
		AvatarUrl: req.User.AvatarUrl,
		Nickname:  req.User.Nickname,
		Motto:     req.User.Motto,
	})
	if err != nil {
		return nil, err
	}

	//发送使用url信息
	var urls []url.URL
	u, _ := url.Parse(req.User.AvatarUrl)
	urls = append(urls, *u)
	go s.SendDelayMessage(urls)

	return &user.UpdateUserResp{}, nil
}

func (s *UserServiceImpl) SearchUser(ctx context.Context, req *user.SearchUserReq) (res *user.SearchUserResp, err error) {
	popts := &paginator.PaginationOptions{
		Limit:     req.Limit,
		Offset:    req.Offset,
		Backward:  req.Backward,
		LastToken: req.LastToken,
	}
	data, total, err := s.UserEsMapper.SearchUser(ctx, req.Nickname, popts, &esp.ScoreSorter{})
	if err != nil {
		return nil, err
	}
	resp := make([]*user.UserPreview, 0, *req.Limit)
	for _, d := range data {
		m := &user.UserPreview{
			Id:        d.ID.Hex(),
			Nickname:  d.Nickname,
			AvatarUrl: d.AvatarUrl,
		}
		resp = append(resp, m)
	}
	res = &user.SearchUserResp{Users: resp, Total: total}
	if popts.LastToken != nil {
		res.Token = *popts.LastToken
	}
	return res, nil
}

func (s *UserServiceImpl) SendDelayMessage(message interface{}) {
	json, _ := jsonx.Marshal(message)
	msg := &mqprimitive.Message{
		Topic: "sts_used_url",
		Body:  json,
	}

	res, err := s.MqProducer.SendSync(context.Background(), msg)
	if err != nil || res.Status != mqprimitive.SendOK {
		for i := 0; i < 2; i++ {
			res, err := s.MqProducer.SendSync(context.Background(), msg)
			if err == nil && res.Status == mqprimitive.SendOK {
				break
			}
		}
	}
}
