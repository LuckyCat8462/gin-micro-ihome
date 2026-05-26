package Learning

//启动redis指令
//redis-cli -h 127.0.0.1 -p 6379
//如果出现/452这种连续的乱码,说明出现了中文,想要查看的花,在启动redis时加上--raw的后缀
//redis-cli -h 127.0.0.1 -p 6379 --raw

//设置一个键值对
//SET mykey myvalue

//获取一个键的值：
//GET mykey

//查看所有键：
//KEYS *
//注意：KEYS命令在生产环境中通常不推荐使用，因为它会阻塞服务器直到列表返回，尤其是在有大量键的情况下。
//更好的选择是使用SCAN命令。

//使用SCAN命令迭代键（推荐）
//SCAN 0 MATCH * COUNT 1000

//删除一个键：
//DEL mykey

//查看当前数据库的键的数量：
//DBSIZE

//// redis.Pool ——  Ctrl-B 查看 redis库， 连接池属性
//type Pool struct {
//	Dial func() (Conn, error)	// 连接数据库使用
//	。。。。
//	MaxIdle int		// 最大空闲数 == 初始化连接数
//	MaxActive int	// 最大存活数 > MaxIdle
//	IdleTimeout time.Duration	// 空闲超时时间。
//	。。。。
//	MaxConnLifetime time.Duration	// 最大生命周期。
//	。。。。
//}

//检查一个键是否存在：
//
//EXISTS mykey
//
//清空当前数据库：
//
//FLUSHDB
//
//清空所有数据库（慎用）：
//
//FLUSHALL
//
//查看Redis服务器的信息：
//
//INFO
//
//退出redis-cli：
//
//exit
