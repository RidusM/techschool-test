package main

import (
	"context"
	"os"
	"testing"
	"time"

	"wbtest/internal/config"
	"wbtest/internal/entity"
	"wbtest/internal/repository"
	"wbtest/internal/service"
	"wbtest/pkg/cache"
	"wbtest/pkg/logger"
	"wbtest/pkg/metric"
	"wbtest/pkg/storage/postgres"
	"wbtest/pkg/storage/postgres/transaction"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type IntegrationTestSuite struct {
	suite.Suite

	db           *postgres.Postgres
	orderService *service.OrderService
	cfg          *config.Config
}

func (s *IntegrationTestSuite) SetupSuite() {
	cfg, err := config.Load()
	require.NoError(s.T(), err, "Failed to load configuration")

	s.cfg = cfg

	testLogger, err := logger.NewAdapter(
		cfg,
	)
	require.NoError(s.T(), err)

	db, err := postgres.NewPostgres(
		&cfg.Postgres,
		testLogger,
	)
	require.NoError(s.T(), err, "Failed to connect to postgres")
	s.db = db

	txManager, err := transaction.NewManager(db, testLogger, metric.NewFactory().Transaction())
	require.NoError(s.T(), err)

	orderRepo := repository.NewOrderRepository(db)
	deliveryRepo := repository.NewDeliveryRepository(db)
	paymentRepo := repository.NewPaymentRepository(db)
	itemRepo := repository.NewItemRepository(db)

	orderCache, err := cache.NewLRUCache[uuid.UUID, *entity.Order](
		cfg.Cache.Capacity,
		testLogger,
		metric.NewFactory().Cache(),
	)
	require.NoError(s.T(), err)

	s.orderService = service.NewOrderService(
		deliveryRepo,
		itemRepo,
		orderRepo,
		paymentRepo,
		txManager,
		testLogger,
		orderCache,
		cfg.Cache.TTL,
	)
}

func (s *IntegrationTestSuite) TearDownTest() {
	ctx := context.Background()
	_, err := s.db.Pool.Exec(
		ctx,
		"TRUNCATE TABLE items, payment, delivery, orders RESTART IDENTITY CASCADE;",
	)
	require.NoError(s.T(), err)
}

func (s *IntegrationTestSuite) TestCreateAndGetOrder() {
	ctx := context.Background()
	fakeOrder := generateFakeOrder()

	createdOrder, err := s.orderService.CreateOrder(ctx, fakeOrder)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), createdOrder)
	require.Equal(s.T(), fakeOrder.OrderUID, createdOrder.OrderUID)

	retrievedOrder, err := s.orderService.GetOrder(ctx, fakeOrder.OrderUID)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), retrievedOrder)

	require.Equal(s.T(), fakeOrder.OrderUID, retrievedOrder.OrderUID)
	require.Equal(s.T(), fakeOrder.TrackNumber, retrievedOrder.TrackNumber)
	require.NotNil(s.T(), retrievedOrder.Delivery)
	require.Equal(s.T(), fakeOrder.Delivery.Email, retrievedOrder.Delivery.Email)
	require.NotNil(s.T(), retrievedOrder.Payment)
	require.Equal(s.T(), fakeOrder.Payment.Transaction, retrievedOrder.Payment.Transaction)
	require.Len(s.T(), retrievedOrder.Items, len(fakeOrder.Items))
	require.Equal(s.T(), fakeOrder.Items[0].ChrtID, retrievedOrder.Items[0].ChrtID)
}

func TestIntegration(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") == "" {
		t.Skip("Skipping integration test; set INTEGRATION_TEST to run.")
	}
	suite.Run(t, new(IntegrationTestSuite))
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
