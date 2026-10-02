package model

import (
	"errors"

	_ "github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// ErrLoginFailed 手机号不存在或密码错误
var ErrLoginFailed = errors.New("login failed")

// ErrUserExists 手机号已注册
var ErrUserExists = errors.New("user already exists")

// 注册用户信息,写 MySQL 数据库.
func RegisterUser(mobile, pwd string) error {
	var count int64
	if err := GlobalConn.Model(&User{}).Where("mobile = ?", mobile).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return ErrUserExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	var user User
	user.Mobile = mobile
	user.Name = mobile // 默认使用手机号作为用户名
	user.Password_hash = string(hash)

	// 插入数据到MySQL
	return GlobalConn.Create(&user).Error
}

// 处理登录业务：按手机号查用户，校验密码（bcrypt；兼容旧明文并升级）
func Login(mobile, pwd string) (string, error) {
	var user User
	err := GlobalConn.Where("mobile = ?", mobile).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", ErrLoginFailed
		}
		return "", err
	}

	if bcrypt.CompareHashAndPassword([]byte(user.Password_hash), []byte(pwd)) == nil {
		return user.Name, nil
	}

	// 兼容学习库中的明文密码：匹配成功后升级为 bcrypt
	if user.Password_hash == pwd {
		hash, genErr := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
		if genErr != nil {
			return "", genErr
		}
		if saveErr := GlobalConn.Model(&user).Update("password_hash", string(hash)).Error; saveErr != nil {
			return "", saveErr
		}
		return user.Name, nil
	}

	return "", ErrLoginFailed
}
