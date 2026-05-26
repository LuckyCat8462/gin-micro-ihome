package handler

import (
	"context"
	"fmt"
	"register/model"
	register "register/proto"
	"register/utils"
)

type Register struct{}

// Return a new handler
func New() *Register {
	return &Register{}
}

// 注册
func (e *Register) Register(ctx context.Context, req *register.RegRequest, rsp *register.RegResponse) error {
	//req数据测试
	fmt.Println("---微服务注册功能测试-register.go-login函数：")
	fmt.Println("req数据:", req)

	//数据库操作
	err := model.RegisterUser(req.Mobile, req.Password)
	if err != nil {
		rsp.Errno = utils.RECODE_DBERR
		rsp.Errmsg = utils.RecodeText(utils.RECODE_DBERR)
	} else {
		rsp.Errno = utils.RECODE_OK
		rsp.Errmsg = utils.RecodeText(utils.RECODE_OK)
	}
	return nil
}

// 登录
func (e *Register) Login(ctx context.Context, req *register.RegRequest, rsp *register.RegResponse) error {
	//req数据测试
	//fmt.Println("---微服务登录功能测试-register.go-login函数：")
	//fmt.Println("req数据:", req)

	//数据库操作
	userName, err := model.Login(req.Mobile, req.Password)
	if err != nil {
		fmt.Println("数据库操作返回用户名：", userName, err)
		rsp.Errno = utils.RECODE_DBERR
		rsp.Errmsg = utils.RecodeText(utils.RECODE_DBERR)
	} else {
		//正常登录
		//fmt.Println("成功登录，loginUser:", userName)
		rsp.Errno = utils.RECODE_OK
		rsp.Errmsg = utils.RecodeText(utils.RECODE_OK)
		rsp.Name = userName
	}
	return nil

}
