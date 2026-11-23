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
const RedisAddr = "127.0.0.1:6379"

// SessionSecret Cookie 签名密钥（本地开发用，生产请更换）
const SessionSecret = "gin-micro-ihome-session-secret"

// RPCTimeoutSeconds 网关调用微服务默认超时（秒）
const RPCTimeoutSeconds = 5

// MinIO 对象存储（S3 兼容，与 house 服务保持一致）
const MinioEndpoint = "127.0.0.1:9000"
const MinioAccessKey = "minioadmin"
const MinioSecretKey = "minioadmin"
const MinioBucket = "ihome"
const MinioUseSSL = false
const MinioPublicBase = "http://127.0.0.1:9000"
