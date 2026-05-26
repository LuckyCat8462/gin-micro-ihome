# gin-micro-ihome

> 基于 Gin 框架和 Micro 微服务架构的在线租房平台后端系统

---

## 🏠 项目简介

这是一个基于 Go 语言开发的在线租房平台后端服务，采用 Gin 框架作为 Web 层，Micro 框架实现微服务架构，支持用户注册登录、房源发布、订单管理等核心功能。

## 🛠️ 技术栈

| 分类 | 技术 | 版本 |
|------|------|------|
| 语言 | Go | 1.18+ |
| Web框架 | Gin | v1.9+ |
| 微服务框架 | Micro | v4+ |
| ORM | GORM | v1.25+ |
| 数据库 | MySQL | 5.7+ |
| 缓存 | Redis | 6.0+ |
| 服务发现 | Consul | 1.15+ |
| 序列化 | Protocol Buffers | 3.0+ |

## 📁 项目结构

| 目录 | 说明 |
|------|------|
| `Learning/` | Go语言学习笔记和示例代码 |
| `MyShell/` | 运维脚本（启动、停止、修复脚本） |
| `conf/` | 全局配置文件 |
| `logs/` | 日志存储目录 |
| `service/getCaptcha/` | 图形验证码微服务 |
| `service/house/` | 房屋信息管理微服务 |
| `service/order/` | 订单管理微服务 |
| `service/register/` | 用户注册微服务 |
| `service/user/` | 用户管理微服务 |
| `web/` | Web层（前端接口、静态资源） |

### 微服务模块结构

每个微服务模块包含以下目录：

| 目录 | 说明 |
|------|------|
| `conf/` | 服务配置文件 |
| `handler/` | 请求处理逻辑 |
| `model/` | 数据模型定义 |
| `proto/` | Protocol Buffers定义 |
| `utils/` | 工具函数 |

### Web层结构

| 目录 | 说明 |
|------|------|
| `conf/` | Web配置文件 |
| `controller/` | 控制器层 |
| `model/` | 数据模型 |
| `proto/` | Proto客户端文件 |
| `utils/` | 工具函数 |
| `view/` | 前端静态资源（HTML/CSS/JS） |

## 🚀 快速开始

### 环境要求

| 依赖 | 版本 | 说明 |
|------|------|------|
| Go | 1.24+ | 编程语言 |
| MySQL | 5.7+ | 数据库 |
| Redis | 6.0+ | 缓存（Session存储） |
| Consul | 1.15+ | 服务发现 |

### 安装依赖

```bash
# 进入项目目录
cd gin-micro-ihome

# 安装Go依赖
go mod tidy

# 安装Micro CLI（可选，用于微服务管理）
go install go-micro.dev/v5@latest
```

### 数据库配置

1. **创建MySQL数据库**：

```bash
# 登录MySQL
mysql -uroot -p

# 创建数据库
CREATE DATABASE search_house CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

# 创建用户并授权
GRANT ALL PRIVILEGES ON search_house.* TO 'u01'@'localhost' IDENTIFIED BY 'u01123456';
FLUSH PRIVILEGES;
```

2. **配置文件说明**（`conf/config.go`）：

```go
const MysqlName = "u01"      // 用户名
const MysqlPwd = "u01123456" // 密码
const MysqlAddr = "127.0.0.1" // 地址
const MysqlPort = "3306"      // 端口
const MysqlDB = "search_house" // 数据库名
```

### 启动依赖服务

```bash
# 启动MySQL
service mysql start
# 或
systemctl start mysql

# 启动Redis（后台运行）
redis-server --daemonize yes

# 验证Redis连接
redis-cli ping  # 应返回 PONG

# 启动Consul（开发模式）
consul agent -dev
```

### 运行服务

#### 方式一：使用一键启动脚本（推荐）

```bash
# 启动所有服务
./MyShell/start-all.sh

# 停止所有服务
./MyShell/stop-all.sh
```

#### 方式二：手动启动（开发调试）

**终端1 - 启动Consul**：
```bash
consul agent -dev
```

**终端2 - 启动微服务**：
```bash
cd service/getCaptcha && go run main.go
```

**终端3 - 启动用户服务**：
```bash
cd service/user && go run main.go
```

**终端4 - 启动房屋服务**：
```bash
cd service/house && go run main.go
```

**终端5 - 启动订单服务**：
```bash
cd service/order && go run main.go
```

**终端6 - 启动注册服务**：
```bash
cd service/register && go run main.go
```

**终端7 - 启动Web网关**：
```bash
cd web && go run main.go
```

### 服务访问

| 服务 | 地址 | 说明 |
|------|------|------|
| Web首页 | http://localhost:8080 | 前端页面 |
| Consul UI | http://localhost:8500 | 服务发现管理 |
| Redis | localhost:6379 | 缓存服务 |

### 验证服务

```bash
# 检查所有服务是否正常运行
curl -s http://localhost:8080/api/v1.0/areas

# 查看Consul服务注册
curl -s http://localhost:8500/v1/catalog/services | python3 -m json.tool
```

## 🔌 API接口

### 用户模块

| 接口 | 方法 | 路径 | 说明 |
|------|------|------|------|
| 用户注册 | POST | /api/v1.0/users | 新用户注册 |
| 用户登录 | POST | /api/v1.0/sessions | 用户登录 |
| 用户信息 | GET | /api/v1.0/user | 获取用户信息 |
| 用户退出 | DELETE | /api/v1.0/sessions | 用户退出 |

### 房屋模块

| 接口 | 方法 | 路径 | 说明 |
|------|------|------|------|
| 获取地区 | GET | /api/v1.0/areas | 获取地区信息 |
| 房屋列表 | GET | /api/v1.0/houses/index | 获取房屋列表 |
| 房屋详情 | GET | /api/v1.0/houses/:id | 获取房屋详情 |
| 发布房源 | POST | /api/v1.0/houses | 发布新房源 |

### 验证码模块

| 接口 | 方法 | 路径 | 说明 |
|------|------|------|------|
| 获取验证码 | GET | /api/v1.0/imagecode/:uuid | 获取图形验证码 |

## ❌ 错误码说明

| 错误码 | 错误信息 | 英文标识 |
|--------|----------|----------|
| 0 | 成功 | RECODE_OK |
| 4001 | 数据库查询错误 | RECODE_DBERR |
| 4002 | 无数据 | RECODE_NODATA |
| 4003 | 数据错误 | RECODE_DATAERR |
| 4101 | 用户未登录 | RECODE_SESSIONERR |
| 4102 | 用户登录失败 | RECODE_LOGINERR |
| 4103 | 参数错误 | RECODE_PARAMERR |
| 4104 | 用户不存在或未激活 | RECODE_USERONERR |
| 4105 | 用户身份错误 | RECODE_ROLEERR |
| 4106 | 密码错误 | RECODE_PWDERR |
| 4107 | 用户已经注册 | RECODE_USERERR |
| 4108 | 手机号错误 | RECODE_MOBILEERR |
| 4201 | 非法请求或请求次数受限 | RECODE_REQERR |
| 4202 | IP受限 | RECODE_IPERR |
| 4301 | 第三方系统错误 | RECODE_THIRDERR |
| 4302 | 文件读写错误 | RECODE_IOERR |
| 4500 | 内部错误 | RECODE_SERVERERR |
| 4501 | 未知错误 | RECODE_UNKNOWERR |

## 📚 学习资料

项目包含 `Learning/` 目录，用于存放 Go 语言学习笔记：

| 文件名 | 学习主题 |
|--------|----------|
| `00_webs.go` | GORM官方文档链接 |
| `01_ORM.go` | ORM概念与使用 |
| `02_GROM.go` | GORM框架特性 |
| `03_MySQL_Linux.go` | Linux下MySQL操作指令 |
| `04_MainAndInit.go` | Go语言特殊函数(main/init) |
| `05_Instraction.go` | 服务启动指令(Redis/Consul/MySQL) |
| `06_redis.go` | Redis命令与连接池配置 |
| `07_CookieAndSeesion.go` | Cookie与Session原理 |
| `08_redis_seesion.go` | Redis Session存储示例 |
| `09_middleware.go` | Gin中间件机制 |

### 学习笔记文档

详细的学习笔记已整理至 [learnings.md](learnings.md)，包含：

- ORM与GORM详解
- MySQL与Redis操作指南
- Go语言特殊函数说明
- Cookie和Session原理
- Gin中间件使用方法
