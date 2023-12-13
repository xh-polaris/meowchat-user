package service

import (
	"context"
	"strconv"
	"time"

	"github.com/google/wire"
	"github.com/xh-polaris/gopkg/pagination"
	"github.com/xh-polaris/gopkg/pagination/esp"
	genuser "github.com/xh-polaris/service-idl-gen-go/kitex_gen/meowchat/user"
	"github.com/zeromicro/go-zero/core/stores/redis"
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
	CheckIn(ctx context.Context, req *genuser.CheckInReq) (res *genuser.CheckInResp, err error)
}

type UserServiceImpl struct {
	Config          *config.Config
	UserMongoMapper usermapper.IMongoMapper
	UserEsMapper    usermapper.IEsMapper
	Redis           *redis.Redis
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
			user, err = s.UserMongoMapper.FindOne(ctx, req.UserId)
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

func (s *UserServiceImpl) CheckIn(ctx context.Context, req *genuser.CheckInReq) (res *genuser.CheckInResp, err error) {

	res = new(genuser.CheckInResp)

	t, err := s.Redis.GetCtx(ctx, "checkInTimes"+req.UserId)
	if err != nil {
		return &genuser.CheckInResp{GetFish: false}, nil
	}
	r, err := s.Redis.GetCtx(ctx, "checkInDates"+req.UserId)
	if err != nil {
		return &genuser.CheckInResp{GetFish: false}, nil
	} else if r == "" {
		res.GetFish = true
		res.GetFishTimes = 1
		err = s.Redis.SetexCtx(ctx, "checkInTimes"+req.UserId, "1", 604800)
		if err != nil {
			return &genuser.CheckInResp{GetFish: false}, nil
		}
		err = s.Redis.SetexCtx(ctx, "checkInDates"+req.UserId, strconv.FormatInt(time.Now().Unix(), 10), 604800)
		if err != nil {
			return &genuser.CheckInResp{GetFish: false}, nil
		}
	} else {
		times, err := strconv.ParseInt(t, 10, 64)
		if err != nil {
			return &genuser.CheckInResp{GetFish: false}, nil
		}
		date, err := strconv.ParseInt(r, 10, 64)
		if err != nil {
			return &genuser.CheckInResp{GetFish: false}, nil
		}
		lastTime := time.Unix(date, 0)
		err = s.Redis.SetexCtx(ctx, "checkInDates"+req.UserId, strconv.FormatInt(time.Now().Unix(), 10), 604800)
		if err != nil {
			return &genuser.CheckInResp{GetFish: false}, nil
		}
		if lastTime.Day() == time.Now().Day() && lastTime.Month() == time.Now().Month() && lastTime.Year() == time.Now().Year() {
			return &genuser.CheckInResp{GetFish: false}, nil
		}
		lastYear, lastWeek := lastTime.ISOWeek()
		nowYear, nowWeek := time.Now().ISOWeek()
		if lastWeek == nowWeek && lastYear == nowYear {
			res.GetFishTimes = times + 1
			err = s.Redis.SetexCtx(ctx, "checkInTimes"+req.UserId, strconv.FormatInt(times+1, 10), 604800)
			if err != nil {
				return &genuser.CheckInResp{GetFish: false}, nil
			}
			res.GetFish = true
		} else {
			err = s.Redis.SetexCtx(ctx, "checkInTimes"+req.UserId, "1", 604800)
			if err != nil {
				return &genuser.CheckInResp{GetFish: false}, nil
			}
			res.GetFish = true
			res.GetFishTimes = 1
		}
	}
	return res, nil
}
