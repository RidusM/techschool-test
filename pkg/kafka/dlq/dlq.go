package dlq

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"time"

	"wbtest/internal/config"
	"wbtest/pkg/logger"
	"wbtest/pkg/metric"

	"github.com/segmentio/kafka-go"
)

const (
	_defaultMaxAttempts    = 10
	_defaultBaseRetryDelay = 100 * time.Millisecond
	_defaultMaxRetryDelay  = 5 * time.Second

	_backoffMultiplier = 2
)

type DLQ struct {
	writer  *kafka.Writer
	log     logger.Logger
	metrics metric.DLQ

	MaxAttempts    int
	baseRetryDelay time.Duration
	maxRetryDelay  time.Duration
}

func NewDLQ(cfg config.DLQ, log logger.Logger, metrics metric.DLQ, opts ...Option) (*DLQ, error) {
	writer := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Brokers...),
		Topic:        cfg.Topic,
		Async:        false,
		BatchSize:    cfg.BatchSize,
		BatchTimeout: cfg.BatchTimeout,
		WriteTimeout: cfg.WriteTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		Logger: kafka.LoggerFunc(func(msg string, args ...any) {
			log.LogAttrs(context.Background(), logger.InfoLevel, "dlq writer info",
				logger.String("message", fmt.Sprintf(msg, args...)),
			)
		}),
		ErrorLogger: kafka.LoggerFunc(func(msg string, args ...any) {
			log.LogAttrs(context.Background(), logger.ErrorLevel, "dlq writer error",
				logger.String("error", fmt.Sprintf(msg, args...)),
			)
		}),
	}

	dlq := &DLQ{
		writer:  writer,
		log:     log,
		metrics: metrics,

		MaxAttempts:    _defaultMaxAttempts,
		baseRetryDelay: _defaultBaseRetryDelay,
		maxRetryDelay:  _defaultMaxRetryDelay,
	}

	for _, opt := range opts {
		opt(dlq)
	}

	if err := dlq.validate(); err != nil {
		return nil, fmt.Errorf("kafka.dlq.NewDLQ: validation: %w", err)
	}

	return dlq, nil
}

func (d *DLQ) Close() error {
	if err := d.writer.Close(); err != nil {
		return fmt.Errorf("kafka.dlq.Close: %w", err)
	}
	return nil
}

func (d *DLQ) Send(
	ctx context.Context,
	originalMsg kafka.Message,
	err error,
	retryCount int,
) error {
	const op = "kafka.dlq.Send"

	defer func() {
		if d.metrics != nil {
			d.metrics.DLSent(d.writer.Topic, originalMsg.Topic, retryCount)
		}
	}()

	metadata := map[string]interface{}{
		"original_topic": originalMsg.Topic,
		"partition":      originalMsg.Partition,
		"offset":         originalMsg.Offset,
		"retry_count":    retryCount,
		"error":          err.Error(),
		"timestamp":      time.Now().UTC().Format(time.RFC3339),
	}

	dlqMessage := map[string]interface{}{
		"metadata": metadata,
		"payload":  string(originalMsg.Value),
	}

	value, err := json.Marshal(dlqMessage)
	if err != nil {
		d.log.Errorw("failed to marshal dlq message",
			"op", op,
			"error", err,
			"original_offset", originalMsg.Offset,
			"payload_base64", base64.StdEncoding.EncodeToString(originalMsg.Value),
			"payload_encoding", "base64",
			"payload_size", len(originalMsg.Value),
		)
		fallbackMsg := map[string]interface{}{
			"error":          "marshal_failed",
			"offset":         originalMsg.Offset,
			"size":           len(originalMsg.Value),
			"key_base64":     base64.StdEncoding.EncodeToString(originalMsg.Key),
			"partition":      originalMsg.Partition,
			"original_topic": originalMsg.Topic,
		}
		fallbackBytes, fallbackErr := json.Marshal(fallbackMsg)
		if fallbackErr != nil {
			d.log.Errorw("critical: failed to marshal even fallback DLQ message",
				"original_offset", originalMsg.Offset,
				"original_size", len(originalMsg.Value),
				"fallback_error", fallbackErr,
			)

			if err = d.writer.WriteMessages(ctx, kafka.Message{
				Key:   originalMsg.Key,
				Value: []byte(fmt.Sprintf("DLQ_FALLOUT:%d", originalMsg.Offset)),
			}); err != nil {
				return fmt.Errorf("%s: write DLQ fallback message: %w", op, err)
			}
		}

		if err = d.writer.WriteMessages(ctx, kafka.Message{
			Key:   originalMsg.Key,
			Value: fallbackBytes,
		}); err != nil {
			return fmt.Errorf("%s: write DLQ message: %w", op, err)
		}
	}

	err = d.writer.WriteMessages(ctx, kafka.Message{
		Key:   originalMsg.Key,
		Value: value,
	})
	if err != nil {
		d.log.Errorw("failed to send message to dlq",
			"op", op,
			"error", err,
			"offset", originalMsg.Offset,
		)

		if d.metrics != nil {
			d.metrics.DLError(d.writer.Topic, "write_failed")
		}

		return fmt.Errorf("%s: send message: %w", op, err)
	}

	d.log.Infow("message sent to dlq",
		"op", op,
		"topic", d.writer.Topic,
		"offset", originalMsg.Offset,
		"retry_count", retryCount,
	)

	return nil
}

func ProcessWithRetry(
	ctx context.Context,
	msg kafka.Message,
	handler func(context.Context, kafka.Message) error,
	dlq *DLQ,
	log logger.Logger,
) error {
	const op = "kafka.dlq.ProcessWithRetry"
	var err error
	var attemptCount int
	currentBackoff := dlq.baseRetryDelay
	for attemptCount = 1; attemptCount <= dlq.MaxAttempts; attemptCount++ {
		if err = ctx.Err(); err != nil {
			return fmt.Errorf("%s: context: %w", op, err)
		}
		jitter := time.Duration(
			rand.Int64N(int64(currentBackoff * _backoffMultiplier)),
		)
		if jitter > dlq.maxRetryDelay {
			jitter = dlq.maxRetryDelay
		}

		log.LogAttrs(ctx, logger.InfoLevel, "Retrying message processing",
			logger.String("operation", op),
			logger.Int("attempt", attemptCount),
			logger.String("retry_after", jitter.String()),
			logger.Any("error", ctx.Err()),
		)
		select {
		case <-time.After(jitter):
		case <-ctx.Done():
			return fmt.Errorf("%s: context done: %w", op, ctx.Err())
		}

		err = handler(ctx, msg)
		if err == nil {
			return nil
		}

		log.LogAttrs(ctx, logger.ErrorLevel, "message processing failed",
			logger.String("operation", op),
			logger.Int64("offset", msg.Offset),
			logger.Int("retry_count", attemptCount),
			logger.Any("error", err),
		)
		nextBackoff := currentBackoff * _backoffMultiplier
		if nextBackoff > dlq.maxRetryDelay {
			nextBackoff = dlq.maxRetryDelay
		}
		currentBackoff = nextBackoff
	}

	return dlq.Send(ctx, msg, err, attemptCount-1)
}
