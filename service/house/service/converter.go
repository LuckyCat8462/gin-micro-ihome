package service

import (
	"house/model"
	house "house/proto"
	"path/filepath"
	"strings"
)

var indexLocalImages = []string{
	"/home/images/home01.jpg",
	"/home/images/home02.jpg",
	"/home/images/home03.jpg",
}

func getLocalBannerImage(index int, indexImageUrl string) string {
	if indexImageUrl != "" {
		name := filepath.Base(indexImageUrl)
		for _, img := range []string{"home01.jpg", "home02.jpg", "home03.jpg"} {
			if name == img || strings.Contains(indexImageUrl, img) {
				return "/home/images/" + img
			}
		}
	}
	return indexLocalImages[index%len(indexLocalImages)]
}

func toHouseProto(h model.House, area model.Area) *house.Houses {
	return &house.Houses{
		Title:      h.Title,
		Address:    h.Address,
		Ctime:      h.CreatedAt.Format("2006-01-02 15:04:05"),
		HouseId:    int32(h.ID),
		OrderCount: int32(h.Order_count),
		Price:      int32(h.Price),
		RoomCount:  int32(h.Room_count),
		AreaName:   area.Name,
	}
}

func toIndexHouseProto(h model.House, area model.Area, index int) *house.Houses {
	p := toHouseProto(h, area)
	p.ImgUrl = getLocalBannerImage(index, h.Index_image_url)
	return p
}

func toDetailProto(h model.House, owner model.User, viewer model.User) house.DetailData {
	detail := &house.HouseDetail{
		Acreage:   int32(h.Acreage),
		Address:   h.Address,
		Beds:      h.Beds,
		Capacity:  int32(h.Capacity),
		Deposit:   int32(h.Deposit),
		Hid:       int32(h.ID),
		MaxDays:   int32(h.Max_days),
		MinDays:   int32(h.Min_days),
		Price:     int32(h.Price),
		RoomCount: int32(h.Room_count),
		Title:     h.Title,
		Unit:      h.Unit,
		UserName:  owner.Name,
		UserId:    int32(owner.ID),
	}
	return house.DetailData{
		House:  detail,
		UserId: int32(viewer.ID),
	}
}
