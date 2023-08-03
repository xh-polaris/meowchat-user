package adaptor

import (
	"context"
	"github.com/xh-polaris/service-idl-gen-go/kitex_gen/meowchat/user"

	"meowchat-user/biz/application/service"
	"meowchat-user/biz/infrastructure/config"
)

type UserServerImpl struct {
	*config.Config
	LikeService service.LikeService
	UserService service.UserService
}

func (s *UserServerImpl) DoLike(ctx context.Context, req *user.DoLikeReq) (res *user.DoLikeResp, err error) {
	return s.LikeService.DoLike(ctx, req)
}

func (s *UserServerImpl) GetUserLike(ctx context.Context, req *user.GetUserLikedReq) (res *user.GetUserLikedResp, err error) {
	return s.LikeService.GetUserLike(ctx, req)
}

func (s *UserServerImpl) GetTargetLikes(ctx context.Context, req *user.GetTargetLikesReq) (res *user.GetTargetLikesResp, err error) {
	return s.LikeService.GetTargetLikes(ctx, req)
}

func (s *UserServerImpl) GetUserLikes(ctx context.Context, req *user.GetUserLikesReq) (res *user.GetUserLikesResp, err error) {
	return s.LikeService.GetUserLikes(ctx, req)
}

func (s *UserServerImpl) GetLikedUsers(ctx context.Context, req *user.GetLikedUsersReq) (res *user.GetLikedUsersResp, err error) {
	return s.LikeService.GetLikedUsers(ctx, req)
}

func (s *UserServerImpl) GetUser(ctx context.Context, req *user.GetUserReq) (res *user.GetUserResp, err error) {
	return s.UserService.GetUser(ctx, req)
}

func (s *UserServerImpl) GetUserDetail(ctx context.Context, req *user.GetUserDetailReq) (res *user.GetUserDetailResp, err error) {
	return s.UserService.GetUserDetail(ctx, req)
}

func (s *UserServerImpl) UpdateUser(ctx context.Context, req *user.UpdateUserReq) (res *user.UpdateUserResp, err error) {
	return s.UserService.UpdateUser(ctx, req)
}

func (s *UserServerImpl) SearchUser(ctx context.Context, req *user.SearchUserReq) (res *user.SearchUserResp, err error) {
	return s.UserService.SearchUser(ctx, req)
}
