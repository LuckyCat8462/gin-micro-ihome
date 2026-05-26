package model

import (
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	house "house/proto"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// 首页轮播使用的本地静态图（对应 web/view/images/）
var indexLocalImages = []string{
	"/home/images/home01.jpg",
	"/home/images/home02.jpg",
	"/home/images/home03.jpg",
}

// getLocalBannerImage 根据数据库 index_image_url 或序号映射到本地图片
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

// GetUserHouse 获取房源信息
func GetUserHouse(userName string) ([]*house.Houses, error) {
	var houseInfos []*house.Houses

	//有用户名
	var user User
	if err := GlobalConn.Where("name = ?", userName).Find(&user).Error; err != nil {
		fmt.Println("获取当前用户信息错误", err)
		return nil, err
	}
	//else {
	//	fmt.Println("用户存在", user)
	//}

	//房源信息   一对多查询
	var houses []House
	//GlobalConn.Model(&user).Related(&houses)
	if err := GlobalConn.Where("user_id = ?", user.ID).Find(&houses).Error; err != nil {
		fmt.Println("获取当前用户信息错误", err)
		return nil, err
	} else {
		//测试用输出
		//fmt.Println("用户", user.Name, "存在房屋")
	}

	for _, v := range houses {
		var houseInfo house.Houses
		houseInfo.Title = v.Title
		houseInfo.Address = v.Address
		houseInfo.Ctime = v.CreatedAt.Format("2006-01-02 15:04:05")
		houseInfo.HouseId = int32(v.ID)
		//houseInfo.ImgUrl = "http://192.168.137.81:8888/" + v.Index_image_url
		houseInfo.OrderCount = int32(v.Order_count)
		houseInfo.Price = int32(v.Price)
		houseInfo.RoomCount = int32(v.Room_count)
		//houseInfo.UserAvatar = "http://192.168.137.81:8888/" + user.Avatar_url

		//获取地域信息
		var area Area
		//related函数可以是以主表关联从表,也可以是以从表关联主表
		GlobalConn.Where("id = ?", v.AreaId).Find(&area)
		houseInfo.AreaName = area.Name

		houseInfos = append(houseInfos, &houseInfo)

	}
	//测试用输出
	//fmt.Println("houseInfos", houseInfos)
	return houseInfos, nil
}

// 发布房屋
func AddHouse(request *house.Request) (int, error) {
	var houseInfo House
	//给house赋值
	houseInfo.Address = request.Address

	//根据userName获取userId
	var user User

	if err := GlobalConn.Where("name = ?", request.UserName).Find(&user).Error; err != nil {
		fmt.Println("查询当前用户失败", err)
		return 0, err
	}

	//sql中一对多插入,只是给外键赋值
	houseInfo.UserId = uint(user.ID)
	houseInfo.Title = request.Title
	//类型转换
	//Atoi(),将字符串转换为整形数的函数，会跳过空白字符。
	price, _ := strconv.Atoi(request.Price)
	roomCount, _ := strconv.Atoi(request.RoomCount)
	houseInfo.Price = price
	houseInfo.Room_count = roomCount
	houseInfo.Unit = request.Unit
	houseInfo.Capacity, _ = strconv.Atoi(request.Capacity)
	houseInfo.Beds = request.Beds
	houseInfo.Deposit, _ = strconv.Atoi(request.Deposit)
	houseInfo.Min_days, _ = strconv.Atoi(request.MinDays)
	houseInfo.Max_days, _ = strconv.Atoi(request.MaxDays)
	houseInfo.Acreage, _ = strconv.Atoi(request.MaxDays)
	//一对多插入
	areaId, _ := strconv.Atoi(request.AreaId)
	houseInfo.AreaId = uint(areaId)

	//request.Facility    所有的家具  房屋
	//for _, v := range request.Facility {
	//	id, _ := strconv.Atoi(v)
	//	fmt.Println(v)
	//	var fac Facility
	//	if err := GlobalConn.Where("id = ?", id).First(&fac).Error; err != nil {
	//		fmt.Println("家具id错误", err)
	//		return 0, err
	//	}
	//	//查询到了数据
	//	context.TODO()
	//	//houseInfo.Facilities = append(houseInfo.Facilities, &fac)
	//}

	if err := GlobalConn.Create(&houseInfo).Error; err != nil {
		fmt.Println("插入房屋信息失败", err)
		return 0, err
	}

	fmt.Println("新增房源", houseInfo)

	return int(houseInfo.ID), nil
}

// 把图片的凭证存储到数据中   更新   主图,次图  第一张图片是主图,剩下的图片是副图

func SaveHouseImg(houseId, imgPath string) error {
	/*return GlobalDB.Model(new(House)).Where("id = ?",houseId).
	Update("index_image_url",imgPath).Error*/
	//如何判断上传的图是当前房屋的第一张图片
	var houseInfo House
	if err := GlobalConn.Where("id = ?", houseId).Find(&houseInfo).Error; err != nil {
		fmt.Println("查询不到房屋信息", err)
		return err
	}

	if houseInfo.Index_image_url == "" {
		//说明没有上传过图片  现在上传的图片是主图
		return GlobalConn.Model(new(House)).Where("id = ?", houseId).
			Update("index_image_url", imgPath).Error
	}

	//上传的幅图
	var houseImg HouseImage
	houseImg.Url = imgPath
	hId, _ := strconv.Atoi(houseId)
	houseImg.HouseId = uint(hId)
	return GlobalConn.Create(&houseImg).Error
}

// GetHouseDetail 房屋详情
func GetHouseDetail(houseId, userName string) (house.DetailData, error) {

	var respData house.DetailData

	//给houseDetail赋值
	var houseDetail house.HouseDetail

	var houseInfo House
	if err := GlobalConn.Where("id = ?", houseId).Find(&houseInfo).Error; err != nil {
		fmt.Println("查询房屋信息错误", err)
		return respData, err
	}
	//else {
	//	//测试输出
	//	//fmt.Println(houseInfo)
	//}

	{
		houseDetail.Acreage = int32(houseInfo.Acreage)
		houseDetail.Address = houseInfo.Address
		houseDetail.Beds = houseInfo.Beds
		houseDetail.Capacity = int32(houseInfo.Capacity)
		houseDetail.Deposit = int32(houseInfo.Deposit)
		houseDetail.Hid = int32(houseInfo.ID)
		houseDetail.MaxDays = int32(houseInfo.Max_days)
		houseDetail.MinDays = int32(houseInfo.Min_days)
		houseDetail.Price = int32(houseInfo.Price)
		houseDetail.RoomCount = int32(houseInfo.Room_count)
		houseDetail.Title = houseInfo.Title
		houseDetail.Unit = houseInfo.Unit
		//if houseInfo.Index_image_url != "" {
		//	houseDetail.ImgUrls = append(houseDetail.ImgUrls, "http://192.168.137.81:8888/"+houseInfo.Index_image_url)
		//}
	}
	//fmt.Println("houseD", houseDetail)

	//获取房屋所有者信息
	var user User
	if err := GlobalConn.Where("id = ?", houseInfo.UserId).Find(&user).Error; err != nil {
		fmt.Println("查询房屋所有者信息错误", err)
		return respData, err
	}
	houseDetail.UserName = user.Name
	//houseDetail.UserAvatar = "http://192.168.137.81:8888/" + user.Avatar_url
	houseDetail.UserId = int32(user.ID)
	//
	respData.House = &houseDetail

	//获取当前浏览人信息
	var nowUser User
	if err := GlobalConn.Where("name = ?", userName).Find(&nowUser).Error; err != nil {
		fmt.Println("查询当前浏览人信息错误", err)
		return respData, err
	}
	//respData.UserId = int32(nowUser.ID)
	respData.UserId = 50
	fmt.Println("respData:", respData)

	return respData, nil
}

// 搜索房屋
func SearchHouse(areaId, sd, ed, sk string) ([]*house.Houses, error) {
	var houseInfos []House

	//   minDays  <  (结束时间  -  开始时间) <  max_days
	//计算一个差值  先把string类型转为time类型
	sdTime, _ := time.Parse("2006-01-02", sd)
	edTime, _ := time.Parse("2006-01-02", ed)
	dur := edTime.Sub(sdTime)
	fmt.Println("durtime:", dur)

	err := GlobalConn.Where("area_id = ?", areaId).
		Order("created_at desc").Find(&houseInfos).Error
	if err != nil {
		fmt.Println("搜索房屋失败", err)
		return nil, err
	}
	fmt.Println("houseinfos", houseInfos)

	//获取[]*house.Houses
	var housesResp []*house.Houses

	for _, v := range houseInfos {
		var houseTemp house.Houses
		houseTemp.Address = v.Address
		//根据房屋信息获取地域信息
		//var user User
		var area Area

		//GlobalDB.Model(&v).Related(&area).Related(&user)

		houseTemp.AreaName = area.Name
		houseTemp.Ctime = v.CreatedAt.Format("2006-01-02 15:04:05")
		houseTemp.HouseId = int32(v.ID)
		houseTemp.OrderCount = int32(v.Order_count)
		houseTemp.Price = int32(v.Price)
		houseTemp.RoomCount = int32(v.Room_count)
		houseTemp.Title = v.Title
		//houseTemp.ImgUrl = "http://192.168.137.81:8888/"+v.Index_image_url
		//houseTemp.UserAvatar = "http://192.168.137.81:8888/"+user.Avatar_url

		housesResp = append(housesResp, &houseTemp)

	}

	return housesResp, nil
}

// 获取轮播图房屋信息
func GetIndexHouse() ([]*house.Houses, error) {

	var housesResp []*house.Houses

	var houses []House
	if err := GlobalConn.Order("created_at desc").Limit(3).Find(&houses).Error; err != nil {
		fmt.Println("获取房屋信息失败", err)
		return nil, err
	}

	for i, v := range houses {
		var houseTemp house.Houses
		houseTemp.Address = v.Address
		var area Area
		if v.AreaId > 0 {
			GlobalConn.First(&area, v.AreaId)
		}
		houseTemp.AreaName = area.Name
		houseTemp.Ctime = v.CreatedAt.Format("2006-01-02 15:04:05")
		houseTemp.HouseId = int32(v.ID)
		// 读取 DB 中的 index_image_url，映射到 web/view/images 下 home01~03
		houseTemp.ImgUrl = getLocalBannerImage(i, v.Index_image_url)
		houseTemp.OrderCount = int32(v.Order_count)
		houseTemp.Price = int32(v.Price)
		houseTemp.RoomCount = int32(v.Room_count)
		houseTemp.Title = v.Title

		housesResp = append(housesResp, &houseTemp)
	}
	fmt.Println(housesResp)

	return housesResp, nil
}
