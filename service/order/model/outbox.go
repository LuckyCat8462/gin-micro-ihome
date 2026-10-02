package model

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrOrderNotFound = errors.New("order not found")
	ErrOrderParam    = errors.New("invalid order param")
	ErrSelfBook      = errors.New("cannot book own house")
)

func InsertOrderWaitPay(houseId, beginDate, endDate, userName string) (int, error) {
	var order OrderHouse

	hid, err := strconv.Atoi(houseId)
	if err != nil || hid <= 0 {
		return 0, ErrOrderParam
	}
	order.HouseId = uint(hid)

	bDate, err := time.Parse("2006-01-02", beginDate)
	if err != nil {
		return 0, ErrOrderParam
	}
	eDate, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return 0, ErrOrderParam
	}
	if !eDate.After(bDate) {
		return 0, ErrOrderParam
	}
	order.Begin_date = bDate
	order.End_date = eDate

	var userData UserData
	if err := GlobalConn.Raw("select id from user where name = ?", userName).Scan(&userData).Error; err != nil {
		fmt.Println("获取用户数据错误", err)
		return 0, err
	}
	if userData.Id == 0 {
		return 0, ErrOrderParam
	}

	order.Days = int(eDate.Sub(bDate).Hours()) / 24
	if order.Days < 1 {
		return 0, ErrOrderParam
	}
	order.Status = OrderStatusWaitPay

	var house House
	if err := GlobalConn.Where("id = ?", hid).First(&house).Error; err != nil {
		return 0, err
	}
	if house.UserId == uint(userData.Id) {
		return 0, ErrSelfBook
	}
	if house.Min_days > 0 && order.Days < house.Min_days {
		return 0, ErrOrderParam
	}
	if house.Max_days > 0 && order.Days > house.Max_days {
		return 0, ErrOrderParam
	}
	order.House_price = house.Price
	order.Amount = house.Price * order.Days
	order.UserId = uint(userData.Id)

	if err := GlobalConn.Create(&order).Error; err != nil {
		fmt.Println("插入订单失败", err)
		return 0, err
	}
	return int(order.ID), nil
}

func MarkOrderCancelled(orderID int) error {
	return GlobalConn.Model(&OrderHouse{}).Where("id = ?", orderID).
		Update("status", OrderStatusCancelled).Error
}

func FindOrderByID(orderID uint) (OrderHouse, error) {
	var o OrderHouse
	err := GlobalConn.Where("id = ?", orderID).First(&o).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return o, ErrOrderNotFound
	}
	return o, err
}

func FindUserIDByName(userName string) (uint, error) {
	var userData UserData
	if err := GlobalConn.Raw("select id from user where name = ?", userName).Scan(&userData).Error; err != nil {
		return 0, err
	}
	if userData.Id == 0 {
		return 0, ErrOrderParam
	}
	return uint(userData.Id), nil
}

// InsertOutbox 写入本地消息表
func InsertOutbox(tx *gorm.DB, orderID uint, eventType, payload string) error {
	msg := LocalMessage{
		MessageId:   uuid.New().String(),
		OrderId:     orderID,
		EventType:   eventType,
		Payload:     payload,
		Status:      MsgStatusPending,
		NextRetryAt: time.Now(),
	}
	return tx.Create(&msg).Error
}

// PayOrderTx 支付：WAIT_PAY → PAID + outbox CONFIRM
func PayOrderTx(orderID uint, userID uint) error {
	return GlobalConn.Transaction(func(tx *gorm.DB) error {
		var o OrderHouse
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", orderID).First(&o).Error; err != nil {
			return err
		}
		if userID == 0 || o.UserId != userID {
			return fmt.Errorf("order owner mismatch")
		}
		if o.Status != OrderStatusWaitPay {
			return fmt.Errorf("order status not WAIT_PAY")
		}
		if err := tx.Model(&o).Update("status", OrderStatusPaid).Error; err != nil {
			return err
		}
		payload := fmt.Sprintf(`{"order_id":%d,"house_id":%d,"begin_date":"%s","end_date":"%s"}`,
			o.ID, o.HouseId, o.Begin_date.Format("2006-01-02"), o.End_date.Format("2006-01-02"))
		return InsertOutbox(tx, o.ID, EventConfirm, payload)
	})
}

// CancelOrderTx 取消：WAIT_PAY/PAID → CANCELLED + outbox CANCEL
func CancelOrderTx(orderID uint) error {
	return GlobalConn.Transaction(func(tx *gorm.DB) error {
		var o OrderHouse
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", orderID).First(&o).Error; err != nil {
			return err
		}
		if o.Status == OrderStatusCancelled {
			return nil
		}
		if o.Status != OrderStatusWaitPay && o.Status != OrderStatusPaid {
			return fmt.Errorf("order cannot cancel, status=%s", o.Status)
		}
		if err := tx.Model(&o).Update("status", OrderStatusCancelled).Error; err != nil {
			return err
		}
		payload := fmt.Sprintf(`{"order_id":%d,"house_id":%d,"begin_date":"%s","end_date":"%s"}`,
			o.ID, o.HouseId, o.Begin_date.Format("2006-01-02"), o.End_date.Format("2006-01-02"))
		return InsertOutbox(tx, o.ID, EventCancel, payload)
	})
}

// MarkOrderWaitAccept PAID → WAIT_ACCEPT（库存确认回调）
func MarkOrderWaitAccept(orderID uint) error {
	return GlobalConn.Model(&OrderHouse{}).
		Where("id = ? AND status = ?", orderID, OrderStatusPaid).
		Update("status", OrderStatusWaitAccept).Error
}

func ListPendingOutbox(limit int) ([]LocalMessage, error) {
	var list []LocalMessage
	err := GlobalConn.Where("status = ? AND next_retry_at <= ?", MsgStatusPending, time.Now()).
		Order("id asc").Limit(limit).Find(&list).Error
	return list, err
}

func MarkOutboxSent(id uint) error {
	return GlobalConn.Model(&LocalMessage{}).Where("id = ?", id).
		Updates(map[string]interface{}{"status": MsgStatusSent}).Error
}

func MarkOutboxRetry(id uint, retryCount int, next time.Time) error {
	status := MsgStatusPending
	if retryCount >= 10 {
		status = MsgStatusFailed
	}
	return GlobalConn.Model(&LocalMessage{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"retry_count":   retryCount,
			"next_retry_at": next,
			"status":        status,
		}).Error
}

func ListTimeoutWaitPayOrders(timeoutMinutes int) ([]OrderHouse, error) {
	deadline := time.Now().Add(-time.Duration(timeoutMinutes) * time.Minute)
	var list []OrderHouse
	err := GlobalConn.Where("status = ? AND created_at < ?", OrderStatusWaitPay, deadline).
		Limit(100).Find(&list).Error
	return list, err
}
