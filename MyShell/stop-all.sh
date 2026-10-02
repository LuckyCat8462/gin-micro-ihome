#!/bin/bash
# 一键停止 start-all.sh 启动的进程（按 PID 文件 + 二进制名精确清理）
# 用法: ./MyShell/stop-all.sh

set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PID_DIR="$PROJECT_ROOT/logs/pids"
BIN_DIR="$PROJECT_ROOT/logs/bin"

log() { echo "[$(date '+%H:%M:%S')] $*"; }

stop_pidfile() {
  local name="$1"
  local pidfile="$PID_DIR/${name}.pid"
  if [[ ! -f "$pidfile" ]]; then
    return 0
  fi
  local pid
  pid="$(cat "$pidfile" 2>/dev/null || true)"
  if [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null; then
    kill "$pid" 2>/dev/null || true
    sleep 1
    if kill -0 "$pid" 2>/dev/null; then
      kill -9 "$pid" 2>/dev/null || true
    fi
    log "已停止 $name (PID $pid)"
  else
    log "$name 未在运行 (陈旧 PID 文件)"
  fi
  rm -f "$pidfile"
}

# 按绝对路径二进制再扫一遍，避免 PID 文件丢失时残留
stop_by_bin() {
  local name="$1"
  local bin="$BIN_DIR/$name"
  [[ -x "$bin" ]] || return 0
  local pids
  pids="$(pgrep -f "^$bin\$" 2>/dev/null || true)"
  if [[ -z "$pids" ]]; then
    # 兼容命令行带路径的情况
    pids="$(pgrep -f "$bin" 2>/dev/null || true)"
  fi
  if [[ -n "$pids" ]]; then
    # shellcheck disable=SC2086
    kill $pids 2>/dev/null || true
    sleep 1
    # shellcheck disable=SC2086
    kill -9 $pids 2>/dev/null || true
    log "已清理残留进程 $name ($pids)"
  fi
}

log "停止 Web 与微服务..."
for name in web order house user register getCaptcha minio; do
  stop_pidfile "$name"
  stop_by_bin "$name"
done

# 仅停止本脚本拉起的 consul（有 pid 文件才停，避免误杀 docker consul）
if [[ -f "$PID_DIR/consul.pid" ]]; then
  stop_pidfile "consul"
fi

log "全部已停止"
