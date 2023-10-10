package main

import (
	"net"

	"github.com/cloudwego/kitex/pkg/rpcinfo"
	"github.com/cloudwego/kitex/server"
	"github.com/kitex-contrib/obs-opentelemetry/tracing"
	"github.com/xh-polaris/gopkg/kitex/middleware"
	"github.com/xh-polaris/service-idl-gen-go/kitex_gen/meowchat/user/userservice"

	"github.com/xh-polaris/meowchat-user/biz/infrastructure/util/log"
	"github.com/xh-polaris/meowchat-user/provider"
)

func main() {
	s, err := provider.NewUserServerImpl()
	if err != nil {
		panic(err)
	}
	addr, err := net.ResolveTCPAddr("tcp", s.ListenOn)
	if err != nil {
		panic(err)
	}
	svr := userservice.NewServer(
		s,
		server.WithServiceAddr(addr),
		server.WithSuite(tracing.NewServerSuite()),
		server.WithServerBasicInfo(&rpcinfo.EndpointBasicInfo{ServiceName: s.Name}),
		server.WithMiddleware(middleware.LogMiddleware(s.Name)),
	)

	err = svr.Run()

	if err != nil {
		log.Error(err.Error())
	}
}
