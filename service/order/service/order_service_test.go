package service

import (
	"context"
	"errors"
	"fmt"
	"order/model"
	"order/utils"
	"strconv"
	"testing"
	"time"

	housePb "house/proto"

	"go-micro.dev/v5/client"
)

func TestPayOrderAndCancelOrderInvalidParam(t *testing.T) {
	if err := PayOrder("not-a-number", "anyone"); !errors.Is(err, ErrParam) {
		t.Fatalf("PayOrder want ErrParam, got %v", err)
	}
	if err := CancelOrder("bad", ""); !errors.Is(err, ErrParam) {
		t.Fatalf("CancelOrder want ErrParam, got %v", err)
	}
}

type mockHouseClient struct {
	reserveErrno string
	reserveErr   error
	released     []string
}

func (m *mockHouseClient) PubHouse(context.Context, *housePb.Request, ...client.CallOption) (*housePb.Response, error) {
	return nil, errors.New("not implemented")
}
func (m *mockHouseClient) UploadHouseImg(context.Context, *housePb.ImgReq, ...client.CallOption) (*housePb.ImgResp, error) {
	return nil, errors.New("not implemented")
}
func (m *mockHouseClient) GetHouseInfo(context.Context, *housePb.GetReq, ...client.CallOption) (*housePb.GetResp, error) {
	return nil, errors.New("not implemented")
}
func (m *mockHouseClient) GetHouseDetail(context.Context, *housePb.DetailReq, ...client.CallOption) (*housePb.DetailResp, error) {
	return nil, errors.New("not implemented")
}
func (m *mockHouseClient) GetIndexHouse(context.Context, *housePb.IndexReq, ...client.CallOption) (*housePb.GetResp, error) {
	return nil, errors.New("not implemented")
}
func (m *mockHouseClient) SearchHouse(context.Context, *housePb.SearchReq, ...client.CallOption) (*housePb.GetResp, error) {
	return nil, errors.New("not implemented")
}
func (m *mockHouseClient) ReserveInventory(context.Context, *housePb.ReserveReq, ...client.CallOption) (*housePb.ReserveResp, error) {
	if m.reserveErr != nil {
		return nil, m.reserveErr
	}
	return &housePb.ReserveResp{Errno: m.reserveErrno, Errmsg: "mock"}, nil
}
func (m *mockHouseClient) ReleaseInventory(_ context.Context, in *housePb.ReleaseReq, _ ...client.CallOption) (*housePb.ReleaseResp, error) {
	m.released = append(m.released, in.GetOrderId())
	return &housePb.ReleaseResp{Errno: utils.RECODE_OK}, nil
}

func ensureOrderServiceDB(t *testing.T) {
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

func seedOrderFixtures(t *testing.T) (user model.User, house model.House) {
	t.Helper()
	suffix := strconv.FormatInt(time.Now().UnixNano(), 10)
	landlord := model.User{
		Name:          "ut_svc_ll_" + suffix,
		Password_hash: "x",
		Mobile:        "182" + suffix[len(suffix)-8:],
	}
	if err := model.GlobalConn.Create(&landlord).Error; err != nil {
		t.Fatalf("seed landlord: %v", err)
	}
	user = model.User{
		Name:          "ut_svc_" + suffix,
		Password_hash: "x",
		Mobile:        "183" + suffix[len(suffix)-8:],
	}
	if err := model.GlobalConn.Create(&user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	house = model.House{
		UserId: uint(landlord.ID),
		Title:  "ut-svc-house-" + suffix,
		Price:  150,
	}
	if err := model.GlobalConn.Create(&house).Error; err != nil {
		t.Fatalf("seed house: %v", err)
	}
	t.Cleanup(func() {
		_ = model.GlobalConn.Where("user_id = ?", user.ID).Delete(&model.OrderHouse{}).Error
		_ = model.GlobalConn.Delete(&house).Error
		_ = model.GlobalConn.Delete(&user).Error
		_ = model.GlobalConn.Delete(&landlord).Error
	})
	return user, house
}

func mobile11(suffix string) string {
	if len(suffix) >= 10 {
		return "1" + suffix[len(suffix)-10:]
	}
	return fmt.Sprintf("1%010s", suffix)
}

func TestCreateOrderConflictCancelsOrder(t *testing.T) {
	ensureOrderServiceDB(t)
	user, house := seedOrderFixtures(t)

	prev := houseClient
	mock := &mockHouseClient{reserveErrno: utils.RECODE_DATAERR}
	houseClient = mock
	t.Cleanup(func() { houseClient = prev })

	oid, errno, err := CreateOrder(strconv.Itoa(int(house.ID)), "2030-06-01", "2030-06-03", user.Name)
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("want ErrConflict, got err=%v errno=%s", err, errno)
	}
	if errno != utils.RECODE_DATAERR {
		t.Fatalf("errno want %s, got %s", utils.RECODE_DATAERR, errno)
	}
	if oid <= 0 {
		t.Fatal("expected order id created before cancel")
	}

	o, err := model.FindOrderByID(uint(oid))
	if err != nil {
		t.Fatalf("FindOrderByID: %v", err)
	}
	if o.Status != model.OrderStatusCancelled {
		t.Fatalf("conflict path should cancel order, got %s", o.Status)
	}
	t.Cleanup(func() {
		_ = model.GlobalConn.Where("order_id = ?", oid).Delete(&model.LocalMessage{}).Error
		_ = model.GlobalConn.Delete(&o).Error
	})
}

func TestCreateOrderRPCErrorCancelsOrder(t *testing.T) {
	ensureOrderServiceDB(t)
	user, house := seedOrderFixtures(t)

	prev := houseClient
	houseClient = &mockHouseClient{reserveErr: errors.New("rpc down")}
	t.Cleanup(func() { houseClient = prev })

	oid, errno, err := CreateOrder(strconv.Itoa(int(house.ID)), "2030-06-10", "2030-06-12", user.Name)
	if err == nil {
		t.Fatal("want rpc error")
	}
	if errno != utils.RECODE_THIRDERR {
		t.Fatalf("errno want %s, got %s", utils.RECODE_THIRDERR, errno)
	}
	o, findErr := model.FindOrderByID(uint(oid))
	if findErr != nil {
		t.Fatalf("FindOrderByID: %v", findErr)
	}
	if o.Status != model.OrderStatusCancelled {
		t.Fatalf("rpc fail should cancel order, got %s", o.Status)
	}
	t.Cleanup(func() {
		_ = model.GlobalConn.Delete(&o).Error
	})
}

func TestCreateOrderSuccess(t *testing.T) {
	ensureOrderServiceDB(t)
	user, house := seedOrderFixtures(t)

	prev := houseClient
	houseClient = &mockHouseClient{reserveErrno: utils.RECODE_OK}
	t.Cleanup(func() { houseClient = prev })

	oid, errno, err := CreateOrder(strconv.Itoa(int(house.ID)), "2030-06-20", "2030-06-22", user.Name)
	if err != nil || errno != utils.RECODE_OK {
		t.Fatalf("CreateOrder failed: errno=%s err=%v", errno, err)
	}
	o, err := model.FindOrderByID(uint(oid))
	if err != nil {
		t.Fatalf("FindOrderByID: %v", err)
	}
	if o.Status != model.OrderStatusWaitPay {
		t.Fatalf("status want WAIT_PAY, got %s", o.Status)
	}
	t.Cleanup(func() {
		_ = model.GlobalConn.Delete(&o).Error
	})
}
