package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"house/conf"
	house "house/proto"

	"github.com/gomodule/redigo/redis"
)

const (
	houseListKey    = "houseList"
	indexHouseKey   = "indexHouse"
	houseDetailKey  = "houseDetail"
)

var ErrCacheMiss = errors.New("cache miss")

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
