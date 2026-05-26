package model

import (
	"fmt"
	"gin-micro-ihome/web/conf"
	"github.com/gomodule/redigo/redis"
)

// 创建全局redis 连接池 句柄
var RedisPool redis.Pool

// 创建函数, 初始化Redis连接池
func InitRedis() {
	RedisPool = redis.Pool{
		MaxIdle:         20,
		MaxActive:       50,
		MaxConnLifetime: 60 * 5,
		IdleTimeout:     60,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", conf.RedisAddr)
		},
	}
}

func SaveImgCode(code, uuid string) error {
	// 1. 链接数据库
	conn, err := redis.Dial("tcp", conf.RedisAddr)
	if err != nil {
		fmt.Println("redis Dial err:", err)
		return err
	}
	defer conn.Close()

	// 2. 写数据库  --- 有效时间 5 分钟
	_, err = conn.Do("setex", uuid, 60*5, code)

	return err // 不需要回复助手!
}

// 校验图片验证码
func CheckImgCode(uuid, imgCode string) bool {
	// 链接 redis --- 从链接池中获取链接
	conn, err := redis.Dial("tcp", conf.RedisAddr)
	if err != nil {
		fmt.Println("redis.Dial err:", err)
		return false
	}
	//conn := RedisPool.Get()
	defer conn.Close()

	// 查询 redis 数据
	code, err := redis.String(conn.Do("get", uuid))
	if err != nil {
		fmt.Println("查询错误 err:", err)
		return false
	}

	// 返回校验结果
	return code == imgCode
}

// 处理登录业务
func Login(mobile, pwd string) (string, error) {
	var user User
	err := GlobalConn.Where("mobile = ?", mobile).Select("name").
		Where("password_hash = ?", pwd).
		Find(&user).Error
	if err != nil {
		fmt.Println("login函数出错", err)
	}
	return user.Name, err
}

// 获取用户信息
func GetUserInfo(userName string) (User, error) {
	var user User
	err := GlobalConn.Where("name = ?", userName).First(&user).Error
	return user, err
}

// 更新用户名
func UpdateUserName(newName, OldName string) error {
	return GlobalConn.Model(&User{}).Where("name = ?", OldName).
		Update("name", newName).Error
}

// 根据用户名更新用户头像
func UpdateAvatar(userName, avatar string) error {
	return GlobalConn.Model(&User{}).Where("name = ?", userName).
		Update("avatar_url", avatar).Error

}
