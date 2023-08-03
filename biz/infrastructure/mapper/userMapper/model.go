package userMapper

import (
	"github.com/google/wire"
	"meowchat-user/biz/infrastructure/config"
	"meowchat-user/biz/infrastructure/mapper/userMapper/es"
	"meowchat-user/biz/infrastructure/mapper/userMapper/mongo"
)

type (
	Model interface {
		mongo.UserMongoModel
		es.UserEsModel
	}
	defaultModel struct {
		mongo.UserMongoModel
		es.UserEsModel
	}
)

func NewModel(config *config.Config) Model {
	return defaultModel{
		UserMongoModel: mongo.NewUserModel(config),
		UserEsModel:    es.NewUserModel(config),
	}
}

var UserSet = wire.NewSet(
	NewModel,
)
