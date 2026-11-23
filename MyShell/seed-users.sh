#!/bin/bash
# 注册 5 个已实名测试账号（幂等：已存在则跳过注册，仍尝试补全实名）
# 用法: ./MyShell/seed-users.sh
# 前置: Web 已启动（./MyShell/start-all.sh）

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
# shellcheck disable=SC1091
source "$ROOT/MyShell/env.sh"

BASE="${BASE_URL:-http://127.0.0.1:8080/api/v1.0}"
PASS="${SEED_PASSWORD:-123456}"
REDIS_HOST="${REDIS_HOST:-}"

resolve_redis() {
  if [[ -n "$REDIS_HOST" ]]; then
    if redis_cli -h "$REDIS_HOST" -p "$REDIS_PORT" ping 2>/dev/null | grep -q PONG; then
      echo "$REDIS_HOST"
      return 0
    fi
    # 即使 host 不通也试试 localhost（容器内 redis-cli 走本地）
    if docker exec ihome-redis redis-cli ping 2>/dev/null | grep -q PONG; then
      echo "127.0.0.1"
      export REDIS_CLI_HOST=127.0.0.1
      return 0
    fi
    log "错误: REDIS_HOST=$REDIS_HOST 不可用"
    exit 1
  fi
  local h
  for h in $REDIS_CANDIDATES; do
    if redis_cli -h "$h" -p "$REDIS_PORT" ping 2>/dev/null | grep -q PONG; then
      echo "$h"
      return 0
    fi
  done
  # 最后 fallback：docker exec 容器内 redis-cli（它走容器自己的 localhost）
  if docker exec "$DOCKER_CONTAINER_REDIS" redis-cli ping 2>/dev/null | grep -q PONG; then
    echo "127.0.0.1"
    export REDIS_CLI_HOST=127.0.0.1
    return 0
  fi
  log "错误: Redis 不可用（写图片验证码需要）"
  exit 1
}

RH="$(resolve_redis)"
USERS=(
  "13900000001:张三:110101199001011234"
  "13900000002:李四:110101199002021235"
  "13900000003:王五:110101199003031236"
  "13900000004:赵六:110101199004041237"
  "13900000005:钱七:110101199005051238"
)

if ! curl -sf --max-time 3 "$BASE/session" >/dev/null; then
  log "错误: Web 未就绪 ($BASE/session)"
  exit 1
fi

for item in "${USERS[@]}"; do
  mobile="${item%%:*}"
  rest="${item#*:}"
  real="${rest%%:*}"
  idcard="${rest##*:}"
  uuid="seed-$mobile"
  code="ABCD"
  cookie="/tmp/ihome_seed_${mobile}.txt"
  rm -f "$cookie"

  log "处理 $mobile ($real)"
  redis_cli -h "$RH" -p "$REDIS_PORT" SETEX "$uuid" 300 "$code" >/dev/null 2>&1 || \
    docker exec "$DOCKER_CONTAINER_REDIS" redis-cli SETEX "$uuid" 300 "$code" >/dev/null 2>&1

  reg="$(curl -s -c "$cookie" -b "$cookie" -H "Content-Type: application/json" \
    -d "{\"mobile\":\"$mobile\",\"password\":\"$PASS\",\"imagecode\":\"$code\",\"uuid\":\"$uuid\"}" \
    "$BASE/users" || true)"
  echo "  register: $reg"

  login="$(curl -s -c "$cookie" -b "$cookie" -H "Content-Type: application/json" \
    -d "{\"mobile\":\"$mobile\",\"password\":\"$PASS\"}" \
    "$BASE/sessions")"
  echo "  login: $login"

  auth="$(curl -s -c "$cookie" -b "$cookie" -H "Content-Type: application/json" \
    -d "{\"real_name\":\"$real\",\"id_card\":\"$idcard\"}" \
    "$BASE/user/auth")"
  echo "  auth: $auth"
done

log "完成。统一密码: $PASS"
log "账号: 13900000001 ~ 13900000005"
