package service

import (
	"fmt"
	"house/conf"
	"house/model"
	house "house/proto"
	"time"
)

// GetHouseList 搜索房源列表：Cache-Aside，支持空结果缓存
func GetHouseList(aid, sd, ed, sk string) ([]*house.Houses, error) {
	key := cacheKey(aid, sd, ed, sk)

	if cached, err := getFromCache(key); err == nil {
		fmt.Println("从 Redis 获取房源列表")
		return cached, nil
	}

	fmt.Println("从 MySQL 获取房源列表")
	houses, err := searchHousesFromDB(aid, sd, ed, sk)
	if err != nil {
		return nil, err
	}

	if err := setCache(key, houses, conf.HouseListExpire); err != nil {
		fmt.Println("写入房源列表缓存失败", err)
	}
	return houses, nil
}

func searchHousesFromDB(areaID, sd, ed, sk string) ([]*house.Houses, error) {
	sdTime, _ := time.Parse("2006-01-02", sd)
	edTime, _ := time.Parse("2006-01-02", ed)
	dur := edTime.Sub(sdTime)
	fmt.Println("durtime:", dur, "sk:", sk)

	houseRows, err := model.FindHousesByAreaID(areaID)
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

// GetIndexHouseList 首页轮播房源，带 Redis 缓存
func GetIndexHouseList() ([]*house.Houses, error) {
	if cached, err := getFromCache(indexHouseKey); err == nil {
		fmt.Println("从 Redis 获取首页房源")
		return cached, nil
	}

	fmt.Println("从 MySQL 获取首页房源")
	houseRows, err := model.FindLatestHouses(3)
	if err != nil {
		return nil, err
	}

	var result []*house.Houses
	for i, h := range houseRows {
		area, _ := model.FindAreaByID(h.AreaId)
		result = append(result, toIndexHouseProto(h, area, i))
	}

	if err := setCache(indexHouseKey, result, conf.IndexHouseExpire); err != nil {
		fmt.Println("写入首页房源缓存失败", err)
	}
	return result, nil
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

// GetHouseDetail 获取房源详情，按 houseId 缓存
func GetHouseDetail(houseID, userName string) (house.DetailData, error) {
	var empty house.DetailData
	key := detailCacheKey(houseID)

	if cached, err := getDetailFromCache(key); err == nil {
		fmt.Println("从 Redis 获取房源详情")
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

	viewer, err := model.FindUserByName(userName)
	if err != nil {
		return empty, err
	}

	detail := toDetailProto(h, owner, viewer)
	if err := setDetailCache(key, &detail, conf.HouseDetailExpire); err != nil {
		fmt.Println("写入房源详情缓存失败", err)
	}
	return detail, nil
}
