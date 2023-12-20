// Code generated by Wire. DO NOT EDIT.

//go:generate go run github.com/google/wire/cmd/wire
//go:build !wireinject
// +build !wireinject

package provider

import (
	"github.com/xh-polaris/meowchat-user/biz/adaptor"
	"github.com/xh-polaris/meowchat-user/biz/application/service"
	"github.com/xh-polaris/meowchat-user/biz/infrastructure/config"
	"github.com/xh-polaris/meowchat-user/biz/infrastructure/mapper/like"
	"github.com/xh-polaris/meowchat-user/biz/infrastructure/mapper/user"
	"github.com/xh-polaris/meowchat-user/biz/infrastructure/stores/redis"
)

// Injectors from wire.go:

func NewUserServerImpl() (*adaptor.UserServerImpl, error) {
	configConfig, err := config.NewConfig()
	if err != nil {
		return nil, err
	}
	iMongoMapper := like.NewMongoModel(configConfig)
	redisRedis := redis.NewRedis(configConfig)
	likeServiceImpl := &service.LikeServiceImpl{
		Config:    configConfig,
		LikeModel: iMongoMapper,
		Redis:     redisRedis,
	}
	userIMongoMapper := user.NewMongoMapper(configConfig)
	iEsMapper := user.NewEsMapper(configConfig)
	userServiceImpl := &service.UserServiceImpl{
		Config:          configConfig,
		UserMongoMapper: userIMongoMapper,
		UserEsMapper:    iEsMapper,
	}
	userServerImpl := &adaptor.UserServerImpl{
		Config:      configConfig,
		LikeService: likeServiceImpl,
		UserService: userServiceImpl,
	}
	return userServerImpl, nil
}
