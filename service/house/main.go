package main

import (
	"go-micro.dev/v5/registry/consul"
	"house/handler"
	"house/model"
	pb "house/proto"

	"go-micro.dev/v5"
)

func main() {
	//初始化consul
	consulReg := consul.NewConsulRegistry()
	model.InitDb()
	// Create service
	service := micro.NewService(
		micro.Name("micro_house"),
		micro.Registry(consulReg),             //添加注册
		micro.Address("127.0.0.1:12314"), //主动添加addr,防止其生成随机port
		micro.Version("latest"),
	)

	// Register handler
	pb.RegisterHouseHandler(service.Server(), handler.New())

	// Run service
	service.Run()
}
