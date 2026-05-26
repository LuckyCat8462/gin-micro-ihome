#!/bin/bash
# 一键启动：Consul + 5 个微服务 + Web 网关
# 用法: ./MyShell/start-all.sh
# 停止: ./MyShell/stop-all.sh

set -e

PROJECT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
LOG_DIR="$PROJECT_ROOT/logs"
PID_DIR="$LOG_DIR/pids"
BIN_DIR="$LOG_DIR/bin"
REDIS_HOST="192.168.81.128"
REDIS_PORT="6379"
CONSUL_HTTP="http://127.0.0.1:8500"

mkdir -p "$LOG_DIR" "$PID_DIR" "$BIN_DIR"

log() { echo "[$(date '+%H:%M:%S')] $*"; }

is_running() {
  local pidfile="$1"
  [[ -f "$pidfile" ]] || return 1
  local pid
  pid="$(cat "$pidfile")"
  kill -0 "$pid" 2>/dev/null
}

check_redis() {
  if redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" ping 2>/dev/null | grep -q PONG; then
    log "Redis OK ($REDIS_HOST:$REDIS_PORT)"
  else
    log "错误: Redis 不可用，请先启动: sudo systemctl start redis-server"
    log "      测试: redis-cli -h $REDIS_HOST -p $REDIS_PORT ping"
    exit 1
  fi
}

check_consul() {
  if curl -sf "$CONSUL_HTTP/v1/status/leader" >/dev/null 2>&1; then
    log "Consul 已在运行"
    return 0
  fi
  if ! command -v consul >/dev/null 2>&1; then
    log "错误: 未安装 consul，请先安装并确保 consul 在 PATH 中"
    exit 1
  fi
  log "启动 Consul (agent -dev)..."
  nohup consul agent -dev >>"$LOG_DIR/consul.log" 2>&1 &
  echo $! >"$PID_DIR/consul.pid"
  sleep 2
  if curl -sf "$CONSUL_HTTP/v1/status/leader" >/dev/null 2>&1; then
    log "Consul 启动成功"
  else
    log "错误: Consul 启动失败，查看 $LOG_DIR/consul.log"
    exit 1
  fi
}

start_micro() {
  local name="$1"
  local dir="$2"
  local pidfile="$PID_DIR/${name}.pid"

  if is_running "$pidfile"; then
    log "$name 已在运行 (PID $(cat "$pidfile"))，跳过"
    return 0
  fi

  log "启动 $name ..."
  (
    cd "$PROJECT_ROOT/$dir"
    nohup go run . >>"$LOG_DIR/${name}.log" 2>&1 &
    echo $! >"$pidfile"
  )
  sleep 2
  if is_running "$pidfile"; then
    log "$name 已启动 (PID $(cat "$pidfile"))，日志: logs/${name}.log"
  else
    log "错误: $name 启动失败，查看 logs/${name}.log"
    exit 1
  fi
}

start_web() {
  local name="web"
  local pidfile="$PID_DIR/${name}.pid"

  if is_running "$pidfile"; then
    log "Web 已在运行 (PID $(cat "$pidfile"))，跳过"
    return 0
  fi

  log "启动 Web 网关 ( :8080 ) ..."
  (
    cd "$PROJECT_ROOT"
    nohup go run ./web >>"$LOG_DIR/${name}.log" 2>&1 &
    echo $! >"$pidfile"
  )
  sleep 2
  if is_running "$pidfile"; then
    log "Web 已启动 → http://localhost:8080/home/index.html"
  else
    log "错误: Web 启动失败，查看 logs/web.log"
    exit 1
  fi
}

cd "$PROJECT_ROOT"
log "项目目录: $PROJECT_ROOT"
log "---------- 检查依赖 ----------"
check_redis
check_consul
log "---------- 启动微服务 ----------"
start_micro "getCaptcha" "service/getCaptcha"
start_micro "register"   "service/register"
start_micro "user"       "service/user"
start_micro "house"      "service/house"
start_micro "order"      "service/order"
log "---------- 启动 Web ----------"
start_web
log "---------- 全部完成 ----------"
log "首页: http://localhost:8080/home/index.html"
log "停止: $PROJECT_ROOT/MyShell/stop-all.sh"
log "查看日志: tail -f $LOG_DIR/house.log"
