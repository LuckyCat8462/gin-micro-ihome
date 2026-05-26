package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"gin-micro-ihome/web/conf"
	"gin-micro-ihome/web/model"
	getCaptcha "gin-micro-ihome/web/proto/getCaptcha"
	houseMicro "gin-micro-ihome/web/proto/house"
	orderMicro "gin-micro-ihome/web/proto/order"
	registerMicro "gin-micro-ihome/web/proto/register"
	userMicro "gin-micro-ihome/web/proto/user"
	"gin-micro-ihome/web/utils"
	"github.com/afocus/captcha"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/gomodule/redigo/redis"
	"image/png"
	"net/http"
	"path"
)

func GetSession(ctx *gin.Context) {
	// 	初始化错误返回的map
	resp := make(map[string]interface{})
	// 调用utils包种的宏，go中称为常量
	s := sessions.Default(ctx)
	userName := s.Get("userName")
	if userName == nil {
		fmt.Println("用户未登录-getSession错了")
		resp["errno"] = utils.RECODE_SESSIONERR
		resp["errmsg"] = utils.RecodeText(utils.RECODE_SESSIONERR)
	} else {
		//测试输出
		//fmt.Println("---GetSession拿到了---")
		resp["errno"] = utils.RECODE_OK
		resp["errmsg"] = utils.RecodeText(utils.RECODE_OK)

		var nameData struct {
			Name string `json:"name"`
		}
		nameData.Name = userName.(string)
		resp["data"] = nameData
		//测试输出
		//fmt.Println("用户名：", nameData)
	}

	ctx.JSON(http.StatusOK, resp)
}

// 获取图片信息
func GetImageCd(ctx *gin.Context) {
	// 获取uuid
	uuid := ctx.Param("uuid")

	//指定consul服务发现
	microService := utils.InitMicro()
	//初始化客户端
	microClient := getCaptcha.NewGetCaptchaService("getCaptcha", microService.Client())
	//调用远程服务
	resp, err := microClient.Call(context.TODO(), &getCaptcha.Request{Uuid: uuid})
	if err != nil {
		fmt.Println("未找到远程服务。。。")
		fmt.Println(err)
		return
	}
	//resp并不能直接使用，因为传入服务端时进行了序列化，此时还需要将得到的数据进行反序列化
	var img captcha.Image
	//img := captcha.Image{}
	json.Unmarshal(resp.Img, &img)
	//将图片写入到浏览器

	png.Encode(ctx.Writer, img)
	//fmt.Println("str", string(resp.Img))
	fmt.Println("uuid", uuid)
}

// 获取验证码
func GetSmscd(ctx *gin.Context) {
	// 获取uuid
	phone := ctx.Param("phone")
	//	拆分GET请求中的url格式		资源路径?key1=v1&key2=v2
	imgCode := ctx.Query("text")
	uuid := ctx.Query("id")
	fmt.Println("out________", phone, imgCode, uuid)
}

// 发送注册信息
func PostReg(ctx *gin.Context) {
	// 获取数据
	var regData struct {
		Mobile   string `json:"mobile"`
		PassWord string `json:"password"`
	}

	ctx.Bind(&regData)

	fmt.Println("测试reg数据：", regData.Mobile, regData.PassWord)

	//调用远程服务
	// 初始化consul
	microService := utils.InitMicro()
	microClient := registerMicro.NewRegisterService("micro_register", microService.Client())

	// 调用远程函数
	resp, err := microClient.Register(context.TODO(), &registerMicro.RegRequest{
		Mobile:   regData.Mobile,
		Password: regData.PassWord,
	})
	if err != nil {
		fmt.Println("注册用户, 找不到远程服务!", err)
		return
	}
	// 写给浏览器
	ctx.JSON(http.StatusOK, resp)
}

// 获取地域信息
func GetArea(ctx *gin.Context) {
	// 先从MySQL中获取数据.
	var areas []model.Area

	// 从缓存redis 中, 获取数据
	//conn := model.RedisPool.Get()
	conn, err := redis.Dial("tcp", conf.RedisAddr)
	if err != nil {
		fmt.Println(err)
		return
	}
	// 当初使用 "字节切片" 存入, 现在使用 切片类型接收
	areaData, _ := redis.Bytes(conn.Do("get", "areaData"))
	//测试提示
	//fmt.Println("house.go-GetArea函数：获取地址信息")
	// 没有从 Redis 中获取到数据
	if len(areaData) == 0 {
		fmt.Println("从 MySQL 中 获取数据...")
		model.GlobalConn.Find(&areas)
		// 把数据写入到 redis 中. , 存储结构体序列化后的 json 串
		areaBuf, _ := json.Marshal(areas)
		conn.Do("set", "areaData", areaBuf)

	} else {
		////测试输出
		//fmt.Println("从 Redis 中 获取数据...")
		// redis 中有数据
		json.Unmarshal(areaData, &areas)
	}

	resp := make(map[string]interface{})

	resp["errno"] = "0"
	resp["errmsg"] = utils.RecodeText(utils.RECODE_OK)
	resp["data"] = areas

	ctx.JSON(http.StatusOK, resp)
}

// 处理登录业务
func PostLogin(ctx *gin.Context) {

	//微服务版本登录功能
	////	获取前端数据
	var loginData struct {
		Mobile   string `json:"mobile"`
		PassWord string `json:"password"`
	}
	ctx.Bind(&loginData)
	//fmt.Println("测试loginData", loginData)

	//指定consul服务发现
	microService := utils.InitMicro()
	//初始化客户端
	microClient := registerMicro.NewRegisterService("micro_register", microService.Client())

	resp, err := microClient.Login(context.TODO(), &registerMicro.RegRequest{Mobile: loginData.Mobile, Password: loginData.PassWord})
	fmt.Println("resp info", resp)
	if err != nil {
		fmt.Println("调用login服务错误", err)
		return
	} else {
		fmt.Println("login服务正常")
		//返回数据  存储session  并返回数据给web端
		session := sessions.Default(ctx)
		session.Set("userName", resp.Name)
		session.Save()
		//fmt.Println("login测试session", session.Get("userName"))
	}

	ctx.JSON(http.StatusOK, resp)

	//非微服务版本
	//resp := make(map[string]interface{})
	//
	////获取数据库数据,查询是否和数据库的数据匹配
	//userName, err := model.Login(loginData.Mobile, loginData.PassWord)
	//
	//if err == nil {
	//	fmt.Println("postLogin登录成功")
	//	resp["errno"] = utils.RECODE_OK
	//	resp["errmsg"] = utils.RecodeText(utils.RECODE_OK)
	//
	//	s := sessions.Default(ctx) //初始化session
	//	//fmt.Println("初始化session")
	//	s.Set("userName", userName) //将用户名设置到session中
	//	s.Save()
	//} else {
	//	fmt.Println("登录失败")
	//	resp["errno"] = utils.RECODE_LOGINERR
	//	resp["errmsg"] = utils.RecodeText(utils.RECODE_LOGINERR)
	//}
	//
	//ctx.JSON(http.StatusOK, resp)
}

// 退出登录
func DeleteSession(ctx *gin.Context) {
	resp := make(map[string]interface{})
	//初始化session对象
	s := sessions.Default(ctx)
	//删除session数据
	s.Delete("userName")
	//	必须使用save保存
	err := s.Save()
	if err != nil {
		fmt.Println("删除失败", err)
		resp["errno"] = utils.RECODE_IOERR
		resp["errmsg"] = utils.RecodeText(utils.RECODE_IOERR)
	} else {
		resp["errno"] = "0"
		resp["errmsg"] = utils.RecodeText(utils.RECODE_OK)
	}
	ctx.JSON(http.StatusOK, resp)
}

// 获取用户信息
func GetUserInfo(ctx *gin.Context) {
	resp := make(map[string]interface{})
	defer ctx.JSON(http.StatusOK, resp)

	// 获取 Session, 得到 当前 用户信息
	s := sessions.Default(ctx)
	userName := s.Get("userName")
	// 判断用户名是否存在.
	if userName == nil { // 用户没登录, 但进入该页面, 恶意进入.
		resp["errno"] = utils.RECODE_SESSIONERR
		resp["errmsg"] = utils.RecodeText(utils.RECODE_SESSIONERR)
		return // 如果出错, 报错, 退出
	}

	// 根据用户名, 获取 用户信息  ---- 查 MySQL 数据库  user 表.
	user, err := model.GetUserInfo(userName.(string))
	if err != nil {
		resp["errno"] = utils.RECODE_DBERR
		resp["errmsg"] = utils.RecodeText(utils.RECODE_DBERR)
		return // 如果出错, 报错, 退出
	}

	resp["errno"] = utils.RECODE_OK
	resp["errmsg"] = utils.RecodeText(utils.RECODE_OK)

	temp := make(map[string]interface{})
	temp["user_id"] = user.ID
	temp["name"] = user.Name
	temp["mobile"] = user.Mobile
	temp["real_name"] = user.Real_name
	temp["id_card"] = user.Id_card
	temp["avatar_url"] = user.Avatar_url

	resp["data"] = temp
}

// 更新用户名
func PutUserInfo(ctx *gin.Context) {
	// 获取当前用户名
	s := sessions.Default(ctx) // 初始化Session 对象
	userName := s.Get("userName")

	// 获取新用户名		---- 处理 Request Payload 类型数据. Bind()
	var nameData struct {
		Name string `json:"name"`
	}
	ctx.Bind(&nameData)

	// 更新用户名
	resp := make(map[string]interface{})
	defer ctx.JSON(http.StatusOK, resp)

	// 更新数据库中的 name
	err := model.UpdateUserName(nameData.Name, userName.(string))
	if err != nil {
		resp["errno"] = utils.RECODE_DBERR
		resp["errmsg"] = utils.RecodeText(utils.RECODE_DBERR)
		return
	}

	// 更新 Session 数据
	s.Set("userName", nameData.Name)
	err = s.Save() // 必须保存
	if err != nil {
		resp["errno"] = utils.RECODE_SESSIONERR
		resp["errmsg"] = utils.RecodeText(utils.RECODE_SESSIONERR)
		return
	}
	resp["errno"] = utils.RECODE_OK
	resp["errmsg"] = utils.RecodeText(utils.RECODE_OK)
	resp["data"] = nameData
}

// 上传头像
func PostAvater(ctx *gin.Context) {
	//	获取图片文件,静态文件
	file, _ := ctx.FormFile("avatar")
	//上传文件到项目中
	err := ctx.SaveUploadedFile(file, "test/"+file.Filename)
	fmt.Println(err)
	//// 检查文件大小 (限制为 10MB)
	//if file.Size > 10<<20 {
	//	ctx.JSON(http.StatusBadRequest, gin.H{
	//		"error": "文件大小不能超过 10MB",
	//	})
	//	return
	//}
	//filepath01 := "test/" + file.Filename
	//
	//////	上传文件到项目中
	////err := ctx.SaveUploadedFile(FileHeader, "test/"+FileHeader.Filename)
	////fmt.Println(err)
	////上传文件到fdfs中
	////clt, _ := fdfs_client.NewClientWithConfig("/etc/fdfs/client.conf")
	////打开文件，读取文件内容
	////f, _ := file.Open()
	////
	////buf := make([]byte, file.Size)
	////f.Read(buf)
	//////go语言根据文件名获取文件后缀
	////fileExt := path.Ext(file.Filename) //传文件名
	//////按字节流上传图片内容
	////remoteId, _ := clt.UploadByBuffer(buf, fileExt[1:]) //fileExt是一个切片，所以可以从下标1取
	//
	////获取session，得到当前用户
	//userName := sessions.Default(ctx).Get("userName")
	//
	////根据用户名,更新用户头像
	//model.UpdateAvatar(userName.(string), filepath01)
	//
	//resp := make(map[string]interface{})
	//resp["errno"] = "0"
	//resp["errmsg"] = utils.RecodeText(utils.RECODE_OK)
	//
	//temp := make(map[string]interface{})
	//temp["avatar_url"] = filepath01
	//
	//resp["data"] = temp
	//ctx.JSON(http.StatusOK, resp)
}

type AuthStu struct {
	IdCard   string `json:"id_card"`
	RealName string `json:"real_name"`
}

// 上传实名认证
func PutUserAuth(ctx *gin.Context) {
	//获取前端数据
	var auth AuthStu
	err := ctx.Bind(&auth)

	//校验数据
	if err != nil {
		fmt.Println("获取数据错误", err)
		return
	}

	session := sessions.Default(ctx)
	userName := session.Get("userName")
	fmt.Println("获取userName：", userName)

	//处理数据 微服务
	microService := utils.InitMicro()
	microClient := userMicro.NewUserService("micro_user", microService.Client())

	//调用远程服务

	resp, err := microClient.AuthUpdate(context.TODO(), &userMicro.AuthReq{
		UserName: userName.(string),
		RealName: auth.RealName,
		IdCard:   auth.IdCard,
	})
	if err != nil {
		fmt.Println("resp情况", err)
	}

	//返回数据
	ctx.JSON(http.StatusOK, resp)
}

// 房屋结构体
type HouseStu struct {
	Acreage   string   `json:"acreage"`
	Address   string   `json:"address"`
	AreaId    string   `json:"area_id"`
	Beds      string   `json:"beds"`
	Capacity  string   `json:"capacity"`
	Deposit   string   `json:"deposit"`
	Facility  []string `json:"facility"`
	MaxDays   string   `json:"max_days"`
	MinDays   string   `json:"min_days"`
	Price     string   `json:"price"`
	RoomCount string   `json:"room_count"`
	Title     string   `json:"title"`
	Unit      string   `json:"unit"`
}

func PostHouses(ctx *gin.Context) {
	//获取数据
	var house HouseStu
	err := ctx.Bind(&house)

	//	校验数据
	if err != nil {
		fmt.Println("获取数据错误", err)
		return
	}

	//	获取用户名
	userName := sessions.Default(ctx).Get("userName")
	fmt.Println("获取userName：", userName)

	//处理数据 微服务
	microService := utils.InitMicro()
	microClient := houseMicro.NewHouseService("micro_house", microService.Client())

	resp, _ := microClient.PubHouse(context.TODO(), &houseMicro.Request{
		Acreage:   house.Acreage,
		Address:   house.Address,
		AreaId:    house.AreaId,
		Beds:      house.Beds,
		Capacity:  house.Capacity,
		Deposit:   house.Deposit,
		Facility:  house.Facility,
		MaxDays:   house.MaxDays,
		MinDays:   house.MinDays,
		Price:     house.Price,
		RoomCount: house.RoomCount,
		Title:     house.Title,
		Unit:      house.Unit,
		UserName:  userName.(string),
	})

	ctx.JSON(http.StatusOK, resp)

}

// 获取用户房源信息
func GetUserHouses(ctx *gin.Context) {
	//获取用户名
	userName := sessions.Default(ctx).Get("userName")
	//测试输出
	//fmt.Println("GetUserHouses函数", userName)
	//微服务
	microService := utils.InitMicro()
	microClient := houseMicro.NewHouseService("micro_house", microService.Client())

	////调用远程服务
	resp, _ := microClient.GetHouseInfo(context.TODO(), &houseMicro.GetReq{UserName: userName.(string)})

	//返回数据
	ctx.JSON(http.StatusOK, resp)
}

func PostHousesImage(ctx *gin.Context) {
	//获取数据
	houseId := ctx.Param("id")
	fileHeader, err := ctx.FormFile("house_image")
	//校验数据
	if houseId == "" || err != nil {
		fmt.Println("传入数据不完整", err)
		return
	}

	//三种校验 大小,类型,防止重名  fastdfs
	if fileHeader.Size > 50000000 {
		fmt.Println("文件过大,请重新选择")
		return
	}

	fileExt := path.Ext(fileHeader.Filename)
	if fileExt != ".png" && fileExt != ".jpg" {
		fmt.Println("文件类型错误,请重新选择")
		return
	}

	//获取文件字节切片
	file, _ := fileHeader.Open()
	buf := make([]byte, fileHeader.Size)
	file.Read(buf)

	//处理数据  服务中实现
	microService := utils.InitMicro()
	microClient := houseMicro.NewHouseService("micro_house", microService.Client())
	//调用服务
	resp, _ := microClient.UploadHouseImg(context.TODO(), &houseMicro.ImgReq{
		HouseId: houseId,
		ImgData: buf,
		FileExt: fileExt,
	})

	//返回数据
	ctx.JSON(http.StatusOK, resp)
}

// 获取房屋详情
func GetHouseInfo(ctx *gin.Context) {
	//获取数据
	houseId := ctx.Param("id")

	//校验数据
	if houseId == "" {
		fmt.Println("获取数据错误")
		return
	}
	userName := sessions.Default(ctx).Get("userName")
	//处理数据
	microService := utils.InitMicro()
	microClient := houseMicro.NewHouseService("micro_house", microService.Client())
	//调用远程服务

	resp, _ := microClient.GetHouseDetail(context.TODO(), &houseMicro.DetailReq{
		HouseId:  houseId,
		UserName: userName.(string),
	})
	fmt.Println("detailresp:", resp)

	//返回数据

	ctx.JSON(http.StatusOK, resp)
}

// 搜索房屋
func GetHouses(ctx *gin.Context) {

	//获取数据
	fmt.Println("测试开始")
	//areaId
	aid := "13"
	//start day
	sd := "2025-8-10"
	//end day
	ed := "2025-8-11"

	//areaId
	//aid := ctx.Query("aid")
	//fmt.Println("aid:", aid)
	////start day
	//sd := ctx.Query("sd")
	////end day
	//ed := ctx.Query("ed")
	//排序方式
	sk := ctx.Query("sk")
	//page  第几页
	//ctx.Query("p")
	//校验数据
	if aid == "" || sd == "" || ed == "" {
		fmt.Println("传入数据不完整")
		return
	}

	//处理数据   服务端  把字符串转换为时间格式,使用函数time.Parse()  第一个参数是转换模板,需要转换的二字符串,两者格式一致
	/*sdTime ,_:=time.Parse("2006-01-02 15:04:05",sd+" 00:00:00")
	edTime,_ := time.Parse("2006-01-02",ed)*/

	/*sdTime,_ :=time.Parse("2006-01-02",sd)
	edTime,_ := time.Parse("2006-01-02",ed)
	d := edTime.Sub(sdTime)
	fmt.Println(d.Hours())*/
	microService := utils.InitMicro()
	microClient := houseMicro.NewHouseService("micro_house", microService.Client())
	//调用远程服务
	resp, _ := microClient.SearchHouse(context.TODO(), &houseMicro.SearchReq{
		Aid: aid,
		Sd:  sd,
		Ed:  ed,
		Sk:  sk,
	})
	fmt.Println(resp)
	//返回数据
	ctx.JSON(http.StatusOK, resp)

}

// 首页轮播图
func GetIndex(ctx *gin.Context) {
	//处理数据
	microService := utils.InitMicro()
	microClient := houseMicro.NewHouseService("micro_house", microService.Client())
	//调用服务
	resp, _ := microClient.GetIndexHouse(context.TODO(), &houseMicro.IndexReq{})

	ctx.JSON(http.StatusOK, resp)
}

type OrderStu struct {
	EndDate   string `json:"end_date"`
	HouseId   string `json:"house_id"`
	StartDate string `json:"start_date"`
}

// 下订单
func PostOrders(ctx *gin.Context) {
	//获取数据
	var order OrderStu
	err := ctx.Bind(&order)

	//校验数据
	if err != nil {
		fmt.Println("获取数据错误", err)
		return
	}
	//获取用户名
	userName := sessions.Default(ctx).Get("userName")
	fmt.Println("orderusername", userName)
	//处理数据  服务端
	microService := utils.InitMicro()
	microClient := orderMicro.NewOrderService("micro_order", microService.Client())
	//调用服务
	resp, _ := microClient.CreateOrder(context.TODO(), &orderMicro.Request{
		StartDate: order.StartDate,
		EndDate:   order.EndDate,
		HouseId:   order.HouseId,
		UserName:  userName.(string),
	})

	//返回数据
	ctx.JSON(http.StatusOK, resp)
}

// 获取订单信息
func GetUserOrder(ctx *gin.Context) {
	//获取get请求传参
	role := ctx.Query("role")
	//校验数据
	if role == "" {
		fmt.Println("获取数据失败")
		return
	}

	//处理数据  服务端
	microService := utils.InitMicro()
	microClient := orderMicro.NewOrderService("micro_order", microService.Client())
	//调用远程服务
	resp, _ := microClient.GetOrderInfo(context.TODO(), &orderMicro.GetReq{
		Role:     role,
		UserName: sessions.Default(ctx).Get("userName").(string),
	})
	fmt.Println("resp:", resp)
	//返回数据
	ctx.JSON(http.StatusOK, resp)
}

type StatusStu struct {
	Action string `json:"action"`
	Reason string `json:"reason"`
}

// 更新订单状态
func PutOrders(ctx *gin.Context) {
	//获取数据
	id := ctx.Param("id")
	var statusStu StatusStu
	err := ctx.Bind(&statusStu)

	//校验数据
	if err != nil || id == "" {
		fmt.Println("获取数据错误", err)
		return
	}
	//处理数据  服务端
	microService := utils.InitMicro()
	microClient := orderMicro.NewOrderService("micro_order", microService.Client())
	//调用元和产能服务
	resp, _ := microClient.UpdateStatus(context.TODO(), &orderMicro.UpdateReq{
		Action: statusStu.Action,
		Reason: statusStu.Reason,
		Id:     id,
	})

	//返回数据
	ctx.JSON(http.StatusOK, resp)
}
