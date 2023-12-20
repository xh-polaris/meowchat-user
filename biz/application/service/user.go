package service

import (
	"context"
	"time"

	"github.com/google/wire"
	"github.com/xh-polaris/gopkg/pagination"
	"github.com/xh-polaris/gopkg/pagination/esp"
	genuser "github.com/xh-polaris/service-idl-gen-go/kitex_gen/meowchat/user"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/xh-polaris/meowchat-user/biz/infrastructure/config"
	"github.com/xh-polaris/meowchat-user/biz/infrastructure/consts"
	usermapper "github.com/xh-polaris/meowchat-user/biz/infrastructure/mapper/user"
)

type UserService interface {
	GetUser(ctx context.Context, req *genuser.GetUserReq) (res *genuser.GetUserResp, err error)
	GetUserDetail(ctx context.Context, req *genuser.GetUserDetailReq) (res *genuser.GetUserDetailResp, err error)
	UpdateUser(ctx context.Context, req *genuser.UpdateUserReq) (res *genuser.UpdateUserResp, err error)
	SearchUser(ctx context.Context, req *genuser.SearchUserReq) (res *genuser.SearchUserResp, err error)
}

type UserServiceImpl struct {
	Config          *config.Config
	UserMongoMapper usermapper.IMongoMapper
	UserEsMapper    usermapper.IEsMapper
}

var UserSet = wire.NewSet(
	wire.Struct(new(UserServiceImpl), "*"),
	wire.Bind(new(UserService), new(*UserServiceImpl)),
)

func (s *UserServiceImpl) GetUser(ctx context.Context, req *genuser.GetUserReq) (res *genuser.GetUserResp, err error) {
	user1, err := s.UserMongoMapper.FindOne(ctx, req.UserId)
	if err != nil {
		return nil, err
	}

	return &genuser.GetUserResp{
		User: &genuser.UserPreview{
			Id:        user1.ID.Hex(),
			AvatarUrl: user1.AvatarUrl,
			Nickname:  user1.Nickname,
		},
	}, nil
}

func (s *UserServiceImpl) GetUserDetail(ctx context.Context, req *genuser.GetUserDetailReq) (res *genuser.GetUserDetailResp, err error) {
	user, err := s.UserMongoMapper.FindOne(ctx, req.UserId)
	if err != nil {
		if err != consts.ErrNotFound {
			return nil, err
		}
		user = &usermapper.User{}
		user.ID, err = primitive.ObjectIDFromHex(req.GetUserId())
		if err != nil {
			return nil, err
		}
		user.AvatarUrl = "https://static.xhpolaris.com/cat_world.jpg"
		user.Nickname = "用户_" + req.GetUserId()[:13]
		user.UpdateAt = time.Now()
		user.CreateAt = time.Now()
		err = s.UserMongoMapper.Insert(ctx, user)
		// 处理并发冲突
		if mongo.IsDuplicateKeyError(err) {
			user, err = s.UserMongoMapper.FindOneNoCache(ctx, req.UserId)
			if err != nil {
				return nil, err
			}
		} else if err != nil {
			return nil, err
		}
	}

	return &genuser.GetUserDetailResp{
		User: &genuser.UserDetail{
			Id:        user.ID.Hex(),
			AvatarUrl: user.AvatarUrl,
			Nickname:  user.Nickname,
			Motto:     user.Motto,
		},
	}, nil
}

func (s *UserServiceImpl) UpdateUser(ctx context.Context, req *genuser.UpdateUserReq) (res *genuser.UpdateUserResp, err error) {
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

	return &genuser.UpdateUserResp{}, nil
}

func (s *UserServiceImpl) SearchUser(ctx context.Context, req *genuser.SearchUserReq) (res *genuser.SearchUserResp, err error) {
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
	resp := make([]*genuser.UserPreview, 0, *req.Limit)
	for _, d := range data {
		m := &genuser.UserPreview{
			Id:        d.ID.Hex(),
			Nickname:  d.Nickname,
			AvatarUrl: d.AvatarUrl,
		}
		resp = append(resp, m)
	}
	res = &genuser.SearchUserResp{Users: resp, Total: total}
	if popts.LastToken != nil {
		res.Token = *popts.LastToken
	}
	return res, nil
}
