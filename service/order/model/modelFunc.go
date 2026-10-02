package model

import (
	"errors"
	"fmt"
	"strconv"

	Order "order/proto"
)

type UserData struct {
	Id int
}

var (
	ErrOrderStatus = errors.New("invalid order status")
	ErrNotLandlord = errors.New("not house landlord")
)

func GetOrderInfo(userName, role string) ([]*Order.OrdersData, error) {
	var userData UserData
	if err := GlobalConn.Raw("select id from user where name = ?", userName).Scan(&userData).Error; err != nil {
		return nil, err
	}
	if userData.Id == 0 {
		return nil, ErrOrderParam
	}

	var orders []OrderHouse
	if role == "custom" {
		if err := GlobalConn.Where("user_id = ?", userData.Id).Order("id desc").Find(&orders).Error; err != nil {
			return nil, err
		}
	} else {
		var houseIDs []uint
		if err := GlobalConn.Model(&House{}).Where("user_id = ?", userData.Id).Pluck("id", &houseIDs).Error; err != nil {
			return nil, err
		}
		if len(houseIDs) == 0 {
			return []*Order.OrdersData{}, nil
		}
		if err := GlobalConn.Where("house_id IN ?", houseIDs).Order("id desc").Find(&orders).Error; err != nil {
			return nil, err
		}
	}

	orderResp := make([]*Order.OrdersData, 0, len(orders))
	for _, v := range orders {
		var house House
		_ = GlobalConn.Where("id = ?", v.HouseId).First(&house).Error

		orderTemp := &Order.OrdersData{
			OrderId:   int32(v.ID),
			EndDate:   v.End_date.Format("2006-01-02"),
			StartDate: v.Begin_date.Format("2006-01-02"),
			Ctime:     v.CreatedAt.Format("2006-01-02"),
			Amount:    int32(v.Amount),
			Comment:   v.Comment,
			Days:      int32(v.Days),
			Status:    v.Status,
			Title:     house.Title,
			ImgUrl:    house.Index_image_url,
			Credit:    v.Credit,
		}
		orderResp = append(orderResp, orderTemp)
	}
	return orderResp, nil
}

// UpdateStatusByLandlord 房东接单/拒单：仅 WAIT_ACCEPT 可操作，且必须是房源房东
func UpdateStatusByLandlord(action, id, reason, userName string) error {
	oid, err := strconv.Atoi(id)
	if err != nil || oid <= 0 {
		return ErrOrderParam
	}
	if action != "accept" && action != "reject" {
		return ErrOrderParam
	}
	if action == "reject" && reason == "" {
		return ErrOrderParam
	}

	uid, err := FindUserIDByName(userName)
	if err != nil {
		return err
	}

	order, err := FindOrderByID(uint(oid))
	if err != nil {
		return err
	}
	if order.Status != OrderStatusWaitAccept {
		return ErrOrderStatus
	}

	var house House
	if err := GlobalConn.Where("id = ?", order.HouseId).First(&house).Error; err != nil {
		return err
	}
	if house.UserId != uid {
		return ErrNotLandlord
	}

	db := GlobalConn.Model(&OrderHouse{}).Where("id = ? AND status = ?", oid, OrderStatusWaitAccept)
	if action == "accept" {
		res := db.Update("status", OrderStatusWaitComment)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrOrderStatus
		}
		return nil
	}

	res := db.Updates(map[string]interface{}{"status": OrderStatusRejected, "comment": reason})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrOrderStatus
	}
	return nil
}

// CommentOrderByTenant 租户评价：仅本人且 WAIT_COMMENT → COMPLETE
func CommentOrderByTenant(orderIDStr, userName, comment string) error {
	if comment == "" {
		return ErrOrderParam
	}
	oid, err := strconv.Atoi(orderIDStr)
	if err != nil || oid <= 0 {
		return ErrOrderParam
	}
	uid, err := FindUserIDByName(userName)
	if err != nil {
		return err
	}
	order, err := FindOrderByID(uint(oid))
	if err != nil {
		return err
	}
	if order.UserId != uid {
		return fmt.Errorf("order owner mismatch")
	}
	if order.Status != OrderStatusWaitComment {
		return ErrOrderStatus
	}
	res := GlobalConn.Model(&OrderHouse{}).
		Where("id = ? AND status = ?", oid, OrderStatusWaitComment).
		Updates(map[string]interface{}{"status": OrderStatusComplete, "comment": comment})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrOrderStatus
	}
	return nil
}
