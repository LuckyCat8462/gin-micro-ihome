#!/bin/bash
# MyShell/env.sh —— 机器差异配置的唯一入口（被 MyShell 下所有脚本 source）
# 本机覆盖: 复制 local.env.example 为 local.env（已被 gitignore），只写需要改的项

SHELL_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [[ -f "$SHELL_DIR/local.env" ]]; then
  # shellcheck disable=SC1091
  source "$SHELL_DIR/local.env"
fi

# ---------- Redis ----------
# 候选主机，空格分隔、按序探测；新机器有内网 Redis 时在 local.env 追加
# 例: REDIS_CANDIDATES="127.0.0.1 192.168.1.100"
REDIS_CANDIDATES="${REDIS_CANDIDATES:-127.0.0.1}"
REDIS_PORT="${REDIS_PORT:-6379}"

# ---------- MySQL ----------
MYSQL_HOST="${MYSQL_HOST:-127.0.0.1}"
MYSQL_PORT="${MYSQL_PORT:-3306}"
MYSQL_USER="${MYSQL_USER:-neko}"
MYSQL_PWD="${MYSQL_PWD:-neko123456}"
MYSQL_DB="${MYSQL_DB:-search_house}"

# ---------- 其他依赖 ----------
CONSUL_HTTP="${CONSUL_HTTP:-http://127.0.0.1:8500}"
MINIO_API="${MINIO_API:-http://127.0.0.1:9000}"
AMQP_HOST="${AMQP_HOST:-127.0.0.1}"
AMQP_PORT="${AMQP_PORT:-5672}"

# ---------- Go ----------
# 留空则自动探测: command -v go → 常见安装路径
GO_BIN="${GO_BIN:-}"
# Go 模块代理: proxy.golang.org 在国内网络不可达，默认走 goproxy.cn；
# 已有自己的代理/私有源时在 local.env 覆盖: GOPROXY="https://your-proxy,direct"
export GOPROXY="${GOPROXY:-https://goproxy.cn,direct}"

# ---------- docker 容器名（与 docker-compose.yml 对齐） ----------
DOCKER_CONTAINER_REDIS="${DOCKER_CONTAINER_REDIS:-ihome-redis}"
DOCKER_CONTAINER_MYSQL="${DOCKER_CONTAINER_MYSQL:-ihome-mysql}"

# ---------- 通用工具 ----------
log() { echo "[$(date '+%H:%M:%S')] $*"; }

mysql_q() {
  # 本机有 mysql 客户端优先，否则 docker exec 进容器执行
  if command -v mysql >/dev/null 2>&1; then
    mysql -h"$MYSQL_HOST" -P"$MYSQL_PORT" -u"$MYSQL_USER" -p"$MYSQL_PWD" "$MYSQL_DB" -N -e "$1" 2>/dev/null
  else
    docker exec "$DOCKER_CONTAINER_MYSQL" mysql -u"$MYSQL_USER" -p"$MYSQL_PWD" "$MYSQL_DB" -N -e "$1" 2>/dev/null
  fi
}

redis_cli() {
  # redis-cli wrapper: 优先宿主机，fallback 到 docker exec
  if command -v redis-cli >/dev/null 2>&1; then
    redis-cli "$@"
  else
    docker exec "$DOCKER_CONTAINER_REDIS" redis-cli "$@"
  fi
}
