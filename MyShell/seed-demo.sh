#!/bin/bash
# 演示数据灌库：清理 ut-* → 发房+MinIO 传图 → 真链路订单 + SQL 状态样例
# 用法:
#   ./MyShell/seed-users.sh   # 建议先跑（幂等）
#   ./MyShell/seed-demo.sh
# 环境变量:
#   BASE_URL=http://127.0.0.1:8080/api/v1.0
#   CLEAN_UT=1                清理单测残留（默认 1）
#   SKIP_ORDERS=0             跳过订单灌数
#   SEED_PASSWORD=123456
# 前置: Web/微服务/MySQL/Redis/MinIO 已启动

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
# shellcheck disable=SC1091
source "$ROOT/MyShell/env.sh"
BASE="${BASE_URL:-http://127.0.0.1:8080/api/v1.0}"
PASS="${SEED_PASSWORD:-123456}"
CLEAN_UT="${CLEAN_UT:-1}"
SKIP_ORDERS="${SKIP_ORDERS:-0}"
ASSET_DIR="$ROOT/MyShell/seed-assets/houses"
AVATAR_DIR="$ROOT/MyShell/seed-assets/avatars"
LOCAL_IMG_DIR="$ROOT/web/view/images"

LANDLORD_A="13900000001"
LANDLORD_B="13900000002"
TENANT_A="13900000003"
TENANT_B="13900000004"
TENANT_C="13900000005"

log() { echo "[$(date '+%H:%M:%S')] $*"; }

mysql_q() {
  local sql="$1"
  if command -v mysql >/dev/null 2>&1; then
    mysql -h"$MYSQL_HOST" -P"$MYSQL_PORT" -u"$MYSQL_USER" -p"$MYSQL_PWD" "$MYSQL_DB" -N -e "$sql" 2>/dev/null
  else
    docker exec "$DOCKER_CONTAINER_MYSQL" mysql -u"$MYSQL_USER" -p"$MYSQL_PWD" "$MYSQL_DB" -N -e "$sql" 2>/dev/null
  fi
}

json_get() {
  # usage: json_get <json> <python-expr using d>
  local raw="$1" expr="$2"
  python3 -c "import json,sys; d=json.loads(sys.argv[1]); print($expr)" "$raw"
}

ensure_ready() {
  if ! curl -sf --max-time 3 "$BASE/session" >/dev/null; then
    log "错误: Web 未就绪 ($BASE/session)"
    exit 1
  fi
  if ! curl -sf --max-time 2 http://127.0.0.1:9000/minio/health/live >/dev/null; then
    log "错误: MinIO 未就绪 (http://127.0.0.1:9000)"
    exit 1
  fi
  local n
  n="$(find "$ASSET_DIR" -name '*.jpg' 2>/dev/null | wc -l | tr -d ' ')"
  if [[ "$n" -lt 12 ]]; then
    log "错误: 素材不足 ($ASSET_DIR 仅 $n 张 jpg，至少需要 12 张)"
    exit 1
  fi
  local u
  for u in "$LANDLORD_A" "$LANDLORD_B" "$TENANT_A" "$TENANT_B" "$TENANT_C"; do
    if [[ -z "$(mysql_q "SELECT id FROM user WHERE mobile='$u' LIMIT 1")" ]]; then
      log "错误: 缺少账号 $u，请先执行 ./MyShell/seed-users.sh"
      exit 1
    fi
  done
}

login() {
  local mobile="$1" cookie="$2"
  rm -f "$cookie"
  local resp
  resp="$(curl -s -c "$cookie" -b "$cookie" -H "Content-Type: application/json" \
    -d "{\"mobile\":\"$mobile\",\"password\":\"$PASS\"}" \
    "$BASE/sessions")"
  local errno
  errno="$(json_get "$resp" "d.get('errno','')")"
  if [[ "$errno" != "0" ]]; then
    log "登录失败 $mobile: $resp"
    exit 1
  fi
}

upload_avatar_if_needed() {
  local mobile="$1" cookie="$2" file="$3"
  [[ -f "$file" ]] || return 0
  local url
  url="$(mysql_q "SELECT avatar_url FROM user WHERE mobile='$mobile' LIMIT 1")"
  if [[ -n "$url" && "$url" == http* ]]; then
    log "头像已存在，跳过 $mobile"
    return 0
  fi
  local resp
  resp="$(curl -s -c "$cookie" -b "$cookie" -F "avatar=@${file};type=image/jpeg" \
    "$BASE/user/avatar")"
  log "头像 $mobile: $(json_get "$resp" "d.get('errno','')+' '+str(d.get('errmsg',''))")"
}

img_path() {
  # 1-based index into asset pool (wrap)
  local idx="$1" n total
  total="$(find "$ASSET_DIR" -name '*.jpg' | wc -l | tr -d ' ')"
  idx=$(( (idx - 1) % total + 1 ))
  printf "%s/%02d.jpg" "$ASSET_DIR" "$idx"
}

house_exists() {
  local title="$1"
  mysql_q "SELECT id FROM house WHERE title='${title}' AND deleted_at IS NULL LIMIT 1"
}

publish_house() {
  local cookie="$1" title="$2" area_id="$3" price="$4" unit="$5" rooms="$6" acreage="$7" capacity="$8" beds="$9" deposit="${10}"
  local addr="上海市演示路 ${area_id}00 号 · ${title}"
  # 设施：按 area+price 选 5 个
  local f1=$(( (area_id % 20) + 1 ))
  local f2=$(( (f1 % 23) + 1 ))
  local f3=$(( (f2 % 23) + 1 ))
  local f4=$(( (f3 % 23) + 1 ))
  local f5=$(( (f4 % 23) + 1 ))
  local payload
  payload="$(python3 - <<PY
import json
print(json.dumps({
  "title": "$title",
  "price": "$price",
  "area_id": "$area_id",
  "address": "$addr",
  "room_count": "$rooms",
  "acreage": "$acreage",
  "unit": "$unit",
  "capacity": "$capacity",
  "beds": "$beds",
  "deposit": "$deposit",
  "min_days": "1",
  "max_days": "0",
  "facility": ["$f1","$f2","$f3","$f4","$f5"],
}, ensure_ascii=False))
PY
)"
  local resp
  resp="$(curl -s -c "$cookie" -b "$cookie" -H "Content-Type: application/json" \
    -d "$payload" "$BASE/houses")"
  local errno hid
  errno="$(json_get "$resp" "d.get('errno','')")"
  if [[ "$errno" != "0" ]]; then
    log "发房失败 [$title]: $resp"
    return 1
  fi
  hid="$(json_get "$resp" "(d.get('data') or {}).get('house_id','')")"
  echo "$hid"
}

upload_house_images() {
  local cookie="$1" house_id="$2" seed_idx="$3"
  local existing
  existing="$(mysql_q "SELECT index_image_url FROM house WHERE id=$house_id LIMIT 1")"
  if [[ -n "$existing" && "$existing" == *"/ihome/"* ]]; then
    local img_cnt
    img_cnt="$(mysql_q "SELECT COUNT(*) FROM house_image WHERE house_id=$house_id")"
    if [[ "${img_cnt:-0}" -ge 2 ]]; then
      log "  房源 $house_id 图片已齐，跳过"
      return 0
    fi
  fi
  local i f resp errno
  for i in 0 1 2; do
    f="$(img_path "$((seed_idx + i))")"
    resp="$(curl -s -c "$cookie" -b "$cookie" \
      -F "house_image=@${f};filename=$(basename "$f");type=image/jpeg" \
      "$BASE/houses/${house_id}/images")"
    errno="$(json_get "$resp" "d.get('errno','')")"
    if [[ "$errno" != "0" ]]; then
      log "  传图失败 house=$house_id file=$f: $resp"
      return 1
    fi
  done
  log "  房源 $house_id 已上传 3 张图"
}

clean_ut() {
  log "清理 ut-* 测试残留..."
  mysql_q "
DELETE r FROM house_reservation r
  INNER JOIN house h ON h.id = r.house_id
  WHERE h.title LIKE 'ut-%';
DELETE hi FROM house_image hi
  INNER JOIN house h ON h.id = hi.house_id
  WHERE h.title LIKE 'ut-%';
DELETE hf FROM house_facility hf
  INNER JOIN house h ON h.id = hf.house_id
  WHERE h.title LIKE 'ut-%';
DELETE o FROM order_house o
  INNER JOIN house h ON h.id = o.house_id
  WHERE h.title LIKE 'ut-%';
DELETE o FROM order_house o
  INNER JOIN user u ON u.id = o.user_id
  WHERE u.name LIKE 'ut\\_%' OR u.name LIKE 'ut\\_svc\\_%' OR u.name LIKE 'ut\\_order\\_%';
DELETE FROM house WHERE title LIKE 'ut-%';
DELETE FROM user WHERE name LIKE 'ut\\_%' OR name LIKE 'ut\\_svc\\_%' OR name LIKE 'ut\\_order\\_%';
DELETE lm FROM local_message lm
  LEFT JOIN order_house o ON o.id = lm.order_id
  WHERE o.id IS NULL;
"
  log "清理完成"
}

# 将仍使用 ./images/... 本地路径的房源主图/副图迁到 MinIO（幂等）
migrate_legacy_local_images() {
  local rows
  mapfile -t rows < <(mysql_q "
SELECT id FROM house
WHERE deleted_at IS NULL
  AND (
    index_image_url LIKE './images/%'
    OR index_image_url LIKE '/home/images/%'
    OR (index_image_url <> '' AND index_image_url NOT LIKE 'http%')
  )
ORDER BY id;
")
  if [[ "${#rows[@]}" -eq 0 || -z "${rows[0]:-}" ]]; then
    log "无本地图房源需要迁移"
    return 0
  fi

  local cookie="/tmp/ihome_seed_migrate_img.txt"
  login "$LANDLORD_A" "$cookie"

  local hid main_file i f resp errno
  for hid in "${rows[@]}"; do
    [[ -n "$hid" ]] || continue
    case "$hid" in
      1) main_file="$LOCAL_IMG_DIR/home01.jpg" ;;
      2) main_file="$LOCAL_IMG_DIR/home02.jpg" ;;
      3) main_file="$LOCAL_IMG_DIR/home03.jpg" ;;
      *) main_file="$(img_path "$hid")" ;;
    esac
    if [[ ! -f "$main_file" ]]; then
      main_file="$(img_path "$hid")"
    fi
    log "迁移本地图房源 id=$hid ← $(basename "$main_file")"

    # UploadHouseImage 仅在 index 为空时写主图，故先清空本地路径
    mysql_q "
UPDATE house SET index_image_url='' WHERE id=$hid;
DELETE FROM house_image WHERE house_id=$hid AND (url LIKE './images/%' OR url LIKE '/home/images/%' OR url NOT LIKE 'http%');
"
    for i in 0 1 2; do
      if [[ "$i" -eq 0 ]]; then
        f="$main_file"
      else
        f="$(img_path "$((hid * 3 + i))")"
      fi
      resp="$(curl -s -c "$cookie" -b "$cookie" \
        -F "house_image=@${f};filename=$(basename "$f");type=image/jpeg" \
        "$BASE/houses/${hid}/images")"
      errno="$(json_get "$resp" "d.get('errno','')")"
      if [[ "$errno" != "0" ]]; then
        log "  传图失败 house=$hid: $resp"
        return 1
      fi
    done
    log "  房源 $hid 已迁至 MinIO"
  done
}

seed_houses() {
  local cookie_a="/tmp/ihome_seed_ll_a.txt"
  local cookie_b="/tmp/ihome_seed_ll_b.txt"
  login "$LANDLORD_A" "$cookie_a"
  login "$LANDLORD_B" "$cookie_b"
  upload_avatar_if_needed "$LANDLORD_A" "$cookie_a" "$AVATAR_DIR/landlord-a.jpg"
  upload_avatar_if_needed "$LANDLORD_B" "$cookie_b" "$AVATAR_DIR/landlord-b.jpg"

  # 12 区 × 3 档 = 36；区名取自 DB
  local areas
  mapfile -t areas < <(mysql_q "SELECT id, name FROM area ORDER BY id LIMIT 12")
  if [[ "${#areas[@]}" -lt 12 ]]; then
    log "错误: area 表不足 12 条"
    exit 1
  fi

  local tiers=("经济" "舒适" "精品")
  local prices=(168 328 688)
  local units=("一室一厅" "两室一厅" "三室两厅")
  local rooms=(1 2 3)
  local acres=(35 68 110)
  local caps=(2 4 6)
  local beds_arr=("单人床×2" "双人床×1+单人床×1" "双人床×2")
  local deps=(200 500 1000)

  local idx=0 area_line aid aname t cookie title hid seed_i
  for area_line in "${areas[@]}"; do
    aid="$(echo "$area_line" | awk '{print $1}')"
    aname="$(echo "$area_line" | cut -f2-)"
    for t in 0 1 2; do
      idx=$((idx + 1))
      title="demo-${aname}-${tiers[$t]}"
      if [[ $((idx % 2)) -eq 1 ]]; then
        cookie="$cookie_a"
      else
        cookie="$cookie_b"
      fi
      hid="$(house_exists "$title")"
      if [[ -n "$hid" ]]; then
        log "[$idx/36] 已存在 $title (id=$hid)"
      else
        log "[$idx/36] 发布 $title → 房东 cookie"
        hid="$(publish_house "$cookie" "$title" "$aid" "${prices[$t]}" "${units[$t]}" \
          "${rooms[$t]}" "${acres[$t]}" "${caps[$t]}" "${beds_arr[$t]}" "${deps[$t]}")"
        log "  得到 house_id=$hid"
      fi
      seed_i=$(( (idx - 1) * 3 + 1 ))
      upload_house_images "$cookie" "$hid" "$seed_i"
    done
  done
  log "房源灌数完成，当前 demo 房源数: $(mysql_q "SELECT COUNT(*) FROM house WHERE title LIKE 'demo-%' AND deleted_at IS NULL")"
}

# 返回某房东名下一套 demo 房源 id
pick_house() {
  local landlord_mobile="$1" offset="${2:-0}"
  mysql_q "
SELECT h.id FROM house h
  INNER JOIN user u ON u.id = h.user_id
  WHERE u.mobile='$landlord_mobile' AND h.title LIKE 'demo-%' AND h.deleted_at IS NULL
  ORDER BY h.id
  LIMIT 1 OFFSET $offset;
"
}

wait_order_status() {
  local oid="$1" want="$2" seconds="${3:-20}"
  local i st
  for ((i = 1; i <= seconds; i++)); do
    st="$(mysql_q "SELECT status FROM order_house WHERE id=$oid LIMIT 1")"
    if [[ "$st" == "$want" ]]; then
      echo "$st"
      return 0
    fi
    sleep 1
  done
  echo "${st:-}"
  return 1
}

seed_orders_api() {
  local c_tenant="/tmp/ihome_seed_tenant.txt"
  local c_ll="/tmp/ihome_seed_ll_accept.txt"
  login "$TENANT_A" "$c_tenant"
  login "$LANDLORD_A" "$c_ll"

  local h1 h2 h3
  h1="$(pick_house "$LANDLORD_A" 0)"
  h2="$(pick_house "$LANDLORD_A" 1)"
  h3="$(pick_house "$LANDLORD_B" 0)"
  if [[ -z "$h1" || -z "$h2" || -z "$h3" ]]; then
    log "错误: 找不到足够的 demo 房源用于下单"
    exit 1
  fi

  # 幂等：若今日已有 seed 标记评论的订单则跳过整段 API
  local marker
  marker="$(mysql_q "SELECT id FROM order_house WHERE comment='seed-api-flow' LIMIT 1")"
  if [[ -n "$marker" ]]; then
    log "API 订单故事线已存在 (order=$marker)，跳过"
    return 0
  fi

  local start1 end1 start2 end2 start3 end3
  start1="$(date -d '+20 days' +%Y-%m-%d 2>/dev/null || date -v+20d +%Y-%m-%d)"
  end1="$(date -d '+22 days' +%Y-%m-%d 2>/dev/null || date -v+22d +%Y-%m-%d)"
  start2="$(date -d '+30 days' +%Y-%m-%d 2>/dev/null || date -v+30d +%Y-%m-%d)"
  end2="$(date -d '+32 days' +%Y-%m-%d 2>/dev/null || date -v+32d +%Y-%m-%d)"
  start3="$(date -d '+40 days' +%Y-%m-%d 2>/dev/null || date -v+40d +%Y-%m-%d)"
  end3="$(date -d '+41 days' +%Y-%m-%d 2>/dev/null || date -v+41d +%Y-%m-%d)"

  log "真链路1: 下单→支付→接单 house=$h1 ($start1~$end1)"
  local resp oid errno
  resp="$(curl -s -c "$c_tenant" -b "$c_tenant" -H "Content-Type: application/json" \
    -d "{\"house_id\":\"$h1\",\"start_date\":\"$start1\",\"end_date\":\"$end1\"}" \
    "$BASE/orders")"
  errno="$(json_get "$resp" "d.get('errno','')")"
  oid="$(json_get "$resp" "(d.get('data') or {}).get('order_id','')")"
  if [[ "$errno" != "0" || -z "$oid" ]]; then
    log "下单失败: $resp"
    return 1
  fi
  resp="$(curl -s -c "$c_tenant" -b "$c_tenant" -X POST "$BASE/orders/${oid}/payment")"
  log "  支付: $(json_get "$resp" "d.get('errno','')+' '+str(d.get('errmsg',''))")"
  local st
  st="$(wait_order_status "$oid" "WAIT_ACCEPT" 25 || true)"
  log "  支付后状态=$st (期望 WAIT_ACCEPT)"
  if [[ "$st" == "WAIT_ACCEPT" || "$st" == "PAID" ]]; then
    # PAID 时再等一会；WAIT_ACCEPT 直接接单
    if [[ "$st" == "PAID" ]]; then
      st="$(wait_order_status "$oid" "WAIT_ACCEPT" 25 || true)"
    fi
    if [[ "$st" == "WAIT_ACCEPT" ]]; then
      resp="$(curl -s -c "$c_ll" -b "$c_ll" -H "Content-Type: application/json" \
        -X PUT -d '{"action":"accept"}' "$BASE/orders/${oid}/status")"
      log "  接单: $(json_get "$resp" "d.get('errno','')+' '+str(d.get('errmsg',''))")"
    fi
  fi
  mysql_q "UPDATE order_house SET comment='seed-api-flow' WHERE id=$oid"

  log "真链路2: 下单→支付→待接单 house=$h2 ($start2~$end2)"
  login "$TENANT_B" "$c_tenant"
  resp="$(curl -s -c "$c_tenant" -b "$c_tenant" -H "Content-Type: application/json" \
    -d "{\"house_id\":\"$h2\",\"start_date\":\"$start2\",\"end_date\":\"$end2\"}" \
    "$BASE/orders")"
  oid="$(json_get "$resp" "(d.get('data') or {}).get('order_id','')")"
  errno="$(json_get "$resp" "d.get('errno','')")"
  if [[ "$errno" == "0" && -n "$oid" ]]; then
    curl -s -c "$c_tenant" -b "$c_tenant" -X POST "$BASE/orders/${oid}/payment" >/dev/null
    wait_order_status "$oid" "WAIT_ACCEPT" 20 >/dev/null || true
    mysql_q "UPDATE order_house SET comment='seed-api-wait-accept' WHERE id=$oid"
    log "  order=$oid status=$(mysql_q "SELECT status FROM order_house WHERE id=$oid")"
  else
    log "  下单失败: $resp"
  fi

  log "真链路3: 下单保留待支付 house=$h3 ($start3~$end3)"
  login "$TENANT_C" "$c_tenant"
  resp="$(curl -s -c "$c_tenant" -b "$c_tenant" -H "Content-Type: application/json" \
    -d "{\"house_id\":\"$h3\",\"start_date\":\"$start3\",\"end_date\":\"$end3\"}" \
    "$BASE/orders")"
  oid="$(json_get "$resp" "(d.get('data') or {}).get('order_id','')")"
  if [[ -n "$oid" ]]; then
    mysql_q "UPDATE order_house SET comment='seed-api-wait-pay' WHERE id=$oid"
    log "  order=$oid WAIT_PAY"
  else
    log "  下单失败: $resp"
  fi
}

seed_orders_sql() {
  local marker
  marker="$(mysql_q "SELECT id FROM order_house WHERE comment='seed-sql-sample' LIMIT 1")"
  if [[ -n "$marker" ]]; then
    log "SQL 订单样例已存在，跳过"
    return 0
  fi

  log "写入 SQL 状态样例订单..."
  mysql_q "
INSERT INTO order_house
  (created_at, updated_at, user_id, house_id, begin_date, end_date, days, house_price, amount, status, comment, credit)
SELECT NOW(), NOW(),
  (SELECT id FROM user WHERE mobile='$TENANT_A'),
  h.id,
  DATE_SUB(CURDATE(), INTERVAL 30 DAY),
  DATE_SUB(CURDATE(), INTERVAL 28 DAY),
  2, h.price, h.price*2, 'CANCELLED', 'seed-sql-sample', 0
FROM house h WHERE h.title LIKE 'demo-%' AND h.deleted_at IS NULL ORDER BY h.id LIMIT 1;

INSERT INTO order_house
  (created_at, updated_at, user_id, house_id, begin_date, end_date, days, house_price, amount, status, comment, credit)
SELECT NOW(), NOW(),
  (SELECT id FROM user WHERE mobile='$TENANT_B'),
  h.id,
  DATE_SUB(CURDATE(), INTERVAL 20 DAY),
  DATE_SUB(CURDATE(), INTERVAL 18 DAY),
  2, h.price, h.price*2, 'WAIT_COMMENT', 'seed-sql-sample', 0
FROM house h WHERE h.title LIKE 'demo-%' AND h.deleted_at IS NULL ORDER BY h.id LIMIT 1 OFFSET 2;

INSERT INTO order_house
  (created_at, updated_at, user_id, house_id, begin_date, end_date, days, house_price, amount, status, comment, credit)
SELECT NOW(), NOW(),
  (SELECT id FROM user WHERE mobile='$TENANT_C'),
  h.id,
  DATE_SUB(CURDATE(), INTERVAL 10 DAY),
  DATE_SUB(CURDATE(), INTERVAL 8 DAY),
  2, h.price, h.price*2, 'COMPLETE', 'seed-sql-sample', 1
FROM house h WHERE h.title LIKE 'demo-%' AND h.deleted_at IS NULL ORDER BY h.id LIMIT 1 OFFSET 4;

INSERT INTO order_house
  (created_at, updated_at, user_id, house_id, begin_date, end_date, days, house_price, amount, status, comment, credit)
SELECT NOW(), NOW(),
  (SELECT id FROM user WHERE mobile='$TENANT_A'),
  h.id,
  DATE_SUB(CURDATE(), INTERVAL 5 DAY),
  DATE_SUB(CURDATE(), INTERVAL 3 DAY),
  2, h.price, h.price*2, 'REJECTED', 'seed-sql-sample', 0
FROM house h WHERE h.title LIKE 'demo-%' AND h.deleted_at IS NULL ORDER BY h.id LIMIT 1 OFFSET 6;
"
  log "SQL 样例写入完成"
}

print_summary() {
  log "==== 灌数结果 ===="
  mysql_q "
SELECT 'demo_houses' t, COUNT(*) c FROM house WHERE title LIKE 'demo-%' AND deleted_at IS NULL
UNION ALL SELECT 'houses_with_minio', COUNT(*) FROM house WHERE index_image_url LIKE '%/ihome/%' AND deleted_at IS NULL
UNION ALL SELECT 'local_img_left', COUNT(*) FROM house WHERE deleted_at IS NULL AND index_image_url <> '' AND index_image_url NOT LIKE 'http%'
UNION ALL SELECT 'house_image', COUNT(*) FROM house_image
UNION ALL SELECT 'ut_houses_left', COUNT(*) FROM house WHERE title LIKE 'ut-%'
UNION ALL SELECT 'orders', COUNT(*) FROM order_house WHERE deleted_at IS NULL;
SELECT status, COUNT(*) c FROM order_house WHERE deleted_at IS NULL GROUP BY status;
"
  log "房东: $LANDLORD_A / $LANDLORD_B  房客: $TENANT_A~$TENANT_C  密码: $PASS"
  log "首页: http://localhost:8080/home/index.html"
}

main() {
  ensure_ready
  if [[ "$CLEAN_UT" == "1" ]]; then
    clean_ut
  else
    log "跳过 ut-* 清理 (CLEAN_UT=$CLEAN_UT)"
  fi
  seed_houses
  migrate_legacy_local_images
  if [[ "$SKIP_ORDERS" != "1" ]]; then
    seed_orders_api
    seed_orders_sql
  else
    log "跳过订单灌数"
  fi
  print_summary
  log "完成"
}

main "$@"
