#!/bin/bash
# 将 Redis 改为同时监听 127.0.0.1 与局域网 IP，便于本地开发使用 127.0.0.1:6379
# 需要 sudo 权限执行

set -e

CONF="/etc/redis/redis.conf"
if [ ! -f "$CONF" ]; then
  echo "未找到 $CONF，请手动修改 Redis 配置"
  exit 1
fi

echo "当前 Redis 绑定："
redis-cli -h 192.168.81.128 -p 6379 CONFIG GET bind 2>/dev/null || redis-cli -p 6379 CONFIG GET bind

echo ""
echo "将把 bind 设置为: 127.0.0.1 192.168.81.128"
echo "执行后请把 web/conf/config.go 中 RedisAddr 改为 127.0.0.1:6379"
echo ""

sudo sed -i 's/^bind .*/bind 127.0.0.1 192.168.81.128/' "$CONF" || {
  echo "若 sed 失败，请手动编辑 $CONF ，设置:"
  echo "  bind 127.0.0.1 192.168.81.128"
  exit 1
}

sudo systemctl restart redis-server
sleep 1

echo "验证："
redis-cli -h 127.0.0.1 -p 6379 ping
redis-cli -h 192.168.81.128 -p 6379 ping

echo "完成。请将 web/conf/config.go 与 service/getCaptcha/conf/config.go 中 RedisAddr 改为 127.0.0.1:6379"
