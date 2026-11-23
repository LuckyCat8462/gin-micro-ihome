package conf

// mysql数据库有关配置
const MysqlName = "neko"
const MysqlPwd = "neko123456"
const MysqlAddr = "127.0.0.1"
const MysqlPort = "3306"
const MysqlDB = "search_house"

// Redis 缓存配置（与 web/conf 保持一致）
const RedisAddr = "127.0.0.1:6379"
const HouseListExpire = 300
const IndexHouseExpire = 300
const HouseDetailExpire = 600
const HouseSearchPageSize = 10

// RabbitMQ
const AmqpURL = "amqp://guest:guest@127.0.0.1:5672/"
const OrderExchange = "ihome.order"
const InventoryQueue = "house.inventory"
const InventoryDLQ = "house.inventory.dlq"
const RoutingOrderPaid = "order.paid"
const RoutingOrderCancelled = "order.cancelled"
const RoutingInventoryConfirmed = "order.inventory.confirmed"

// 消费失败最多重试次数，超过后 Nack(requeue=false) 进入 DLQ
const ConsumeRetryMax = 3

// 预留过期时间（分钟）
const ReserveExpireMinutes = 15

// MinIO 对象存储（S3 兼容）
const MinioEndpoint = "127.0.0.1:9000"
const MinioAccessKey = "minioadmin"
const MinioSecretKey = "minioadmin"
const MinioBucket = "ihome"
const MinioUseSSL = false
const MinioPublicBase = "http://127.0.0.1:9000"
