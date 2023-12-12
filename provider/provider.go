package provider

import (
	"github.com/google/wire"

	"github.com/xh-polaris/meowchat-user/biz/application/service"
	"github.com/xh-polaris/meowchat-user/biz/infrastructure/config"
	"github.com/xh-polaris/meowchat-user/biz/infrastructure/mapper/like"
	"github.com/xh-polaris/meowchat-user/biz/infrastructure/mapper/user"
	"github.com/xh-polaris/meowchat-user/biz/infrastructure/stores/redis"
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
	redis.NewRedis,
	MapperSet,
)

var MapperSet = wire.NewSet(
	like.NewMongoModel,
	user.NewMongoMapper,
	user.NewEsMapper,
)
