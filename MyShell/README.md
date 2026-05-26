# 爱家租房 - 本地一键启动

## 前置条件

1. **MySQL** 已启动，库 `search_house`，账号见 `web/conf/config.go`
2. **Redis** 已启动（本机默认 `192.168.81.128:6379`）
3. **Consul** 已安装（脚本可自动 `consul agent -dev`）
4. **Go 1.24+** 已配置，`go mod download` 已完成

## 一键启动

```bash
cd /home/neko/Learning/WorkTools/Go_WorkSapce/src/gin-micro-ihome
chmod +x MyShell/start-all.sh MyShell/stop-all.sh
./MyShell/start-all.sh
```

启动后访问：**http://localhost:8080/home/index.html**

## 一键停止

```bash
./MyShell/stop-all.sh
```

## 日志与 PID

| 路径 | 说明 |
|------|------|
| `logs/getCaptcha.log` | 验证码微服务 |
| `logs/register.log` | 注册微服务 |
| `logs/user.log` | 用户微服务 |
| `logs/house.log` | 房源微服务 |
| `logs/order.log` | 订单微服务 |
| `logs/web.log` | Web 网关 |
| `logs/consul.log` | Consul |
| `logs/pids/*.pid` | 进程号 |

查看某服务日志：

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

## 其他脚本

- `fix-redis-bind.sh`：让 Redis 同时监听 127.0.0.1
- `ScriptHouses.sh`：旧脚本（已过时，请用 `start-all.sh`）
