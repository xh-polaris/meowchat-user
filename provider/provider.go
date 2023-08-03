package provider

import (
	"github.com/google/wire"
	"meowchat-user/biz/application/service"
	"meowchat-user/biz/infrastructure/config"
	"meowchat-user/biz/infrastructure/mapper/likeMapper"
	"meowchat-user/biz/infrastructure/mapper/userMapper"
)

var AllProvider = wire.NewSet(
	ApplicationSet,
	InfrastructureSet,
)

var ApplicationSet = wire.NewSet(
	service.LikeSet,
	service.UserSet,
)

var InfrastructureSet = wire.NewSet(
	config.NewConfig,
	MapperSet,
)

var MapperSet = wire.NewSet(
	likeMapper.LikeSet,
	userMapper.UserSet,
)
