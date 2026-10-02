package controller

import (
	"encoding/json"
	"fmt"
	"gin-micro-ihome/web/conf"
	"gin-micro-ihome/web/model"
	getCaptcha "gin-micro-ihome/web/proto/getCaptcha"
	houseMicro "gin-micro-ihome/web/proto/house"
	orderMicro "gin-micro-ihome/web/proto/order"
	registerMicro "gin-micro-ihome/web/proto/register"
	userMicro "gin-micro-ihome/web/proto/user"
	"gin-micro-ihome/web/storage"
	"gin-micro-ihome/web/utils"
	"image/png"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/afocus/captcha"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/gomodule/redigo/redis"
	"github.com/google/uuid"
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
	resp, err := microClient.Call(utils.RPCCtx(), &getCaptcha.Request{Uuid: uuid})
	if err != nil {
		fmt.Println("未找到远程服务。。。", err)
		utils.AbortRPC(ctx)
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

// 获取验证码（学习项目已停用短信，保留路由避免旧前端报 404）
func GetSmscd(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"errno":  utils.RECODE_PARAMERR,
		"errmsg": "短信验证码已停用，请使用图片验证码完成注册",
	})
}

// 发送注册信息
func PostReg(ctx *gin.Context) {
	var regData struct {
		Mobile    string `json:"mobile"`
		PassWord  string `json:"password"`
		ImageCode string `json:"imagecode"`
		UUID      string `json:"uuid"`
	}
	if err := ctx.Bind(&regData); err != nil {
		ctx.JSON(http.StatusOK, gin.H{
			"errno":  utils.RECODE_PARAMERR,
			"errmsg": utils.RecodeText(utils.RECODE_PARAMERR),
		})
		return
	}
	if regData.Mobile == "" || regData.PassWord == "" || regData.ImageCode == "" || regData.UUID == "" {
		ctx.JSON(http.StatusOK, gin.H{
			"errno":  utils.RECODE_PARAMERR,
			"errmsg": utils.RecodeText(utils.RECODE_PARAMERR),
		})
		return
	}
	if !model.CheckImgCode(regData.UUID, regData.ImageCode) {
		ctx.JSON(http.StatusOK, gin.H{
			"errno":  utils.RECODE_PARAMERR,
			"errmsg": "图片验证码错误",
		})
		return
	}

	microService := utils.InitMicro()
	microClient := registerMicro.NewRegisterService("micro_register", microService.Client())

	resp, err := microClient.Register(utils.RPCCtx(), &registerMicro.RegRequest{
		Mobile:   regData.Mobile,
		Password: regData.PassWord,
	})
	if err != nil {
		fmt.Println("注册用户, 找不到远程服务!", err)
		ctx.JSON(http.StatusOK, gin.H{
			"errno":  utils.RECODE_SERVERERR,
			"errmsg": utils.RecodeText(utils.RECODE_SERVERERR),
		})
		return
	}

	if resp.Errno == utils.RECODE_OK {
		name := resp.Name
		if name == "" {
			name = regData.Mobile
		}
		session := sessions.Default(ctx)
		session.Clear()
		session.Set("userName", name)
		if err := session.Save(); err != nil {
			fmt.Println("注册成功但保存session失败", err)
			ctx.JSON(http.StatusOK, gin.H{
				"errno":  utils.RECODE_SERVERERR,
				"errmsg": utils.RecodeText(utils.RECODE_SERVERERR),
			})
			return
		}
	}

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
		utils.AbortRPC(ctx)
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
	var loginData struct {
		Mobile   string `json:"mobile"`
		PassWord string `json:"password"`
	}
	if err := ctx.Bind(&loginData); err != nil {
		ctx.JSON(http.StatusOK, gin.H{
			"errno":  utils.RECODE_PARAMERR,
			"errmsg": utils.RecodeText(utils.RECODE_PARAMERR),
		})
		return
	}

	microService := utils.InitMicro()
	microClient := registerMicro.NewRegisterService("micro_register", microService.Client())

	resp, err := microClient.Login(utils.RPCCtx(), &registerMicro.RegRequest{Mobile: loginData.Mobile, Password: loginData.PassWord})
	if err != nil {
		fmt.Println("调用login服务错误", err)
		ctx.JSON(http.StatusOK, gin.H{
			"errno":  utils.RECODE_SERVERERR,
			"errmsg": utils.RecodeText(utils.RECODE_SERVERERR),
		})
		return
	}

	if resp.Errno == utils.RECODE_OK && resp.Name != "" {
		session := sessions.Default(ctx)
		session.Clear()
		session.Set("userName", resp.Name)
		if err := session.Save(); err != nil {
			fmt.Println("保存session失败", err)
			ctx.JSON(http.StatusOK, gin.H{
				"errno":  utils.RECODE_SERVERERR,
				"errmsg": utils.RecodeText(utils.RECODE_SERVERERR),
			})
			return
		}
	}

	ctx.JSON(http.StatusOK, resp)
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
	temp["avatar"] = user.Avatar_url

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
	fileHeader, err := ctx.FormFile("avatar")
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{
			"errno":  utils.RECODE_PARAMERR,
			"errmsg": utils.RecodeText(utils.RECODE_PARAMERR),
		})
		return
	}
	if fileHeader.Size > 5<<20 {
		ctx.JSON(http.StatusOK, gin.H{
			"errno":  utils.RECODE_PARAMERR,
			"errmsg": "头像大小不能超过 5MB",
		})
		return
	}

	fileExt := strings.ToLower(path.Ext(fileHeader.Filename))
	if fileExt != ".png" && fileExt != ".jpg" && fileExt != ".jpeg" {
		ctx.JSON(http.StatusOK, gin.H{
			"errno":  utils.RECODE_PARAMERR,
			"errmsg": "仅支持 jpg/png 图片",
		})
		return
	}

	userName := sessions.Default(ctx).Get("userName")
	name, ok := userName.(string)
	if !ok || name == "" {
		ctx.JSON(http.StatusOK, gin.H{
			"errno":  utils.RECODE_SESSIONERR,
			"errmsg": utils.RecodeText(utils.RECODE_SESSIONERR),
		})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{
			"errno":  utils.RECODE_IOERR,
			"errmsg": utils.RecodeText(utils.RECODE_IOERR),
		})
		return
	}
	defer file.Close()

	buf := make([]byte, fileHeader.Size)
	if _, err := file.Read(buf); err != nil {
		ctx.JSON(http.StatusOK, gin.H{
			"errno":  utils.RECODE_IOERR,
			"errmsg": utils.RecodeText(utils.RECODE_IOERR),
		})
		return
	}

	ext := strings.TrimPrefix(fileExt, ".")
	objectKey := fmt.Sprintf("avatars/%s/%s.%s", name, uuid.NewString(), ext)
	url, err := storage.Upload(utils.RPCCtx(), objectKey, buf, storage.ContentTypeByExt(ext))
	if err != nil {
		fmt.Println("上传头像到 MinIO 失败", err)
		ctx.JSON(http.StatusOK, gin.H{
			"errno":  utils.RECODE_IOERR,
			"errmsg": utils.RecodeText(utils.RECODE_IOERR),
		})
		return
	}

	if err := model.UpdateAvatar(name, url); err != nil {
		fmt.Println("更新头像失败", err)
		ctx.JSON(http.StatusOK, gin.H{
			"errno":  utils.RECODE_DBERR,
			"errmsg": utils.RecodeText(utils.RECODE_DBERR),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"errno":  utils.RECODE_OK,
		"errmsg": utils.RecodeText(utils.RECODE_OK),
		"data": gin.H{
			"avatar_url": url,
		},
	})
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

	resp, err := microClient.AuthUpdate(utils.RPCCtx(), &userMicro.AuthReq{
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
	var house HouseStu
	if err := ctx.Bind(&house); err != nil {
		fmt.Println("获取数据错误", err)
		utils.AbortParam(ctx, "")
		return
	}

	userNameVal := sessions.Default(ctx).Get("userName")
	userName, ok := userNameVal.(string)
	if !ok || userName == "" {
		ctx.JSON(http.StatusOK, gin.H{
			"errno":  utils.RECODE_SESSIONERR,
			"errmsg": utils.RecodeText(utils.RECODE_SESSIONERR),
		})
		return
	}

	microService := utils.InitMicro()
	microClient := houseMicro.NewHouseService("micro_house", microService.Client())
	resp, err := microClient.PubHouse(utils.RPCCtx(), &houseMicro.Request{
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
		UserName:  userName,
	})
	if err != nil {
		fmt.Println("PubHouse RPC 失败", err)
		utils.AbortRPC(ctx)
		return
	}
	if resp == nil {
		utils.AbortRPC(ctx)
		return
	}
	ctx.JSON(http.StatusOK, resp)
}

// 获取用户房源信息
func GetUserHouses(ctx *gin.Context) {
	userNameVal := sessions.Default(ctx).Get("userName")
	userName, ok := userNameVal.(string)
	if !ok || userName == "" {
		ctx.JSON(http.StatusOK, gin.H{
			"errno":  utils.RECODE_SESSIONERR,
			"errmsg": utils.RecodeText(utils.RECODE_SESSIONERR),
		})
		return
	}
	microService := utils.InitMicro()
	microClient := houseMicro.NewHouseService("micro_house", microService.Client())
	resp, err := microClient.GetHouseInfo(utils.RPCCtx(), &houseMicro.GetReq{UserName: userName})
	if err != nil {
		fmt.Println("GetHouseInfo RPC 失败", err)
		utils.AbortRPC(ctx)
		return
	}
	if resp == nil {
		utils.AbortRPC(ctx)
		return
	}
	ctx.JSON(http.StatusOK, resp)
}

func PostHousesImage(ctx *gin.Context) {
	houseId := ctx.Param("id")
	fileHeader, err := ctx.FormFile("house_image")
	if houseId == "" || err != nil {
		fmt.Println("传入数据不完整", err)
		utils.AbortParam(ctx, "")
		return
	}
	if fileHeader.Size > 50000000 {
		utils.AbortParam(ctx, "文件过大,请重新选择")
		return
	}
	fileExt := path.Ext(fileHeader.Filename)
	if fileExt != ".png" && fileExt != ".jpg" {
		utils.AbortParam(ctx, "文件类型错误,请重新选择")
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		utils.AbortParam(ctx, "读取文件失败")
		return
	}
	defer file.Close()
	buf := make([]byte, fileHeader.Size)
	if _, err := file.Read(buf); err != nil {
		utils.AbortParam(ctx, "读取文件失败")
		return
	}
	microService := utils.InitMicro()
	microClient := houseMicro.NewHouseService("micro_house", microService.Client())
	resp, err := microClient.UploadHouseImg(utils.RPCCtx(), &houseMicro.ImgReq{
		HouseId: houseId,
		ImgData: buf,
		FileExt: fileExt,
	})
	if err != nil {
		fmt.Println("UploadHouseImg RPC 失败", err)
		utils.AbortRPC(ctx)
		return
	}
	if resp == nil {
		utils.AbortRPC(ctx)
		return
	}
	ctx.JSON(http.StatusOK, resp)
}

// 获取房屋详情
func GetHouseInfo(ctx *gin.Context) {
	houseId := ctx.Param("id")
	if houseId == "" {
		ctx.JSON(http.StatusOK, gin.H{
			"errno":  utils.RECODE_PARAMERR,
			"errmsg": utils.RecodeText(utils.RECODE_PARAMERR),
		})
		return
	}
	userName := ""
	if v := sessions.Default(ctx).Get("userName"); v != nil {
		if name, ok := v.(string); ok {
			userName = name
		}
	}

	microService := utils.InitMicro()
	microClient := houseMicro.NewHouseService("micro_house", microService.Client())
	resp, err := microClient.GetHouseDetail(utils.RPCCtx(), &houseMicro.DetailReq{
		HouseId:  houseId,
		UserName: userName,
	})
	if err != nil {
		fmt.Println("获取房屋详情失败", err)
		ctx.JSON(http.StatusOK, gin.H{
			"errno":  utils.RECODE_SERVERERR,
			"errmsg": utils.RecodeText(utils.RECODE_SERVERERR),
		})
		return
	}
	if resp == nil {
		ctx.JSON(http.StatusOK, gin.H{
			"errno":  utils.RECODE_NODATA,
			"errmsg": utils.RecodeText(utils.RECODE_NODATA),
		})
		return
	}
	ctx.JSON(http.StatusOK, resp)
}

// 搜索房屋
func GetHouses(ctx *gin.Context) {
	aid := strings.TrimSpace(ctx.Query("aid"))
	sd := strings.TrimSpace(ctx.Query("sd"))
	ed := strings.TrimSpace(ctx.Query("ed"))
	sk := strings.TrimSpace(ctx.Query("sk"))
	page := strings.TrimSpace(ctx.Query("p"))
	if page == "" {
		page = "1"
	}
	if sk == "" {
		sk = "new"
	}

	// 日期允许双空；若填写则必须成对且合法
	if sd != "" || ed != "" {
		if sd == "" || ed == "" {
			ctx.JSON(http.StatusOK, gin.H{
				"errno":  utils.RECODE_PARAMERR,
				"errmsg": "请同时选择入住和离店日期",
			})
			return
		}
		startDate, err1 := time.Parse("2006-01-02", sd)
		endDate, err2 := time.Parse("2006-01-02", ed)
		if err1 != nil || err2 != nil {
			ctx.JSON(http.StatusOK, gin.H{
				"errno":  utils.RECODE_PARAMERR,
				"errmsg": "日期格式错误",
			})
			return
		}
		if !endDate.After(startDate) {
			ctx.JSON(http.StatusOK, gin.H{
				"errno":  utils.RECODE_PARAMERR,
				"errmsg": "离店日期须晚于入住日期",
			})
			return
		}
	}

	microService := utils.InitMicro()
	microClient := houseMicro.NewHouseService("micro_house", microService.Client())
	resp, err := microClient.SearchHouse(utils.RPCCtx(), &houseMicro.SearchReq{
		Aid: aid,
		Sd:  sd,
		Ed:  ed,
		Sk:  sk,
		P:   page,
	})
	if err != nil {
		fmt.Println("SearchHouse RPC 失败", err)
		ctx.JSON(http.StatusOK, gin.H{
			"errno":  utils.RECODE_SERVERERR,
			"errmsg": utils.RecodeText(utils.RECODE_SERVERERR),
		})
		return
	}
	if resp == nil {
		ctx.JSON(http.StatusOK, gin.H{
			"errno":  utils.RECODE_SERVERERR,
			"errmsg": utils.RecodeText(utils.RECODE_SERVERERR),
		})
		return
	}
	ctx.JSON(http.StatusOK, resp)
}

// 首页轮播图
func GetIndex(ctx *gin.Context) {
	microService := utils.InitMicro()
	microClient := houseMicro.NewHouseService("micro_house", microService.Client())
	resp, err := microClient.GetIndexHouse(utils.RPCCtx(), &houseMicro.IndexReq{})
	if err != nil {
		fmt.Println("GetIndexHouse RPC 失败", err)
		utils.AbortRPC(ctx)
		return
	}
	if resp == nil {
		utils.AbortRPC(ctx)
		return
	}
	ctx.JSON(http.StatusOK, resp)
}

type OrderStu struct {
	EndDate   string `json:"end_date"`
	HouseId   string `json:"house_id"`
	StartDate string `json:"start_date"`
}

// 下订单
func PostOrders(ctx *gin.Context) {
	var order OrderStu
	if err := ctx.Bind(&order); err != nil {
		fmt.Println("获取订单数据错误", err)
		ctx.JSON(http.StatusOK, gin.H{
			"errno":  utils.RECODE_PARAMERR,
			"errmsg": utils.RecodeText(utils.RECODE_PARAMERR),
		})
		return
	}
	if order.HouseId == "" || order.StartDate == "" || order.EndDate == "" {
		ctx.JSON(http.StatusOK, gin.H{
			"errno":  utils.RECODE_PARAMERR,
			"errmsg": utils.RecodeText(utils.RECODE_PARAMERR),
		})
		return
	}
	startDate, err := time.Parse("2006-01-02", order.StartDate)
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{
			"errno":  utils.RECODE_PARAMERR,
			"errmsg": "入住日期格式错误",
		})
		return
	}
	endDate, err := time.Parse("2006-01-02", order.EndDate)
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{
			"errno":  utils.RECODE_PARAMERR,
			"errmsg": "离店日期格式错误",
		})
		return
	}
	if !endDate.After(startDate) {
		ctx.JSON(http.StatusOK, gin.H{
			"errno":  utils.RECODE_PARAMERR,
			"errmsg": "离店日期须晚于入住日期",
		})
		return
	}

	userNameVal := sessions.Default(ctx).Get("userName")
	userName, ok := userNameVal.(string)
	if !ok || userName == "" {
		ctx.JSON(http.StatusOK, gin.H{
			"errno":  utils.RECODE_SESSIONERR,
			"errmsg": utils.RecodeText(utils.RECODE_SESSIONERR),
		})
		return
	}

	microService := utils.InitMicro()
	microClient := orderMicro.NewOrderService("micro_order", microService.Client())
	resp, err := microClient.CreateOrder(utils.RPCCtx(), &orderMicro.Request{
		StartDate: order.StartDate,
		EndDate:   order.EndDate,
		HouseId:   order.HouseId,
		UserName:  userName,
	})
	if err != nil {
		fmt.Println("CreateOrder RPC 失败", err)
		ctx.JSON(http.StatusOK, gin.H{
			"errno":  utils.RECODE_SERVERERR,
			"errmsg": utils.RecodeText(utils.RECODE_SERVERERR),
		})
		return
	}
	if resp == nil {
		ctx.JSON(http.StatusOK, gin.H{
			"errno":  utils.RECODE_SERVERERR,
			"errmsg": utils.RecodeText(utils.RECODE_SERVERERR),
		})
		return
	}
	ctx.JSON(http.StatusOK, resp)
}

// 获取订单信息
// 获取订单信息
func GetUserOrder(ctx *gin.Context) {
	role := ctx.Query("role")
	if role == "" {
		ctx.JSON(http.StatusOK, gin.H{
			"errno":  utils.RECODE_PARAMERR,
			"errmsg": utils.RecodeText(utils.RECODE_PARAMERR),
		})
		return
	}
	userNameVal := sessions.Default(ctx).Get("userName")
	userName, ok := userNameVal.(string)
	if !ok || userName == "" {
		ctx.JSON(http.StatusOK, gin.H{
			"errno":  utils.RECODE_SESSIONERR,
			"errmsg": utils.RecodeText(utils.RECODE_SESSIONERR),
		})
		return
	}

	microService := utils.InitMicro()
	microClient := orderMicro.NewOrderService("micro_order", microService.Client())
	resp, err := microClient.GetOrderInfo(utils.RPCCtx(), &orderMicro.GetReq{
		Role:     role,
		UserName: userName,
	})
	if err != nil {
		fmt.Println("GetOrderInfo RPC 失败", err)
		ctx.JSON(http.StatusOK, gin.H{
			"errno":  utils.RECODE_SERVERERR,
			"errmsg": utils.RecodeText(utils.RECODE_SERVERERR),
		})
		return
	}
	if resp == nil {
		ctx.JSON(http.StatusOK, gin.H{
			"errno":  utils.RECODE_SERVERERR,
			"errmsg": utils.RecodeText(utils.RECODE_SERVERERR),
		})
		return
	}
	ctx.JSON(http.StatusOK, resp)
}

type StatusStu struct {
	Action string `json:"action"`
	Reason string `json:"reason"`
}

// 更新订单状态（房东接单/拒单）
func PutOrders(ctx *gin.Context) {
	id := ctx.Param("id")
	var statusStu StatusStu
	err := ctx.Bind(&statusStu)
	if err != nil || id == "" {
		fmt.Println("获取订单状态数据错误", err)
		ctx.JSON(http.StatusOK, gin.H{
			"errno":  utils.RECODE_PARAMERR,
			"errmsg": utils.RecodeText(utils.RECODE_PARAMERR),
		})
		return
	}
	userNameVal := sessions.Default(ctx).Get("userName")
	userName, ok := userNameVal.(string)
	if !ok || userName == "" {
		ctx.JSON(http.StatusOK, gin.H{
			"errno":  utils.RECODE_SESSIONERR,
			"errmsg": utils.RecodeText(utils.RECODE_SESSIONERR),
		})
		return
	}

	microService := utils.InitMicro()
	microClient := orderMicro.NewOrderService("micro_order", microService.Client())
	resp, err := microClient.UpdateStatus(utils.RPCCtx(), &orderMicro.UpdateReq{
		Action:   statusStu.Action,
		Reason:   statusStu.Reason,
		Id:       id,
		UserName: userName,
	})
	if err != nil {
		fmt.Println("UpdateStatus RPC 失败", err)
		ctx.JSON(http.StatusOK, gin.H{
			"errno":  utils.RECODE_SERVERERR,
			"errmsg": utils.RecodeText(utils.RECODE_SERVERERR),
		})
		return
	}
	if resp == nil {
		ctx.JSON(http.StatusOK, gin.H{
			"errno":  utils.RECODE_SERVERERR,
			"errmsg": utils.RecodeText(utils.RECODE_SERVERERR),
		})
		return
	}
	ctx.JSON(http.StatusOK, resp)
}

// PostOrderPayment 模拟支付
func PostOrderPayment(ctx *gin.Context) {
	id := ctx.Param("id")
	userNameVal := sessions.Default(ctx).Get("userName")
	userName, ok := userNameVal.(string)
	if id == "" || !ok || userName == "" {
		ctx.JSON(http.StatusOK, gin.H{
			"errno":  utils.RECODE_PARAMERR,
			"errmsg": utils.RecodeText(utils.RECODE_PARAMERR),
		})
		return
	}
	microService := utils.InitMicro()
	microClient := orderMicro.NewOrderService("micro_order", microService.Client())
	resp, err := microClient.PayOrder(utils.RPCCtx(), &orderMicro.PayReq{
		OrderId:  id,
		UserName: userName,
	})
	if err != nil {
		fmt.Println("PayOrder RPC 失败", err)
		ctx.JSON(http.StatusOK, gin.H{
			"errno":  utils.RECODE_SERVERERR,
			"errmsg": utils.RecodeText(utils.RECODE_SERVERERR),
		})
		return
	}
	if resp == nil {
		ctx.JSON(http.StatusOK, gin.H{
			"errno":  utils.RECODE_SERVERERR,
			"errmsg": utils.RecodeText(utils.RECODE_SERVERERR),
		})
		return
	}
	ctx.JSON(http.StatusOK, resp)
}

// PutOrderCancel 取消待支付订单
func PutOrderCancel(ctx *gin.Context) {
	id := ctx.Param("id")
	userNameVal := sessions.Default(ctx).Get("userName")
	userName, ok := userNameVal.(string)
	if id == "" || !ok || userName == "" {
		ctx.JSON(http.StatusOK, gin.H{
			"errno":  utils.RECODE_PARAMERR,
			"errmsg": utils.RecodeText(utils.RECODE_PARAMERR),
		})
		return
	}
	microService := utils.InitMicro()
	microClient := orderMicro.NewOrderService("micro_order", microService.Client())
	resp, err := microClient.CancelOrder(utils.RPCCtx(), &orderMicro.CancelReq{
		OrderId:  id,
		UserName: userName,
	})
	if err != nil {
		fmt.Println("CancelOrder RPC 失败", err)
		ctx.JSON(http.StatusOK, gin.H{
			"errno":  utils.RECODE_SERVERERR,
			"errmsg": utils.RecodeText(utils.RECODE_SERVERERR),
		})
		return
	}
	if resp == nil {
		ctx.JSON(http.StatusOK, gin.H{
			"errno":  utils.RECODE_SERVERERR,
			"errmsg": utils.RecodeText(utils.RECODE_SERVERERR),
		})
		return
	}
	ctx.JSON(http.StatusOK, resp)
}

// PutOrderComment 租户评价订单
func PutOrderComment(ctx *gin.Context) {
	id := ctx.Param("id")
	var body struct {
		Comment string `json:"comment"`
	}
	if err := ctx.Bind(&body); err != nil || id == "" || body.Comment == "" {
		ctx.JSON(http.StatusOK, gin.H{
			"errno":  utils.RECODE_PARAMERR,
			"errmsg": utils.RecodeText(utils.RECODE_PARAMERR),
		})
		return
	}
	userNameVal := sessions.Default(ctx).Get("userName")
	userName, ok := userNameVal.(string)
	if !ok || userName == "" {
		ctx.JSON(http.StatusOK, gin.H{
			"errno":  utils.RECODE_SESSIONERR,
			"errmsg": utils.RecodeText(utils.RECODE_SESSIONERR),
		})
		return
	}
	microService := utils.InitMicro()
	microClient := orderMicro.NewOrderService("micro_order", microService.Client())
	resp, err := microClient.CommentOrder(utils.RPCCtx(), &orderMicro.CommentReq{
		OrderId:  id,
		UserName: userName,
		Comment:  body.Comment,
	})
	if err != nil {
		fmt.Println("CommentOrder RPC 失败", err)
		ctx.JSON(http.StatusOK, gin.H{
			"errno":  utils.RECODE_SERVERERR,
			"errmsg": utils.RecodeText(utils.RECODE_SERVERERR),
		})
		return
	}
	if resp == nil {
		ctx.JSON(http.StatusOK, gin.H{
			"errno":  utils.RECODE_SERVERERR,
			"errmsg": utils.RecodeText(utils.RECODE_SERVERERR),
		})
		return
	}
	ctx.JSON(http.StatusOK, resp)
}
