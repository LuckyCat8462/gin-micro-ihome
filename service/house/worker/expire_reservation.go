package worker

import (
	"fmt"
	"house/model"
	houseservice "house/service"
	"time"
)

// StartExpireReservationScanner 定期释放已过期的 RESERVED 预留
func StartExpireReservationScanner() {
	go func() {
		for {
			n, err := model.ReleaseExpiredReservations()
			if err != nil {
				fmt.Println("expire reservation scanner error", err)
			} else if n > 0 {
				fmt.Println("released expired reservations", n)
				houseservice.InvalidateHouseListCache()
			}
			time.Sleep(30 * time.Second)
		}
	}()
}
