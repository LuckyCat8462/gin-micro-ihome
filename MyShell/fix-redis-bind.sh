#!/bin/bash
# [已废弃] 本脚本为旧虚拟机（192.168.81.128）定制，不再需要。
# 现状:
#   - Go conf 全部使用 127.0.0.1
#   - docker compose 起的 Redis 容器天然监听所有接口
#   - 机器差异配置统一放 MyShell/env.sh（本机覆盖: MyShell/local.env）
# 如需局域网访问本机 Redis，在 MyShell/local.env 的 REDIS_CANDIDATES 追加内网 IP，
# 或使用 docker compose up -d redis。
echo "[fix-redis-bind.sh] 已废弃: 现行配置全部走 127.0.0.1 + docker compose，无需修改 redis bind。"
exit 0
