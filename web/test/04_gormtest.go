package main

import (
	"fmt"
	"gorm.io/gorm/schema"
	"time"

	//_ "github.com/go-sql-driver/mysql"
	//"_"表示不直接使用包，底层连接要使用
	_ "gin-micro-ihome/web/test2"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// 3.创建全局结构体
type Student struct {
	gorm.Model //go语言中匿名成员--继承
	Id         int
	Name       string `gorm:"size:50;default:'tester1'"` //string 默认大小是255,在创建时，可以指定为其他的大小
	Age        int
	Join       time.Time `gorm:"type:datetime"`
}

// 外键案例
type User1 struct {
	gorm.Model
	Name     string
	Articles []Article `gorm:"foreignKey:UserID"` // UserID 是外键
}

type Article struct {
	gorm.Model
	Title   string
	Content string
	UserID  uint // 引用 User 表的主键
}

/* 用户 table_name = user */
type User struct {
	gorm.Model
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
	HouseId         uint         //房屋编号
	UserId          uint         `gorm:"foreignKey:UserID"` //房屋主人的用户编号  与用户进行关联
	AreaId          uint         //归属地的区域编号   和地区表进行关联
	Title           string       `gorm:"size:64" `                 //房屋标题
	Address         string       `gorm:"size:512"`                 //地址
	Room_count      int          `gorm:"default:1" `               //房间数目
	Acreage         int          `gorm:"default:0" json:"acreage"` //房屋总面积
	Price           int          `json:"price"`
	Unit            string       `gorm:"size:32;default:''" json:"unit"`               //房屋单元,如 几室几厅
	Capacity        int          `gorm:"default:1" json:"capacity"`                    //房屋容纳的总人数
	Beds            string       `gorm:"size:64;default:''" json:"beds"`               //房屋床铺的配置
	Deposit         int          `gorm:"default:0" json:"deposit"`                     //押金
	Min_days        int          `gorm:"default:1" json:"min_days"`                    //最少入住的天数
	Max_days        int          `gorm:"default:0" json:"max_days"`                    //最多入住的天数 0表示不限制
	Order_count     int          `gorm:"default:0" json:"order_count"`                 //预定完成的该房屋的订单数
	Index_image_url string       `gorm:"size:256;default:''" json:"index_image_url"`   //房屋主图片路径
	Facilities      []Facility   `gorm:"many2many:house_facilities" json:"facilities"` //房屋设施   与设施表进行关联
	Images          []HouseImage `json:"img_urls"`                                     //房屋的图片   除主要图片之外的其他图片地址
	Orders          []OrderHouse `json:"orders"`                                       //房屋的订单    与房屋表进行管理
}

/* 区域信息 table_name = area */ //区域信息是需要我们手动添加到数据库中的
type Area struct {
	Id     int     `json:"aid"`                  //区域编号     1    2
	Name   string  `gorm:"size:32" json:"aname"` //区域名字     昌平 海淀
	Houses []House `json:"houses"`               //区域所有的房屋   与房屋表进行关联
}

/* 设施信息 table_name = "facility"*/ //设施信息 需要我们提前手动添加的
type Facility struct {
	Id     int     `json:"fid"`     //设施编号
	Name   string  `gorm:"size:32"` //设施名字
	Houses []House //都有哪些房屋有此设施  与房屋表进行关联的
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

//- 默认MySQL数据库 有 3 种时间：
//  - date：
//  - datetime：
//  - timeStamp：时间戳。 ——  gorm 中，只有 timeStamp
//
//- 如果必须使用 MySQL 数据库特有的 “数据类型”， 使用 “type” 关键字来设置。

// 4.连接池-1-创建全局的连接池句柄
var GlobalConn *gorm.DB

func main() {
	fmt.Println("this is mian")

	//	1.连接数据库	格式：	用户名:密码@协议（ip：port）/数据库名
	//conn, err := gorm.Open("mysql", "neko:neko123456@tcp(127.0.0.1:3306/test)")
	dsn := "neko:neko123456@tcp(127.0.0.1:3306)/t1?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{ //	gorm参数
		//不要复数表名
		NamingStrategy:                           schema.NamingStrategy{SingularTable: true},
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		panic(err)
		return
	}

	fmt.Println("连接数据库成功。。。")

	GlobalConn = db
	//2.借助gorm创建数据库表
	sqlDB, err := GlobalConn.DB()
	if err != nil {
		fmt.Println("获取数据库对象失败")
		return
	}
	//连接池-2
	// SetMaxIdleConns 用于设置连接池中空闲连接的最大数量。
	sqlDB.SetMaxIdleConns(10)
	// SetMaxOpenConns 设置打开数据库连接的最大数量。
	sqlDB.SetMaxOpenConns(100)

	////创建表
	//GlobalConn.AutoMigrate(&User{}, &House{})
	GlobalConn.AutoMigrate(&User1{}, &Article{}, &User{}, &House{}, &Area{}, &Facility{}, &HouseImage{}, &OrderHouse{})

	//
	//
	//
	//
	//
	//
	//
	//
	//
	//
	//
	//
	//
	//
	//
	//
	//
	//
	//
	////创建数据
	//Stu := Student{Name: "neko", Age: 20}
	//result := GlobalConn.Create(&Stu) //通过指针创建

	//返回插入数据的name
	//name := Stu.Name
	//返回插入记录的条数
	//affect := result.RowsAffected
	//fmt.Println(name, affect)
	//ct := db.AutoMigrate(new(Student)).Error()
	//fmt.Println(ct)

	//增删改查
	//InsertData()
	//SelectData()
	//UpdateData()
	//DeleteData()
}

// gorm增加数据
func InsertData() {
	Stu := Student{Name: "zhi", Age: 20}
	result := GlobalConn.Create(&Stu) //通过指针创建

	//返回插入数据的name
	fmt.Println(result.RowsAffected)
}

// gorm查询数据
func SelectData() {
	//获取第一条记录（主键升序）
	//SELECT * FROM users ORDER BY id LIMIT 1;
	GlobalConn.First(&Student{})

	//获取一条记录，没有指定排序字段
	// SELECT * FROM users LIMIT 1;
	GlobalConn.Take(&Student{})

	// 获取最后一条记录（主键降序）
	//SELECT * FROM users ORDER BY id DESC LIMIT 1;
	GlobalConn.Last(&Student{})

	//获取所有记录
	//需要将stu改为切片
	var stu1 []Student
	GlobalConn.Select("name", "age").Find(&stu1)
	fmt.Println(stu1)

	//where简易sql
	GlobalConn.Where("name = ?", "zhi").First(&stu1)
}

func UpdateData() {
	var Stu Student

	Stu.Name = "lisi4"
	Stu.Age = 22
	fmt.Println(GlobalConn.Model(&Student{}).
		Where("name = ?", "lisi4").
		Updates("lisi5").Error)
}

//	### 删除
//
// - 物理删除。真正的执行 Delete。
//
// ### 软删除
// - 逻辑删除。不真正删。不执行Delete。 —— 数据无价的！
//   - 创建表时，在表中添加一个 “删除字段” 。当需要删除时，更新 “删除字段”， 更新为 ：true
//   - 查询时，不查询 “删除字段” 为 null 的值。
//   - 举例：淘宝，删除订单。 手机，删除图片。
//
// - 实现 软删除：
func DeleteData() {
	GlobalConn.Where("name = ?", "lisi4").Delete(&Student{})

}
