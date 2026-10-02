package main

import (
	"go-micro.dev/v5/registry/consul"
	"house/handler"
	"house/model"
	pb "house/proto"
	"house/worker"

	"go-micro.dev/v5"
)

func main() {
	consulReg := consul.NewConsulRegistry()
	model.InitDb()

	worker.StartInventoryConsumer()
	worker.StartExpireReservationScanner()

	svc := micro.NewService(
		micro.Name("micro_house"),
		micro.Registry(consulReg),
		micro.Address("127.0.0.1:12314"),
		micro.Version("latest"),
	)

	pb.RegisterHouseHandler(svc.Server(), handler.New())
	svc.Run()
}
