#!/bin/bash
# 初始化 area 城区表（新机器空库必跑一次）
# 表结构由服务 GORM AutoMigrate 自动创建，本脚本只灌数据；幂等（同名跳过）
# 用法:
#   ./MyShell/seed-area.sh                # 灌入内置默认城区
#   ./MyShell/seed-area.sh dump.sql       # 导入旧机器导出的 area 数据
# 旧机器导出数据（在旧 VM 上执行）:
#   mysqldump -uneko -p search_house area > area.sql

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
# shellcheck disable=SC1091
source "$ROOT/MyShell/env.sh"

# 带 dump 参数：直接导入旧机器数据
if [[ -n "${1:-}" ]]; then
  dump="$1"
  [[ -f "$dump" ]] || { log "错误: 找不到 $dump"; exit 1; }
  log "导入 $dump → $MYSQL_DB ..."
  if command -v mysql >/dev/null 2>&1; then
    mysql -h"$MYSQL_HOST" -P"$MYSQL_PORT" -u"$MYSQL_USER" -p"$MYSQL_PWD" "$MYSQL_DB" <"$dump"
  else
    docker exec -i "$DOCKER_CONTAINER_MYSQL" mysql -u"$MYSQL_USER" -p"$MYSQL_PWD" "$MYSQL_DB" <"$dump"
  fi
  log "导入完成，area 总数: $(mysql_q "SELECT COUNT(*) FROM area")"
  exit 0
fi

# 前置: area 表需已由 AutoMigrate 创建（先 start-all.sh 再灌数据）
if [[ -z "$(mysql_q "SHOW TABLES LIKE 'area'")" ]]; then
  log "错误: area 表不存在。请先 ./MyShell/start-all.sh 启动（AutoMigrate 建表）后重试"
  exit 1
fi

# 默认城区（与 seed-demo.sh 演示地址一致；如与旧数据不同，编辑此列表或用 dump 导入）
AREAS=(黄浦区 徐汇区 长宁区 静安区 普陀区 虹口区 杨浦区 闵行区 宝山区 嘉定区 浦东新区 松江区)

inserted=0
for name in "${AREAS[@]}"; do
  if [[ -n "$(mysql_q "SELECT id FROM area WHERE name='$name' LIMIT 1")" ]]; then
    log "已存在，跳过: $name"
  else
    mysql_q "INSERT INTO area (name) VALUES ('$name')" >/dev/null
    inserted=$((inserted + 1))
    log "新增: $name"
  fi
done
log "完成: 新增 $inserted 个，area 总数: $(mysql_q "SELECT COUNT(*) FROM area")"