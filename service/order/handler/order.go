package handler

import (
	"context"
	"errors"
	"fmt"
	"order/model"
	"order/service"
	"order/utils"
	"strconv"

	userOrder "order/proto"
)

type Order struct{}

func New() *Order {
	return &Order{}
}

func mapOrderErr(err error) (errno, errmsg string) {
	switch {
	case err == nil:
		return utils.RECODE_OK, utils.RecodeText(utils.RECODE_OK)
	case errors.Is(err, service.ErrParam), errors.Is(err, model.ErrOrderParam):
		return utils.RECODE_PARAMERR, utils.RecodeText(utils.RECODE_PARAMERR)
	case errors.Is(err, model.ErrNotLandlord):
		return utils.RECODE_ROLEERR, "仅房东可操作该订单"
	case errors.Is(err, model.ErrOrderStatus):
		return utils.RECODE_DATAERR, "订单状态不允许此操作"
	case errors.Is(err, model.ErrOrderNotFound):
		return utils.RECODE_NODATA, utils.RecodeText(utils.RECODE_NODATA)
	default:
		return utils.RECODE_DATAERR, utils.RecodeText(utils.RECODE_DATAERR)
	}
}

func (e *Order) CreateOrder(ctx context.Context, req *userOrder.Request, rsp *userOrder.Response) error {
	orderId, errno, err := service.CreateOrder(req.HouseId, req.StartDate, req.EndDate, req.UserName)
	if err != nil || errno != utils.RECODE_OK {
		rsp.Errno = errno
		switch {
		case errno == utils.RECODE_DATAERR:
			rsp.Errmsg = "房间已被抢定"
		case errors.Is(err, model.ErrSelfBook):
			rsp.Errmsg = "不能预订自己的房源"
		case errno == utils.RECODE_PARAMERR:
			rsp.Errmsg = "预订参数错误，请检查日期与入住天数"
		default:
			rsp.Errmsg = utils.RecodeText(errno)
		}
		return nil
	}
	rsp.Errno = utils.RECODE_OK
	rsp.Errmsg = utils.RecodeText(utils.RECODE_OK)
	rsp.Data = &userOrder.OrderData{OrderId: strconv.Itoa(orderId)}
	return nil
}

func (e *Order) GetOrderInfo(ctx context.Context, req *userOrder.GetReq, resp *userOrder.GetResp) error {
	respData, err := model.GetOrderInfo(req.UserName, req.Role)
	if err != nil {
		errno, errmsg := mapOrderErr(err)
		resp.Errno = errno
		resp.Errmsg = errmsg
		return nil
	}

	resp.Errno = utils.RECODE_OK
	resp.Errmsg = utils.RecodeText(utils.RECODE_OK)
	resp.Data = &userOrder.GetData{Orders: respData}
	return nil
}

func (e *Order) UpdateStatus(ctx context.Context, req *userOrder.UpdateReq, resp *userOrder.UpdateResp) error {
	err := service.UpdateStatus(req.Action, req.Id, req.Reason, req.UserName)
	if err != nil {
		fmt.Println("更新订单状态错误", err)
		errno, errmsg := mapOrderErr(err)
		resp.Errno = errno
		resp.Errmsg = errmsg
		return nil
	}

	resp.Errno = utils.RECODE_OK
	resp.Errmsg = utils.RecodeText(utils.RECODE_OK)
	return nil
}

func (e *Order) PayOrder(ctx context.Context, req *userOrder.PayReq, resp *userOrder.PayResp) error {
	err := service.PayOrder(req.OrderId, req.UserName)
	errno, errmsg := mapOrderErr(err)
	if err != nil {
		fmt.Println("PayOrder error", err)
		resp.Errno = errno
		resp.Errmsg = errmsg
		return nil
	}
	resp.Errno = utils.RECODE_OK
	resp.Errmsg = utils.RecodeText(utils.RECODE_OK)
	return nil
}

func (e *Order) CancelOrder(ctx context.Context, req *userOrder.CancelReq, resp *userOrder.CancelResp) error {
	err := service.CancelOrder(req.OrderId, req.UserName)
	if err != nil {
		fmt.Println("CancelOrder error", err)
		errno, errmsg := mapOrderErr(err)
		resp.Errno = errno
		resp.Errmsg = errmsg
		return nil
	}
	resp.Errno = utils.RECODE_OK
	resp.Errmsg = utils.RecodeText(utils.RECODE_OK)
	return nil
}

func (e *Order) CommentOrder(ctx context.Context, req *userOrder.CommentReq, resp *userOrder.CommentResp) error {
	err := service.CommentOrder(req.OrderId, req.UserName, req.Comment)
	if err != nil {
		fmt.Println("CommentOrder error", err)
		errno, errmsg := mapOrderErr(err)
		resp.Errno = errno
		resp.Errmsg = errmsg
		return nil
	}
	resp.Errno = utils.RECODE_OK
	resp.Errmsg = utils.RecodeText(utils.RECODE_OK)
	return nil
}
