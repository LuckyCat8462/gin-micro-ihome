#!/bin/bash
# 一键启动：依赖检查 + 编译二进制 + Consul/微服务/Web
# 用法: ./MyShell/start-all.sh
# 环境变量（可选，默认值集中在 MyShell/env.sh，本机差异写 MyShell/local.env）:
#   REDIS_CANDIDATES / REDIS_HOST / REDIS_PORT
#   MYSQL_HOST / MYSQL_USER / MYSQL_PWD / MYSQL_DB
#   GO_BIN                    go 不在 PATH 时指定可执行文件路径
#   SKIP_BUILD=1              跳过 go build（使用已有 logs/bin）
#   REQUIRE_MINIO=1           MinIO 不可用时退出（默认仅警告）
#   ALLOW_NO_MQ=1             RabbitMQ 不可用时仅警告不退出
# 停止: ./MyShell/stop-all.sh

set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
LOG_DIR="$PROJECT_ROOT/logs"
PID_DIR="$LOG_DIR/pids"
BIN_DIR="$LOG_DIR/bin"

# 机器差异配置唯一入口（含本机覆盖 local.env）
# shellcheck disable=SC1091
source "$PROJECT_ROOT/MyShell/env.sh"

mkdir -p "$LOG_DIR" "$PID_DIR" "$BIN_DIR"

log() { echo "[$(date '+%H:%M:%S')] $*"; }

preflight() {
  local hard_fail=0
  command -v curl >/dev/null 2>&1 || { log "缺少 curl: sudo apt install curl"; hard_fail=1; }
  command -v python3 >/dev/null 2>&1 || log "提示: 缺 python3（seed-demo.sh 需要）: sudo apt install python3"
  if command -v docker >/dev/null 2>&1; then
    if docker compose version >/dev/null 2>&1; then
      log "docker compose OK"
    else
      log "提示: docker compose v2 插件缺失: sudo apt install docker-compose-plugin"
    fi
  else
    log "提示: 未安装 docker。依赖可由 docker compose 兜底，建议安装: curl -fsSL https://get.docker.com | sudo sh"
  fi
  if [[ "$hard_fail" -eq 1 ]]; then
    log "错误: 体检未通过，请安装缺失项后重试"
    exit 1
  fi
}

detect_go() {
  if [[ -n "$GO_BIN" ]]; then
    [[ -x "$GO_BIN" ]] && { echo "$GO_BIN"; return 0; }
    log "错误: GO_BIN=$GO_BIN 不可执行"
    return 1
  fi
  if command -v go >/dev/null 2>&1; then
    command -v go
    return 0
  fi
  local p
  for p in /usr/local/go/bin "$HOME/go/bin" "$HOME/.local/go/bin" /usr/lib/go/bin /opt/homebrew/bin /snap/bin; do
    if [[ -x "$p/go" ]]; then
      echo "$p/go"
      return 0
    fi
  done
  return 1
}

ensure_go() {
  local go_bin
  if ! go_bin="$(detect_go)"; then
    log "错误: 未找到 go。安装 Go 1.24+ 后重试，或指定: GO_BIN=/path/to/go ./MyShell/start-all.sh"
    exit 1
  fi
  export PATH="$(dirname "$go_bin"):$PATH"
  log "Go: $go_bin ($(go version 2>/dev/null | awk '{print $3}'))"
}

is_running() {
  local pidfile="$1"
  [[ -f "$pidfile" ]] || return 1
  local pid
  pid="$(cat "$pidfile" 2>/dev/null || true)"
  [[ -n "$pid" ]] || return 1
  kill -0 "$pid" 2>/dev/null
}

port_open() {
  local host="$1" port="$2"
  (echo >/dev/tcp/"$host"/"$port") >/dev/null 2>&1
}

wait_port() {
  local name="$1" host="$2" port="$3" seconds="${4:-30}"
  local i
  for ((i = 1; i <= seconds; i++)); do
    if port_open "$host" "$port"; then
      log "$name 端口 $host:$port 就绪"
      return 0
    fi
    sleep 1
  done
  log "错误: 等待 $name ($host:$port) 超时 ${seconds}s"
  return 1
}

http_ok() {
  local url="$1"
  curl -sf --max-time 3 "$url" >/dev/null 2>&1
}

resolve_redis_host() {
  local cli_ok=0
  if command -v redis-cli >/dev/null 2>&1; then
    cli_ok=1
  fi

  try_host() {
    local h="$1"
    if [[ $cli_ok -eq 1 ]]; then
      if redis-cli -h "$h" -p "$REDIS_PORT" ping 2>/dev/null | grep -q PONG; then
        return 0
      fi
    fi
    if port_open "$h" "$REDIS_PORT"; then
      return 0
    fi
    return 1
  }

  if [[ -n "${REDIS_HOST:-}" ]]; then
    if try_host "$REDIS_HOST"; then
      echo "$REDIS_HOST"
      return 0
    fi
    log "错误: REDIS_HOST=$REDIS_HOST 不可用"
    return 1
  fi
  local h
  for h in $REDIS_CANDIDATES; do
    if try_host "$h"; then
      echo "$h"
      return 0
    fi
  done
  return 1
}

try_compose_deps() {
  local services=("$@")
  if ! command -v docker >/dev/null 2>&1; then
    return 1
  fi
  if ! docker info >/dev/null 2>&1; then
    return 1
  fi
  log "尝试 docker compose 启动依赖: ${services[*]}"
  (
    cd "$PROJECT_ROOT"
    docker compose up -d "${services[@]}"
  ) || return 1
  return 0
}

check_mysql() {
  if command -v mysqladmin >/dev/null 2>&1; then
    if mysqladmin ping -h"$MYSQL_HOST" -P"$MYSQL_PORT" -u"$MYSQL_USER" -p"$MYSQL_PWD" --silent 2>/dev/null; then
      log "MySQL OK ($MYSQL_HOST:$MYSQL_PORT/$MYSQL_DB)"
      return 0
    fi
  fi
  if command -v mysql >/dev/null 2>&1; then
    if mysql -h"$MYSQL_HOST" -P"$MYSQL_PORT" -u"$MYSQL_USER" -p"$MYSQL_PWD" -e "USE $MYSQL_DB; SELECT 1;" >/dev/null 2>&1; then
      log "MySQL OK ($MYSQL_HOST:$MYSQL_PORT/$MYSQL_DB)"
      return 0
    fi
  fi
  if port_open "$MYSQL_HOST" "$MYSQL_PORT"; then
    log "MySQL 端口可达，跳过账号校验（未安装 mysql 客户端）"
    return 0
  fi
  log "尝试用 docker compose 启动 MySQL..."
  if try_compose_deps mysql; then
    sleep 5
    if port_open "$MYSQL_HOST" "$MYSQL_PORT"; then
      log "MySQL OK (docker)"
      return 0
    fi
  fi
  log "错误: MySQL 不可用 ($MYSQL_HOST:$MYSQL_PORT)，库名 $MYSQL_DB"
  exit 1
}

check_redis() {
  local host
  if ! host="$(resolve_redis_host)"; then
    log "尝试用 docker compose 启动 Redis..."
    if try_compose_deps redis; then
      sleep 2
      if host="$(resolve_redis_host)"; then
        :
      else
        log "错误: Redis 不可用"
        exit 1
      fi
    else
      log "错误: Redis 不可用。可执行: sudo systemctl start redis-server"
      log "      或: docker compose up -d redis"
      log "      测试: redis-cli -h 127.0.0.1 -p $REDIS_PORT ping"
      exit 1
    fi
  fi
  REDIS_HOST_RESOLVED="$host"
  export REDIS_ADDR="$host:$REDIS_PORT"
  log "Redis OK ($REDIS_HOST_RESOLVED:$REDIS_PORT)"
  if [[ "$REDIS_HOST_RESOLVED" != "127.0.0.1" ]]; then
    log "提示: Redis 在 $REDIS_HOST_RESOLVED:$REDIS_PORT（已导出 REDIS_ADDR，Go conf 参数化后可直接继承）"
  fi
}

check_rabbitmq() {
  if port_open "$AMQP_HOST" "$AMQP_PORT"; then
    log "RabbitMQ OK ($AMQP_HOST:$AMQP_PORT)"
    return 0
  fi
  if command -v rabbitmqctl >/dev/null 2>&1; then
    log "尝试启动本机 RabbitMQ..."
    sudo rabbitmq-server -detached >/dev/null 2>&1 || sudo systemctl start rabbitmq-server >/dev/null 2>&1 || true
    sleep 3
    if port_open "$AMQP_HOST" "$AMQP_PORT"; then
      log "RabbitMQ 启动成功"
      return 0
    fi
  fi
  log "尝试 docker compose 启动 RabbitMQ..."
  if try_compose_deps rabbitmq; then
    local i
    for ((i = 1; i <= 30; i++)); do
      if port_open "$AMQP_HOST" "$AMQP_PORT"; then
        log "RabbitMQ OK (docker)"
        return 0
      fi
      sleep 1
    done
  fi
  if [[ "${ALLOW_NO_MQ:-0}" == "1" ]]; then
    log "警告: RabbitMQ 不可用（ALLOW_NO_MQ=1），支付确认/库存释放将异常"
    return 0
  fi
  log "错误: RabbitMQ 不可用。请: docker compose up -d rabbitmq"
  log "      或 ALLOW_NO_MQ=1 ./MyShell/start-all.sh 跳过（不推荐）"
  exit 1
}

check_minio() {
  if http_ok "$MINIO_API/minio/health/live" || http_ok "$MINIO_API/minio/health/ready"; then
    log "MinIO OK ($MINIO_API)"
    return 0
  fi
  if port_open 127.0.0.1 9000; then
    log "MinIO 端口 9000 可达"
    return 0
  fi
  log "尝试 docker compose 启动 MinIO..."
  if try_compose_deps minio minio-init; then
    local i
    for ((i = 1; i <= 40; i++)); do
      if http_ok "$MINIO_API/minio/health/live" || port_open 127.0.0.1 9000; then
        log "MinIO OK (docker)"
        return 0
      fi
      sleep 1
    done
  fi

  # 无 Docker 时下载官方二进制本地拉起（仅开发用）
  if start_minio_binary; then
    return 0
  fi

  if [[ "${REQUIRE_MINIO:-0}" == "1" ]]; then
    log "错误: MinIO 不可用（REQUIRE_MINIO=1）。请安装 docker 后: docker compose up -d minio minio-init"
    exit 1
  fi
  log "警告: MinIO 不可用，图片上传将失败；核心业务仍可启动"
  log "      安装后可: docker compose up -d minio minio-init"
  return 0
}

start_minio_binary() {
  local bin="$BIN_DIR/minio"
  local data="$LOG_DIR/minio-data"
  local pidfile="$PID_DIR/minio.pid"
  mkdir -p "$data"

  if [[ ! -x "$bin" ]]; then
    log "下载 MinIO 二进制到 logs/bin/minio ..."
    if ! curl -fsSL -o "$bin" "https://dl.min.io/server/minio/release/linux-amd64/minio"; then
      log "下载 MinIO 失败"
      return 1
    fi
    chmod +x "$bin"
  fi

  if is_running "$pidfile"; then
    log "MinIO 已在运行 (PID $(cat "$pidfile"))"
    return 0
  fi

  log "启动本地 MinIO 二进制..."
  export MINIO_ROOT_USER=minioadmin
  export MINIO_ROOT_PASSWORD=minioadmin
  nohup "$bin" server "$data" --address ":9000" --console-address ":9001" \
    >>"$LOG_DIR/minio.log" 2>&1 &
  echo $! >"$pidfile"
  local i
  for ((i = 1; i <= 30; i++)); do
    if http_ok "$MINIO_API/minio/health/live" || port_open 127.0.0.1 9000; then
      log "MinIO OK (local binary, 控制台 :9001)"
      # 尽量创建 bucket（可选，失败不阻断）
      if command -v mc >/dev/null 2>&1; then
        mc alias set ihome local http://127.0.0.1:9000 minioadmin minioadmin >/dev/null 2>&1 || true
        mc mb -p ihome/ihome >/dev/null 2>&1 || true
        mc anonymous set download ihome/ihome >/dev/null 2>&1 || true
      fi
      return 0
    fi
    sleep 1
  done
  log "本地 MinIO 启动失败，见 logs/minio.log"
  return 1
}

check_consul() {
  if http_ok "$CONSUL_HTTP/v1/status/leader"; then
    log "Consul 已在运行"
    return 0
  fi
  if command -v consul >/dev/null 2>&1; then
    log "启动 Consul (agent -dev)..."
    nohup consul agent -dev >>"$LOG_DIR/consul.log" 2>&1 &
    echo $! >"$PID_DIR/consul.pid"
    wait_port "Consul" 127.0.0.1 8500 20
    if http_ok "$CONSUL_HTTP/v1/status/leader"; then
      log "Consul 启动成功"
      return 0
    fi
  fi
  log "尝试 docker compose 启动 Consul..."
  if try_compose_deps consul; then
    wait_port "Consul" 127.0.0.1 8500 30
    if http_ok "$CONSUL_HTTP/v1/status/leader"; then
      log "Consul OK (docker)"
      return 0
    fi
  fi
  log "错误: Consul 不可用"
  exit 1
}

build_all() {
  ensure_go
  if [[ "${SKIP_BUILD:-0}" == "1" ]]; then
    log "SKIP_BUILD=1，跳过编译"
    local missing=0
    local b
    for b in getCaptcha register user house order web; do
      if [[ ! -x "$BIN_DIR/$b" ]]; then
        log "缺少二进制: $BIN_DIR/$b"
        missing=1
      fi
    done
    if [[ "$missing" -eq 1 ]]; then
      log "错误: 二进制不完整，请去掉 SKIP_BUILD 重新启动"
      exit 1
    fi
    return 0
  fi

  log "编译微服务与 Web（输出到 logs/bin）..."
  (
    cd "$PROJECT_ROOT/service/getCaptcha" && go build -o "$BIN_DIR/getCaptcha" .
  )
  (
    cd "$PROJECT_ROOT/service/register" && go build -o "$BIN_DIR/register" .
  )
  (
    cd "$PROJECT_ROOT/service/user" && go build -o "$BIN_DIR/user" .
  )
  (
    cd "$PROJECT_ROOT/service/house" && go build -o "$BIN_DIR/house" .
  )
  (
    cd "$PROJECT_ROOT/service/order" && go build -o "$BIN_DIR/order" .
  )
  (
    cd "$PROJECT_ROOT" && go build -o "$BIN_DIR/web" ./web
  )
  log "编译完成"
}

start_bin() {
  local name="$1"
  local port="$2"
  local pidfile="$PID_DIR/${name}.pid"
  local bin="$BIN_DIR/$name"

  if [[ ! -x "$bin" ]]; then
    log "错误: 找不到可执行文件 $bin"
    exit 1
  fi

  if is_running "$pidfile"; then
    log "$name 已在运行 (PID $(cat "$pidfile"))，先停止再拉起以保证新二进制"
    local old
    old="$(cat "$pidfile")"
    kill "$old" 2>/dev/null || true
    sleep 1
    kill -9 "$old" 2>/dev/null || true
    rm -f "$pidfile"
  fi

  # 端口被非本脚本进程占用时提示
  if port_open 127.0.0.1 "$port"; then
    log "警告: 端口 $port 已被占用，尝试继续启动 $name（若失败请先 stop-all）"
  fi

  log "启动 $name ..."
  nohup "$bin" >>"$LOG_DIR/${name}.log" 2>&1 &
  echo $! >"$pidfile"
  wait_port "$name" 127.0.0.1 "$port" 25
  if is_running "$pidfile"; then
    log "$name 已启动 (PID $(cat "$pidfile"))，日志: logs/${name}.log"
  else
    log "错误: $name 启动失败，查看 logs/${name}.log"
    exit 1
  fi
}

verify_stack() {
  log "---------- 联通性抽检 ----------"
  local ok=1
  if http_ok "http://127.0.0.1:8080/api/v1.0/session"; then
    log "Web API OK (/api/v1.0/session)"
  else
    log "错误: Web API 无响应"
    ok=0
  fi
  if http_ok "http://127.0.0.1:8080/api/v1.0/house/index"; then
    log "首页房源接口 OK"
  else
    log "警告: /api/v1.0/house/index 未成功（可能暂无数据或 house 未注册完）"
  fi
  if http_ok "http://127.0.0.1:8080/api/v1.0/areas"; then
    log "城区接口 OK"
  else
    log "警告: /api/v1.0/areas 未成功"
  fi
  if [[ "$ok" -ne 1 ]]; then
    exit 1
  fi
}

ensure_areas() {
  local n
  n="$(mysql_q "SELECT COUNT(*) FROM area" || echo 0)"
  if [[ "${n:-0}" -eq 0 ]]; then
    log "警告: area 表为空（城区下拉/发房/seed-demo 都会失败）"
    log "      请执行: ./MyShell/seed-area.sh   （或导入旧机器 dump: ./MyShell/seed-area.sh dump.sql）"
  fi
}

cd "$PROJECT_ROOT"
log "项目目录: $PROJECT_ROOT"
preflight
log "---------- 检查依赖 ----------"
check_mysql
check_redis
check_consul
check_rabbitmq
check_minio
log "---------- 编译 ----------"
build_all
log "---------- 启动微服务 ----------"
start_bin "getCaptcha" 12311
start_bin "register" 12312
start_bin "user" 12313
start_bin "house" 12314
start_bin "order" 12315
log "---------- 启动 Web ----------"
start_bin "web" 8080
# 给 Consul 注册一点时间
sleep 2
verify_stack
ensure_areas
log "---------- 全部完成 ----------"
log "首页: http://localhost:8080/home/index.html"
log "Consul: $CONSUL_HTTP"
log "MinIO:  http://127.0.0.1:9000  控制台 :9001 (minioadmin/minioadmin)"
log "停止: $PROJECT_ROOT/MyShell/stop-all.sh"
log "状态: $PROJECT_ROOT/MyShell/status.sh"
log "日志: tail -f $LOG_DIR/house.log"
