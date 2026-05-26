package model

import (
	"fmt"
	"getCaptcha/conf"
	"github.com/gomodule/redigo/redis"
)

//- 长连接：
//  - TCP。—— 有状态！
//- 短连接：
//  - http。—— 无状态！断开后，再发送请求，与上次发送无关！
//    - 选用带有 “时效性” 的介质，存储。 —— redis数据库。setex --- expire。

// 存储图片id 到redis数据库
func SaveImageCode(code, uuid string) error {
	//	1.连接数据库
	conn, err := redis.Dial("tcp", conf.RedisAddr)
	if err != nil {
		fmt.Println(err)
		return err
	}
	defer conn.Close()
	//	2.操作数据库
	//Do() 函数【推荐】;  Send()函数, 需要配合Flush()、Receive() 3 个函数使用。
	//set后可以直接跟key value
	//reply, err := conn.Do("SET", uuid, "this is a test")
	//setex后key要跟上时长
	_, err = conn.Do("setex", uuid, 60*5, code)

	//	3.回复助手类
	//	此处无需恢复助手
	return err
}
