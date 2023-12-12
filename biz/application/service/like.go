package service

import (
	"context"
	"strconv"
	"time"

	"github.com/xh-polaris/gopkg/pagination/mongop"
	"github.com/xh-polaris/service-idl-gen-go/kitex_gen/meowchat/user"
	"github.com/zeromicro/go-zero/core/stores/redis"

	"github.com/xh-polaris/meowchat-user/biz/infrastructure/config"
	"github.com/xh-polaris/meowchat-user/biz/infrastructure/consts"
	"github.com/xh-polaris/meowchat-user/biz/infrastructure/mapper/like"
	"github.com/xh-polaris/meowchat-user/biz/infrastructure/util"

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
	Config    *config.Config
	LikeModel like.IMongoMapper
	Redis     *redis.Redis
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
		res.Liked = true

		t, err := s.Redis.GetCtx(ctx, "likeTimes"+req.UserId)
		if err != nil {
			return &user.DoLikeResp{GetFish: false, Liked: true}, nil
		}
		r, err := s.Redis.GetCtx(ctx, "likeDates"+req.UserId)
		if err != nil {
			return &user.DoLikeResp{GetFish: false, Liked: true}, nil
		} else if r == "" {
			res.GetFish = true
			res.GetFishTimes = 1
			err = s.Redis.SetexCtx(ctx, "likeTimes"+req.UserId, "1", 86400)
			if err != nil {
				return &user.DoLikeResp{GetFish: false, Liked: true}, nil
			}
			err = s.Redis.SetexCtx(ctx, "likeDates"+req.UserId, strconv.FormatInt(time.Now().Unix(), 10), 86400)
			if err != nil {
				return &user.DoLikeResp{GetFish: false, Liked: true}, nil
			}
		} else {
			times, err := strconv.ParseInt(t, 10, 64)
			if err != nil {
				return &user.DoLikeResp{GetFish: false, Liked: true}, nil
			}
			res.GetFishTimes = times + 1
			date, err := strconv.ParseInt(r, 10, 64)
			if err != nil {
				return &user.DoLikeResp{GetFish: false, Liked: true}, nil
			}
			lastTime := time.Unix(date, 0)
			err = s.Redis.SetexCtx(ctx, "likeTimes"+req.UserId, strconv.FormatInt(times+1, 10), 86400)
			if err != nil {
				return &user.DoLikeResp{GetFish: false, Liked: true}, nil
			}
			err = s.Redis.SetexCtx(ctx, "likeDates"+req.UserId, strconv.FormatInt(time.Now().Unix(), 10), 86400)
			if err != nil {
				return &user.DoLikeResp{GetFish: false, Liked: true}, nil
			}
			if lastTime.Day() == time.Now().Day() && lastTime.Month() == time.Now().Month() && lastTime.Year() == time.Now().Year() {
				err = s.Redis.SetexCtx(ctx, "likeTimes"+req.UserId, strconv.FormatInt(times+1, 10), 86400)
				if err != nil {
					return &user.DoLikeResp{GetFish: false, Liked: true}, nil
				}
				if times >= s.Config.LikeTimes {
					res.GetFish = false
				} else {
					res.GetFish = true
				}
			} else {
				err = s.Redis.SetexCtx(ctx, "likeTimes"+req.UserId, "1", 86400)
				if err != nil {
					return &user.DoLikeResp{GetFish: false, Liked: true}, nil
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
	p := util.ParsePagination(req.PaginationOptions)

	data, total, err := s.LikeModel.GetUserLikes(ctx, req.UserId, int64(req.Type), p, mongop.IdCursorType)
	if err != nil {
		return nil, err
	}
	res = new(user.GetUserLikesResp)
	res.Total = total
	if p.LastToken != nil {
		res.Token = *p.LastToken
	}
	likes := make([]*user.Like, 0)
	for _, alike := range data {
		likes = append(likes, &user.Like{
			TargetId:     alike.TargetId,
			AssociatedId: alike.AssociatedId,
		})
	}
	res.Likes = likes
	return res, nil
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
