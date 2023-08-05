//go:build wireinject
// +build wireinject

package provider

import (
	"github.com/google/wire"

	"github.com/xh-polaris/meowchat-user/biz/adaptor"
)

func NewUserServerImpl() (*adaptor.UserServerImpl, error) {
	wire.Build(
		wire.Struct(new(adaptor.UserServerImpl), "*"),
		AllProvider,
	)
	return nil, nil
}
