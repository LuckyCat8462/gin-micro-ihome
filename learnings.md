# Learning Notes

> GORM中文文档：[https://gorm.io/zh_CN/docs/index.html](https://gorm.io/zh_CN/docs/index.html)

---

## 第一章 ORM与GORM

### 1.1 ORM基本概念

**ORM**：Object Relational Mapping —— 对象关系映射

**作用**：
- 通过操作结构体对象，来达到操作数据库表的目的
- 通过结构体对象，来生成数据库表

**优点**：
- SQL有可能比较复杂（Oracle --- 子查询 -- 嵌套），ORM操作数据库不需要使用SQL
- 不同开发者，书写的SQL语句执行效率不同

**Go语言支持的ORM**：
- gORM：http://gorm.book.jasperxu.com/
- xORM

### 1.2 ORM操作数据库

#### 1.2.1 连接数据库

GORM支持多种数据库，以下是常见数据库的连接示例：

**连接MySQL**：
```go
package main

import (
  "gorm.io/driver/mysql"
  "gorm.io/gorm"
)

func main() {
  // dsn格式：user:password@tcp(host:port)/dbname?charset=utf8mb4&parseTime=True&loc=Local
  dsn := "root:123456@tcp(127.0.0.1:3306)/testdb?charset=utf8mb4&parseTime=True&loc=Local"
  db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
  if err != nil {
    panic("failed to connect database")
  }
  
  // 自动迁移 - 根据结构体创建或更新表
  db.AutoMigrate(&User{})
}
```

**连接PostgreSQL**：
```go
import (
  "gorm.io/driver/postgres"
  "gorm.io/gorm"
)

dsn := "host=localhost user=gorm password=gorm dbname=gorm port=5432 sslmode=disable TimeZone=Asia/Shanghai"
db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
```

**连接SQLite**：
```go
import (
  "gorm.io/driver/sqlite"
  "gorm.io/gorm"
)

db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
```

#### 1.2.2 连接池配置

```go
import (
  "gorm.io/driver/mysql"
  "gorm.io/gorm"
  "github.com/go-sql-driver/mysql"
)

// 配置连接池
sqlDB, err := db.DB()
sqlDB.SetMaxIdleConns(10)           // 最大空闲数
sqlDB.SetMaxOpenConns(100)          // 最大打开连接数
sqlDB.SetConnLifetime(time.Hour)    // 连接最大生命周期
```

#### 1.2.3 GORM配置选项

```go
db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
  NamingStrategy: schema.NamingStrategy{
    TablePrefix:   "t_",        // 表名前缀
    SingularTable: true,        // 使用单数表名
  },
  Logger: logger.Default.LogMode(logger.Info), // 日志模式
})
```

### 1.3 GORM特点

- gorm操作的都是**表、表数据** —— 不能操作数据库！需使用SQL配合完成

---

## 第二章 MySQL与Redis操作指南

### 2.1 MySQL操作指令

**测试用户信息**：
| 项目 | 值 |
|------|-----|
| 用户名 | neko |
| 密码 | neko123456 |

#### 2.1.1 服务管理

```bash
# 启动MySQL服务
service mysql start
# 或
systemctl start mysql

# 确认MySQL服务状态
ps xua | grep mysql
```

#### 2.1.2 登录与连接

```bash
# 连接MySQL数据库（root用户）
mysql -uroot -p123456

# 普通用户登录
mysql -u username -p

# 登录root用户（密码为空）
sudo mysql
```

#### 2.1.3 数据库操作

```bash
# 查看所有数据库
show databases;

# 创建数据库
create database test charset=utf8;

# 删除数据库（慎用）
drop database t1;  # t1代表库名

# 选择数据库
use 数据库名;

# 查看当前数据库中的表
show tables;
```

#### 2.1.4 用户权限管理

```bash
# 给用户授予某张表的全部权限
grant all privileges on search_house.* to 'neko'@'localhost';
```

---

### 2.2 Redis操作指令

#### 2.2.1 服务启动

```bash
# 启动Redis服务
redis-server

# 连接Redis客户端
redis-cli -h 127.0.0.1 -p 6379

# 如果出现乱码（中文），使用--raw参数
redis-cli -h 127.0.0.1 -p 6379 --raw
```

#### 2.2.2 基本数据操作

| 命令 | 说明 |
|------|------|
| `SET mykey myvalue` | 设置键值对 |
| `GET mykey` | 获取键的值 |
| `KEYS *` | 查看所有键（生产环境不推荐） |
| `SCAN 0 MATCH * COUNT 1000` | 安全地迭代键（推荐） |
| `DEL mykey` | 删除键 |
| `EXISTS mykey` | 检查键是否存在 |
| `DBSIZE` | 查看当前数据库键的数量 |

#### 2.2.3 数据库管理

```bash
# 清空当前数据库
FLUSHDB

# 清空所有数据库（慎用）
FLUSHALL

# 查看Redis服务器信息
INFO

# 退出客户端
exit
```

#### 2.2.4 连接池配置

```go
type Pool struct {
    Dial func() (Conn, error)       // 连接数据库使用
    MaxIdle int                     // 最大空闲数 == 初始化连接数
    MaxActive int                   // 最大存活数 > MaxIdle
    IdleTimeout time.Duration       // 空闲超时时间
    MaxConnLifetime time.Duration   // 最大生命周期
}
```

---

## 第三章 Go语言特殊函数

Go语言中有两个特殊函数：

### 3.1 main() 函数

- **作用**：项目的入口函数
- **特点**：每个可执行程序必须且只能有一个main函数

### 3.2 init() 函数

- **作用**：当导入包但未在程序中使用时，在main()调用之前自动被调用
- **示例**：查看MySQL包的"mysql"源码，在driver.go底部包含init()函数的定义
- **典型用途**：实现注册MySQL驱动

> **注意**：init()函数会先于main()函数被调用

---

## 第四章 服务启动指令汇总

### 4.1 Redis

```bash
redis-server
# 或
redis-cli -h 127.0.0.1 -p 6379
# 中文乱码处理
redis-cli -h 127.0.0.1 -p 6379 --raw
```

### 4.2 Consul

```bash
consul agent -dev
```

### 4.3 MySQL

```bash
service mysql start
# 或
systemctl start mysql

# 登录
mysql -u username -p
# root用户登录（密码为空）
sudo mysql
```

### 4.4 用户权限

```bash
grant all privileges on search_house.* to 'neko'@'localhost';
```

---

## 第五章 Cookie和Session

### 5.1 HTTP协议版本

| 版本 | 特点 |
|------|------|
| http/1.0 | 无状态，短连接 |
| http/1.1 | 可记录状态，默认支持 |
| http/2.0 | 支持长连接，协议头：`Connection: keep-alive` |

### 5.2 Cookie

**基本特性**：
- 最早的http/1.0版提供Cookie机制，但没有Session
- **作用**：一定时间内存储用户的连接信息（如用户名、登录时间等不敏感信息）
- **出身**：HTTP自带机制，Session不是！
- **存储位置**：客户端（浏览器）中，浏览器可以存储少量数据
- **存储形式**：key-value格式
- **安全性**：不安全，数据直接存储在浏览器上，可被查看

### 5.3 Cookie与Session对比

| 对比项 | Cookie | Session |
|--------|--------|---------|
| 存储位置 | 浏览器 | 服务器 |
| 生成位置 | Web服务 | Web服务（以Cookie加密为key） |
| 安全性 | 较低 | 较高 |

**工作流程**：
1. 浏览器发送请求（不携带数据）到Web服务
2. Web服务产生Cookie，携带Cookie返回到浏览器；浏览器存储Cookie
3. Cookie加密作为key，生成Session作为value，存入容器中
4. 浏览器携带上次的Cookie发送到Web服务；Web服务以Cookie加密为key查询Session

### 5.4 Session操作

Gin框架默认不支持Session功能，需添加插件：

**安装Session插件**：
```bash
go get github.com/gin-contrib/sessions
```

**context.SetCookie()参数说明**：
| 参数 | 说明 |
|------|------|
| name | Cookie名称 |
| value | Cookie值 |
| maxAge | 最大生命周期（=0未指定，<0删除，>0指定秒数） |
| path | 路径（通常传""） |
| domain | 域名/IP地址 |
| secure | 是否安全保护（true：仅HTTPS） |
| httpOnly | 是否仅HTTP协议访问 |

---

## 第六章 Redis Session存储

### 6.1 使用Redis作为Session存储示例

```go
package main

import (
  "github.com/gin-contrib/sessions"
  "github.com/gin-contrib/sessions/redis"
  "github.com/gin-gonic/gin"
)

func main() {
  r := gin.Default()
  
  // NewStore参数说明：
  // size: 容器大小
  // network: 协议（tcp/udp）
  // address: IP:port
  // password: Redis密码（无则传""）
  // []byte("secret"): 加密密钥
  store, _ := redis.NewStore(10, "tcp", "localhost:6379", "", []byte("secret"))
  r.Use(sessions.Sessions("mysession", store))

  r.GET("/incr", func(c *gin.Context) {
    session := sessions.Default(c)
    var count int
    v := session.Get("count")
    if v == nil {
      count = 0
    } else {
      count = v.(int)
      count++
    }
    session.Set("count", count)
    session.Save()
    c.JSON(200, gin.H{"count": count})
  })
  
  r.Run(":8000")
}
```

---

## 第七章 中间件（Middleware）

### 7.1 中间件概念

**定义**：
- 早期：用于系统和应用之间，内核 —— 中间件 —— 用户应用
- 现在：用于两个模块之间的功能软件，路由 ——> 中间件（过滤作用）——> 控制器

**特性**：
- 中间件对指定位置**以下**的路由全部生效
- 指定位置**以上**的路由不生效
- 设置好中间件后，所有后续路由都会使用该中间件

### 7.2 中间件的三种控制方式

#### 7.2.1 Next()

- **作用**：跳过当前中间件剩余内容，执行下一个中间件
- **返回行为**：当所有操作执行完之后，以出栈的执行顺序返回，执行剩余代码

#### 7.2.2 return

- **作用**：终止执行当前中间件剩余内容，执行下一个中间件
- **返回行为**：当所有函数执行结束后，以出栈的顺序执行返回，但**不执行return后的代码**

#### 7.2.3 Abort()

- **作用**：只执行当前中间件
- **返回行为**：操作完成后，以出栈的顺序，依次返回上一级中间件

---

*文档整理完成*
