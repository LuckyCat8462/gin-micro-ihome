package service

import (
	"house/model"
	house "house/proto"
	"strconv"
)

// PublishHouse 发布房源，写后清除读缓存
func PublishHouse(req *house.Request) (int, error) {
	user, err := model.FindUserByName(req.UserName)
	if err != nil {
		return 0, err
	}

	price, _ := strconv.Atoi(req.Price)
	roomCount, _ := strconv.Atoi(req.RoomCount)
	capacity, _ := strconv.Atoi(req.Capacity)
	deposit, _ := strconv.Atoi(req.Deposit)
	minDays, _ := strconv.Atoi(req.MinDays)
	maxDays, _ := strconv.Atoi(req.MaxDays)
	acreage, _ := strconv.Atoi(req.Acreage)
	areaId, _ := strconv.Atoi(req.AreaId)

	h := model.House{
		Address:    req.Address,
		UserId:     uint(user.ID),
		Title:      req.Title,
		Price:      price,
		Room_count: roomCount,
		Unit:       req.Unit,
		Capacity:   capacity,
		Beds:       req.Beds,
		Deposit:    deposit,
		Min_days:   minDays,
		Max_days:   maxDays,
		Acreage:    acreage,
		AreaId:     uint(areaId),
	}

	if err := model.InsertHouse(&h); err != nil {
		return 0, err
	}

	InvalidateAllHouseCache("")
	return int(h.ID), nil
}

// UploadHouseImage 保存房源图片路径，写后清除相关缓存
func UploadHouseImage(houseID, imgPath string) error {
	h, err := model.FindHouseByID(houseID)
	if err != nil {
		return err
	}

	if h.Index_image_url == "" {
		if err := model.UpdateHouseIndexImage(houseID, imgPath); err != nil {
			return err
		}
	} else {
		if err := model.InsertHouseImage(houseID, imgPath); err != nil {
			return err
		}
	}

	InvalidateAllHouseCache(houseID)
	return nil
}
