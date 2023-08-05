package main

import (
	"github.com/xh-polaris/meowchat-user/provider"
	"net"

	"github.com/xh-polaris/meowchat-user/biz/infrastructure/util/log"

	"github.com/cloudwego/kitex/pkg/rpcinfo"
	"github.com/cloudwego/kitex/server"
	"github.com/kitex-contrib/obs-opentelemetry/tracing"
	"github.com/xh-polaris/service-idl-gen-go/kitex_gen/meowchat/user/userservice"
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
	)

	err = svr.Run()

	if err != nil {
		log.Error(err.Error())
	}
}
