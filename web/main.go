package main

import (
	"gin-micro-ihome/web/conf"
	"gin-micro-ihome/web/controller"
	"gin-micro-ihome/web/model"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/redis"
	"github.com/gin-gonic/gin"
)

func LoginFilter() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// 初始化 Session 对象
		s := sessions.Default(ctx)
		userName := s.Get("userName")

		if userName == nil {
			ctx.Abort() // 从这里返回, 不必继续执行了
		} else {
			ctx.Next() // 继续向下
		}
	}
}

func main() {
	//初始化操作
	//#
	//初始化Mysql连接池
	model.InitDb()
	//初始化redis连接池
	model.InitRedis()
	//1.初始化路由
	router := gin.Default()

	//初始化容器
	store, _ := redis.NewStore(10, "tcp", conf.RedisAddr, "", "", []byte("secret"))
	//使用容器
	router.Use(sessions.Sessions("loginSession", store)) //使用中间件--指定容器

	//2.路由匹配
	router.Static("/home", "web/view")
	//添加路由组
	r1 := router.Group("/api/v1.0")
	{
		r1.GET("/session", controller.GetSession) // 此处GetSession是回调函数，只用写函数名

		r1.GET("/imagecode/:uuid", controller.GetImageCd)
		r1.GET("/smscode/:phone", controller.GetSmscd)
		r1.POST("/users", controller.PostReg)
		r1.GET("/areas", controller.GetArea)
		r1.POST("/sessions", controller.PostLogin)
		//展示首页轮播图
		r1.GET("/house/index", controller.GetIndex)

		r1.Use(LoginFilter()) //调用中间件--下方的路由都不需要校验session了,可以直接获取数据
		r1.DELETE("session", controller.DeleteSession)
		r1.GET("/user", controller.GetUserInfo)
		r1.PUT("/user/name", controller.PutUserInfo)
		r1.POST("user/avatar", controller.PostAvater)

		//	上传实名认证
		r1.POST("/user/auth", controller.PutUserAuth)
		// 获取实名认证
		r1.GET("/user/auth", controller.GetUserInfo)

		//获取房源信息
		r1.GET("/user/houses", controller.GetUserHouses)
		//	上传房屋信息
		r1.POST("/houses", controller.PostHouses)
		//添加房源图片
		r1.POST("/houses/:id/images", controller.PostHousesImage)
		//展示房屋详情
		r1.GET("/houses/:id", controller.GetHouseInfo)
		//搜索房屋
		r1.GET("/houses", controller.GetHouses)

		//提交订单
		r1.POST("/orders", controller.PostOrders)
		//获取订单
		r1.GET("/user/orders", controller.GetUserOrder)
		//同意/拒绝订单
		r1.PUT("/orders/:id/status", controller.PutOrders)

	}

	//3.启动运行
	router.Run(":8080")
}
