package worker

import (
	"fmt"
	"order/conf"
	"order/model"
	"order/mq"
	"time"
)

// StartOutboxPublisher 定时扫描 Outbox 并投递 RabbitMQ
func StartOutboxPublisher() {
	go func() {
		for {
			publishPending()
			time.Sleep(2 * time.Second)
		}
	}()
}

func publishPending() {
	list, err := model.ListPendingOutbox(50)
	if err != nil || len(list) == 0 {
		return
	}

	conn, err := mq.Dial(conf.AmqpURL)
	if err != nil {
		fmt.Println("outbox dial amqp failed", err)
		return
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		fmt.Println("outbox channel failed", err)
		return
	}
	defer ch.Close()

	topo := mq.Topology{
		Exchange:            conf.OrderExchange,
		InventoryQueue:      conf.InventoryQueue,
		StatusQueue:         conf.StatusQueue,
		RoutingPaid:         conf.RoutingOrderPaid,
		RoutingCancelled:    conf.RoutingOrderCancelled,
		RoutingInvConfirmed: conf.RoutingInventoryConfirmed,
	}
	_ = mq.SetupOrderTopology(ch, topo)

	for _, msg := range list {
		routingKey := conf.RoutingOrderPaid
		eventType := msg.EventType
		if msg.EventType == model.EventCancel {
			routingKey = conf.RoutingOrderCancelled
		}

		o, err := model.FindOrderByID(msg.OrderId)
		houseID := uint(0)
		begin, end := "", ""
		if err == nil {
			houseID = o.HouseId
			begin = o.Begin_date.Format("2006-01-02")
			end = o.End_date.Format("2006-01-02")
		}

		ev := mq.Event{
			MessageID: msg.MessageId,
			OrderID:   msg.OrderId,
			EventType: eventType,
			HouseID:   houseID,
			BeginDate: begin,
			EndDate:   end,
		}
		if err := mq.Publish(ch, conf.OrderExchange, routingKey, ev); err != nil {
			fmt.Println("outbox publish failed", msg.MessageId, err)
			next := time.Now().Add(time.Duration(1<<uint(msg.RetryCount)) * time.Second)
			if next.After(time.Now().Add(5 * time.Minute)) {
				next = time.Now().Add(5 * time.Minute)
			}
			_ = model.MarkOutboxRetry(msg.ID, msg.RetryCount+1, next)
			continue
		}
		_ = model.MarkOutboxSent(msg.ID)
		fmt.Println("outbox sent", msg.MessageId, routingKey)
	}
}
