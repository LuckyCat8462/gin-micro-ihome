package conf

// mysql数据库有关配置
const MysqlName = "neko"
const MysqlPwd = "neko123456"
const MysqlAddr = "127.0.0.1"
const MysqlPort = "3306"
const MysqlDB = "search_house"

// RedisAddr Redis 连接地址（host:port）
// 本机 Redis 若仅绑定局域网 IP，使用 192.168.81.128:6379；
// 执行 MyShell/fix-redis-bind.sh 后可改为 127.0.0.1:6379
const RedisAddr = "192.168.81.128:6379"
