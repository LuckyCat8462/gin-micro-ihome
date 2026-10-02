package handler

import (
	"context"
	"errors"
	"fmt"
	"house/storage"
	house "house/proto"
	houseservice "house/service"
	"house/utils"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

type House struct{}

// Return a new handler
func New() *House {
	return &House{}
}

func (e *House) PubHouse(ctx context.Context, req *house.Request, rsp *house.Response) error {
	houseId, err := houseservice.PublishHouse(req)
	fmt.Println(req)
	fmt.Println("pubHouse")

	if err != nil {
		rsp.Errno = utils.RECODE_DBERR
		rsp.Errmsg = utils.RecodeText(utils.RECODE_DBERR)
		return nil
	}

	rsp.Errno = utils.RECODE_OK
	rsp.Errmsg = utils.RecodeText(utils.RECODE_OK)

	var h house.HouseData
	h.HouseId = strconv.Itoa(houseId)
	rsp.Data = &h

	return nil
}

func (e *House) UploadHouseImg(ctx context.Context, req *house.ImgReq, resp *house.ImgResp) error {
	if req.HouseId == "" || len(req.ImgData) == 0 {
		resp.Errno = utils.RECODE_PARAMERR
		resp.Errmsg = utils.RecodeText(utils.RECODE_PARAMERR)
		return nil
	}

	ext := strings.TrimPrefix(strings.ToLower(req.FileExt), ".")
	if ext == "" {
		ext = "jpg"
	}
	if ext != "jpg" && ext != "jpeg" && ext != "png" {
		resp.Errno = utils.RECODE_PARAMERR
		resp.Errmsg = "仅支持 jpg/png 图片"
		return nil
	}

	objectKey := fmt.Sprintf("houses/%s/%s.%s", req.HouseId, uuid.NewString(), ext)
	url, err := storage.Upload(ctx, objectKey, req.ImgData, storage.ContentTypeByExt(ext))
	if err != nil {
		fmt.Println("上传 MinIO 失败", err)
		resp.Errno = utils.RECODE_IOERR
		resp.Errmsg = utils.RecodeText(utils.RECODE_IOERR)
		return nil
	}

	if err := houseservice.UploadHouseImage(req.HouseId, url); err != nil {
		fmt.Println("保存房源图片失败", err)
		resp.Errno = utils.RECODE_DBERR
		resp.Errmsg = utils.RecodeText(utils.RECODE_DBERR)
		return nil
	}

	resp.Errno = utils.RECODE_OK
	resp.Errmsg = utils.RecodeText(utils.RECODE_OK)
	resp.Data = &house.ImgData{Url: url}
	return nil
}

func (e *House) GetHouseInfo(ctx context.Context, req *house.GetReq, resp *house.GetResp) error {
	houseInfos, err := houseservice.GetUserHouseList(req.UserName)
	if err != nil {
		resp.Errno = utils.RECODE_DBERR
		resp.Errmsg = utils.RecodeText(utils.RECODE_DBERR)
		return nil
	}

	resp.Errno = utils.RECODE_OK
	resp.Errmsg = utils.RecodeText(utils.RECODE_OK)

	var getData house.GetData
	getData.Houses = houseInfos
	resp.Data = &getData

	return nil
}

func (e *House) GetHouseDetail(ctx context.Context, req *house.DetailReq, resp *house.DetailResp) error {
	fmt.Println(req)
	respData, err := houseservice.GetHouseDetail(req.HouseId, req.UserName)

	if err != nil {
		resp.Errno = utils.RECODE_DATAERR
		resp.Errmsg = utils.RecodeText(utils.RECODE_DATAERR)
		return nil
	}

	resp.Errno = utils.RECODE_OK
	resp.Errmsg = utils.RecodeText(utils.RECODE_OK)
	resp.Data = &respData

	return nil
}

func (e *House) GetIndexHouse(ctx context.Context, req *house.IndexReq, resp *house.GetResp) error {
	houseResp, err := houseservice.GetIndexHouseList()
	if err != nil {
		resp.Errno = utils.RECODE_DBERR
		resp.Errmsg = utils.RecodeText(utils.RECODE_DBERR)
		return nil
	}

	resp.Errno = utils.RECODE_OK
	resp.Errmsg = utils.RecodeText(utils.RECODE_OK)

	resp.Data = &house.GetData{Houses: houseResp}

	return nil
}

func (e *House) SearchHouse(ctx context.Context, req *house.SearchReq, resp *house.GetResp) error {
	houseResp, totalPage, currentPage, err := houseservice.GetHouseList(req.Aid, req.Sd, req.Ed, req.Sk, req.P)
	if err != nil {
		if errors.Is(err, houseservice.ErrSearchParam) {
			resp.Errno = utils.RECODE_PARAMERR
			resp.Errmsg = "搜索日期参数错误"
			return nil
		}
		resp.Errno = utils.RECODE_DATAERR
		resp.Errmsg = utils.RecodeText(utils.RECODE_DATAERR)
		return nil
	}

	resp.Errno = utils.RECODE_OK
	resp.Errmsg = utils.RecodeText(utils.RECODE_OK)
	resp.Data = &house.GetData{
		Houses:      houseResp,
		TotalPage:   strconv.Itoa(totalPage),
		CurrentPage: strconv.Itoa(currentPage),
	}
	return nil
}

func (e *House) ReserveInventory(ctx context.Context, req *house.ReserveReq, resp *house.ReserveResp) error {
	err := houseservice.ReserveInventory(req.HouseId, req.OrderId, req.StartDate, req.EndDate)
	if errors.Is(err, houseservice.ErrConflict) {
		resp.Errno = utils.RECODE_DATAERR
		resp.Errmsg = "房间已被抢定"
		return nil
	}
	if errors.Is(err, houseservice.ErrParam) {
		resp.Errno = utils.RECODE_PARAMERR
		resp.Errmsg = utils.RecodeText(utils.RECODE_PARAMERR)
		return nil
	}
	if err != nil {
		resp.Errno = utils.RECODE_DBERR
		resp.Errmsg = utils.RecodeText(utils.RECODE_DBERR)
		return nil
	}
	resp.Errno = utils.RECODE_OK
	resp.Errmsg = utils.RecodeText(utils.RECODE_OK)
	return nil
}

func (e *House) ReleaseInventory(ctx context.Context, req *house.ReleaseReq, resp *house.ReleaseResp) error {
	orderID, _ := strconv.Atoi(req.OrderId)
	if err := houseservice.ReleaseReservation(uint(orderID)); err != nil {
		resp.Errno = utils.RECODE_DBERR
		resp.Errmsg = utils.RecodeText(utils.RECODE_DBERR)
		return nil
	}
	resp.Errno = utils.RECODE_OK
	resp.Errmsg = utils.RecodeText(utils.RECODE_OK)
	return nil
}
