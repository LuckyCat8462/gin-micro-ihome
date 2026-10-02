package service

import (
	"context"
	"errors"
	"fmt"
	"order/model"
	"order/utils"
	"strconv"

	housePb "house/proto"

	"go-micro.dev/v5"
	"go-micro.dev/v5/client"
	"go-micro.dev/v5/registry/consul"
)

var (
	ErrConflict = errors.New("house reserved")
	ErrParam    = errors.New("param error")
)

var houseClient housePb.HouseService

func InitHouseClient() {
	consulReg := consul.NewConsulRegistry()
	svc := micro.NewService(micro.Registry(consulReg))
	svc.Init()
	houseClient = housePb.NewHouseService("micro_house", svc.Client())
}

func getHouseClient() housePb.HouseService {
	if houseClient == nil {
		InitHouseClient()
	}
	return houseClient
}

func releaseInventoryBestEffort(orderID string) {
	_, err := getHouseClient().ReleaseInventory(context.TODO(), &housePb.ReleaseReq{OrderId: orderID})
	if err != nil {
		fmt.Println("ReleaseInventory failed", orderID, err)
	}
}

// CreateOrder 先写 WAIT_PAY 订单，再同步 RPC 预留；冲突则取消订单并返回 4004
func CreateOrder(houseID, startDate, endDate, userName string) (orderID int, errno string, err error) {
	orderID, err = model.InsertOrderWaitPay(houseID, startDate, endDate, userName)
	if err != nil {
		if errors.Is(err, model.ErrOrderParam) {
			return 0, utils.RECODE_PARAMERR, err
		}
		if errors.Is(err, model.ErrSelfBook) {
			return 0, utils.RECODE_ROLEERR, err
		}
		return 0, utils.RECODE_DBERR, err
	}

	resp, err := getHouseClient().ReserveInventory(context.TODO(), &housePb.ReserveReq{
		HouseId:   houseID,
		OrderId:   strconv.Itoa(orderID),
		StartDate: startDate,
		EndDate:   endDate,
	}, client.WithRetries(1))
	if err != nil {
		_ = model.MarkOrderCancelled(orderID)
		fmt.Println("ReserveInventory RPC 失败", err)
		return orderID, utils.RECODE_THIRDERR, err
	}
	if resp.Errno == utils.RECODE_DATAERR {
		_ = model.MarkOrderCancelled(orderID)
		return orderID, utils.RECODE_DATAERR, ErrConflict
	}
	if resp.Errno != utils.RECODE_OK {
		_ = model.MarkOrderCancelled(orderID)
		return orderID, resp.Errno, fmt.Errorf("%s", resp.Errmsg)
	}
	return orderID, utils.RECODE_OK, nil
}

// PayOrder 模拟支付
func PayOrder(orderIDStr, userName string) error {
	oid, err := strconv.Atoi(orderIDStr)
	if err != nil {
		return ErrParam
	}
	uid, err := model.FindUserIDByName(userName)
	if err != nil {
		if errors.Is(err, model.ErrOrderParam) {
			return ErrParam
		}
		return err
	}
	return model.PayOrderTx(uint(oid), uid)
}

// CancelOrder 用户取消或超时取消；写库后同步释放库存（MQ 仍作补偿）
func CancelOrder(orderIDStr, userName string) error {
	oid, err := strconv.Atoi(orderIDStr)
	if err != nil {
		return ErrParam
	}
	if userName != "" {
		uid, err := model.FindUserIDByName(userName)
		if err != nil {
			if errors.Is(err, model.ErrOrderParam) {
				return ErrParam
			}
			return err
		}
		o, err := model.FindOrderByID(uint(oid))
		if err != nil {
			return err
		}
		if o.UserId != uid {
			return fmt.Errorf("order owner mismatch")
		}
	}
	if err := model.CancelOrderTx(uint(oid)); err != nil {
		return err
	}
	releaseInventoryBestEffort(orderIDStr)
	return nil
}

// UpdateStatus 房东接单/拒单；拒单时同步释放库存
func UpdateStatus(action, id, reason, userName string) error {
	err := model.UpdateStatusByLandlord(action, id, reason, userName)
	if err != nil {
		return err
	}
	if action == "reject" {
		releaseInventoryBestEffort(id)
	}
	return nil
}

// CommentOrder 租户评价
func CommentOrder(orderID, userName, comment string) error {
	err := model.CommentOrderByTenant(orderID, userName, comment)
	if errors.Is(err, model.ErrOrderParam) {
		return ErrParam
	}
	return err
}

// OnInventoryConfirmed 库存确认回调：PAID → WAIT_ACCEPT
func OnInventoryConfirmed(orderID uint) error {
	return model.MarkOrderWaitAccept(orderID)
}
