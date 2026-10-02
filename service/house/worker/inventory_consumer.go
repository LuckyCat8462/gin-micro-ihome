package worker

import (
	"fmt"
	"house/conf"
	"house/mq"
	"house/service"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/streadway/amqp"
)

// StartInventoryConsumer 消费支付确认/取消消息
func StartInventoryConsumer() {
	go func() {
		for {
			if err := runInventoryConsumer(); err != nil {
				fmt.Println("inventory consumer error", err)
				time.Sleep(3 * time.Second)
			}
		}
	}()
}

func runInventoryConsumer() error {
	conn, err := mq.Dial(conf.AmqpURL)
	if err != nil {
		return err
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	topo := mq.Topology{
		Exchange:            conf.OrderExchange,
		InventoryQueue:      conf.InventoryQueue,
		InventoryDLQ:        conf.InventoryDLQ,
		RoutingPaid:         conf.RoutingOrderPaid,
		RoutingCancelled:    conf.RoutingOrderCancelled,
		RoutingInvConfirmed: conf.RoutingInventoryConfirmed,
	}
	if err := mq.SetupOrderTopology(ch, topo); err != nil {
		return err
	}
	if err := ch.Qos(1, 0, false); err != nil {
		return err
	}

	deliveries, err := ch.Consume(conf.InventoryQueue, "house-inventory", false, false, false, false, nil)
	if err != nil {
		return err
	}
	fmt.Println("house inventory consumer started")

	for d := range deliveries {
		handleInventoryDelivery(ch, d)
	}
	return fmt.Errorf("inventory consumer channel closed")
}

func handleInventoryDelivery(ch *amqp.Channel, d amqp.Delivery) {
	ev, err := mq.ParseEvent(d.Body)
	if err != nil {
		_ = d.Nack(false, false)
		return
	}

	switch d.RoutingKey {
	case conf.RoutingOrderPaid:
		houseID, err := service.ConfirmReservation(ev.OrderID)
		if err != nil {
			fmt.Println("ConfirmReservation failed", ev.OrderID, err)
			mq.FailOrDeadLetter(ch, d, conf.InventoryQueue, conf.InventoryDLQ, conf.ConsumeRetryMax)
			return
		}
		confirmEv := mq.Event{
			MessageID: uuid.New().String(),
			OrderID:   ev.OrderID,
			EventType: "INVENTORY_CONFIRMED",
			HouseID:   houseID,
			BeginDate: ev.BeginDate,
			EndDate:   ev.EndDate,
		}
		if err := mq.Publish(ch, conf.OrderExchange, conf.RoutingInventoryConfirmed, confirmEv); err != nil {
			fmt.Println("publish inventory confirmed failed", err)
			mq.FailOrDeadLetter(ch, d, conf.InventoryQueue, conf.InventoryDLQ, conf.ConsumeRetryMax)
			return
		}
		_ = d.Ack(false)
		fmt.Println("reservation confirmed", ev.OrderID, "house", houseID)

	case conf.RoutingOrderCancelled:
		if err := service.ReleaseReservation(ev.OrderID); err != nil {
			fmt.Println("ReleaseReservation failed", ev.OrderID, err)
			mq.FailOrDeadLetter(ch, d, conf.InventoryQueue, conf.InventoryDLQ, conf.ConsumeRetryMax)
			return
		}
		_ = d.Ack(false)
		fmt.Println("reservation released", ev.OrderID, strconv.FormatUint(uint64(ev.OrderID), 10))

	default:
		_ = d.Ack(false)
	}
}
