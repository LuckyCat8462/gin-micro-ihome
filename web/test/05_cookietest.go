package main

import (
	"fmt"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/redis"

	//"github.com/gomodule/redigo/redis"
	//"github.com/gin-contrib/sessions/redis"
	"github.com/gin-gonic/gin"
)

func main() {
	//初始化容器
	router := gin.Default()

	store, _ := redis.NewStore(10, "tcp", "127.0.0.1:6379", "", "", []byte("secret"))
	//可以设置临时session
	store.Options(sessions.Options{MaxAge: 60 * 60 * 24})
	//使用容器
	router.Use(sessions.Sessions("sessionTest1", store))
	//此处的sessionTest1为cookie值

	router.GET("/test", func(c *gin.Context) {
		session := sessions.Default(c)

		//设置session
		session.Set("session1", "sessionTest1")
		session.Save()
		c.Writer.WriteString("测试session...")

		//设置cookie
		//c.SetCookie("test1", "nekotest1", 60*60, "", "", true, true)
		//c.Writer.WriteString("测试cookie...")

		//	获取session
		v := session.Get("session1")
		fmt.Println("获取session", v)
	})

	router.Run(":8089")
}
