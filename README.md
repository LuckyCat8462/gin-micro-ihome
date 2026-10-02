# gin-micro-ihome

基于 Gin + go-micro 的爱家租房微服务项目：Web 网关 + 5 个独立微服务，Consul 服务发现，Protobuf RPC；支持房源发布/搜索、日历库存预留、订单支付超时、MinIO 存图。

---

## 项目简介

在线租房学习项目，覆盖注册登录、实名认证、房源管理、预订下单与房东接单等主流程。

**架构口径（面试请对齐）：** 本仓库是 **Gin + go-micro 的服务化拆分（独立进程 + Consul RPC）**；**数据层仍为共享 MySQL 库 `search_house`，学习阶段故意未按域拆库**。谈「微服务」时请说明是进程/RPC 边界，而非已完成库表归属与独立部署。

相对早期版本，当前重点补齐了：

- **日历库存预留**：下单同步占房，避免超售
- **Outbox + RabbitMQ**：支付/取消后异步确认或释放库存（至少一次 + 状态机幂等）
- **支付超时**：未支付订单自动取消并释放预留
- **MinIO 对象存储**：头像与房源图落桶，接口返回可访问 URL
- **Docker Compose / 一键脚本**：本地依赖与服务可快速拉起

## 技术栈

| 分类 | 技术 |
|------|------|
| 语言 | Go 1.24.4 |
| Web 网关 | Gin、gin-contrib/sessions（Redis Store） |
| 微服务 | go-micro.dev/v5、Consul Registry |
| RPC | Protocol Buffers（`.proto` + 生成 `*.pb.go`） |
| 数据库 | MySQL 8 + GORM |
| 缓存 | Redis（Session / 验证码 / 房源缓存） |
| 消息队列 | RabbitMQ（订单事件 / 库存确认） |
| 对象存储 | MinIO（S3 兼容） |

## 架构概览

```text
浏览器 ──► Web 网关 (:8080)
              │  RPC / Consul
              ├── getCaptcha (:12311)  图形验证码
              ├── register   (:12312)  注册 / 登录（bcrypt）
              ├── user       (:12313)  用户信息 / 实名
              ├── house      (:12314)  房源 + 日历预留 + MinIO 上传
              └── order      (:12315)  订单 + Outbox 发布

依赖：MySQL（共享库 search_house）· Redis · Consul · RabbitMQ · MinIO
```

下单主链路（简化）：

```text
创建订单 → house.ReserveInventory（RESERVED，约 15 分钟）
     → 支付 → 写 Outbox → MQ(order.paid)
     → house 确认预留 → MQ(inventory.confirmed)
     → 订单进入 WAIT_ACCEPT（房东同意/拒绝）

超时未付 / 取消 → 释放预留（RELEASED）
```

## 功能状态与已知限制

演示路径（应能走通）：注册 → 登录 → 搜房/详情 → 下单预留 → 支付 → 房东接单 / 取消或超时释放。

| 能力 | 状态 | 说明 |
|------|------|------|
| 注册登录（bcrypt） | ✅ | 明文旧数据登录成功后自动升级 hash |
| 房源发布 / 搜索分页 / 详情 | ✅ | 搜索、详情游客可访问 |
| 日历预留 + 冲突检测 | ✅ | 同步 RPC；见下方已知限制 |
| 支付 / 取消 / 评价 / 接单拒单 | ✅ | 模拟支付；拒单校验房东身份 |
| Outbox + RabbitMQ 确认/释放 | ✅ | 最小闭环已具备；有限重试进 DLQ 已做 |
| 支付超时扫描 | ✅ | 默认约 15 分钟 |
| MinIO 头像 / 房源图 | ✅ | 私有桶预签名、删图同步等未做 |
| 短信验证码 | ❌ | 接口已返回停用说明，演示请用图片验证码 |
| Publisher Confirm / `message_id` 去重表 | ❌ | 见已知限制与 `docs/rabbitMQ.md` |
| 按域拆库 / 独立 proto 发布 | ❌ | 学习阶段不做；演进方向可口述 |
| CI / 结构化日志 / health | ⏳ | 有余力再补 |

**已知限制（面试常问，请主动说清）：**

1. **MQ**：投递语义是 **至少一次**；幂等靠订单/库存 **状态机（`order_id`）**；消费失败按 `x-retry-count` 有限重试（默认 3 次）后**显式写入 DLQ**。**尚未**开 Publisher Confirm，**尚未**做 `message_id` 消费去重表。
2. **并发预留**：已有行可 `FOR UPDATE`；**该房源尚无预留行时**存在 gap 锁风险，可能双订（未做日期维度唯一约束）。
3. **网关**：RPC 默认 **5s 超时**，传输失败返回标准 `errno`；业务错误透传下游 `errno/errmsg`。
4. **共享库**：order / house / user 等仍共享 `search_house`，各服务 `AutoMigrate` 范围较宽；order 本地 `replace` house 模块。
5. **凭据**：MySQL / MinIO / Session 等仍写在各层 `conf/config.go`（本地学习用，勿当生产范式）。

更细的 MQ 对照表见本地文档 `docs/rabbitMQ.md`（默认 gitignore，不入库）。

## 目录结构

```text
gin-micro-ihome/
├── conf/                 # 根级配置（部分共用）
├── web/                  # Gin Web 网关（:8080）
│   ├── controller/       # HTTP 接口
│   ├── model/            # MySQL / Redis
│   ├── conf/             # Web 配置（MySQL / Redis / MinIO / Session）
│   ├── proto/            # 微服务客户端 stub
│   ├── storage/          # MinIO 客户端（头像上传）
│   ├── utils/            # 错误码等
│   └── view/             # 静态前端 HTML/CSS/JS
├── service/
│   ├── getCaptcha/       # 图形验证码
│   ├── register/         # 注册登录
│   ├── user/             # 用户 / 实名
│   ├── house/            # 房源（含 service / mq / worker / storage）
│   └── order/            # 订单（含 service / mq / worker / outbox）
├── MyShell/              # 一键启停 / 状态 / 种测试账号
├── docker-compose.yml    # 本地中间件 + 可选全栈编排
├── Dockerfile            # 按 TARGET 构建 web 或各微服务
├── docs/                 # 本地学习/设计文档（默认 gitignore）
├── Learning/             # 独立学习示例（非业务代码）
└── logs/                 # 运行日志、PID、编译产物
```

各微服务常见子目录：`conf/`、`handler/`、`model/`、`proto/`、`utils/`；house / order 另有 `service/`、`mq/`、`worker/`。

## 快速开始

### 环境要求

| 依赖 | 说明 |
|------|------|
| Go 1.24+ | 编译与运行 |
| Docker / Compose | 推荐：一键起中间件 |
| MySQL 8 | 库名 `search_house` |
| Redis | Session / 缓存 |
| Consul | 服务注册发现 |
| RabbitMQ | 订单与库存事件（强烈建议） |
| MinIO | 图片上传（缺省时脚本可能仅警告） |

### 方式一：Compose 起依赖 + 脚本起业务（推荐）

```bash
# 1. 仅启动中间件
docker compose up -d mysql redis consul rabbitmq minio minio-init

# 2. 编译并启动 5 个微服务 + Web
chmod +x MyShell/*.sh
./MyShell/start-all.sh

# 3（可选）. 写入测试账号 + 演示房源/订单（图进 MinIO）
./MyShell/seed-users.sh
./MyShell/seed-demo.sh
```

首页：**http://localhost:8080/home/index.html**

常用脚本：

| 命令 | 说明 |
|------|------|
| `./MyShell/start-all.sh` | 检查依赖 → `go build` 到 `logs/bin/` → 启动服务 |
| `./MyShell/stop-all.sh` | 停止业务进程 |
| `./MyShell/status.sh` | 查看依赖端口与进程状态 |
| `./MyShell/seed-users.sh` | 注册测试账号 `13900000001`～`05`，密码 `123456` |
| `./MyShell/seed-demo.sh` | 清理 `ut-*`、发 36 套跨区房源（MinIO 图）、订单故事线 |

环境变量（可选）：`REDIS_HOST`、`SKIP_BUILD=1`、`REQUIRE_MINIO=1`、`ALLOW_NO_MQ=1`。更多说明见 [MyShell/README.md](MyShell/README.md)。

### 方式二：Compose 全栈

```bash
docker compose up -d --build
```

说明：业务容器使用 host 网络以兼容代码里大量 `127.0.0.1` 配置；若 Redis 配成局域网 IP，请确保本机可达，或改为 `127.0.0.1:6379`。

### 方式三：手动启动（调试）

分别在独立终端执行：

```bash
consul agent -dev
cd service/getCaptcha && go run .
cd service/register   && go run .
cd service/user       && go run .
cd service/house      && go run .
cd service/order      && go run .
cd web                && go run .
```

需自行保证 MySQL / Redis / RabbitMQ / MinIO 可用。

### 数据库与配置

默认账号与 Compose 一致（以 `web/conf/config.go` 为准）：

| 项 | 值 |
|----|-----|
| 库名 | `search_house` |
| 用户 / 密码 | `neko` / `neko123456` |
| 地址 | `127.0.0.1:3306` |

表结构由各服务 GORM `AutoMigrate` 自动创建（含 `house_reservation`、`local_message` 等）。若本机已有 MySQL，也可手动建库后直接跑服务。

Redis 默认配置可能是局域网地址（如 `192.168.81.128:6379`）；不可达时改 `web/conf/config.go` / `service/house/conf/config.go`，或执行 `MyShell/fix-redis-bind.sh`。

### 服务与控制台地址

| 服务 | 地址 |
|------|------|
| Web 首页 | http://localhost:8080/home/index.html |
| API 前缀 | http://localhost:8080/api/v1.0 |
| Consul UI | http://localhost:8500 |
| RabbitMQ 管理台 | http://localhost:15672（guest/guest） |
| MinIO API / 控制台 | http://127.0.0.1:9000 / :9001（minioadmin/minioadmin） |
| getCaptcha / register / user / house / order | `:12311`～`:12315` |

验证：

```bash
curl -s http://localhost:8080/api/v1.0/areas
curl -s http://localhost:8500/v1/catalog/services
./MyShell/status.sh
```

## API 一览

统一 JSON：`errno`（字符串）、`errmsg`；成功 `errno = "0"`。需登录接口在 `LoginFilter` 之后注册；未登录返回会话错误。

### 公开接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1.0/session` | 当前登录态 |
| GET | `/api/v1.0/imagecode/:uuid` | 图形验证码 |
| GET | `/api/v1.0/smscode/:phone` | 短信验证码（已停用，返回说明） |
| POST | `/api/v1.0/users` | 注册 |
| POST | `/api/v1.0/sessions` | 登录 |
| GET | `/api/v1.0/areas` | 城区列表 |
| GET | `/api/v1.0/house/index` | 首页房源 |
| GET | `/api/v1.0/houses` | 搜索房源（支持分页等） |
| GET | `/api/v1.0/houses/:id` | 房源详情 |

### 需登录

| 方法 | 路径 | 说明 |
|------|------|------|
| DELETE | `/api/v1.0/session` | 退出 |
| GET | `/api/v1.0/user` | 用户信息 |
| PUT | `/api/v1.0/user/name` | 修改用户名 |
| POST | `/api/v1.0/user/avatar` | 上传头像（MinIO） |
| POST / GET | `/api/v1.0/user/auth` | 实名认证 / 查询 |
| GET | `/api/v1.0/user/houses` | 我的房源 |
| POST | `/api/v1.0/houses` | 发布房源 |
| POST | `/api/v1.0/houses/:id/images` | 上传房源图（MinIO） |
| POST | `/api/v1.0/orders` | 下单（同步预留库存） |
| POST | `/api/v1.0/orders/:id/payment` | 模拟支付 |
| PUT | `/api/v1.0/orders/:id/cancel` | 取消订单 |
| PUT | `/api/v1.0/orders/:id/comment` | 评价订单 |
| GET | `/api/v1.0/user/orders` | 我的订单 / 客户订单 |
| PUT | `/api/v1.0/orders/:id/status` | 房东同意 / 拒绝 |

## 订单状态（摘要）

| 状态 | 含义 |
|------|------|
| `WAIT_PAY` | 待支付（已预留日历，默认约 15 分钟超时） |
| `PAID` | 已支付（等待库存确认推进） |
| `WAIT_ACCEPT` | 待房东接单 |
| `WAIT_COMMENT` | 待评价 |
| `COMPLETE` | 完成 |
| `CANCELLED` / `REJECTED` | 取消 / 拒绝 |

预留状态：`RESERVED` → `CONFIRMED` 或 `RELEASED`。

## 错误码

| 错误码 | 说明 |
|--------|------|
| 0 | 成功 |
| 4001 | 数据库错误 |
| 4002 | 无数据 |
| 4003 | 数据错误 |
| 4101 | 用户未登录 |
| 4102 | 登录失败 |
| 4103 | 参数错误 |
| 4104 | 用户不存在或未激活 |
| 4105 | 用户身份错误 |
| 4106 | 密码错误 |
| 4107 | 用户已注册 |
| 4108 | 手机号错误 |
| 4201 | 非法请求或频率受限 |
| 4202 | IP 受限 |
| 4301 | 第三方系统错误 |
| 4302 | 文件读写错误 |
| 4500 | 内部错误 |
| 4501 | 未知错误 |

完整映射见 `web/utils/` 与各服务 `utils/`。

## 相关文档

| 文档 | 说明 |
|------|------|
| [MyShell/README.md](MyShell/README.md) | 一键启停、环境变量、端口 |
| `docs/对象存储与图片上传.md` | MinIO 上传与展示链路（本地 docs，默认不入库） |
| `docs/rabbitMQ.md` | 订单事件与库存消息说明（本地 docs） |
| `docker-compose.yml` | 中间件与全栈编排注释 |

## 开发约定（简）

- Web 以 RPC 调微服务为主；部分接口（如 `/areas`）可直接访问 MySQL/Redis（网关直连只读例外，非拆库完成）
- 新业务优先走 service 层（house / order 已有）；register / user 仍为 handler→model，逐步迁移即可
- 改 RPC 时同步更新 `service/*/proto/*.proto` 与 `web/proto/` 生成代码
- 配置写在各层 `conf/config.go`（学习阶段）；生产应改为环境变量 / 本地未提交配置
- 密码使用 bcrypt；旧明文库登录成功后会自动升级为 hash
- `Learning/` 为实验代码，业务改动以 `web/` 与 `service/` 为准

## License

个人 / 学习用途项目。
