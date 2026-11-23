package model

import (
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"house/conf"
	"time"
)

/* 用户 table_name = user */
type User struct {
	ID            int    `gorm:"primaryKey"`      //用户编号
	Name          string `gorm:"size:32;unique"`  //用户名
	Password_hash string `gorm:"size:128" `       //用户密码加密的
	Mobile        string `gorm:"size:11;unique" ` //手机号
	Real_name     string `gorm:"size:32" `        //真实姓名  实名认证
	Id_card       string `gorm:"size:20" `        //身份证号  实名认证
	Avatar_url    string `gorm:"size:256" `       //用户头像路径       通过fastdfs进行图片存储
}

/* 房屋信息 table_name = house */
type House struct {
	gorm.Model
	HouseId         uint   //房屋编号
	UserId          uint   `gorm:"foreignKey:UserID"` //房屋主人的用户编号  与用户进行关联
	AreaId          uint   //归属地的区域编号   和地区表进行关联
	Title           string `gorm:"size:64" `                 //房屋标题
	Address         string `gorm:"size:512"`                 //地址
	Room_count      int    `gorm:"default:1" `               //房间数目
	Acreage         int    `gorm:"default:0" json:"acreage"` //房屋总面积
	Price           int    `json:"price"`
	Unit            string `gorm:"size:32;default:''" json:"unit"`             //房屋单元,如 几室几厅
	Capacity        int    `gorm:"default:1" json:"capacity"`                  //房屋容纳的总人数
	Beds            string `gorm:"size:64;default:''" json:"beds"`             //房屋床铺的配置
	Deposit         int    `gorm:"default:0" json:"deposit"`                   //押金
	Min_days        int    `gorm:"default:1" json:"min_days"`                  //最少入住的天数
	Max_days        int    `gorm:"default:0" json:"max_days"`                  //最多入住的天数 0表示不限制
	Order_count     int    `gorm:"default:0" json:"order_count"`               //预定完成的该房屋的订单数
	Index_image_url string `gorm:"size:256;default:''" json:"index_image_url"` //房屋主图片路径
	Facilities      []Facility `gorm:"many2many:house_facility;"`              //房屋设施
}

/* 区域信息 table_name = area */ //区域信息是需要我们手动添加到数据库中的
type Area struct {
	Id   int    `json:"aid"`                  //区域编号     1    2
	Name string `gorm:"size:32" json:"aname"` //区域名字     昌平 海淀

}

/* 设施信息 table_name = "facility"*/ //设施信息 需要我们提前手动添加的
type Facility struct {
	Id   int    `json:"fid"`     //设施编号
	Name string `gorm:"size:32"` //设施名字

}

/* 房屋图片 table_name = "house_image"*/
type HouseImage struct {
	Id      int    `json:"house_image_id"`      //图片id
	Url     string `gorm:"size:256" json:"url"` //图片url     存放我们房屋的图片
	HouseId uint   `json:"house_id"`            //图片所属房屋编号
}

/* 订单 table_name = order */
type OrderHouse struct {
	gorm.Model            //订单编号
	UserId      uint      `json:"user_id"`       //下单的用户编号   //与用户表进行关联
	HouseId     uint      `json:"house_id"`      //预定的房间编号   //与房屋信息进行关联
	Begin_date  time.Time `gorm:"type:datetime"` //预定的起始时间
	End_date    time.Time `gorm:"type:datetime"` //预定的结束时间
	Days        int       //预定总天数
	House_price int       //房屋的单价
	Amount      int       //订单总金额
	Status      string    `gorm:"default:'WAIT_ACCEPT'"` //订单状态
	Comment     string    `gorm:"size:512"`              //订单评论
	Credit      bool      //表示个人征信情况 true表示良好
}

// 日历库存预留状态
const (
	ReserveStatusReserved  = "RESERVED"
	ReserveStatusConfirmed = "CONFIRMED"
	ReserveStatusReleased  = "RELEASED"
)

/* 房源日历预留 table_name = house_reservation */
type HouseReservation struct {
	ID        uint      `gorm:"primaryKey"`
	HouseId   uint      `gorm:"index;not null"`
	OrderId   uint      `gorm:"uniqueIndex;not null"` // 幂等键
	BeginDate time.Time `gorm:"type:datetime;not null"`
	EndDate   time.Time `gorm:"type:datetime;not null"`
	Status    string    `gorm:"size:32;default:'RESERVED';index"`
	ExpiredAt time.Time `gorm:"type:datetime"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// 创建 数据库链接句柄
var GlobalConn *gorm.DB

func InitDb() (*gorm.DB, error) {
	//字符串拼接
	dsn := conf.MysqlName + ":" + conf.MysqlPwd + "@tcp(" + conf.MysqlAddr + ":" + conf.MysqlPort + ")/" + conf.MysqlDB + "?charset=utf8mb4&parseTime=True&loc=Local"

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{ //	gorm参数
		//不要复数表名
		NamingStrategy:                           schema.NamingStrategy{SingularTable: true},
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		fmt.Println("数据库初始化错误", err)
		return nil, err
	}

	// 初始化 全局连接池句柄
	GlobalConn = db
	sqlDB, err := GlobalConn.DB()
	if err != nil {
		fmt.Println("获取数据库对象失败")
		//return
	}
	// SetMaxIdleConns 用于设置连接池中空闲连接的最大数量。
	sqlDB.SetMaxIdleConns(10)
	// SetMaxOpenConns 设置打开数据库连接的最大数量。
	sqlDB.SetMaxOpenConns(100)

	db.AutoMigrate(&User{}, &House{}, &Area{}, &Facility{}, &HouseImage{}, &OrderHouse{}, &HouseReservation{})
	return nil, err
}
