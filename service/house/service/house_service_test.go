package service

import (
	house "house/proto"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestGetOrLoadHousesSingleflight(t *testing.T) {
	conn, err := dialRedis()
	if err != nil {
		t.Skip("redis unavailable:", err)
	}
	defer conn.Close()

	key := "test_singleflight_key"
	if _, err := conn.Do("DEL", key); err != nil {
		t.Fatalf("del key: %v", err)
	}

	var loads int32
	const n = 50
	var wg sync.WaitGroup
	wg.Add(n)

	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			_, err := getOrLoadHouses(key, 60, func() ([]*house.Houses, error) {
				atomic.AddInt32(&loads, 1)
				time.Sleep(20 * time.Millisecond)
				return []*house.Houses{{Title: "test"}}, nil
			})
			if err != nil {
				t.Errorf("getOrLoadHouses: %v", err)
			}
		}()
	}
	wg.Wait()

	if got := atomic.LoadInt32(&loads); got != 1 {
		t.Fatalf("expected loader called once, got %d", got)
	}
}
