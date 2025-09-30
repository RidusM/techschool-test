//nolint:mnd
package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"os/signal"
	"syscall"
	"time"

	"wbtest/internal/entity"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func main() {
	kafkaBrokers := flag.String(
		"brokers",
		"kafka:29092",
		"Kafka bootstrap brokers to connect to, as a comma separated list",
	)
	kafkaTopic := flag.String("topic", "orders-dev", "Kafka topic to write messages to")
	numMessages := flag.Int("count", 1, "Number of messages to send")
	interval := flag.Duration("interval", 1*time.Second, "Interval between sending messages")

	flag.Parse()

	writer := &kafka.Writer{
		Addr:     kafka.TCP(*kafkaBrokers),
		Topic:    *kafkaTopic,
		Balancer: &kafka.LeastBytes{},
	}
	defer writer.Close()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	log.Printf(
		"Starting Kafka producer. Will send %d messages to topic '%s' at broker(s) '%s' every %v\n",
		*numMessages,
		*kafkaTopic,
		*kafkaBrokers,
		*interval,
	)

	ticker := time.NewTicker(*interval)
	defer ticker.Stop()

	messagesSent := 0

	sendMessage(ctx, writer)

	messagesSent++
	if messagesSent >= *numMessages {
		log.Printf("Sent all %d messages. Exiting.\n", *numMessages)
		return
	}

	for {
		select {
		case <-ctx.Done():
			log.Println("Shutting down producer...")
			return
		case <-ticker.C:
			sendMessage(ctx, writer)
			messagesSent++
			if messagesSent >= *numMessages {
				log.Printf("Sent all %d messages. Exiting.\n", *numMessages)
				return
			}
		}
	}
}

func sendMessage(ctx context.Context, writer *kafka.Writer) {
	order := generateFakeOrder()
	orderBytes, err := json.Marshal(order)
	if err != nil {
		log.Printf("Failed to marshal order: %v", err)
		return
	}

	msg := kafka.Message{
		Key:   []byte(order.OrderUID.String()),
		Value: orderBytes,
	}

	writeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err = writer.WriteMessages(writeCtx, msg)
	if err != nil {
		log.Printf("Failed to write message to Kafka: %v", err)
	}

	log.Printf("Successfully sent order UID: %s", order.OrderUID.String())
}

func generateFakeDelivery() *entity.Delivery {
	return &entity.Delivery{
		Name:    gofakeit.Name(),
		Phone:   gofakeit.Phone(),
		Zip:     gofakeit.Zip(),
		City:    gofakeit.City(),
		Address: gofakeit.Address().Address,
		Region:  gofakeit.State(),
		Email:   gofakeit.Email(),
	}
}

func generateFakePayment() *entity.Payment {
	return &entity.Payment{
		Transaction:  uuid.New(),
		RequestID:    uuid.New(),
		Currency:     gofakeit.CurrencyShort(),
		Provider:     gofakeit.Word(),
		Amount:       uint64(gofakeit.UintRange(1000, 10000)),
		PaymentDt:    gofakeit.DateRange(time.Now().AddDate(-1, 0, 0), time.Now()).Unix(),
		Bank:         gofakeit.BS(),
		DeliveryCost: uint64(gofakeit.UintRange(100, 500)),
		GoodsTotal:   uint64(gofakeit.UintRange(500, 9000)),
		CustomFee:    uint64(gofakeit.UintRange(0, 100)),
	}
}

func generateFakeItem() *entity.Item {
	return &entity.Item{
		ChrtID:      uint64(gofakeit.UintRange(10000, 99999)),
		TrackNumber: gofakeit.UUID(),
		Price:       uint64(gofakeit.UintRange(100, 1000)),
		Rid:         uuid.New(),
		Name:        gofakeit.ProductName(),
		Sale:        gofakeit.Number(0, 50),
		Size:        gofakeit.Word(),
		TotalPrice:  uint64(gofakeit.UintRange(50, 950)),
		NMID:        uint64(gofakeit.UintRange(1000000, 9999999)),
		Brand:       gofakeit.Company(),
		Status:      gofakeit.Number(1, 5),
	}
}

func generateFakeOrder() *entity.Order {
	orderUID := uuid.New()
	itemsCount := gofakeit.Number(1, 5)
	items := make([]*entity.Item, 0, itemsCount)

	for range itemsCount {
		items = append(items, generateFakeItem())
	}

	return &entity.Order{
		OrderUID:          orderUID,
		TrackNumber:       gofakeit.UUID(),
		Entry:             gofakeit.LetterN(10),
		Delivery:          generateFakeDelivery(),
		Payment:           generateFakePayment(),
		Items:             items,
		Locale:            gofakeit.LetterN(2),
		InternalSignature: gofakeit.UUID(),
		CustomerID:        gofakeit.Username(),
		DeliveryService:   gofakeit.Word(),
		Shardkey:          gofakeit.Word(),
		SmID:              gofakeit.Number(1, 10),
		DateCreated:       gofakeit.Date(),
		OofShard:          gofakeit.LetterN(1),
	}
}
