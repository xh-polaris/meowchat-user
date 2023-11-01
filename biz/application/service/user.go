package service

import (
	"context"
	"strconv"
	"time"

	"github.com/apache/rocketmq-client-go/v2"
	mqprimitive "github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/google/wire"
	"github.com/xh-polaris/gopkg/pagination"
	"github.com/xh-polaris/gopkg/pagination/esp"
	"github.com/xh-polaris/service-idl-gen-go/kitex_gen/meowchat/user"
	"github.com/zeromicro/go-zero/core/jsonx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/xh-polaris/meowchat-user/biz/infrastructure/consts"
	usermapper "github.com/xh-polaris/meowchat-user/biz/infrastructure/mapper/user"
)

type UserService interface {
	GetUser(ctx context.Context, req *user.GetUserReq) (res *user.GetUserResp, err error)
	GetUserDetail(ctx context.Context, req *user.GetUserDetailReq) (res *user.GetUserDetailResp, err error)
	UpdateUser(ctx context.Context, req *user.UpdateUserReq) (res *user.UpdateUserResp, err error)
	SearchUser(ctx context.Context, req *user.SearchUserReq) (res *user.SearchUserResp, err error)
	CheckIn(ctx context.Context, req *user.CheckInReq) (res *user.CheckInResp, err error)
}

type UserServiceImpl struct {
	UserMongoMapper usermapper.IMongoMapper
	UserEsMapper    usermapper.IEsMapper
	MqProducer      rocketmq.Producer
	Redis           *redis.Redis
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
		if err != consts.ErrNotFound {
			return nil, err
		}
		data := &usermapper.User{}
		data.ID, err = primitive.ObjectIDFromHex(req.GetUserId())
		if err != nil {
			return nil, err
		}
		data.AvatarUrl = "https://static.xhpolaris.com/cat_world.jpg"
		data.Nickname = "用户_" + req.GetUserId()[:13]
		data.UpdateAt = time.Now()
		data.CreateAt = time.Now()
		err = s.UserMongoMapper.Insert(ctx, data)
		if err != nil {
			return nil, err
		}
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
	//var urls []url.URL
	//u, _ := url.Parse(req.User.AvatarUrl)
	//urls = append(urls, *u)
	//go s.SendDelayMessage(urls)

	return &user.UpdateUserResp{}, nil
}

func (s *UserServiceImpl) SearchUser(ctx context.Context, req *user.SearchUserReq) (res *user.SearchUserResp, err error) {
	popts := &pagination.PaginationOptions{
		Limit:     req.Limit,
		Offset:    req.Offset,
		Backward:  req.Backward,
		LastToken: req.LastToken,
	}
	data, total, err := s.UserEsMapper.SearchUser(ctx, req.Nickname, popts, esp.ScoreCursorType)
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

func (s *UserServiceImpl) CheckIn(ctx context.Context, req *user.CheckInReq) (res *user.CheckInResp, err error) {

	res = new(user.CheckInResp)
	r, err := s.Redis.GetCtx(ctx, "signIn"+req.UserId)
	if err != nil {
		return nil, err
	} else if r == "" {
		res.IsFirst = true
		err = s.Redis.SetexCtx(ctx, "signIn"+req.UserId, strconv.FormatInt(time.Now().Unix(), 10), 86400)
		if err != nil {
			res.IsFirst = false
			return res, err
		}
	} else {
		m, err := strconv.ParseInt(r, 10, 64)
		if err != nil {
			return nil, err
		}
		lastTime := time.Unix(m, 0)
		err = s.Redis.SetexCtx(ctx, "signIn"+req.UserId, strconv.FormatInt(time.Now().Unix(), 10), 86400)
		if err != nil {
			return nil, err
		}
		if lastTime.Day() == time.Now().Day() && lastTime.Month() == time.Now().Month() && lastTime.Year() == time.Now().Year() {
			res.IsFirst = false
		} else {
			res.IsFirst = true
		}
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
