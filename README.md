# gin-micro-ihome

> 基于 Gin + go-micro 的微服务租房平台（仿 Wellcee）：Web 网关 + 5 个微服务，Consul 服务发现，gRPC 通信，覆盖房源预订与库存一致性完整链路。**Go 后端面试 / 学习项目，非生产系统。**

***

## 📌 项目概述

### 项目是什么、解决什么业务场景

本项目是一个面向 **Go 后端求职面试** 的微服务实战项目，模仿 Wellcee 租房平台，覆盖主链路：**注册登录 → 实名认证 → 房源发布与搜索 → 预订下单 → 日历库存预留 → 支付超时处理 → 房东接单/拒单**。

它解决的核心业务场景是租房平台的 **房源预订与库存一致性问题**——重点演示「下单时同步占房避免超售、支付后异步确认库存、超时/取消自动释放」这条完整闭环，而不只是一个 CRUD 壳子。

### 核心架构模式、通信方式

整体采用 **API 网关 + 多微服务拆分** 架构：

- 服务间通过 **Protocol Buffers + gRPC** 通信，**Consul** 承担服务注册与发现，RPC 默认 5s 超时。
- 下单时 order 服务同步调用 house 占房（`RESERVED`）；支付后经 **Outbox + RabbitMQ** 异步确认（`CONFIRMED`）；超时/取消自动释放（`RELEASED`），幂等靠订单/库存**状态机 + order\_id**。
- MySQL 8 + GORM 持久化，Redis 做 Session / 验证码 / 房源缓存，MinIO 存头像与房源图；网关用 `gin-contrib/sessions`（Redis Store）管理登录态。

### 边界声明：学习 / 面试项目，非生产系统（避坑）

> ⚠️ 为在有限代码量内讲清楚微服务链路，本项目做了有意识的简化，面试时建议主动说明：

1. **MQ 投递语义**：RabbitMQ 为**至少一次**；幂等靠状态机，消费失败按 `x-retry-count` 有限重试（默认 3 次）后写 DLQ。**尚未**开启 Publisher Confirm，**尚未**做 `message_id` 消费去重表。
2. **并发预留**：已有预留行可走 `FOR UPDATE`；但房源尚无预留行时存在间隙锁风险，可能双订（未做日期维度唯一约束）。
3. **共享数据库**：order / house / user 仍共享 `search_house` 库，并非严格的按服务拆库；部分只读接口（如 `/areas`）由网关直连 MySQL。
4. **凭据管理**：MySQL / Redis / MinIO / Session 等配置写在各层 `conf/config.go`，仅适用于本地学习，**不能作为生产范式**。
5. **功能完整性**：MinIO 私有桶预签名、删图同步、分布式锁等生产级能力未实现；支付为模拟支付，短信验证码接口已停用。

## 🧱 系统架构

```text
浏览器 ──► Web 网关 (Gin, :8080)
              │  gRPC / Consul 服务发现
              ├── getCaptcha (:12311)  图形验证码
              ├── register   (:12312)  注册 / 登录（bcrypt）
              ├── user       (:12313)  用户信息 / 实名认证
              ├── house      (:12314)  房源 + 日历预留 + MinIO 上传
              └── order      (:12315)  订单 + Outbox 事件发布

依赖：MySQL（共享库 search_house）· Redis · Consul · RabbitMQ · MinIO
```

**请求流转**：浏览器请求先到达 Gin 网关，公开接口直接转发；需登录接口经 `LoginFilter` 校验 Session 后，由网关通过 Consul 寻址、以 gRPC 调用对应微服务。业务错误透传下游 `errno/errmsg`，传输失败返回标准错误码。

**下单主链路**（同步占房 + 异步确认）：

```text
创建订单 → house.ReserveInventory（RESERVED，约 15 分钟）
     → 支付 → 写 Outbox → MQ(order.paid)
     → house 确认预留 → MQ(inventory.confirmed)
     → 订单进入 WAIT_ACCEPT（房东同意/拒绝）

超时未付 / 取消 → 释放预留（RELEASED）
```

订单状态机：`WAIT_PAY → PAID → WAIT_ACCEPT → WAIT_COMMENT → COMPLETE`，分支状态 `CANCELLED / REJECTED`；预留状态：`RESERVED → CONFIRMED / RELEASED`。

## ✨ 功能清单

### 租客侧

- [x] 注册 / 登录 / 退出（bcrypt，旧明文数据登录成功后自动升级 hash）
- [x] 房源搜索（分页）、首页房源、房源详情（游客可访问）
- [x] 下单（同步预留日历库存，冲突检测）
- [x] 模拟支付、取消订单、评价订单
- [x] 我的订单列表

### 房东侧

- [x] 实名认证 / 查询
- [x] 发布房源、上传房源图（MinIO）
- [x] 我的房源列表
- [x] 客户订单列表、同意接单 / 拒绝订单（校验房东身份）

### 平台通用

- [x] 图形验证码、用户信息修改、头像上传（MinIO）
- [x] 支付超时扫描：约 15 分钟未支付自动取消并释放预留
- [x] Outbox + RabbitMQ：支付/取消后异步确认或释放库存（至少一次 + 状态机幂等 + 有限重试进 DLQ）

### 简化实现 / Mock 模块（面试不踩坑）

- **支付**：模拟支付接口，未对接真实支付网关，无支付回调验签。
- **短信验证码**：接口保留但已停用，返回说明文案；注册登录不依赖真实短信通道。
- **对象存储**：MinIO 返回可访问 URL，私有桶预签名与删图同步未实现。
- **库存并发**：极端并发下无预留行时可能双订，见「边界声明」第 2 条。

## 🧩 技术栈

| 分类     | 技术                                        |
| ------ | ----------------------------------------- |
| 语言     | Go 1.24.4                                 |
| Web 网关 | Gin、gin-contrib/sessions（Redis Store）     |
| 微服务    | go-micro.dev/v5、Consul Registry           |
| RPC    | Protocol Buffers（`.proto` + 生成 `*.pb.go`） |
| 数据库    | MySQL 8 + GORM                            |
| 缓存     | Redis（Session / 验证码 / 房源缓存）               |
| 消息队列   | RabbitMQ（订单事件 / 库存确认）                     |
| 对象存储   | MinIO（S3 兼容）                              |
| 工具链    | Docker Compose、一键启停脚本、bcrypt              |

## 📋 前置依赖

| 依赖               | 推荐版本 / 说明                 |
| ---------------- | ------------------------- |
| Go               | 1.24+                     |
| Docker / Compose | 推荐：一键起中间件                 |
| MySQL            | 8.x，库名 `search_house`     |
| Redis            | Session / 缓存              |
| Consul           | 服务注册发现（必须最先启动）            |
| RabbitMQ         | 订单与库存事件（强烈建议，含管理台 :15672） |
| MinIO            | 图片上传（缺省时脚本可能仅警告）          |

**启动顺序提示**：先启动 Consul 与 MySQL / Redis / RabbitMQ / MinIO 等依赖，再启动 5 个微服务（向 Consul 注册），最后启动 Web 网关。`MyShell/start-all.sh` 已按此顺序编排并做依赖检查。

## 🚀 快速启动

### 方式一：Compose 起依赖 + 脚本起业务（推荐）

```bash
# 1. 克隆代码
git clone <your-repo-url> && cd gin-micro-ihome

# 2. 仅启动中间件
docker compose up -d mysql redis consul rabbitmq minio minio-init

# 3. 编译并启动 5 个微服务 + Web
chmod +x MyShell/*.sh
./MyShell/start-all.sh

# 4（可选）. 写入测试账号 + 演示房源/订单（图进 MinIO）
./MyShell/seed-users.sh
./MyShell/seed-demo.sh
```

首页：**<http://localhost:8080/home/index.html>**

### 方式二：Compose 全栈

```bash
docker compose up -d --build
```

说明：业务容器使用 host 网络以兼容代码里的 `127.0.0.1` 配置；若 Redis 配成局域网 IP，请确保本机可达或改为 `127.0.0.1:6379`。

### 配置与数据库

- 各层配置位于 `conf/config.go` 与 `web/conf/config.go`（学习阶段硬编码）。默认 MySQL：`127.0.0.1:3306`，用户/密码 `neko / neko123456`，库 `search_house`，与 Compose 一致。
- 表结构由各服务 GORM `AutoMigrate` 自动创建（含 `house_reservation`、`local_message` 等），无需手动执行 SQL。
- 可选环境变量：`REDIS_HOST`、`SKIP_BUILD=1`、`REQUIRE_MINIO=1`、`ALLOW_NO_MQ=1`，详见 [MyShell/README.md](MyShell/README.md)。

### 常用脚本

| 命令                        | 说明                                     |
| ------------------------- | -------------------------------------- |
| `./MyShell/start-all.sh`  | 检查依赖 → `go build` 到 `logs/bin/` → 启动服务 |
| `./MyShell/stop-all.sh`   | 停止业务进程                                 |
| `./MyShell/status.sh`     | 查看依赖端口与进程状态                            |
| `./MyShell/seed-users.sh` | 注册测试账号 `13900000001`～`05`，密码 `123456`  |
| `./MyShell/seed-demo.sh`  | 清理 `ut-*`、发 36 套跨区房源（MinIO 图）、订单故事线    |

### 服务验证

```bash
curl -s http://localhost:8080/api/v1.0/areas
curl -s http://localhost:8500/v1/catalog/services
./MyShell/status.sh
```

控制台：Consul UI <http://localhost:8500>；RabbitMQ 管理台 <http://localhost:15672>（guest/guest）；MinIO API <http://127.0.0.1:9000> / 控制台 :9001（minioadmin/minioadmin）。

## 📂 项目目录结构

```text
gin-micro-ihome/
├── conf/                 # 根级配置（部分共用）
├── web/                  # Gin Web 网关（:8080）
│   ├── controller/       # HTTP 接口（参数校验、RPC 转发）
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
│   └── order/            # 订单（含 service / mq / worker，Outbox 落库与投递）
├── MyShell/              # 一键启停 / 状态 / 种测试账号
├── docker-compose.yml    # 本地中间件 + 可选全栈编排
├── Dockerfile            # 按 TARGET 构建 web 或各微服务
├── Learning/             # 独立学习示例（非业务代码）
└── logs/                 # 运行日志、PID、编译产物
```

各微服务常见子目录：`conf/`、`handler/`（RPC 入口）、`model/`（GORM）、`proto/`、`utils/`；house / order 另有 `service/`（业务逻辑）、`mq/`（消息收发）、`worker/`（Outbox 投递、超时扫描等后台任务）。分层约定：**handler 只做协议适配，业务规则集中在 service 层，数据访问集中在 model 层**。

## 💡 设计思路 & 重难点说明

### 服务拆分思路

- **按业务域拆分**：验证码、注册登录、用户、房源、订单各自独立进程与端口，通过 Consul 注册发现。
- **网关统一入口**：会话管理、登录鉴权、参数校验集中在网关，微服务内部不重复处理鉴权。
- **house 与 order 的边界**：库存归属 house，订单状态机归属 order。下单时 order 同步调用 house 占房，支付后经 MQ 异步通知确认——**同步保证不超售，异步保证解耦**，是拆分时最核心的边界决策。
- **坦白的妥协**：各服务共享 `search_house` 库，属于「服务拆分先行、拆库留作演进」的学习阶段决策。

### 关键方案选型理由

- **Consul 而非 Nacos**：go-micro 原生支持，接入成本最低；本项目不需要 Nacos 的配置中心能力，服务发现 + 健康检查刚好够用。
- **gRPC + Protobuf 而非 REST**：强类型契约，`.proto` 即接口文档，改 RPC 时同步重新生成代码，编译期即可发现契约破坏。
- **RabbitMQ 而非 Kafka**：订单事件是业务消息而非日志流，需要路由、重试、死信等 RabbitMQ 原生能力。
- **Session + Redis 而非 JWT**：需要服务端主动失效会话（退出即删 Session），登录态集中在网关校验，Session 更简单直接；JWT 的无状态优势在单网关架构下收益不明显。
- **MinIO 而非云 OSS**：S3 兼容，本地 Docker 一键起，上传/访问链路完整可复现，不依赖云账号。
- **bcrypt 而非 MD5/SHA**：自适应哈希 + 加盐，抗彩虹表，密码存储的业界标准做法。

### 难点 1：下单如何防止超售？

- **问题**：同一房源同一日期可能被多个租客同时下单，纯「查再插」在并发下会双订。
- **方案**：下单时 order **同步 RPC** 调用 `house.ReserveInventory`，在 house 侧事务内做冲突检测 + 写入预留（`RESERVED`，约 15 分钟），已有预留行走 `SELECT ... FOR UPDATE` 行锁串行化。
- **权衡与不足**：已有行可安全加锁，但**房源尚无预留行时间隙锁可能失效**，极端并发下仍可能双订。生产方案应补「房源 + 日期」维度唯一约束兜底，本项目未做。

### 难点 2：支付成功后，如何可靠地通知 house 确认库存？

- **问题**：「订单库提交」和「发 MQ」是两次操作，先提交后发消息，崩溃即丢消息；先发消息再提交，又会出现消息先于事务的脏数据——经典**双写不一致**。
- **方案**：引入 **Outbox 模式**：支付时在同一个本地事务里更新订单状态 + 写入 `local_message` 表，由后台 worker 扫描 Outbox 投递到 RabbitMQ（`order.paid`）；house 消费确认预留后回发 `inventory.confirmed`，推进订单到 `WAIT_ACCEPT`。
- **权衡与不足**：投递语义为**至少一次**，幂等由消费方保证（见难点 3）。未开 Publisher Confirm，极端情况靠 Outbox 重扫兜底。

### 难点 3：MQ 至少一次投递，消费者如何幂等？

- **问题**：网络重试、worker 重投会导致同一消息被消费多次，重复确认/释放库存会破坏状态。
- **方案**：不靠消息去重表，靠**状态机幂等**——消费时以 `order_id` 定位记录，只有当前状态匹配前置状态才执行流转，否则直接 ack 丢弃，重复消息被状态机自然吸收。
- **配套兜底**：消费失败按 `x-retry-count` 有限重试（默认 3 次），仍失败则**显式写入 DLQ**，避免无限重试阻塞队列。
- **权衡与不足**：状态机幂等只覆盖状态流转类消息，通用场景仍建议 `message_id` 去重表。

### 难点 4：待支付订单超时，库存如何自动释放？

- **问题**：租客下单占房后不付款，日历一直被占，需自动取消并释放。
- **方案**：order 服务内置**定时扫描器**，找出超过约 15 分钟仍为 `WAIT_PAY` 的订单执行取消，并经 MQ 通知 house 释放预留（复用取消链路，不另写一套释放逻辑）。
- **权衡与不足**：扫描有分钟级延迟，订单量大时全表扫描有压力，但实现简单、不依赖 MQ 插件。替代方案为 RabbitMQ 延迟消息（TTL + DLX），精度更高但引入额外复杂度。

### 难点 5：历史明文密码如何平滑迁移到 bcrypt？

- **问题**：早期库里有明文密码，直接作废会强制所有用户重置。
- **方案**：登录时先按 bcrypt 校验，失败则兼容比对明文；**明文校验通过的瞬间就地重新计算 bcrypt hash 落库**。用户无感知，随登录逐步完成迁移。
- **亮点**：真实项目常见的「存量数据平滑迁移」思路，比一次性脚本刷库更稳。

### 踩坑记录

- **间隙锁陷阱**：InnoDB 的 `FOR UPDATE` 只能锁住已存在的行，「不存在所以插入」的场景在 RR 隔离级别下有间隙锁竞争，压测双订问题由此暴露——也是难点 1 唯一约束兜底方案的来源。
- **配置硬编码反噬部署**：代码里大量地址写死 `127.0.0.1`，全栈 Compose 时业务容器被迫使用 host 网络；Redis 曾配置局域网 IP，换网络环境后全链路连不上（可用 `MyShell/fix-redis-bind.sh` 修复）。这是后续把配置抽为环境变量的直接动因。

## 📚 API 一览

统一 JSON：`errno`（字符串）、`errmsg`，成功 `errno = "0"`；需登录接口未登录返回会话错误。完整错误码映射见 `web/utils/` 与各服务 `utils/`。

主链路接口（其余接口以代码路由为准）：

| 方法   | 路径                             | 说明               |
| ---- | ------------------------------ | ---------------- |
| POST | `/api/v1.0/users`              | 注册               |
| POST | `/api/v1.0/sessions`           | 登录               |
| GET  | `/api/v1.0/houses`             | 搜索房源（分页，游客可访问）   |
| GET  | `/api/v1.0/houses/:id`         | 房源详情（游客可访问）      |
| POST | `/api/v1.0/houses`             | 发布房源（需登录）        |
| POST | `/api/v1.0/orders`             | 下单，同步预留库存（需登录）   |
| POST | `/api/v1.0/orders/:id/payment` | 模拟支付（需登录）        |
| PUT  | `/api/v1.0/orders/:id/cancel`  | 取消订单（需登录）        |
| PUT  | `/api/v1.0/orders/:id/status`  | 房东同意 / 拒绝（需登录）   |
| GET  | `/api/v1.0/user/orders`        | 我的订单 / 客户订单（需登录） |

## ❓ FAQ / 常见问题

- **Redis 连接失败**：配置里可能是局域网 IP（如 `192.168.81.128:6379`）。改为 `127.0.0.1:6379`（`web/conf/config.go`、`service/house/conf/config.go`），或执行 `MyShell/fix-redis-bind.sh`；也可用环境变量 `REDIS_HOST` 覆盖。
- **Consul 连不上 / 服务发现失败**：确认 Consul 已最先启动，`curl -s http://localhost:8500/v1/catalog/services` 应能看到 5 个服务；手动调试时先 `consul agent -dev` 再启微服务。
- **RabbitMQ 暂不可用**：可用 `ALLOW_NO_MQ=1 ./MyShell/start-all.sh` 跳过强制检查，但订单异步确认/释放链路不工作，仅适合调通页面与同步接口。
- **MinIO 未就绪**：脚本默认可能仅警告，图片上传会失败；`REQUIRE_MINIO=1` 可让启动检查强制要求 MinIO。
- **gRPC 调用超时**：网关 RPC 默认 5s 超时，优先检查目标微服务是否存活（`./MyShell/status.sh`）、Consul 注册是否正常，再查 MySQL/Redis 依赖。
