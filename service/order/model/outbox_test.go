package model

import (
	"fmt"
	"strconv"
	"testing"
	"time"
)

func ensureOrderDB(t *testing.T) {
	t.Helper()
	if GlobalConn != nil {
		sqlDB, err := GlobalConn.DB()
		if err == nil && sqlDB.Ping() == nil {
			return
		}
	}
	_, err := InitDb()
	if err != nil || GlobalConn == nil {
		t.Skip("mysql unavailable:", err)
	}
	if sqlDB, err := GlobalConn.DB(); err != nil || sqlDB.Ping() != nil {
		t.Skip("mysql unavailable:", err)
	}
}

type seedBundle struct {
	User  User
	House House
}

func seedUserAndHouse(t *testing.T) seedBundle {
	t.Helper()
	suffix := strconv.FormatInt(time.Now().UnixNano(), 10)
	landlord := User{
		Name:          "ut_order_ll_" + suffix,
		Password_hash: "x",
		Mobile:        "180" + suffix[len(suffix)-8:],
	}
	if err := GlobalConn.Create(&landlord).Error; err != nil {
		t.Fatalf("seed landlord: %v", err)
	}
	u := User{
		Name:          "ut_order_" + suffix,
		Password_hash: "x",
		Mobile:        "181" + suffix[len(suffix)-8:],
	}
	if err := GlobalConn.Create(&u).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	h := House{
		UserId:   uint(landlord.ID),
		Title:    "ut-order-house-" + suffix,
		Price:    200,
		Min_days: 1,
		Max_days: 0,
	}
	if err := GlobalConn.Create(&h).Error; err != nil {
		t.Fatalf("seed house: %v", err)
	}
	t.Cleanup(func() {
		var orderIDs []uint
		_ = GlobalConn.Model(&OrderHouse{}).Where("user_id = ?", u.ID).Pluck("id", &orderIDs)
		if len(orderIDs) > 0 {
			_ = GlobalConn.Where("order_id IN ?", orderIDs).Delete(&LocalMessage{}).Error
			_ = GlobalConn.Where("id IN ?", orderIDs).Delete(&OrderHouse{}).Error
		}
		_ = GlobalConn.Delete(&h).Error
		_ = GlobalConn.Delete(&u).Error
		_ = GlobalConn.Delete(&landlord).Error
	})
	return seedBundle{User: u, House: h}
}

func mobileFromSuffix(suffix string) string {
	if len(suffix) >= 10 {
		return "1" + suffix[len(suffix)-10:]
	}
	return fmt.Sprintf("1%010s", suffix)
}

func createWaitPayOrder(t *testing.T, b seedBundle, begin, end string) OrderHouse {
	t.Helper()
	oid, err := InsertOrderWaitPay(strconv.Itoa(int(b.House.ID)), begin, end, b.User.Name)
	if err != nil {
		t.Fatalf("InsertOrderWaitPay: %v", err)
	}
	o, err := FindOrderByID(uint(oid))
	if err != nil {
		t.Fatalf("FindOrderByID: %v", err)
	}
	t.Cleanup(func() {
		_ = GlobalConn.Where("order_id = ?", o.ID).Delete(&LocalMessage{}).Error
		_ = GlobalConn.Delete(&o).Error
	})
	return o
}

func TestPayOrderTxWritesPendingOutbox(t *testing.T) {
	ensureOrderDB(t)
	b := seedUserAndHouse(t)
	o := createWaitPayOrder(t, b, "2030-03-01", "2030-03-03")

	if err := PayOrderTx(o.ID, uint(b.User.ID)); err != nil {
		t.Fatalf("PayOrderTx: %v", err)
	}

	got, err := FindOrderByID(o.ID)
	if err != nil {
		t.Fatalf("reload order: %v", err)
	}
	if got.Status != OrderStatusPaid {
		t.Fatalf("status want PAID, got %s", got.Status)
	}

	var msgs []LocalMessage
	if err := GlobalConn.Where("order_id = ? AND event_type = ?", o.ID, EventConfirm).Find(&msgs).Error; err != nil {
		t.Fatalf("list outbox: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("want 1 CONFIRM outbox, got %d", len(msgs))
	}
	if msgs[0].Status != MsgStatusPending {
		t.Fatalf("outbox status want PENDING, got %s", msgs[0].Status)
	}

	if err := MarkOutboxSent(msgs[0].ID); err != nil {
		t.Fatalf("MarkOutboxSent: %v", err)
	}
	var sent LocalMessage
	if err := GlobalConn.First(&sent, msgs[0].ID).Error; err != nil {
		t.Fatalf("reload outbox: %v", err)
	}
	if sent.Status != MsgStatusSent {
		t.Fatalf("outbox status want SENT, got %s", sent.Status)
	}
}

func TestCancelOrderTxIdempotentWritesOneCancelOutbox(t *testing.T) {
	ensureOrderDB(t)
	b := seedUserAndHouse(t)
	o := createWaitPayOrder(t, b, "2030-03-10", "2030-03-12")

	if err := CancelOrderTx(o.ID); err != nil {
		t.Fatalf("CancelOrderTx#1: %v", err)
	}
	if err := CancelOrderTx(o.ID); err != nil {
		t.Fatalf("CancelOrderTx#2: %v", err)
	}

	got, err := FindOrderByID(o.ID)
	if err != nil {
		t.Fatalf("reload order: %v", err)
	}
	if got.Status != OrderStatusCancelled {
		t.Fatalf("status want CANCELLED, got %s", got.Status)
	}

	var cnt int64
	if err := GlobalConn.Model(&LocalMessage{}).
		Where("order_id = ? AND event_type = ?", o.ID, EventCancel).
		Count(&cnt).Error; err != nil {
		t.Fatalf("count cancel outbox: %v", err)
	}
	if cnt != 1 {
		t.Fatalf("want 1 CANCEL outbox after idempotent cancel, got %d", cnt)
	}
}

func TestListTimeoutWaitPayOrdersAndCancel(t *testing.T) {
	ensureOrderDB(t)
	b := seedUserAndHouse(t)
	o := createWaitPayOrder(t, b, "2030-04-01", "2030-04-03")

	past := time.Now().Add(-30 * time.Minute)
	if err := GlobalConn.Model(&o).Update("created_at", past).Error; err != nil {
		t.Fatalf("backdate created_at: %v", err)
	}

	list, err := ListTimeoutWaitPayOrders(15)
	if err != nil {
		t.Fatalf("ListTimeoutWaitPayOrders: %v", err)
	}
	found := false
	for _, item := range list {
		if item.ID == o.ID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("timeout list missing order %d", o.ID)
	}

	if err := CancelOrderTx(o.ID); err != nil {
		t.Fatalf("CancelOrderTx timeout path: %v", err)
	}
	got, err := FindOrderByID(o.ID)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if got.Status != OrderStatusCancelled {
		t.Fatalf("status want CANCELLED, got %s", got.Status)
	}

	var cnt int64
	_ = GlobalConn.Model(&LocalMessage{}).
		Where("order_id = ? AND event_type = ?", o.ID, EventCancel).
		Count(&cnt)
	if cnt != 1 {
		t.Fatalf("want CANCEL outbox for timeout release path, got %d", cnt)
	}
}

func TestConfirmOutboxIdempotentOnPaidStatus(t *testing.T) {
	ensureOrderDB(t)
	b := seedUserAndHouse(t)
	o := createWaitPayOrder(t, b, "2030-05-01", "2030-05-02")

	if err := PayOrderTx(o.ID, uint(b.User.ID)); err != nil {
		t.Fatalf("PayOrderTx: %v", err)
	}
	// 重复支付应失败（状态已非 WAIT_PAY），不会再写第二条 CONFIRM
	if err := PayOrderTx(o.ID, uint(b.User.ID)); err == nil {
		t.Fatal("second PayOrderTx should fail")
	}

	var cnt int64
	if err := GlobalConn.Model(&LocalMessage{}).
		Where("order_id = ? AND event_type = ?", o.ID, EventConfirm).
		Count(&cnt).Error; err != nil {
		t.Fatalf("count confirm: %v", err)
	}
	if cnt != 1 {
		t.Fatalf("want exactly 1 CONFIRM outbox, got %d", cnt)
	}
}
