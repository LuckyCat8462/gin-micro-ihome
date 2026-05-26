package main

import (
	"go-micro.dev/v5/registry/consul"
	"user/handler"
	"user/model"
	pb "user/proto"

	"go-micro.dev/v5"
)

func main() {
	//初始化consul
	consulReg := consul.NewConsulRegistry()
	model.InitDb()
	// Create service
	service := micro.NewService(
		micro.Name("micro_user"),
		micro.Registry(consulReg),             //添加注册
		micro.Address("127.0.0.1:12313"), //主动添加addr,防止其生成随机port
		micro.Version("latest"),
	)

	// Register handler
	pb.RegisterUserHandler(service.Server(), handler.New())

	// Run service
	service.Run()
}
