package model

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrReservationConflict = errors.New("reservation conflict")
	ErrReservationNotFound = errors.New("reservation not found")
)

// HasDateConflict 检查同房源是否存在日期重叠的有效预留（过期 RESERVED 不计）
func HasDateConflict(tx *gorm.DB, houseID uint, begin, end time.Time, excludeOrderID uint) (bool, error) {
	var count int64
	now := time.Now()
	q := tx.Model(&HouseReservation{}).
		Where("house_id = ?", houseID).
		Where("(status = ? OR (status = ? AND expired_at > ?))",
			ReserveStatusConfirmed, ReserveStatusReserved, now).
		Where("begin_date < ? AND end_date > ?", end, begin)
	if excludeOrderID > 0 {
		q = q.Where("order_id <> ?", excludeOrderID)
	}
	if err := q.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// InsertReservation 事务内锁定并插入预留；冲突返回 ErrReservationConflict
func InsertReservation(houseID, orderID uint, begin, end, expiredAt time.Time) error {
	return GlobalConn.Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		// 锁同房源已有有效预留行，避免并发穿透
		var existing []HouseReservation
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("house_id = ? AND (status = ? OR (status = ? AND expired_at > ?))",
				houseID, ReserveStatusConfirmed, ReserveStatusReserved, now).
			Find(&existing).Error; err != nil {
			return err
		}
		for _, r := range existing {
			if r.BeginDate.Before(end) && begin.Before(r.EndDate) {
				return ErrReservationConflict
			}
		}

		// 幂等：同 order_id 已存在则视为成功
		var old HouseReservation
		err := tx.Where("order_id = ?", orderID).First(&old).Error
		if err == nil {
			if old.Status == ReserveStatusReleased {
				return tx.Model(&old).Updates(map[string]interface{}{
					"status":     ReserveStatusReserved,
					"begin_date": begin,
					"end_date":   end,
					"expired_at": expiredAt,
					"house_id":   houseID,
				}).Error
			}
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		r := HouseReservation{
			HouseId:   houseID,
			OrderId:   orderID,
			BeginDate: begin,
			EndDate:   end,
			Status:    ReserveStatusReserved,
			ExpiredAt: expiredAt,
		}
		return tx.Create(&r).Error
	})
}

// ConfirmReservationByOrderID RESERVED → CONFIRMED，幂等
func ConfirmReservationByOrderID(orderID uint) error {
	var r HouseReservation
	err := GlobalConn.Where("order_id = ?", orderID).First(&r).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrReservationNotFound
	}
	if err != nil {
		return err
	}
	if r.Status == ReserveStatusConfirmed {
		return nil
	}
	if r.Status == ReserveStatusReleased {
		return fmt.Errorf("reservation already released")
	}
	return GlobalConn.Model(&r).Update("status", ReserveStatusConfirmed).Error
}

// ReleaseReservationByOrderID 释放预留，幂等；若曾确认则回滚成交计数
func ReleaseReservationByOrderID(orderID uint) error {
	var r HouseReservation
	err := GlobalConn.Where("order_id = ?", orderID).First(&r).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if r.Status == ReserveStatusReleased {
		return nil
	}
	wasConfirmed := r.Status == ReserveStatusConfirmed
	if err := GlobalConn.Model(&r).Update("status", ReserveStatusReleased).Error; err != nil {
		return err
	}
	if wasConfirmed {
		_ = DecrHouseOrderCount(r.HouseId)
	}
	return nil
}

// FindOccupiedHouseIDs 返回在 [begin,end) 区间内被占用的房源 ID
func FindOccupiedHouseIDs(begin, end time.Time) ([]uint, error) {
	var ids []uint
	now := time.Now()
	err := GlobalConn.Model(&HouseReservation{}).
		Where("(status = ? OR (status = ? AND expired_at > ?))",
			ReserveStatusConfirmed, ReserveStatusReserved, now).
		Where("begin_date < ? AND end_date > ?", end, begin).
		Distinct("house_id").
		Pluck("house_id", &ids).Error
	return ids, err
}

// IncrHouseOrderCount 确认预订后增加成交计数
func IncrHouseOrderCount(houseID uint) error {
	return GlobalConn.Model(&House{}).Where("id = ?", houseID).
		UpdateColumn("order_count", gorm.Expr("order_count + 1")).Error
}

// DecrHouseOrderCount 拒单/取消已确认预留时回滚成交计数
func DecrHouseOrderCount(houseID uint) error {
	return GlobalConn.Model(&House{}).Where("id = ? AND order_count > 0", houseID).
		UpdateColumn("order_count", gorm.Expr("order_count - 1")).Error
}

// ReleaseExpiredReservations 将过期 RESERVED 标记为 RELEASED
func ReleaseExpiredReservations() (int64, error) {
	res := GlobalConn.Model(&HouseReservation{}).
		Where("status = ? AND expired_at <= ?", ReserveStatusReserved, time.Now()).
		Update("status", ReserveStatusReleased)
	return res.RowsAffected, res.Error
}
