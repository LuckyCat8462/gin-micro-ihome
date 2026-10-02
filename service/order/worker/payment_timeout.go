package worker

import (
	"fmt"
	"order/conf"
	"order/model"
	"order/mq"
	"order/service"
	"time"

	"github.com/streadway/amqp"
)

// StartPaymentTimeoutScanner 扫描超时未支付订单并取消
func StartPaymentTimeoutScanner() {
	go func() {
		for {
			scanTimeoutOrders()
			time.Sleep(30 * time.Second)
		}
	}()
}

func scanTimeoutOrders() {
	list, err := model.ListTimeoutWaitPayOrders(conf.PayTimeoutMinutes)
	if err != nil {
		fmt.Println("scan timeout orders error", err)
		return
	}
	for _, o := range list {
		fmt.Println("支付超时取消订单", o.ID)
		if err := service.CancelOrder(fmt.Sprintf("%d", o.ID), ""); err != nil {
			fmt.Println("cancel timeout order failed", o.ID, err)
		}
	}
}

// StartStatusConsumer 消费库存确认消息，推进订单到 WAIT_ACCEPT
func StartStatusConsumer() {
	go func() {
		for {
			if err := runStatusConsumer(); err != nil {
				fmt.Println("status consumer error", err)
				time.Sleep(3 * time.Second)
			}
		}
	}()
}

func runStatusConsumer() error {
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
		StatusQueue:         conf.StatusQueue,
		StatusDLQ:           conf.StatusDLQ,
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

	deliveries, err := ch.Consume(conf.StatusQueue, "order-status", false, false, false, false, nil)
	if err != nil {
		return err
	}
	fmt.Println("order status consumer started")

	for d := range deliveries {
		handleStatusDelivery(ch, d)
	}
	return fmt.Errorf("status consumer channel closed")
}

func handleStatusDelivery(ch *amqp.Channel, d amqp.Delivery) {
	ev, err := mq.ParseEvent(d.Body)
	if err != nil {
		_ = d.Nack(false, false)
		return
	}
	if err := service.OnInventoryConfirmed(ev.OrderID); err != nil {
		fmt.Println("OnInventoryConfirmed failed", ev.OrderID, err)
		mq.FailOrDeadLetter(ch, d, conf.StatusQueue, conf.StatusDLQ, conf.ConsumeRetryMax)
		return
	}
	_ = d.Ack(false)
	fmt.Println("order inventory confirmed", ev.OrderID)
}

// StartReconcileJob 简单对账：PAID 但长时间未确认的订单重投 CONFIRM
func StartReconcileJob() {
	go func() {
		for {
			reconcilePaidOrders()
			time.Sleep(5 * time.Minute)
		}
	}()
}

func reconcilePaidOrders() {
	deadline := time.Now().Add(-2 * time.Minute)
	var list []model.OrderHouse
	err := model.GlobalConn.Where("status = ? AND updated_at < ?", model.OrderStatusPaid, deadline).
		Limit(50).Find(&list).Error
	if err != nil {
		return
	}
	for _, o := range list {
		var cnt int64
		model.GlobalConn.Model(&model.LocalMessage{}).
			Where("order_id = ? AND event_type = ?", o.ID, model.EventConfirm).
			Count(&cnt)
		if cnt == 0 {
			payload := fmt.Sprintf(`{"order_id":%d,"house_id":%d}`, o.ID, o.HouseId)
			_ = model.InsertOutbox(model.GlobalConn, o.ID, model.EventConfirm, payload)
			fmt.Println("reconcile re-enqueue CONFIRM", o.ID)
		}
	}
}
