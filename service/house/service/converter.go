package service

import (
	"house/model"
	house "house/proto"
	"strings"
)

var defaultHouseImages = []string{
	"/home/images/home01.jpg",
	"/home/images/home02.jpg",
	"/home/images/home03.jpg",
}

func resolveHouseImgURL(indexImageURL string, fallbackIndex int) string {
	if strings.TrimSpace(indexImageURL) != "" {
		return indexImageURL
	}
	return defaultHouseImages[fallbackIndex%len(defaultHouseImages)]
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
		ImgUrl:     resolveHouseImgURL(h.Index_image_url, int(h.ID)),
	}
}

func toIndexHouseProto(h model.House, area model.Area, index int) *house.Houses {
	p := toHouseProto(h, area)
	if strings.TrimSpace(h.Index_image_url) == "" {
		p.ImgUrl = defaultHouseImages[index%len(defaultHouseImages)]
	}
	return p
}

func toDetailProto(h model.House, owner model.User, viewerID int32, images []model.HouseImage, comments []*house.CommentData) house.DetailData {
	var urls []string
	if strings.TrimSpace(h.Index_image_url) != "" {
		urls = append(urls, h.Index_image_url)
	}
	for _, img := range images {
		if strings.TrimSpace(img.Url) == "" {
			continue
		}
		if h.Index_image_url != "" && img.Url == h.Index_image_url {
			continue
		}
		urls = append(urls, img.Url)
	}
	if len(urls) == 0 {
		urls = append(urls, defaultHouseImages[int(h.ID)%len(defaultHouseImages)])
	}

	avatar := owner.Avatar_url
	if strings.TrimSpace(avatar) == "" {
		avatar = "/home/images/landlord01.jpg"
	}

	var facilities []int32
	for _, f := range h.Facilities {
		facilities = append(facilities, int32(f.Id))
	}

	detail := &house.HouseDetail{
		Acreage:    int32(h.Acreage),
		Address:    h.Address,
		Beds:       h.Beds,
		Capacity:   int32(h.Capacity),
		Deposit:    int32(h.Deposit),
		Hid:        int32(h.ID),
		MaxDays:    int32(h.Max_days),
		MinDays:    int32(h.Min_days),
		Price:      int32(h.Price),
		RoomCount:  int32(h.Room_count),
		Title:      h.Title,
		Unit:       h.Unit,
		UserName:   owner.Name,
		UserId:     int32(owner.ID),
		ImgUrls:    urls,
		UserAvatar: avatar,
		Facilities: facilities,
		Comments:   comments,
	}
	return house.DetailData{
		House:  detail,
		UserId: viewerID,
	}
}
