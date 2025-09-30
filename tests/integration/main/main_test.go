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
	"github.com/stretchr/testify/suite"
)

type IntegrationTestSuite struct {
	suite.Suite

	db           *postgres.Postgres
	orderService *service.OrderService
	cfg          *config.Config
}

func (s *IntegrationTestSuite) SetupSuite() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cfg, err := config.Load()
	s.Require().NoError(err, "Failed to load configuration")
	s.cfg = cfg

	testLogger, err := logger.NewAdapter(cfg)
	s.Require().NoError(err)

	maxRetries := 10
	var db *postgres.Postgres

	for i := range maxRetries {
		db, err = postgres.NewPostgres(&cfg.Postgres, testLogger)
		if err == nil {
			break
		}
		testLogger.Info("Waiting for database to be ready...", "attempt", i+1, "error", err.Error())
		time.Sleep(5 * time.Second)
	}
	s.Require().NoError(err, "Failed to connect to postgres after retries")
	s.db = db

	err = db.Pool.Ping(ctx)
	s.Require().NoError(err, "Failed to ping database")

	txManager, err := transaction.NewManager(db, testLogger, metric.NewFactory().Transaction())
	s.Require().NoError(err)

	orderRepo := repository.NewOrderRepository(db)
	deliveryRepo := repository.NewDeliveryRepository(db)
	paymentRepo := repository.NewPaymentRepository(db)
	itemRepo := repository.NewItemRepository(db)

	orderCache, err := cache.NewLRUCache[uuid.UUID, *entity.Order](
		cfg.Cache.Capacity,
		testLogger,
		metric.NewFactory().Cache(),
	)
	s.Require().NoError(err)

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

func (s *IntegrationTestSuite) TearDownSuite() {
	if s.db != nil {
		s.db.Pool.Close()
	}
}

func (s *IntegrationTestSuite) TearDownTest() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := s.db.Pool.Exec(
		ctx,
		"TRUNCATE TABLE items, payment, delivery, orders RESTART IDENTITY CASCADE;",
	)
	s.Require().NoError(err)
}

func (s *IntegrationTestSuite) TestCreateAndGetOrder() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fakeOrder := generateFakeOrder()

	createdOrder, err := s.orderService.CreateOrder(ctx, fakeOrder)
	s.Require().NoError(err)
	s.Require().NotNil(createdOrder)
	s.Require().Equal(fakeOrder.OrderUID, createdOrder.OrderUID)

	retrievedOrder, err := s.orderService.GetOrder(ctx, fakeOrder.OrderUID)
	s.Require().NoError(err)
	s.Require().NotNil(retrievedOrder)

	s.Require().Equal(fakeOrder.OrderUID, retrievedOrder.OrderUID)
	s.Require().Equal(fakeOrder.TrackNumber, retrievedOrder.TrackNumber)

	s.Require().NotNil(retrievedOrder.Delivery)
	s.Require().Equal(fakeOrder.Delivery.Email, retrievedOrder.Delivery.Email)

	s.Require().NotNil(retrievedOrder.Payment)
	s.Require().Equal(fakeOrder.Payment.Transaction, retrievedOrder.Payment.Transaction)

	s.Require().Len(retrievedOrder.Items, len(fakeOrder.Items))
	if len(fakeOrder.Items) > 0 && len(retrievedOrder.Items) > 0 {
		s.Require().Equal(fakeOrder.Items[0].ChrtID, retrievedOrder.Items[0].ChrtID)
	}
}

func TestIntegration(t *testing.T) {
	t.Parallel()
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
