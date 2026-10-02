package main

import (
	"go-micro.dev/v5/registry/consul"
	"order/handler"
	"order/model"
	pb "order/proto"
	"order/service"
	"order/worker"

	"go-micro.dev/v5"
)

func main() {
	consulReg := consul.NewConsulRegistry()
	model.InitDb()
	service.InitHouseClient()

	worker.StartOutboxPublisher()
	worker.StartPaymentTimeoutScanner()
	worker.StartStatusConsumer()
	worker.StartReconcileJob()

	svc := micro.NewService(
		micro.Name("micro_order"),
		micro.Registry(consulReg),
		micro.Address("127.0.0.1:12315"),
		micro.Version("latest"),
	)

	pb.RegisterOrderHandler(svc.Server(), handler.New())
	svc.Run()
}
