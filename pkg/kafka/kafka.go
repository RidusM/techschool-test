package kafka

import (
	"context"
	"fmt"

	"wbtest/internal/config"
	"wbtest/pkg/logger"

	"github.com/segmentio/kafka-go"
)

type contextKey string

const kafkaMetadataKey contextKey = "kafka_metadata"

func NewKafkaReader(cfg config.Kafka, log logger.Logger) (*kafka.Reader, error) {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: cfg.Brokers,
		Topic:   cfg.Topic,
		GroupID: cfg.GroupID,
		Logger: kafka.LoggerFunc(func(msg string, args ...any) {
			ctx := context.WithValue(context.Background(), kafkaMetadataKey, map[string]string{
				"topic":    cfg.Topic,
				"group_id": cfg.GroupID,
			})
			log.LogAttrs(ctx, logger.InfoLevel, "kafka reader info",
				logger.String("message", fmt.Sprintf(msg, args...)),
			)
		}),
		ErrorLogger: kafka.LoggerFunc(func(msg string, args ...any) {
			ctx := context.WithValue(context.Background(), kafkaMetadataKey, map[string]string{
				"topic":    cfg.Topic,
				"group_id": cfg.GroupID,
			})
			log.LogAttrs(ctx, logger.ErrorLevel, "kafka reader error",
				logger.String("error", fmt.Sprintf(msg, args...)),
			)
		}),
	})

	if err := checkKafkaConnection(cfg.Brokers, log); err != nil {
		return nil, err
	}

	return reader, nil
}

func checkKafkaConnection(brokers []string, log logger.Logger) error {
	const op = "kafka.checkKafkaConnection"

	dialer := &kafka.Dialer{}
	for _, broker := range brokers {
		conn, err := dialer.Dial("tcp", broker)
		if err != nil {
			return fmt.Errorf("%s: connect to %s: %w", op, broker, err)
		}

		if err = conn.Close(); err != nil {
			log.Warnw("failed to close connection",
				"operation", op,
				"broker", broker,
				"error", err)
		}
	}
	return nil
}
