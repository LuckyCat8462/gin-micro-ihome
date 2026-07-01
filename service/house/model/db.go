package model

import (
	"fmt"
	"strconv"
)

// FindUserByID 按 ID 查询用户
func FindUserByID(id int) (User, error) {
	var user User
	err := GlobalConn.Where("id = ?", id).First(&user).Error
	if err != nil {
		fmt.Println("查询用户失败", err)
	}
	return user, err
}

func FindUserByName(name string) (User, error) {
	var user User
	err := GlobalConn.Where("name = ?", name).First(&user).Error
	if err != nil {
		fmt.Println("查询用户失败", err)
	}
	return user, err
}

// FindHousesByUserID 按用户 ID 查询其名下房源
func FindHousesByUserID(userID int) ([]House, error) {
	var houses []House
	err := GlobalConn.Where("user_id = ?", userID).Find(&houses).Error
	if err != nil {
		fmt.Println("查询用户房源失败", err)
	}
	return houses, err
}

// FindHouseByID 按 ID 查询单条房源
func FindHouseByID(houseID string) (House, error) {
	var h House
	err := GlobalConn.Where("id = ?", houseID).First(&h).Error
	if err != nil {
		fmt.Println("查询房屋信息错误", err)
	}
	return h, err
}

// FindHousesByAreaID 按区域 ID 查询房源，按创建时间倒序
func FindHousesByAreaID(areaID string) ([]House, error) {
	var houses []House
	err := GlobalConn.Where("area_id = ?", areaID).
		Order("created_at desc").Find(&houses).Error
	if err != nil {
		fmt.Println("搜索房屋失败", err)
	}
	return houses, err
}

// FindLatestHouses 查询最新 N 条房源
func FindLatestHouses(limit int) ([]House, error) {
	var houses []House
	err := GlobalConn.Order("created_at desc").Limit(limit).Find(&houses).Error
	if err != nil {
		fmt.Println("获取房屋信息失败", err)
	}
	return houses, err
}

// FindAreaByID 按 ID 查询区域
func FindAreaByID(id uint) (Area, error) {
	var area Area
	if id == 0 {
		return area, nil
	}
	err := GlobalConn.First(&area, id).Error
	return area, err
}

// InsertHouse 插入房源记录，成功后 h.ID 为自增主键
func InsertHouse(h *House) error {
	if err := GlobalConn.Create(h).Error; err != nil {
		fmt.Println("插入房屋信息失败", err)
		return err
	}
	fmt.Println("新增房源", h)
	return nil
}

// UpdateHouseIndexImage 更新房源主图
func UpdateHouseIndexImage(houseID, imgPath string) error {
	return GlobalConn.Model(new(House)).Where("id = ?", houseID).
		Update("index_image_url", imgPath).Error
}

// InsertHouseImage 插入房源副图
func InsertHouseImage(houseID, imgPath string) error {
	hId, _ := strconv.Atoi(houseID)
	img := HouseImage{
		Url:     imgPath,
		HouseId: uint(hId),
	}
	return GlobalConn.Create(&img).Error
}
