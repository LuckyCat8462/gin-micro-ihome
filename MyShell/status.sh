#!/bin/bash
# 查看本地依赖与服务状态
# 用法: ./MyShell/status.sh

set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PID_DIR="$PROJECT_ROOT/logs/pids"
BIN_DIR="$PROJECT_ROOT/logs/bin"
CONSUL_HTTP="${CONSUL_HTTP:-http://127.0.0.1:8500}"

log() { echo "[$(date '+%H:%M:%S')] $*"; }

port_open() {
  local host="$1" port="$2"
  (echo >/dev/tcp/"$host"/"$port") >/dev/null 2>&1
}

check_one() {
  local name="$1" port="$2"
  local pidfile="$PID_DIR/${name}.pid"
  local state="DOWN"
  local pid="-"
  if [[ -f "$pidfile" ]]; then
    pid="$(cat "$pidfile" 2>/dev/null || echo -)"
    if [[ -n "$pid" && "$pid" != "-" ]] && kill -0 "$pid" 2>/dev/null; then
      state="UP"
    else
      state="STALE_PID"
    fi
  fi
  local port_state="closed"
  if port_open 127.0.0.1 "$port"; then
    port_state="open"
  fi
  printf "%-12s pid=%-8s state=%-10s port=%s\n" "$name" "$pid" "$state" "$port:$port_state"
}

log "==== 依赖端口 ===="
for item in "mysql:3306" "redis:6379" "consul:8500" "rabbitmq:5672" "minio:9000"; do
  name="${item%%:*}"
  port="${item##*:}"
  if port_open 127.0.0.1 "$port"; then
    printf "%-12s 127.0.0.1:%s open\n" "$name" "$port"
  else
    # redis 可能只绑局域网
    if [[ "$name" == "redis" ]] && port_open 192.168.81.128 6379; then
      printf "%-12s 192.168.81.128:6379 open\n" "$name"
    else
      printf "%-12s :%s closed\n" "$name" "$port"
    fi
  fi
done

log "==== 应用进程 ===="
check_one getCaptcha 12311
check_one register 12312
check_one user 12313
check_one house 12314
check_one order 12315
check_one web 8080
check_one minio 9000

log "==== HTTP 抽检 ===="
if curl -sf --max-time 3 http://127.0.0.1:8080/api/v1.0/session >/dev/null; then
  echo "web /api/v1.0/session OK"
else
  echo "web /api/v1.0/session FAIL"
fi
if curl -sf --max-time 3 "$CONSUL_HTTP/v1/status/leader" >/dev/null; then
  echo "consul leader OK"
else
  echo "consul leader FAIL"
fi

log "==== 二进制 ===="
ls -l "$BIN_DIR" 2>/dev/null || echo "(尚无 logs/bin)"
