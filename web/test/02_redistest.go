package main

import (
	"fmt"
	"github.com/gomodule/redigo/redis"
)

func main() {
	//	1.连接数据库
	conn, err := redis.Dial("tcp", "127.0.0.1:6379")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer conn.Close()
	//	2.操作数据库
	//Do() 函数【推荐】;  Send()函数, 需要配合Flush()、Receive() 3 个函数使用。
	reply, err := conn.Do("SET", "t02", "this is a test")
	//	3.回复助手类
	//	确定为具体的数据类型
	r, e := redis.String(reply, err)
	fmt.Println(r, e)
}
