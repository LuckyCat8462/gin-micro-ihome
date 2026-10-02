package model

import (
	"fmt"
	"strconv"
	"time"
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

// FindHouseByID 按 ID 查询单条房源（含设施）
func FindHouseByID(houseID string) (House, error) {
	var h House
	err := GlobalConn.Preload("Facilities").Where("id = ?", houseID).First(&h).Error
	if err != nil {
		fmt.Println("查询房屋信息错误", err)
	}
	return h, err
}

// FindHousesByAreaID 按区域 ID 查询房源，按创建时间倒序
func FindHousesByAreaID(areaID string) ([]House, error) {
	return FindHousesForSearch(areaID, "new")
}

// FindHousesForSearch 按区域（可空=全部）与排序键查询房源
func FindHousesForSearch(areaID, sortKey string) ([]House, error) {
	var houses []House
	q := GlobalConn.Model(&House{})
	if areaID != "" {
		q = q.Where("area_id = ?", areaID)
	}
	switch sortKey {
	case "booking":
		q = q.Order("order_count desc, created_at desc")
	case "price-inc":
		q = q.Order("price asc, created_at desc")
	case "price-des":
		q = q.Order("price desc, created_at desc")
	default:
		q = q.Order("created_at desc")
	}
	err := q.Find(&houses).Error
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

// FindHouseImages 查询房源副图
func FindHouseImages(houseID string) ([]HouseImage, error) {
	var imgs []HouseImage
	err := GlobalConn.Where("house_id = ?", houseID).Find(&imgs).Error
	return imgs, err
}

// ReplaceHouseFacilities 替换房源设施关联
func ReplaceHouseFacilities(houseID uint, facilityIDs []int) error {
	var facs []Facility
	for _, id := range facilityIDs {
		if id <= 0 {
			continue
		}
		facs = append(facs, Facility{Id: id})
	}
	h := House{}
	h.ID = houseID
	return GlobalConn.Model(&h).Association("Facilities").Replace(facs)
}

// HouseComment 详情页评价展示结构
type HouseComment struct {
	Comment   string
	UserName  string
	CreatedAt time.Time
}

// FindHouseComments 查询房源有效评价
func FindHouseComments(houseID string, limit int) ([]HouseComment, error) {
	if limit <= 0 {
		limit = 20
	}
	var rows []HouseComment
	err := GlobalConn.Table("order_house").
		Select("order_house.comment as comment, user.name as user_name, order_house.created_at as created_at").
		Joins("LEFT JOIN user ON user.id = order_house.user_id").
		Where("order_house.house_id = ? AND order_house.comment <> '' AND order_house.deleted_at IS NULL", houseID).
		Order("order_house.id desc").
		Limit(limit).
		Scan(&rows).Error
	return rows, err
}
