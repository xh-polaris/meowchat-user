package service

import (
	"context"

	"github.com/xh-polaris/meowchat-user/biz/infrastructure/consts"
	"github.com/xh-polaris/meowchat-user/biz/infrastructure/mapper/like"
	"github.com/xh-polaris/service-idl-gen-go/kitex_gen/meowchat/user"

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
	LikeModel like.IMongoMapper
}

var LikeSet = wire.NewSet(
	wire.Struct(new(LikeServiceImpl), "*"),
	wire.Bind(new(LikeService), new(*LikeServiceImpl)),
)

func (s *LikeServiceImpl) DoLike(ctx context.Context, req *user.DoLikeReq) (res *user.DoLikeResp, err error) {
	// 判断是否点过赞
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
		err := likeModel.Insert(ctx, alike)
		if err == nil {
			return &user.DoLikeResp{}, nil
		} else {
			return &user.DoLikeResp{}, consts.ErrDataBase
		}
	case true:
		likeModel := s.LikeModel
		ID, err := likeModel.GetId(ctx, req.UserId, req.TargetId, int64(req.Type))
		if err != nil {
			return &user.DoLikeResp{}, consts.ErrDataBase
		}
		err = likeModel.Delete(ctx, ID)
		if err == nil {
			return &user.DoLikeResp{}, nil
		} else {
			return &user.DoLikeResp{}, consts.ErrDataBase
		}
	default:
		return &user.DoLikeResp{}, nil
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
