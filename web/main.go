package main

import (
	"fmt"
	"log"
	"net/http"

	"gin-micro-ihome/web/conf"
	"gin-micro-ihome/web/controller"
	"gin-micro-ihome/web/model"
	"gin-micro-ihome/web/utils"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/redis"
	"github.com/gin-gonic/gin"
)

func LoginFilter() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		s := sessions.Default(ctx)
		userName := s.Get("userName")

		name, ok := userName.(string)
		if !ok || name == "" {
			ctx.JSON(http.StatusOK, gin.H{
				"errno":  utils.RECODE_SESSIONERR,
				"errmsg": utils.RecodeText(utils.RECODE_SESSIONERR),
			})
			ctx.Abort()
			return
		}
		ctx.Next()
	}
}

func main() {
	model.InitDb()
	model.InitRedis()
	router := gin.Default()

	store, err := redis.NewStore(10, "tcp", conf.RedisAddr, "", "", []byte(conf.SessionSecret))
	if err != nil {
		log.Fatalf("初始化 redis session store 失败: %v", err)
	}
	router.Use(sessions.Sessions("loginSession", store))

	router.Static("/home", "web/view")
	r1 := router.Group("/api/v1.0")
	{
		r1.GET("/session", controller.GetSession)

		r1.GET("/imagecode/:uuid", controller.GetImageCd)
		r1.GET("/smscode/:phone", controller.GetSmscd)
		r1.POST("/users", controller.PostReg)
		r1.GET("/areas", controller.GetArea)
		r1.POST("/sessions", controller.PostLogin)
		r1.GET("/house/index", controller.GetIndex)
		// 游客可访问：房源搜索与详情
		r1.GET("/houses", controller.GetHouses)
		r1.GET("/houses/:id", controller.GetHouseInfo)

		r1.Use(LoginFilter())
		r1.DELETE("session", controller.DeleteSession)
		r1.GET("/user", controller.GetUserInfo)
		r1.PUT("/user/name", controller.PutUserInfo)
		r1.POST("user/avatar", controller.PostAvater)

		r1.POST("/user/auth", controller.PutUserAuth)
		r1.GET("/user/auth", controller.GetUserInfo)

		r1.GET("/user/houses", controller.GetUserHouses)
		r1.POST("/houses", controller.PostHouses)
		r1.POST("/houses/:id/images", controller.PostHousesImage)

		r1.POST("/orders", controller.PostOrders)
		r1.POST("/orders/:id/payment", controller.PostOrderPayment)
		r1.PUT("/orders/:id/cancel", controller.PutOrderCancel)
		r1.PUT("/orders/:id/comment", controller.PutOrderComment)
		r1.GET("/user/orders", controller.GetUserOrder)
		r1.PUT("/orders/:id/status", controller.PutOrders)
	}

	fmt.Println("web gateway listening on :8080")
	router.Run(":8080")
}
