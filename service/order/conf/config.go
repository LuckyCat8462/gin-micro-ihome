package conf

// mysql数据库有关配置
const MysqlName = "neko"
const MysqlPwd = "neko123456"
const MysqlAddr = "127.0.0.1"
const MysqlPort = "3306"
const MysqlDB = "search_house"

// RabbitMQ
const AmqpURL = "amqp://guest:guest@127.0.0.1:5672/"
const OrderExchange = "ihome.order"
const InventoryQueue = "house.inventory"
const StatusQueue = "order.status"
const StatusDLQ = "order.status.dlq"
const RoutingOrderPaid = "order.paid"
const RoutingOrderCancelled = "order.cancelled"
const RoutingInventoryConfirmed = "order.inventory.confirmed"

// 消费失败最多重试次数，超过后进入 DLQ
const ConsumeRetryMax = 3

// 支付超时（分钟）
const PayTimeoutMinutes = 15
