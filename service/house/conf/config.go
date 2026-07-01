package conf

// mysql数据库有关配置
const MysqlName = "neko"
const MysqlPwd = "neko123456"
const MysqlAddr = "127.0.0.1"
const MysqlPort = "3306"
const MysqlDB = "search_house"

// Redis 缓存配置（与 web/conf 保持一致）
const RedisAddr = "192.168.81.128:6379"
const HouseListExpire = 300
const IndexHouseExpire = 300
const HouseDetailExpire = 600
