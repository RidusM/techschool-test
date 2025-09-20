package e2e_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"wbtest/internal/entity"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type E2ETestSuite struct {
	suite.Suite

	kafkaWriter *kafka.Writer
	httpClient  *http.Client
	appHost     string
	appPort     string
}

func (s *E2ETestSuite) SetupSuite() {
	kafkaBrokers := getEnvOrDefault("KAFKA_BROKERS", "localhost:9092")
	s.appHost = getEnvOrDefault("APP_HOST", "localhost")
	s.appPort = getEnvOrDefault("APP_PORT", "8080")

	s.kafkaWriter = &kafka.Writer{
		Addr:     kafka.TCP(kafkaBrokers),
		Topic:    "orders",
		Balancer: &kafka.LeastBytes{},
	}
	s.httpClient = &http.Client{
		Timeout: 10 * time.Second,
	}

	s.waitForApp()
}

func (s *E2ETestSuite) waitForApp() {
	const maxRetries = 30
	const retryDelay = 2 * time.Second
	hostport := net.JoinHostPort(s.appHost, s.appPort)
	healthURL := fmt.Sprintf(
		"http://%s/healthz",
		hostport,
	)

	for i := range maxRetries {
		req, err := http.NewRequestWithContext(context.Background(), "GET", healthURL, nil)
		if err != nil {
			s.T().Logf("Failed to create health check request: %v", err)
			time.Sleep(retryDelay)
			continue
		}

		resp, err := s.httpClient.Do(req)
		if err != nil {
			s.T().Logf("Health check failed (attempt %d/%d): %v", i+1, maxRetries, err)
			time.Sleep(retryDelay)
			continue
		}
		resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			s.T().Log("App is healthy")
			return
		} else {
			s.T().Logf("App health check status %d (attempt %d/%d)", resp.StatusCode, i+1, maxRetries)
		}
		time.Sleep(retryDelay)
	}
	s.T().Fatalf("App did not become healthy after %d attempts", maxRetries)
}

func (s *E2ETestSuite) TearDownSuite() {
	if s.kafkaWriter != nil {
		s.kafkaWriter.Close()
	}
}

func (s *E2ETestSuite) TestOrderFlow() {
	order := generateFakeOrder()
	orderBytes, err := json.Marshal(order)
	require.NoError(s.T(), err)

	err = s.kafkaWriter.WriteMessages(context.Background(),
		kafka.Message{
			Key:   []byte(order.OrderUID.String()),
			Value: orderBytes,
		},
	)
	require.NoError(s.T(), err, "Failed to write message to Kafka")

	time.Sleep(5 * time.Second)

	hostport := net.JoinHostPort(s.appHost, s.appPort)
	url := fmt.Sprintf(
		"http://%s/api/v1/order/%s",
		hostport,
		order.OrderUID.String(),
	)
	s.T().Logf("Making request to: %s", url)
	req, err := http.NewRequestWithContext(context.Background(), "GET", url, nil)
	require.NoError(s.T(), err)

	resp, err := s.httpClient.Do(req)
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	require.Equal(
		s.T(),
		http.StatusOK,
		resp.StatusCode,
		"Expected status OK, got %d. Response: %+v",
		resp.StatusCode,
		resp,
	)

	body, err := io.ReadAll(resp.Body)
	require.NoError(s.T(), err)

	s.T().Logf("Response body: %s", string(body))

	var responseOrder entity.Order
	err = json.Unmarshal(body, &responseOrder)
	require.NoError(s.T(), err, "Failed to unmarshal response body: %s", string(body))

	require.Equal(s.T(), order.OrderUID, responseOrder.OrderUID)
	require.Equal(s.T(), order.TrackNumber, responseOrder.TrackNumber)
	require.Equal(s.T(), order.Delivery.Name, responseOrder.Delivery.Name)
	require.Equal(s.T(), order.Payment.Amount, responseOrder.Payment.Amount)
	require.Len(s.T(), responseOrder.Items, len(order.Items))
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func TestE2E(t *testing.T) {
	if os.Getenv("E2E_TEST") == "" {
		t.Skip("Skipping E2E test; set E2E_TEST to run.")
	}
	suite.Run(t, new(E2ETestSuite))
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
		PaymentDt:    int64(gofakeit.DateRange(time.Now().AddDate(-1, 0, 0), time.Now()).Unix()),
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
		Entry:             gofakeit.Word(),
		Delivery:          generateFakeDelivery(),
		Payment:           generateFakePayment(),
		Items:             items,
		Locale:            gofakeit.Country(),
		InternalSignature: gofakeit.UUID(),
		CustomerID:        gofakeit.Username(),
		DeliveryService:   gofakeit.Word(),
		Shardkey:          gofakeit.Word(),
		SmID:              gofakeit.Number(1, 10),
		DateCreated:       gofakeit.Date().Format(time.RFC3339),
		OofShard:          gofakeit.Word(),
	}
}
