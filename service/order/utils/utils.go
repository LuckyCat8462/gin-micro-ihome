package utils

import (
	"go-micro.dev/v5"
	"go-micro.dev/v5/registry/consul"
)

// 封装初始化微服务客户端
func InitMicro() micro.Service {
	//指定consul服务发现并初始化微服务
	consulReg := consul.NewConsulRegistry()
	consulService := micro.NewService(micro.Registry(consulReg))
	return consulService
}
