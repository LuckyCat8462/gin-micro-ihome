package handler

import (
	"context"
	"fmt"
	house "house/proto"
	houseservice "house/service"
	"house/utils"
	"strconv"
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
	fmt.Println("fdfs房屋图片待开发")

	////把图片存储到fastdfs中
	////初始化fdfs的客户端
	//fClient, _ := fdfs_client.NewFdfsClient("/etc/fdfs/client.conf")
	////上传图片到fdfs
	//fdfsResp, err := fClient.UploadByBuffer(req.ImgData, req.FileExt[1:])
	//if err != nil {
	//	resp.Errno = utils.RECODE_DATAERR
	//	resp.Errmsg = utils.RecodeText(utils.RECODE_DATAERR)
	//	return nil
	//}
	//
	//err = houseservice.UploadHouseImage(req.HouseId, fdfsResp.RemoteFileId)
	//if err != nil {
	//	resp.Errno = utils.RECODE_DBERR
	//	resp.Errmsg = utils.RecodeText(utils.RECODE_DBERR)
	//	return nil
	//}
	//
	//resp.Errno = utils.RECODE_OK
	//resp.Errmsg = utils.RecodeText(utils.RECODE_OK)
	//
	//var img house.ImgData
	//img.Url = "http://192.168.137.81:8888/" + fdfsResp.RemoteFileId
	//
	//resp.Data = &img

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
	fmt.Println("req", req)
	houseResp, err := houseservice.GetHouseList(req.Aid, req.Sd, req.Ed, req.Sk)
	fmt.Println("houseResp:", houseResp)
	if err != nil {
		resp.Errno = utils.RECODE_DATAERR
		resp.Errmsg = utils.RecodeText(utils.RECODE_DATAERR)
		return nil
	}

	resp.Errno = utils.RECODE_OK
	resp.Errmsg = utils.RecodeText(utils.RECODE_OK)

	resp.Data = &house.GetData{Houses: houseResp}
	return nil
}
