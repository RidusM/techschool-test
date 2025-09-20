package kafkat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"wbtest/internal/entity"
	"wbtest/internal/service"
	"wbtest/pkg/kafka/dlq"
	"wbtest/pkg/logger"

	"github.com/segmentio/kafka-go"
)

const (
	_defaultCacheBatchSize    = 10
	_defualtDLQProcessTimeout = 30 * time.Second
	_defaultDLQHandleTimeout  = 2 * time.Second
)

type DLQProcessor struct {
	dlqReader  *kafka.Reader
	dlq        *dlq.DLQ
	svc        *service.OrderService
	maxRetries int
	log        logger.Logger
}

type OrderService interface {
	CreateOrder(ctx context.Context, order *entity.Order) (*entity.Order, error)
	GetOrder(ctx context.Context, orderUID string) (*entity.Order, error)
}

func NewDLQProcessor(
	reader *kafka.Reader,
	dlq *dlq.DLQ,
	svc *service.OrderService,
	maxRetries int,
	log logger.Logger,
) *DLQProcessor {
	return &DLQProcessor{
		dlqReader:  reader,
		dlq:        dlq,
		svc:        svc,
		maxRetries: maxRetries,
		log:        log,
	}
}

func (p *DLQProcessor) Start(ctx context.Context) error {
	ticker := time.NewTicker(_defaultCacheBatchSize)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			p.log.Infow("dlq processor shutting down")
			if err := ctx.Err(); err != nil {
				return fmt.Errorf("transport.kafka.dlq_processor.Start: %w", err)
			}
			return nil
		case <-ticker.C:
			p.processBatch(ctx)
		}
	}
}

func (p *DLQProcessor) processBatch(ctx context.Context) {
	processCtx, cancel := context.WithTimeout(ctx, _defualtDLQProcessTimeout)
	defer cancel()

	msg, err := p.dlqReader.ReadMessage(processCtx)
	if err != nil {
		if errors.Is(processCtx.Err(), context.Canceled) {
			return
		}
		p.log.Errorw("read dlq message", "error", err)
		return
	}

	var dlqMsg struct {
		Metadata struct {
			RetryCount int `json:"retry_count"`
		} `json:"metadata"`
		Payload string `json:"payload"`
	}

	if err = json.Unmarshal(msg.Value, &dlqMsg); err != nil {
		p.log.Errorw("unmarshal dlq message",
			"error", err,
			"offset", msg.Offset,
		)
		return
	}

	if dlqMsg.Metadata.RetryCount >= p.maxRetries {
		p.log.Infow("skipping dlq message after max retries",
			"offset", msg.Offset,
			"retry_count", dlqMsg.Metadata.RetryCount,
		)
		return
	}

	var order entity.Order
	if err = json.Unmarshal([]byte(dlqMsg.Payload), &order); err != nil {
		p.log.Errorw("unmarshal dlq payload",
			"error", err,
			"offset", msg.Offset,
		)
		return
	}

	_, err = p.svc.GetOrder(processCtx, order.OrderUID)
	if err == nil {
		p.log.Infow("order already exists, skipping",
			"order_uid", order.OrderUID,
			"offset", msg.Offset)
		return
	}

	handleCtx, handleCancel := context.WithTimeout(processCtx, _defaultDLQHandleTimeout)
	defer handleCancel()

	if _, err = p.svc.CreateOrder(handleCtx, &order); err != nil {
		p.log.Errorw("retry dlq message",
			"error", err,
			"offset", msg.Offset,
			"retry_count", dlqMsg.Metadata.RetryCount,
		)

		var dlqSendErr error
		for i := range 3 {
			dlqSendErr = p.dlq.Send(handleCtx, kafka.Message{
				Topic:     msg.Topic,
				Partition: msg.Partition,
				Offset:    msg.Offset,
				Key:       msg.Key,
				Value:     msg.Value,
			}, err, dlqMsg.Metadata.RetryCount+1)

			if dlqSendErr == nil {
				break
			}

			p.log.Warnw("failed to send to DLQ, retrying",
				"retry", i+1,
				"error", dlqSendErr)

			time.Sleep(100 * time.Millisecond * time.Duration(i+1))
		}

		if dlqSendErr != nil {
			p.log.Errorw("failed to send to DLQ after retries",
				"offset", msg.Offset,
				"retry_count", dlqMsg.Metadata.RetryCount+1,
				"error", dlqSendErr,
			)
		}
	} else {
		p.log.Infow("dlq message processed successfully",
			"offset", msg.Offset,
			"order_uid", order.OrderUID.String(),
		)
	}
}
