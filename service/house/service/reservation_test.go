package service

import (
	"errors"
	"fmt"
	"house/model"
	"strconv"
	"testing"
	"time"
)

func TestReserveInventoryInvalidParams(t *testing.T) {
	cases := []struct {
		name                        string
		houseID, orderID, begin, end string
	}{
		{"bad house id", "x", "1", "2026-08-01", "2026-08-03"},
		{"zero house id", "0", "1", "2026-08-01", "2026-08-03"},
		{"bad order id", "1", "x", "2026-08-01", "2026-08-03"},
		{"bad begin", "1", "1", "08-01", "2026-08-03"},
		{"bad end", "1", "1", "2026-08-01", "08-03"},
		{"end not after begin", "1", "1", "2026-08-03", "2026-08-01"},
		{"same day", "1", "1", "2026-08-01", "2026-08-01"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ReserveInventory(tc.houseID, tc.orderID, tc.begin, tc.end)
			if !errors.Is(err, ErrParam) {
				t.Fatalf("want ErrParam, got %v", err)
			}
		})
	}
}

func ensureHouseDB(t *testing.T) {
	t.Helper()
	if model.GlobalConn != nil {
		sqlDB, err := model.GlobalConn.DB()
		if err == nil && sqlDB.Ping() == nil {
			return
		}
	}
	_, err := model.InitDb()
	if err != nil || model.GlobalConn == nil {
		t.Skip("mysql unavailable:", err)
	}
	if sqlDB, err := model.GlobalConn.DB(); err != nil || sqlDB.Ping() != nil {
		t.Skip("mysql unavailable:", err)
	}
}

func seedHouse(t *testing.T) model.House {
	t.Helper()
	suffix := strconv.FormatInt(time.Now().UnixNano(), 10)
	u := model.User{
		Name:          "ut_house_" + suffix,
		Password_hash: "x",
		Mobile:        mobileFromSuffix(suffix),
	}
	if err := model.GlobalConn.Create(&u).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	h := model.House{
		UserId:   uint(u.ID),
		Title:    "ut-house-" + suffix,
		Price:    100,
		Min_days: 1,
		Max_days: 0,
	}
	if err := model.GlobalConn.Create(&h).Error; err != nil {
		t.Fatalf("seed house: %v", err)
	}
	t.Cleanup(func() {
		_ = model.GlobalConn.Where("house_id = ?", h.ID).Delete(&model.HouseReservation{}).Error
		_ = model.GlobalConn.Delete(&h).Error
		_ = model.GlobalConn.Delete(&u).Error
	})
	return h
}

func mobileFromSuffix(suffix string) string {
	// 11-digit unique-ish mobile for unique index
	if len(suffix) >= 10 {
		return "1" + suffix[len(suffix)-10:]
	}
	return fmt.Sprintf("1%010s", suffix)
}

func TestReserveInventoryDateConflict(t *testing.T) {
	ensureHouseDB(t)
	h := seedHouse(t)

	oid1 := uint(time.Now().UnixNano()%900000000 + 100000000)
	oid2 := oid1 + 1

	if err := ReserveInventory(strconv.Itoa(int(h.ID)), strconv.Itoa(int(oid1)), "2030-01-10", "2030-01-15"); err != nil {
		t.Fatalf("first reserve: %v", err)
	}
	err := ReserveInventory(strconv.Itoa(int(h.ID)), strconv.Itoa(int(oid2)), "2030-01-12", "2030-01-18")
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("overlap want ErrConflict, got %v", err)
	}
}

func TestConfirmAndReleaseReservationIdempotent(t *testing.T) {
	ensureHouseDB(t)
	h := seedHouse(t)

	oid := uint(time.Now().UnixNano()%900000000 + 200000000)
	if err := ReserveInventory(strconv.Itoa(int(h.ID)), strconv.Itoa(int(oid)), "2030-02-01", "2030-02-05"); err != nil {
		t.Fatalf("reserve: %v", err)
	}

	hid1, err := ConfirmReservation(oid)
	if err != nil {
		t.Fatalf("confirm#1: %v", err)
	}
	hid2, err := ConfirmReservation(oid)
	if err != nil {
		t.Fatalf("confirm#2: %v", err)
	}
	if hid1 != h.ID || hid2 != h.ID {
		t.Fatalf("house id mismatch: %d %d want %d", hid1, hid2, h.ID)
	}

	var r model.HouseReservation
	if err := model.GlobalConn.Where("order_id = ?", oid).First(&r).Error; err != nil {
		t.Fatalf("load reservation: %v", err)
	}
	if r.Status != model.ReserveStatusConfirmed {
		t.Fatalf("status want CONFIRMED, got %s", r.Status)
	}

	if err := ReleaseReservation(oid); err != nil {
		t.Fatalf("release#1: %v", err)
	}
	if err := ReleaseReservation(oid); err != nil {
		t.Fatalf("release#2: %v", err)
	}
	if err := model.GlobalConn.Where("order_id = ?", oid).First(&r).Error; err != nil {
		t.Fatalf("reload reservation: %v", err)
	}
	if r.Status != model.ReserveStatusReleased {
		t.Fatalf("status want RELEASED, got %s", r.Status)
	}
}
