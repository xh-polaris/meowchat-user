package service

import (
	"context"
	"strconv"
	"time"

	"github.com/apache/rocketmq-client-go/v2"
	mqprimitive "github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/bytedance/sonic"
	"github.com/xh-polaris/service-idl-gen-go/kitex_gen/meowchat/system"
	"github.com/xh-polaris/service-idl-gen-go/kitex_gen/meowchat/user"
	"github.com/zeromicro/go-zero/core/stores/redis"

	"github.com/xh-polaris/meowchat-user/biz/infrastructure/config"
	"github.com/xh-polaris/meowchat-user/biz/infrastructure/consts"
	"github.com/xh-polaris/meowchat-user/biz/infrastructure/mapper/like"

	"github.com/google/wire"
	"github.com/zeromicro/go-zero/core/stores/monc"
)

type LikeService interface {
	DoLike(ctx context.Context, req *user.DoLikeReq) (res *user.DoLikeResp, err error)
	GetUserLike(ctx context.Context, req *user.GetUserLikedReq) (res *user.GetUserLikedResp, err error)
	GetTargetLikes(ctx context.Context, req *user.GetTargetLikesReq) (res *user.GetTargetLikesResp, err error)
	GetUserLikes(ctx context.Context, req *user.GetUserLikesReq) (res *user.GetUserLikesResp, err error)
	GetLikedUsers(ctx context.Context, req *user.GetLikedUsersReq) (res *user.GetLikedUsersResp, err error)
}

type LikeServiceImpl struct {
	Config     *config.Config
	LikeModel  like.IMongoMapper
	MqProducer rocketmq.Producer
	Redis      *redis.Redis
}

var LikeSet = wire.NewSet(
	wire.Struct(new(LikeServiceImpl), "*"),
	wire.Bind(new(LikeService), new(*LikeServiceImpl)),
)

func (s *LikeServiceImpl) DoLike(ctx context.Context, req *user.DoLikeReq) (res *user.DoLikeResp, err error) {
	// 判断是否点过赞
	res = new(user.DoLikeResp)

	data := &user.GetUserLikedReq{
		UserId:   req.UserId,
		TargetId: req.TargetId,
		Type:     req.Type,
	}
	response, _ := s.GetUserLike(ctx, data)
	switch response.Liked {
	case false:
		// 插入数据
		likeModel := s.LikeModel
		alike := &like.Like{
			UserId:       req.UserId,
			TargetId:     req.TargetId,
			TargetType:   int64(req.Type),
			AssociatedId: req.AssociatedId,
		}
		err = likeModel.Insert(ctx, alike)
		if err != nil {
			return &user.DoLikeResp{}, consts.ErrDataBase
		}

		s.SendNotificationMessage(req)

		t, err := s.Redis.GetCtx(ctx, "likeTimes"+req.UserId)
		if err != nil {
			return &user.DoLikeResp{GetFish: false}, nil
		}
		r, err := s.Redis.GetCtx(ctx, "likeDates"+req.UserId)
		if err != nil {
			return &user.DoLikeResp{GetFish: false}, nil
		} else if r == "" {
			res.GetFish = true
			res.GetFishTimes = 1
			err = s.Redis.SetexCtx(ctx, "likeTimes"+req.UserId, "1", 86400)
			if err != nil {
				res.GetFish = false
				return &user.DoLikeResp{GetFish: false}, nil
			}
			err = s.Redis.SetexCtx(ctx, "likeDates"+req.UserId, strconv.FormatInt(time.Now().Unix(), 10), 86400)
			if err != nil {
				res.GetFish = false
				return &user.DoLikeResp{GetFish: false}, nil
			}
		} else {
			times, err := strconv.ParseInt(t, 10, 64)
			if err != nil {
				return &user.DoLikeResp{GetFish: false}, nil
			}
			res.GetFishTimes = times + 1
			date, err := strconv.ParseInt(r, 10, 64)
			if err != nil {
				return &user.DoLikeResp{GetFish: false}, nil
			}
			lastTime := time.Unix(date, 0)
			err = s.Redis.SetexCtx(ctx, "likeTimes"+req.UserId, strconv.FormatInt(times+1, 10), 86400)
			if err != nil {
				return &user.DoLikeResp{GetFish: false}, nil
			}
			err = s.Redis.SetexCtx(ctx, "likeDates"+req.UserId, strconv.FormatInt(time.Now().Unix(), 10), 86400)
			if err != nil {
				return &user.DoLikeResp{GetFish: false}, nil
			}
			if lastTime.Day() == time.Now().Day() && lastTime.Month() == time.Now().Month() && lastTime.Year() == time.Now().Year() {
				err = s.Redis.SetexCtx(ctx, "likeTimes"+req.UserId, strconv.FormatInt(times+1, 10), 86400)
				if err != nil {
					return &user.DoLikeResp{GetFish: false}, nil
				}
				if times >= s.Config.LikeTimes {
					res.GetFish = false
				} else {
					res.GetFish = true
				}
			} else {
				err = s.Redis.SetexCtx(ctx, "likeTimes"+req.UserId, "1", 86400)
				if err != nil {
					return &user.DoLikeResp{GetFish: false}, nil
				}
				res.GetFish = true
				res.GetFishTimes = 1
			}
		}

		return res, nil
	case true:
		likeModel := s.LikeModel
		ID, err := likeModel.GetId(ctx, req.UserId, req.TargetId, int64(req.Type))
		if err != nil {
			return &user.DoLikeResp{}, consts.ErrDataBase
		}
		err = likeModel.Delete(ctx, ID)
		if err == nil {
			return &user.DoLikeResp{GetFish: false}, nil
		} else {
			return &user.DoLikeResp{}, consts.ErrDataBase
		}
	default:
		return &user.DoLikeResp{GetFish: false}, nil
	}
}

func (s *LikeServiceImpl) GetUserLike(ctx context.Context, req *user.GetUserLikedReq) (res *user.GetUserLikedResp, err error) {
	likeModel := s.LikeModel
	err = likeModel.GetUserLike(ctx, req.UserId, req.TargetId, int64(req.Type))
	switch err {
	case nil:
		return &user.GetUserLikedResp{Liked: true}, nil
	case monc.ErrNotFound:
		return &user.GetUserLikedResp{Liked: false}, nil
	}
	return &user.GetUserLikedResp{}, nil
}

func (s *LikeServiceImpl) GetTargetLikes(ctx context.Context, req *user.GetTargetLikesReq) (res *user.GetTargetLikesResp, err error) {
	likeModel := s.LikeModel
	count, err := likeModel.GetTargetLikes(ctx, req.TargetId, int64(req.Type))
	if err != nil {
		return &user.GetTargetLikesResp{}, consts.ErrDataBase
	} else {
		return &user.GetTargetLikesResp{Count: int64(len(count))}, nil
	}
}

func (s *LikeServiceImpl) GetUserLikes(ctx context.Context, req *user.GetUserLikesReq) (res *user.GetUserLikesResp, err error) {
	data, err := s.LikeModel.GetUserLikes(ctx, req.UserId, int64(req.Type))
	if err != nil {
		return nil, err
	}

	likes := make([]*user.Like, 0)
	for _, alike := range data {
		likes = append(likes, &user.Like{
			TargetId:     alike.TargetId,
			AssociatedId: alike.AssociatedId,
		})
	}
	return &user.GetUserLikesResp{Likes: likes}, nil
}

func (s *LikeServiceImpl) GetLikedUsers(ctx context.Context, req *user.GetLikedUsersReq) (res *user.GetLikedUsersResp, err error) {
	data, err := s.LikeModel.GetTargetLikes(ctx, req.TargetId, int64(req.Type))
	if err != nil {
		return nil, err
	}

	userIds := make([]string, 0, len(data))
	for _, like := range data {
		userIds = append(userIds, like.UserId)
	}

	return &user.GetLikedUsersResp{UserIds: userIds}, nil
}

func (s *LikeServiceImpl) SendNotificationMessage(req *user.DoLikeReq) {

	message := &system.Notification{
		TargetUserId:    req.LikedUserId,
		SourceUserId:    req.UserId,
		SourceContentId: req.TargetId,
		Type:            0,
		Text:            "",
		IsRead:          false,
	}
	if req.GetType() == 1 {
		message.Type = 1
	} else if req.GetType() == 2 {
		message.Type = 3
	} else if req.GetType() == 4 {
		message.Type = 2
	} else if req.GetType() == 6 {
		message.Type = 4
	} else {
		return
	}

	json, _ := sonic.Marshal(message)
	msg := &mqprimitive.Message{
		Topic: "notification",
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
