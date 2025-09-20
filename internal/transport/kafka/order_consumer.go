package kafkat

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"

	"wbtest/internal/entity"
	"wbtest/internal/service"
	"wbtest/pkg/kafka/dlq"
	"wbtest/pkg/logger"
	"wbtest/pkg/metric"

	"github.com/segmentio/kafka-go"
	"golang.org/x/sync/errgroup"
)

type DLQ interface {
	Send(ctx context.Context, msg kafka.Message, err error, retryCount int) error
}

type OrderConsumer struct {
	reader *kafka.Reader
	dlq    *dlq.DLQ
	svc    *service.OrderService
	metric metric.Kafka
	log    logger.Logger
}

func NewOrderConsumer(
	reader *kafka.Reader,
	dlq *dlq.DLQ,
	svc *service.OrderService,
	metric metric.Kafka,
	log logger.Logger,
) *OrderConsumer {
	return &OrderConsumer{
		reader: reader,
		dlq:    dlq,
		svc:    svc,
		metric: metric,
		log:    log,
	}
}

func (c *OrderConsumer) Start(ctx context.Context) error {
	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		return c.run(ctx)
	})

	eg.Go(func() error {
		<-ctx.Done()
		c.log.Infow("shutting down consumer")
		return c.reader.Close()
	})

	if err := eg.Wait(); err != nil {
		return fmt.Errorf("transport.kafka.order_consumer.Start: %w", err)
	}
	return nil
}

func (c *OrderConsumer) run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			if err := ctx.Err(); err != nil {
				return fmt.Errorf("transport.kafka.order_consumer.run: %w", err)
			}
			return nil
		default:
			msg, err := c.reader.ReadMessage(ctx)
			if err != nil {
				if errors.Is(ctx.Err(), context.Canceled) {
					return nil
				}
				c.log.Errorw("kafka read failed",
					"error", err,
				)
				continue
			}

			c.metric.MessageProcessed(msg.Topic, msg.Partition)
			c.processMessage(ctx, msg)
		}
	}
}

func (c *OrderConsumer) processMessage(ctx context.Context, msg kafka.Message) {
	c.log.Infow("processing kafka message",
		"topic", msg.Topic,
		"partition", msg.Partition,
		"offset", msg.Offset,
	)

	err := dlq.ProcessWithRetry(
		ctx,
		msg,
		c.handleMessage,
		c.dlq,
		c.log,
	)
	if err != nil {
		dlqErr := c.dlq.Send(ctx, msg, err, c.dlq.MaxAttempts)
		if dlqErr != nil {
			c.log.Errorw("critical: failed to send to DLQ after retries",
				"offset", msg.Offset,
				"original_error", err,
				"dlq_error", dlqErr,
			)
			c.log.Errorw("dlq fallback",
				"payload_hash", sha256.Sum256(msg.Value),
				"offset", msg.Offset,
			)
		} else {
			c.log.Infow("message sent to DLQ after max retries",
				"offset", msg.Offset,
				"retry_count", c.dlq.MaxAttempts,
			)
		}
		c.metric.MessageFailed(msg.Topic, msg.Partition, "retry_limit_exceeded")
	}
}

func (c *OrderConsumer) handleMessage(ctx context.Context, msg kafka.Message) error {
	const op = "transport.kafka.order_consumer.handleMessage"
	var order entity.Order
	if err := json.Unmarshal(msg.Value, &order); err != nil {
		return fmt.Errorf("%s: unmarshal order: %w", op, err)
	}

	if _, err := c.svc.CreateOrder(ctx, &order); err != nil {
		return fmt.Errorf("%s: create order: %w", op, err)
	}

	c.log.Infow("order saved from kafka",
		"order_uid", order.OrderUID.String(),
		"offset", msg.Offset,
	)

	return nil
}
