# 爱家租房 - 本地一键启动

## 前置条件

1. **Go 1.24+** 已安装并在 `PATH` 中
2. **MySQL**（库 `search_house`，账号见 `web/conf/config.go`）
3. **Redis**（默认探测 `192.168.81.128:6379` / `127.0.0.1:6379`）
4. **Consul**（未安装时脚本会尝试 `docker compose` 或本机 `consul agent -dev`）
5. **RabbitMQ**（支付确认 / 库存释放；可用 docker compose）
6. **MinIO**（头像与房源图；可用 docker compose）

无本机中间件时，可先只起依赖：

```bash
docker compose up -d mysql redis consul rabbitmq minio minio-init
```

## 一键启动

```bash
cd /path/to/gin-micro-ihome
chmod +x MyShell/*.sh
./MyShell/start-all.sh
```

脚本会：检查依赖 → `go build` 到 `logs/bin/` → 启动 5 个微服务 + Web → HTTP 抽检。

启动后访问：**http://localhost:8080/home/index.html**

### 常用环境变量

| 变量 | 含义 |
|------|------|
| `REDIS_HOST` | 指定 Redis 地址 |
| `SKIP_BUILD=1` | 跳过编译，复用 `logs/bin` |
| `REQUIRE_MINIO=1` | MinIO 不可用时退出（默认仅警告） |
| `ALLOW_NO_MQ=1` | RabbitMQ 不可用时仅警告（不推荐） |

## 一键停止 / 状态 / 演示数据

```bash
./MyShell/stop-all.sh
./MyShell/status.sh
./MyShell/seed-users.sh   # 5 个已实名账号，密码 123456
./MyShell/seed-demo.sh    # 演示房源 + MinIO 图片 + 订单（幂等，需 MinIO）
```

### 测试账号角色

| 手机号 | 角色 | 密码 |
|--------|------|------|
| `13900000001` 张三 | 房东 A | `123456` |
| `13900000002` 李四 | 房东 B | `123456` |
| `13900000003`～`05` | 房客 | `123456` |

`seed-demo.sh` 会：清理 `ut-*` 单测残留 → 发布约 36 套 `demo-*` 房源（12 区 × 经济/舒适/精品）→ 每套上传 3 张 MinIO 图（素材在 `seed-assets/`）→ 将仍使用 `./images/` 的旧房源主图迁到 MinIO → 跑下单/支付/接单真链路，并补 SQL 状态样例。

可选环境变量：`CLEAN_UT=0`（不清单测数据）、`SKIP_ORDERS=1`（只灌房源）。

## 日志与 PID

| 路径 | 说明 |
|------|------|
| `logs/<服务>.log` | 运行日志 |
| `logs/pids/*.pid` | 进程号 |
| `logs/bin/*` | 编译出的可执行文件 |

```bash
tail -f logs/house.log
```

## 服务端口

| 服务 | 地址 |
|------|------|
| Web | http://127.0.0.1:8080 |
| getCaptcha | 127.0.0.1:12311 |
| register | 127.0.0.1:12312 |
| user | 127.0.0.1:12313 |
| house | 127.0.0.1:12314 |
| order | 127.0.0.1:12315 |
| Consul | http://127.0.0.1:8500 |
| RabbitMQ | amqp://127.0.0.1:5672 管理台 :15672 |
| MinIO | http://127.0.0.1:9000 控制台 :9001 |

## 其他脚本

- `fix-redis-bind.sh`：让 Redis 同时监听 127.0.0.1 与局域网 IP
- `ScriptHouses.sh`：已弃用，转发到 `start-all.sh`
