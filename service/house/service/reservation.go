package service

import (
	"errors"
	"fmt"
	"house/conf"
	"house/model"
	"strconv"
	"time"
)

var (
	ErrConflict = errors.New("house already reserved")
	ErrParam    = errors.New("invalid reservation param")
)

// ReserveInventory 同步预留日历库存
func ReserveInventory(houseID, orderID, startDate, endDate string) error {
	hid, err := strconv.Atoi(houseID)
	if err != nil || hid <= 0 {
		return ErrParam
	}
	oid, err := strconv.Atoi(orderID)
	if err != nil || oid <= 0 {
		return ErrParam
	}
	begin, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return ErrParam
	}
	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return ErrParam
	}
	if !end.After(begin) {
		return ErrParam
	}

	h, err := model.FindHouseByID(houseID)
	if err != nil {
		return err
	}
	days := int(end.Sub(begin).Hours()) / 24
	if days < 1 {
		days = 1
	}
	if h.Min_days > 0 && days < h.Min_days {
		return ErrParam
	}
	if h.Max_days > 0 && days > h.Max_days {
		return ErrParam
	}

	expiredAt := time.Now().Add(time.Duration(conf.ReserveExpireMinutes) * time.Minute)
	err = model.InsertReservation(uint(hid), uint(oid), begin, end, expiredAt)
	if errors.Is(err, model.ErrReservationConflict) {
		return ErrConflict
	}
	if err != nil {
		fmt.Println("预留失败", err)
		return err
	}
	InvalidateHouseListCache()
	return nil
}

// ConfirmReservation 支付确认：RESERVED → CONFIRMED
func ConfirmReservation(orderID uint) (houseID uint, err error) {
	var r model.HouseReservation
	err = model.GlobalConn.Where("order_id = ?", orderID).First(&r).Error
	if err != nil {
		return 0, err
	}
	if r.Status == model.ReserveStatusConfirmed {
		return r.HouseId, nil
	}
	if err := model.ConfirmReservationByOrderID(orderID); err != nil {
		return 0, err
	}
	_ = model.IncrHouseOrderCount(r.HouseId)
	InvalidateAllHouseCache(strconv.Itoa(int(r.HouseId)))
	return r.HouseId, nil
}

// ReleaseReservation 释放预留
func ReleaseReservation(orderID uint) error {
	var r model.HouseReservation
	err := model.GlobalConn.Where("order_id = ?", orderID).First(&r).Error
	if err == nil {
		houseID := r.HouseId
		if err := model.ReleaseReservationByOrderID(orderID); err != nil {
			return err
		}
		InvalidateAllHouseCache(strconv.Itoa(int(houseID)))
		return nil
	}
	return model.ReleaseReservationByOrderID(orderID)
}
