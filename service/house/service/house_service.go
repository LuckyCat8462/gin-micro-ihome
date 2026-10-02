package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"house/conf"
	house "house/proto"

	"github.com/gomodule/redigo/redis"
	"golang.org/x/sync/singleflight"
)

const (
	houseListKey   = "houseList"
	indexHouseKey  = "indexHouse:v3"
	houseDetailKey = "houseDetail:v2"
)

var (
	ErrCacheMiss    = errors.New("cache miss")
	houseLoadGroup  singleflight.Group
)

func dialRedis() (redis.Conn, error) {
	return redis.Dial("tcp", conf.RedisAddr)
}

func cacheKey(aid, sd, ed, sk string) string {
	return fmt.Sprintf("%s:%s:%s:%s:%s", houseListKey, aid, sd, ed, sk)
}

func detailCacheKey(houseID string) string {
	return fmt.Sprintf("%s:%s", houseDetailKey, houseID)
}

func getFromCache(key string) ([]*house.Houses, error) {
	conn, err := dialRedis()
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	data, err := redis.Bytes(conn.Do("GET", key))
	if err == redis.ErrNil {
		return nil, ErrCacheMiss
	}
	if err != nil {
		return nil, err
	}

	var houses []*house.Houses
	if err := json.Unmarshal(data, &houses); err != nil {
		return nil, err
	}
	return houses, nil
}

func setCache(key string, houses []*house.Houses, expire int) error {
	conn, err := dialRedis()
	if err != nil {
		return err
	}
	defer conn.Close()

	buf, err := json.Marshal(houses)
	if err != nil {
		return err
	}
	_, err = conn.Do("SET", key, buf, "EX", expire)
	return err
}

// getOrLoadHouses Cache-Aside + singleflight，同一 key 过期时仅一个请求穿透 MySQL
func getOrLoadHouses(key string, expire int, loader func() ([]*house.Houses, error)) ([]*house.Houses, error) {
	if cached, err := getFromCache(key); err == nil {
		fmt.Println("从 Redis 获取", key)
		return cached, nil
	}

	v, err, shared := houseLoadGroup.Do(key, func() (interface{}, error) {
		if cached, err := getFromCache(key); err == nil {
			return cached, nil
		}
		data, err := loader()
		if err != nil {
			return nil, err
		}
		if err := setCache(key, data, expire); err != nil {
			fmt.Println("写入缓存失败", key, err)
		}
		return data, nil
	})
	if err != nil {
		return nil, err
	}
	if shared {
		fmt.Println("singleflight 等待共享结果", key)
	}
	return v.([]*house.Houses), nil
}

func getDetailFromCache(key string) (*house.DetailData, error) {
	conn, err := dialRedis()
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	data, err := redis.Bytes(conn.Do("GET", key))
	if err == redis.ErrNil {
		return nil, ErrCacheMiss
	}
	if err != nil {
		return nil, err
	}

	var detail house.DetailData
	if err := json.Unmarshal(data, &detail); err != nil {
		return nil, err
	}
	return &detail, nil
}

func setDetailCache(key string, detail *house.DetailData, expire int) error {
	conn, err := dialRedis()
	if err != nil {
		return err
	}
	defer conn.Close()

	buf, err := json.Marshal(detail)
	if err != nil {
		return err
	}
	_, err = conn.Do("SET", key, buf, "EX", expire)
	return err
}

func deleteKeysByPattern(pattern string) {
	conn, err := dialRedis()
	if err != nil {
		return
	}
	defer conn.Close()

	keys, _ := redis.Strings(conn.Do("KEYS", pattern))
	for _, k := range keys {
		conn.Do("DEL", k)
	}
}

// InvalidateHouseListCache 清除搜索列表缓存
func InvalidateHouseListCache() {
	deleteKeysByPattern(houseListKey + ":*")
}

// InvalidateIndexCache 清除首页轮播缓存
func InvalidateIndexCache() {
	conn, err := dialRedis()
	if err != nil {
		return
	}
	defer conn.Close()
	conn.Do("DEL", indexHouseKey)
}

// InvalidateDetailCache 清除指定房源详情缓存
func InvalidateDetailCache(houseID string) {
	conn, err := dialRedis()
	if err != nil {
		return
	}
	defer conn.Close()
	conn.Do("DEL", detailCacheKey(houseID))
}

// InvalidateAllDetailCache 清除全部详情缓存
func InvalidateAllDetailCache() {
	deleteKeysByPattern(houseDetailKey + ":*")
}

// InvalidateAllHouseCache 写操作后清除所有读缓存
func InvalidateAllHouseCache(houseID string) {
	InvalidateHouseListCache()
	InvalidateIndexCache()
	if houseID != "" {
		InvalidateDetailCache(houseID)
	} else {
		InvalidateAllDetailCache()
	}
}
