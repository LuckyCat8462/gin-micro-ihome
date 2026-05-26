#!/bin/bash
# 一键停止 start-all.sh 启动的进程
# 用法: ./MyShell/stop-all.sh

PROJECT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PID_DIR="$PROJECT_ROOT/logs/pids"

log() { echo "[$(date '+%H:%M:%S')] $*"; }

stop_one() {
  local name="$1"
  local pidfile="$PID_DIR/${name}.pid"
  if [[ ! -f "$pidfile" ]]; then
    return 0
  fi
  local pid
  pid="$(cat "$pidfile")"
  if kill -0 "$pid" 2>/dev/null; then
    # 结束子进程（go run 产生的实际服务）
    pkill -P "$pid" 2>/dev/null || true
    kill "$pid" 2>/dev/null || true
    sleep 1
    kill -9 "$pid" 2>/dev/null || true
    log "已停止 $name (PID $pid)"
  else
    log "$name 未在运行 (陈旧 PID 文件)"
  fi
  rm -f "$pidfile"
}

log "停止 Web 与微服务..."
stop_one "web"
stop_one "order"
stop_one "house"
stop_one "user"
stop_one "register"
stop_one "getCaptcha"

if [[ -f "$PID_DIR/consul.pid" ]]; then
  stop_one "consul"
fi

# 清理可能残留的 go run 子进程（按需）
for pattern in "service/getCaptcha" "service/register" "service/user" "service/house" "service/order" "gin-micro-ihome/web"; do
  pkill -f "$pattern" 2>/dev/null || true
done

log "全部已停止"
