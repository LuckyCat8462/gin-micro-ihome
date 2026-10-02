package service

import (
	"errors"
	"fmt"
	"house/conf"
	"house/model"
	house "house/proto"
	"strconv"
	"strings"
	"time"
)

var ErrSearchParam = errors.New("invalid search param")

// GetHouseList 搜索房源列表：Cache-Aside + singleflight，返回当前页与总页数
func GetHouseList(aid, sd, ed, sk, pageStr string) ([]*house.Houses, int, int, error) {
	if err := validateSearchDates(sd, ed); err != nil {
		return nil, 0, 0, err
	}
	if sk == "" {
		sk = "new"
	}
	page := parsePage(pageStr)
	key := cacheKey(aid, sd, ed, sk)
	all, err := getOrLoadHouses(key, conf.HouseListExpire, func() ([]*house.Houses, error) {
		fmt.Println("从 MySQL 获取房源列表")
		return searchHousesFromDB(aid, sd, ed, sk)
	})
	if err != nil {
		return nil, 0, 0, err
	}

	total := len(all)
	pageSize := conf.HouseSearchPageSize
	if pageSize <= 0 {
		pageSize = 10
	}
	if total == 0 {
		return []*house.Houses{}, 0, page, nil
	}
	totalPage := (total + pageSize - 1) / pageSize
	if page > totalPage {
		page = totalPage
	}
	start := (page - 1) * pageSize
	end := start + pageSize
	if end > total {
		end = total
	}
	return all[start:end], totalPage, page, nil
}

func parsePage(pageStr string) int {
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		return 1
	}
	return page
}

// validateSearchDates 允许双空（仅浏览）；若填写则必须成对且离店晚于入住
func validateSearchDates(sd, ed string) error {
	sd = strings.TrimSpace(sd)
	ed = strings.TrimSpace(ed)
	if sd == "" && ed == "" {
		return nil
	}
	if sd == "" || ed == "" {
		return ErrSearchParam
	}
	sdTime, err := time.Parse("2006-01-02", sd)
	if err != nil {
		return ErrSearchParam
	}
	edTime, err := time.Parse("2006-01-02", ed)
	if err != nil {
		return ErrSearchParam
	}
	if !edTime.After(sdTime) {
		return ErrSearchParam
	}
	return nil
}

func searchHousesFromDB(areaID, sd, ed, sk string) ([]*house.Houses, error) {
	houseRows, err := model.FindHousesForSearch(areaID, sk)
	if err != nil {
		return nil, err
	}

	occupied := map[uint]bool{}
	sd = strings.TrimSpace(sd)
	ed = strings.TrimSpace(ed)
	if sd != "" && ed != "" {
		sdTime, _ := time.Parse("2006-01-02", sd)
		edTime, _ := time.Parse("2006-01-02", ed)
		ids, err := model.FindOccupiedHouseIDs(sdTime, edTime)
		if err != nil {
			fmt.Println("查询占用房源失败", err)
			return nil, err
		}
		for _, id := range ids {
			occupied[id] = true
		}
	}

	var result []*house.Houses
	for _, h := range houseRows {
		if occupied[h.ID] {
			continue
		}
		area, _ := model.FindAreaByID(h.AreaId)
		result = append(result, toHouseProto(h, area))
	}
	return result, nil
}

// GetIndexHouseList 首页轮播房源，Cache-Aside + singleflight 防击穿
func GetIndexHouseList() ([]*house.Houses, error) {
	return getOrLoadHouses(indexHouseKey, conf.IndexHouseExpire, func() ([]*house.Houses, error) {
		fmt.Println("从 MySQL 获取首页房源")
		houseRows, err := model.FindLatestHouses(9)
		if err != nil {
			return nil, err
		}
		var result []*house.Houses
		for i, h := range houseRows {
			area, _ := model.FindAreaByID(h.AreaId)
			result = append(result, toIndexHouseProto(h, area, i))
		}
		return result, nil
	})
}

// GetUserHouseList 获取用户名下房源，不缓存
func GetUserHouseList(userName string) ([]*house.Houses, error) {
	user, err := model.FindUserByName(userName)
	if err != nil {
		return nil, err
	}

	houseRows, err := model.FindHousesByUserID(user.ID)
	if err != nil {
		return nil, err
	}
	var result []*house.Houses
	for _, h := range houseRows {
		area, _ := model.FindAreaByID(h.AreaId)
		result = append(result, toHouseProto(h, area))
	}
	return result, nil
}

// GetHouseDetail 获取房源详情，按 houseId 缓存（UserId 按当前访问者覆盖）
func GetHouseDetail(houseID, userName string) (house.DetailData, error) {
	var empty house.DetailData
	key := detailCacheKey(houseID)

	viewerID := int32(0)
	if strings.TrimSpace(userName) != "" {
		if viewer, err := model.FindUserByName(userName); err == nil {
			viewerID = int32(viewer.ID)
		}
	}

	if cached, err := getDetailFromCache(key); err == nil {
		fmt.Println("从 Redis 获取房源详情")
		cached.UserId = viewerID
		return *cached, nil
	}

	h, err := model.FindHouseByID(houseID)
	if err != nil {
		return empty, err
	}

	owner, err := model.FindUserByID(int(h.UserId))
	if err != nil {
		return empty, err
	}

	images, _ := model.FindHouseImages(houseID)

	var comments []*house.CommentData
	if rows, err := model.FindHouseComments(houseID, 20); err == nil {
		for _, row := range rows {
			comments = append(comments, &house.CommentData{
				Comment:  row.Comment,
				UserName: row.UserName,
				Ctime:    row.CreatedAt.Format("2006-01-02 15:04:05"),
			})
		}
	}

	detail := toDetailProto(h, owner, viewerID, images, comments)
	if err := setDetailCache(key, &detail, conf.HouseDetailExpire); err != nil {
		fmt.Println("写入房源详情缓存失败", err)
	}
	return detail, nil
}
