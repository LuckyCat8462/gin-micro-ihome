package handler

import (
	"context"
	"errors"
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
	err := model.RegisterUser(req.Mobile, req.Password)
	if err != nil {
		if errors.Is(err, model.ErrUserExists) {
			rsp.Errno = utils.RECODE_USERONERR
			rsp.Errmsg = utils.RecodeText(utils.RECODE_USERONERR)
			return nil
		}
		fmt.Println("注册数据库错误：", err)
		rsp.Errno = utils.RECODE_DBERR
		rsp.Errmsg = utils.RecodeText(utils.RECODE_DBERR)
		return nil
	}

	rsp.Errno = utils.RECODE_OK
	rsp.Errmsg = utils.RecodeText(utils.RECODE_OK)
	rsp.Name = req.Mobile // 默认用户名为手机号
	return nil
}

// 登录
func (e *Register) Login(ctx context.Context, req *register.RegRequest, rsp *register.RegResponse) error {
	userName, err := model.Login(req.Mobile, req.Password)
	if err != nil {
		if errors.Is(err, model.ErrLoginFailed) {
			rsp.Errno = utils.RECODE_LOGINERR
			rsp.Errmsg = utils.RecodeText(utils.RECODE_LOGINERR)
			return nil
		}
		fmt.Println("登录数据库错误：", err)
		rsp.Errno = utils.RECODE_DBERR
		rsp.Errmsg = utils.RecodeText(utils.RECODE_DBERR)
		return nil
	}

	rsp.Errno = utils.RECODE_OK
	rsp.Errmsg = utils.RecodeText(utils.RECODE_OK)
	rsp.Name = userName
	return nil
}
