package model

import (
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	Order "order/proto"
	"strconv"
	"time"
)

type UserData struct {
	Id int
}
type HouseData struct {
	Id int
}

func InsertOrder(houseId, beginDate, endDate, userName string) (int, error) {
	//获取插入对象
	var order OrderHouse

	//给对象赋值
	hid, _ := strconv.Atoi(houseId)
	order.HouseId = uint(hid)

	fmt.Println("order.houseid", order.HouseId)
	//把string类型的时间转换为time类型
	bDate, _ := time.Parse("2006-01-02", beginDate)
	order.Begin_date = bDate

	eDate, _ := time.Parse("2006-01-02", endDate)
	order.End_date = eDate

	//需要userId
	/*var user User
	GlobalConn.Where("name = ?",userName).Find(&user)*/
	//select id form user where name = userName

	var userData UserData
	if err := GlobalConn.Raw("select id from user where name = ?", userName).Scan(&userData).Error; err != nil {
		fmt.Println("获取用户数据错误", err)
		return 0, err
	}

	//获取days
	dur := eDate.Sub(bDate)
	order.Days = int(dur.Hours()) / 24
	order.Status = "WAIT_ACCEPT"

	//房屋的单价和总价
	var house House
	GlobalConn.Where("id = ?", hid).Find(&house).Select("price")
	order.House_price = house.Price
	order.Amount = house.Price * order.Days

	order.UserId = uint(userData.Id)
	if err := GlobalConn.Create(&order).Error; err != nil {
		fmt.Println("插入订单失败", err)
		return 0, err
	}
	return int(order.ID), nil
}

// 获取房东订单如何实现?
func GetOrderInfo(userName, role string) ([]*Order.OrdersData, error) {
	//最终需要的数据
	var orderResp []*Order.OrdersData
	//获取当前用户的所有订单
	var orders []OrderHouse

	var userData UserData
	//用原生查询的时候,查询的字段必须跟数据库中的字段保持一致
	GlobalConn.Raw("select id from user where name = ?", userName).Scan(&userData)
	var houseData HouseData

	//fmt.Println("测试role", userData.Id)
	//查询租户的所有的订单
	if role == "custom" {
		if err := GlobalConn.Where("user_id = ?", userData.Id).Find(&orders).Error; err != nil {
			fmt.Println("获取当前用户所有订单失败")
			return nil, err
		} else {
			for _, v := range orders {
				var orderTemp Order.OrdersData
				orderTemp.OrderId = int32(v.ID)
				orderTemp.EndDate = v.End_date.Format("2006-01-02")
				orderTemp.StartDate = v.Begin_date.Format("2006-01-02")
				orderTemp.Ctime = v.CreatedAt.Format("2006-01-02")
				orderTemp.Amount = int32(v.Amount)
				orderTemp.Comment = v.Comment
				orderTemp.Days = int32(v.Days)
				orderTemp.Status = v.Status
				fmt.Println("orderTemp", orderTemp)
				//关联house表
				var house House
				orderTemp.Title = house.Title
				orderResp = append(orderResp, &orderTemp)
			}
			return orderResp, nil
		}
	} else {
		//查询房东的订单  以房东视角来查看订单
		var houses []House
		if err := GlobalConn.Where("user_id = ?", userData.Id).Find(&houses).Error; err != nil {
			fmt.Println("查询houses无果，检查orderServiceModelFuc")
			return nil, err
		}
		for _, v := range houses {
			var tempOrders OrderHouse
			tempOrders.HouseId = v.ID
			tempOrders.UserId = v.UserId
			tempOrders.House_price = v.Price
			//GlobalConn.Model(&v).Related(&tempOrders)
			orders = append(orders, tempOrders)
			GlobalConn.Raw("select id from house where id = ?", tempOrders.HouseId).Scan(&houseData)
		}
	}

	if err := GlobalConn.Where("house_id = ?", houseData.Id).Find(&orders).Error; err != nil {
		fmt.Println("查询order无果，检查orderServiceModelFuc")
		return nil, err
	}

	//循环遍历一下orders
	for _, v := range orders {
		var orderTemp Order.OrdersData
		orderTemp.OrderId = int32(v.ID)
		orderTemp.EndDate = v.End_date.Format("2006-01-02")
		orderTemp.StartDate = v.Begin_date.Format("2006-01-02")
		orderTemp.Ctime = v.CreatedAt.Format("2006-01-02")
		orderTemp.Amount = int32(v.Amount)
		orderTemp.Comment = v.Comment
		orderTemp.Days = int32(v.Days)
		orderTemp.Status = v.Status
		fmt.Println("orderTemp", orderTemp)
		//关联house表
		var house House

		orderTemp.Title = house.Title

		orderResp = append(orderResp, &orderTemp)
	}
	//fmt.Println("orderResp_________", orderResp)

	return orderResp, nil
}

// 更新订单状态
func UpdateStatus(action, id, reason string) error {
	db := GlobalConn.Model(new(OrderHouse)).Where("id = ?", id)

	if action == "accept" {
		//标示房东同意订单
		return db.Update("status", "WAIT_COMMENT").Error
	} else {
		//表示房东不同意订单  如果拒单把拒绝的原因写到comment中
		return db.Updates(map[string]interface{}{"status": "REJECTED", "comment": reason}).Error
	}
}
