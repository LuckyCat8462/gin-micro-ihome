package mq

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/streadway/amqp"
)

// Event 统一消息信封
type Event struct {
	MessageID  string `json:"message_id"`
	OrderID    uint   `json:"order_id"`
	EventType  string `json:"event_type"` // CONFIRM / CANCEL / INVENTORY_CONFIRMED
	HouseID    uint   `json:"house_id"`
	BeginDate  string `json:"begin_date"`
	EndDate    string `json:"end_date"`
	CreatedAt  string `json:"created_at"`
}

type Topology struct {
	Exchange              string
	InventoryQueue        string
	InventoryDLQ          string
	StatusQueue           string
	RoutingPaid           string
	RoutingCancelled      string
	RoutingInvConfirmed   string
}

func Dial(url string) (*amqp.Connection, error) {
	return amqp.Dial(url)
}

// SetupOrderTopology 声明 exchange / queue / binding（idempotent）
// 注意：主队列不挂 x-dead-letter-*，避免与已存在队列参数冲突；毒消息由 FailOrDeadLetter 显式投递到 DLQ。
func SetupOrderTopology(ch *amqp.Channel, t Topology) error {
	if err := ch.ExchangeDeclare(t.Exchange, "topic", true, false, false, false, nil); err != nil {
		return err
	}

	if t.InventoryDLQ != "" {
		if _, err := ch.QueueDeclare(t.InventoryDLQ, true, false, false, false, nil); err != nil {
			return err
		}
	}

	if _, err := ch.QueueDeclare(t.InventoryQueue, true, false, false, false, nil); err != nil {
		return err
	}
	if err := ch.QueueBind(t.InventoryQueue, t.RoutingPaid, t.Exchange, false, nil); err != nil {
		return err
	}
	if err := ch.QueueBind(t.InventoryQueue, t.RoutingCancelled, t.Exchange, false, nil); err != nil {
		return err
	}

	if t.StatusQueue != "" {
		if _, err := ch.QueueDeclare(t.StatusQueue, true, false, false, false, nil); err != nil {
			return err
		}
		if err := ch.QueueBind(t.StatusQueue, t.RoutingInvConfirmed, t.Exchange, false, nil); err != nil {
			return err
		}
	}
	return nil
}

func Publish(ch *amqp.Channel, exchange, routingKey string, ev Event) error {
	if ev.CreatedAt == "" {
		ev.CreatedAt = time.Now().Format(time.RFC3339)
	}
	body, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	return ch.Publish(exchange, routingKey, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		MessageId:    ev.MessageID,
		Body:         body,
	})
}

func ParseEvent(body []byte) (Event, error) {
	var ev Event
	err := json.Unmarshal(body, &ev)
	if err != nil {
		fmt.Println("parse event error", err)
	}
	return ev, err
}

// HeaderRetryCount 读取自定义重试次数（非 x-death）
func HeaderRetryCount(h amqp.Table) int {
	if h == nil {
		return 0
	}
	v, ok := h["x-retry-count"]
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case int32:
		return int(n)
	case int64:
		return int(n)
	case int:
		return n
	case byte:
		return int(n)
	default:
		return 0
	}
}

// FailOrDeadLetter 失败重试：未超限则带 x-retry-count 重入队；超限则显式投递到 dlq 并 Ack
func FailOrDeadLetter(ch *amqp.Channel, d amqp.Delivery, queue, dlq string, maxRetry int) {
	n := HeaderRetryCount(d.Headers)
	if maxRetry <= 0 {
		maxRetry = 3
	}
	if n >= maxRetry {
		fmt.Println("consume retry exceeded, dead-letter", "queue", queue, "dlq", dlq, "retry", n, "msg", d.MessageId)
		if dlq != "" {
			_ = ch.Publish("", dlq, false, false, amqp.Publishing{
				ContentType:  d.ContentType,
				DeliveryMode: amqp.Persistent,
				MessageId:    d.MessageId,
				Headers:      d.Headers,
				Body:         d.Body,
			})
		}
		_ = d.Ack(false)
		return
	}
	headers := amqp.Table{}
	for k, v := range d.Headers {
		headers[k] = v
	}
	headers["x-retry-count"] = int32(n + 1)
	err := ch.Publish("", queue, false, false, amqp.Publishing{
		ContentType:  d.ContentType,
		DeliveryMode: amqp.Persistent,
		MessageId:    d.MessageId,
		Headers:      headers,
		Body:         d.Body,
	})
	if err != nil {
		fmt.Println("requeue with retry header failed", err)
		_ = d.Nack(false, true)
		return
	}
	_ = d.Ack(false)
	fmt.Println("requeued with retry", n+1, "msg", d.MessageId)
}
