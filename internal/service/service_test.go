package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"wbtest/internal/entity"
	mock_repository "wbtest/internal/repository/mock"
	"wbtest/internal/service"
	mock_cache "wbtest/pkg/cache/mock"
	mock_logger "wbtest/pkg/logger/mock"
	"wbtest/pkg/storage/postgres"
	mock_transaction "wbtest/pkg/storage/postgres/transaction/mock"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
)

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

type createOrderTestInput struct {
	order *entity.Order
}

type createOrderTestExpected struct {
	order *entity.Order
	err   error
}

func TestOrderService_CreateOrder(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		desc  string
		setup func() *entity.Order
		mocks func(
			orderRepo *mock_repository.MockOrderRepository,
			deliveryRepo *mock_repository.MockDeliveryRepository,
			paymentRepo *mock_repository.MockPaymentRepository,
			itemRepo *mock_repository.MockItemRepository,
			txManager *mock_transaction.MockManager,
			logger *mock_logger.MockLogger,
			cache *mock_cache.MockCache,
			order *entity.Order,
		)
		input    createOrderTestInput
		expected createOrderTestExpected
	}{
		{
			desc:  "Success",
			setup: generateFakeOrder,
			mocks: func(
				orderRepo *mock_repository.MockOrderRepository,
				deliveryRepo *mock_repository.MockDeliveryRepository,
				paymentRepo *mock_repository.MockPaymentRepository,
				itemRepo *mock_repository.MockItemRepository,
				txManager *mock_transaction.MockManager,
				logger *mock_logger.MockLogger,
				cache *mock_cache.MockCache,
				order *entity.Order,
			) {
				logger.EXPECT().Ctx(gomock.Any()).Return(logger).AnyTimes()

				orderRepo.EXPECT().GetByOrderUID(ctx, order.OrderUID).
					Return(nil, entity.ErrDataNotFound).Times(1)

				logger.EXPECT().
					LogAttrs(ctx, gomock.Any(), "create order started", gomock.Any()).
					Times(1)

				txManager.EXPECT().ExecuteInTransaction(
					ctx, "CreateOrder", gomock.Any(),
				).DoAndReturn(func(
					ctx context.Context,
					opName string,
					txFunc func(postgres.QueryExecuter) error,
				) error {
					return txFunc(nil)
				}).Times(1)

				orderRepo.EXPECT().Create(ctx, nil, gomock.Eq(order)).
					Return(order, nil).Times(1)

				deliveryRepo.EXPECT().
					Create(ctx, nil, order.OrderUID, gomock.Eq(order.Delivery)).
					Return(order.Delivery, nil).Times(1)

				paymentRepo.EXPECT().
					Create(ctx, nil, order.OrderUID, gomock.Eq(order.Payment)).
					Return(order.Payment, nil).Times(1)

				itemRepo.EXPECT().Create(
					ctx, nil, gomock.Eq(order.OrderUID), gomock.Eq(order.Items),
				).Return(nil).Times(1)

				cache.EXPECT().Put(order.OrderUID, gomock.Eq(order), gomock.Any()).Times(1)

				logger.EXPECT().
					LogAttrs(ctx, gomock.Any(), "order created successfully", gomock.Any()).
					Times(1)
			},
			input: createOrderTestInput{
				order: nil,
			},
			expected: createOrderTestExpected{
				order: nil,
				err:   nil,
			},
		},
		{
			desc:  "DuplicateOrder",
			setup: generateFakeOrder,
			mocks: func(
				orderRepo *mock_repository.MockOrderRepository,
				deliveryRepo *mock_repository.MockDeliveryRepository,
				paymentRepo *mock_repository.MockPaymentRepository,
				itemRepo *mock_repository.MockItemRepository,
				txManager *mock_transaction.MockManager,
				logger *mock_logger.MockLogger,
				cache *mock_cache.MockCache,
				order *entity.Order,
			) {
				logger.EXPECT().Ctx(gomock.Any()).Return(logger).AnyTimes()

				orderRepo.EXPECT().GetByOrderUID(ctx, order.OrderUID).
					Return(order, nil).Times(1)
			},
			input: createOrderTestInput{order: nil},
			expected: createOrderTestExpected{
				order: nil,
				err:   nil,
			},
		},
		{
			desc: "InvalidOrder_MissingDelivery",
			setup: func() *entity.Order {
				order := generateFakeOrder()
				order.Delivery = nil
				return order
			},
			mocks: func(
				orderRepo *mock_repository.MockOrderRepository,
				deliveryRepo *mock_repository.MockDeliveryRepository,
				paymentRepo *mock_repository.MockPaymentRepository,
				itemRepo *mock_repository.MockItemRepository,
				txManager *mock_transaction.MockManager,
				logger *mock_logger.MockLogger,
				cache *mock_cache.MockCache,
				order *entity.Order,
			) {
				logger.EXPECT().Ctx(gomock.Any()).Return(logger).AnyTimes()

				orderRepo.EXPECT().GetByOrderUID(ctx, order.OrderUID).
					Return(nil, entity.ErrDataNotFound).Times(1)

				logger.EXPECT().
					LogAttrs(ctx, gomock.Any(), "create order started", gomock.Any()).
					Times(1)

				logger.EXPECT().
					LogAttrs(ctx, gomock.Any(), "order validation failed", gomock.Any()).
					Times(1)
			},
			input: createOrderTestInput{
				order: nil,
			},
			expected: createOrderTestExpected{
				order: nil,
				err:   entity.ErrInvalidData,
			},
		},
		{
			desc: "InvalidOrder_MissingPayment",
			setup: func() *entity.Order {
				order := generateFakeOrder()
				order.Payment = nil
				return order
			},
			mocks: func(
				orderRepo *mock_repository.MockOrderRepository,
				deliveryRepo *mock_repository.MockDeliveryRepository,
				paymentRepo *mock_repository.MockPaymentRepository,
				itemRepo *mock_repository.MockItemRepository,
				txManager *mock_transaction.MockManager,
				logger *mock_logger.MockLogger,
				cache *mock_cache.MockCache,
				order *entity.Order,
			) {
				logger.EXPECT().Ctx(gomock.Any()).Return(logger).AnyTimes()

				orderRepo.EXPECT().GetByOrderUID(ctx, order.OrderUID).
					Return(nil, entity.ErrDataNotFound).Times(1)

				logger.EXPECT().
					LogAttrs(ctx, gomock.Any(), "create order started", gomock.Any()).
					Times(1)

				logger.EXPECT().
					LogAttrs(ctx, gomock.Any(), "order validation failed", gomock.Any()).
					Times(1)
			},
			input: createOrderTestInput{
				order: nil,
			},
			expected: createOrderTestExpected{
				order: nil,
				err:   entity.ErrInvalidData,
			},
		},
		{
			desc: "InvalidOrder_EmptyItems",
			setup: func() *entity.Order {
				order := generateFakeOrder()
				order.Items = []*entity.Item{}
				return order
			},
			mocks: func(
				orderRepo *mock_repository.MockOrderRepository,
				deliveryRepo *mock_repository.MockDeliveryRepository,
				paymentRepo *mock_repository.MockPaymentRepository,
				itemRepo *mock_repository.MockItemRepository,
				txManager *mock_transaction.MockManager,
				logger *mock_logger.MockLogger,
				cache *mock_cache.MockCache,
				order *entity.Order,
			) {
				logger.EXPECT().Ctx(gomock.Any()).Return(logger).AnyTimes()

				orderRepo.EXPECT().GetByOrderUID(ctx, order.OrderUID).
					Return(nil, entity.ErrDataNotFound).Times(1)

				logger.EXPECT().
					LogAttrs(ctx, gomock.Any(), "create order started", gomock.Any()).
					Times(1)

				logger.EXPECT().
					LogAttrs(ctx, gomock.Any(), "order validation failed", gomock.Any()).
					Times(1)
			},
			input: createOrderTestInput{
				order: nil,
			},
			expected: createOrderTestExpected{
				order: nil,
				err:   entity.ErrInvalidData,
			},
		},
		{
			desc: "InvalidOrder_InvalidUID",
			setup: func() *entity.Order {
				order := generateFakeOrder()
				order.OrderUID = uuid.Nil
				return order
			},
			mocks: func(
				orderRepo *mock_repository.MockOrderRepository,
				deliveryRepo *mock_repository.MockDeliveryRepository,
				paymentRepo *mock_repository.MockPaymentRepository,
				itemRepo *mock_repository.MockItemRepository,
				txManager *mock_transaction.MockManager,
				logger *mock_logger.MockLogger,
				cache *mock_cache.MockCache,
				order *entity.Order,
			) {
				logger.EXPECT().Ctx(gomock.Any()).Return(logger).AnyTimes()

				orderRepo.EXPECT().GetByOrderUID(ctx, order.OrderUID).
					Return(nil, entity.ErrDataNotFound).Times(1)

				logger.EXPECT().
					LogAttrs(ctx, gomock.Any(), "create order started", gomock.Any()).
					Times(1)

				logger.EXPECT().
					LogAttrs(ctx, gomock.Any(), "order validation failed", gomock.Any()).
					Times(1)
			},
			input: createOrderTestInput{
				order: nil,
			},
			expected: createOrderTestExpected{
				order: nil,
				err:   entity.ErrInvalidData,
			},
		},
		{
			desc:  "TransactionError",
			setup: generateFakeOrder,
			mocks: func(
				orderRepo *mock_repository.MockOrderRepository,
				deliveryRepo *mock_repository.MockDeliveryRepository,
				paymentRepo *mock_repository.MockPaymentRepository,
				itemRepo *mock_repository.MockItemRepository,
				txManager *mock_transaction.MockManager,
				logger *mock_logger.MockLogger,
				cache *mock_cache.MockCache,
				order *entity.Order,
			) {
				logger.EXPECT().Ctx(gomock.Any()).Return(logger).AnyTimes()

				orderRepo.EXPECT().GetByOrderUID(ctx, order.OrderUID).
					Return(nil, entity.ErrDataNotFound).Times(1)

				logger.EXPECT().
					LogAttrs(ctx, gomock.Any(), "create order started", gomock.Any()).
					Times(1)

				txManager.EXPECT().ExecuteInTransaction(
					ctx, "CreateOrder", gomock.Any(),
				).Return(errors.New("transaction error")).Times(1)

				logger.EXPECT().
					LogAttrs(ctx, gomock.Any(), "order creation failed", gomock.Any()).
					Times(1)
			},
			input: createOrderTestInput{order: nil},
			expected: createOrderTestExpected{
				order: nil,
				err:   errors.New("transaction error"),
			},
		},
		{
			desc:  "SlowOperationLogging",
			setup: generateFakeOrder,
			mocks: func(
				orderRepo *mock_repository.MockOrderRepository,
				deliveryRepo *mock_repository.MockDeliveryRepository,
				paymentRepo *mock_repository.MockPaymentRepository,
				itemRepo *mock_repository.MockItemRepository,
				txManager *mock_transaction.MockManager,
				logger *mock_logger.MockLogger,
				cache *mock_cache.MockCache,
				order *entity.Order,
			) {
				logger.EXPECT().Ctx(gomock.Any()).Return(logger).AnyTimes()

				orderRepo.EXPECT().GetByOrderUID(ctx, order.OrderUID).
					Return(nil, entity.ErrDataNotFound).Times(1)

				logger.EXPECT().
					LogAttrs(ctx, gomock.Any(), "create order started", gomock.Any()).
					Times(1)

				txManager.EXPECT().ExecuteInTransaction(
					ctx, "CreateOrder", gomock.Any(),
				).DoAndReturn(func(
					ctx context.Context,
					opName string,
					txFunc func(postgres.QueryExecuter) error,
				) error {
					time.Sleep(300 * time.Millisecond)
					return txFunc(nil)
				}).Times(1)

				orderRepo.EXPECT().Create(ctx, nil, gomock.Eq(order)).
					Return(order, nil).Times(1)

				deliveryRepo.EXPECT().
					Create(ctx, nil, order.OrderUID, gomock.Eq(order.Delivery)).
					Return(order.Delivery, nil).Times(1)

				paymentRepo.EXPECT().
					Create(ctx, nil, order.OrderUID, gomock.Eq(order.Payment)).
					Return(order.Payment, nil).Times(1)

				itemRepo.EXPECT().Create(
					ctx, nil, gomock.Eq(order.OrderUID), gomock.Eq(order.Items),
				).Return(nil).Times(1)

				cache.EXPECT().Put(order.OrderUID, gomock.Eq(order), gomock.Any()).Times(1)

				logger.EXPECT().
					LogAttrs(gomock.Any(), gomock.Any(), "slow service operation", gomock.Any()).
					Times(1)

				logger.EXPECT().
					LogAttrs(ctx, gomock.Any(), "order created successfully", gomock.Any()).
					Times(1)
			},
			input: createOrderTestInput{
				order: nil,
			},
			expected: createOrderTestExpected{
				order: nil,
				err:   nil,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			order := tc.setup()
			tc.input.order = order

			orderRepo := mock_repository.NewMockOrderRepository(ctrl)
			deliveryRepo := mock_repository.NewMockDeliveryRepository(ctrl)
			paymentRepo := mock_repository.NewMockPaymentRepository(ctrl)
			itemRepo := mock_repository.NewMockItemRepository(ctrl)
			txManager := mock_transaction.NewMockManager(ctrl)
			logger := mock_logger.NewMockLogger(ctrl)
			cache := mock_cache.NewMockCache(ctrl)

			cache.EXPECT().SetOnEvicted(gomock.Any()).AnyTimes()

			tc.mocks(
				orderRepo,
				deliveryRepo,
				paymentRepo,
				itemRepo,
				txManager,
				logger,
				cache,
				order,
			)

			s := service.NewOrderService(
				deliveryRepo,
				itemRepo,
				orderRepo,
				paymentRepo,
				txManager,
				logger,
				cache,
				time.Minute*5,
			)

			resultOrder, err := s.CreateOrder(context.Background(), tc.input.order)

			if tc.expected.err != nil {
				if err == nil {
					t.Fatalf("expected error %v, got nil", tc.expected.err)
				}

				if tc.desc == "TransactionError" {
					if err.Error() != "transaction error" {
						t.Fatalf("expected 'transaction error', got %q", err.Error())
					}
				} else {
					if !errors.Is(err, tc.expected.err) {
						t.Fatalf("expected error to contain %v, got %v", tc.expected.err, err)
					}
				}

				if resultOrder != nil {
					t.Error("expected nil order on error, got non-nil")
				}
			} else {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if resultOrder == nil {
					t.Fatal("expected non-nil order on success")
				}
			}
		})
	}
}

type getOrderTestInput struct {
	orderUID uuid.UUID
}

type getOrderTestExpected struct {
	order *entity.Order
	err   error
}

func TestOrderService_GetOrder(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		desc  string
		setup func() *entity.Order
		mocks func(
			orderRepo *mock_repository.MockOrderRepository,
			deliveryRepo *mock_repository.MockDeliveryRepository,
			paymentRepo *mock_repository.MockPaymentRepository,
			itemRepo *mock_repository.MockItemRepository,
			txManager *mock_transaction.MockManager,
			logger *mock_logger.MockLogger,
			cache *mock_cache.MockCache,
			order *entity.Order,
		)
		input    getOrderTestInput
		expected getOrderTestExpected
	}{
		{
			desc:  "FromCache",
			setup: generateFakeOrder,
			mocks: func(
				orderRepo *mock_repository.MockOrderRepository,
				deliveryRepo *mock_repository.MockDeliveryRepository,
				paymentRepo *mock_repository.MockPaymentRepository,
				itemRepo *mock_repository.MockItemRepository,
				txManager *mock_transaction.MockManager,
				logger *mock_logger.MockLogger,
				cache *mock_cache.MockCache,
				order *entity.Order,
			) {
				logger.EXPECT().Ctx(gomock.Any()).Return(logger).AnyTimes()

				logger.EXPECT().
					LogAttrs(ctx, gomock.Any(), "get order requested", gomock.Any()).
					Times(1)

				cache.EXPECT().Get(order.OrderUID).
					Return(order, true).Times(1)

				logger.EXPECT().
					LogAttrs(ctx, gomock.Any(), "order served from cache", gomock.Any()).
					Times(1)
			},
			input: func() getOrderTestInput {
				return getOrderTestInput{orderUID: uuid.Nil}
			}(),
			expected: getOrderTestExpected{
				order: nil,
				err:   nil,
			},
		},
		{
			desc:  "FromDatabase",
			setup: generateFakeOrder,
			mocks: func(
				orderRepo *mock_repository.MockOrderRepository,
				deliveryRepo *mock_repository.MockDeliveryRepository,
				paymentRepo *mock_repository.MockPaymentRepository,
				itemRepo *mock_repository.MockItemRepository,
				txManager *mock_transaction.MockManager,
				logger *mock_logger.MockLogger,
				cache *mock_cache.MockCache,
				order *entity.Order,
			) {
				logger.EXPECT().Ctx(gomock.Any()).Return(logger).AnyTimes()

				logger.EXPECT().
					LogAttrs(ctx, gomock.Any(), "get order requested", gomock.Any()).
					Times(1)

				cache.EXPECT().Get(order.OrderUID).
					Return(nil, false).Times(1)

				logger.EXPECT().
					LogAttrs(ctx, gomock.Any(), "cache miss", gomock.Any()).
					Times(1)

				orderRepo.EXPECT().GetByOrderUID(gomock.Any(), order.OrderUID).
					Return(order, nil).Times(1)
				deliveryRepo.EXPECT().GetByOrderUID(gomock.Any(), order.OrderUID).
					Return(order.Delivery, nil).Times(1)
				paymentRepo.EXPECT().GetByOrderUID(gomock.Any(), order.OrderUID).
					Return(order.Payment, nil).Times(1)
				itemRepo.EXPECT().GetListByOrderUID(gomock.Any(), order.OrderUID).
					Return(order.Items, nil).Times(1)

				cache.EXPECT().Put(order.OrderUID, gomock.Eq(order), gomock.Any()).Times(1)

				logger.EXPECT().
					LogAttrs(ctx, gomock.Any(), "order served from database", gomock.Any()).
					Times(1)
			},
			input: func() getOrderTestInput {
				return getOrderTestInput{orderUID: uuid.Nil}
			}(),
			expected: getOrderTestExpected{
				order: nil,
				err:   nil,
			},
		},
		{
			desc: "IncompleteOrder_SkipCache",
			setup: func() *entity.Order {
				order := generateFakeOrder()
				order.Delivery = nil
				return order
			},
			mocks: func(
				orderRepo *mock_repository.MockOrderRepository,
				deliveryRepo *mock_repository.MockDeliveryRepository,
				paymentRepo *mock_repository.MockPaymentRepository,
				itemRepo *mock_repository.MockItemRepository,
				txManager *mock_transaction.MockManager,
				logger *mock_logger.MockLogger,
				cache *mock_cache.MockCache,
				order *entity.Order,
			) {
				logger.EXPECT().Ctx(gomock.Any()).Return(logger).AnyTimes()

				logger.EXPECT().
					LogAttrs(ctx, gomock.Any(), "get order requested", gomock.Any()).
					Times(1)

				cache.EXPECT().Get(order.OrderUID).
					Return(nil, false).Times(1)

				logger.EXPECT().
					LogAttrs(ctx, gomock.Any(), "cache miss", gomock.Any()).
					Times(1)

				orderRepo.EXPECT().GetByOrderUID(gomock.Any(), order.OrderUID).
					Return(order, nil).Times(1)

				deliveryRepo.EXPECT().GetByOrderUID(gomock.Any(), order.OrderUID).
					Return(nil, entity.ErrDataNotFound).Times(1)

				paymentRepo.EXPECT().GetByOrderUID(gomock.Any(), order.OrderUID).
					Return(order.Payment, nil).Times(1)

				itemRepo.EXPECT().GetListByOrderUID(gomock.Any(), order.OrderUID).
					Return(order.Items, nil).Times(1)

				logger.EXPECT().
					LogAttrs(ctx, gomock.Any(), "failed to get order from database", gomock.Any()).
					Times(1)
			},
			input: func() getOrderTestInput {
				return getOrderTestInput{orderUID: uuid.Nil}
			}(),
			expected: getOrderTestExpected{
				order: nil,
				err:   errors.New("service.GetOrder: fetch order from db: invalid data"),
			},
		},
		{
			desc:  "OrderNotFound",
			setup: generateFakeOrder,
			mocks: func(
				orderRepo *mock_repository.MockOrderRepository,
				deliveryRepo *mock_repository.MockDeliveryRepository,
				paymentRepo *mock_repository.MockPaymentRepository,
				itemRepo *mock_repository.MockItemRepository,
				txManager *mock_transaction.MockManager,
				logger *mock_logger.MockLogger,
				cache *mock_cache.MockCache,
				order *entity.Order,
			) {
				logger.EXPECT().Ctx(gomock.Any()).Return(logger).AnyTimes()

				logger.EXPECT().
					LogAttrs(ctx, gomock.Any(), "get order requested", gomock.Any()).
					Times(1)

				cache.EXPECT().Get(order.OrderUID).
					Return(nil, false).Times(1)

				logger.EXPECT().
					LogAttrs(ctx, gomock.Any(), "cache miss", gomock.Any()).
					Times(1)

				orderRepo.EXPECT().GetByOrderUID(gomock.Any(), order.OrderUID).
					Return(nil, entity.ErrDataNotFound).Times(1)

				logger.EXPECT().
					LogAttrs(ctx, gomock.Any(), "failed to get order from database", gomock.Any()).
					Times(1)
			},
			input: func() getOrderTestInput {
				return getOrderTestInput{orderUID: uuid.Nil}
			}(),
			expected: getOrderTestExpected{
				order: nil,
				err:   errors.New("service.GetOrder: fetch order from db: data not found"),
			},
		},
		{
			desc:  "DatabaseError",
			setup: generateFakeOrder,
			mocks: func(
				orderRepo *mock_repository.MockOrderRepository,
				deliveryRepo *mock_repository.MockDeliveryRepository,
				paymentRepo *mock_repository.MockPaymentRepository,
				itemRepo *mock_repository.MockItemRepository,
				txManager *mock_transaction.MockManager,
				logger *mock_logger.MockLogger,
				cache *mock_cache.MockCache,
				order *entity.Order,
			) {
				logger.EXPECT().Ctx(gomock.Any()).Return(logger).AnyTimes()

				logger.EXPECT().
					LogAttrs(ctx, gomock.Any(), "get order requested", gomock.Any()).
					Times(1)

				cache.EXPECT().Get(order.OrderUID).
					Return(nil, false).Times(1)

				logger.EXPECT().
					LogAttrs(ctx, gomock.Any(), "cache miss", gomock.Any()).
					Times(1)

				orderRepo.EXPECT().GetByOrderUID(gomock.Any(), order.OrderUID).
					Return(nil, errors.New("database error")).Times(1)

				logger.EXPECT().
					LogAttrs(ctx, gomock.Any(), "failed to get order from database", gomock.Any()).
					Times(1)
			},
			input: func() getOrderTestInput {
				return getOrderTestInput{orderUID: uuid.Nil}
			}(),
			expected: getOrderTestExpected{
				order: nil,
				err:   errors.New("service.GetOrder: fetch order from db: database error"),
			},
		},
		{
			desc:  "SlowGetOrder",
			setup: generateFakeOrder,
			mocks: func(
				orderRepo *mock_repository.MockOrderRepository,
				deliveryRepo *mock_repository.MockDeliveryRepository,
				paymentRepo *mock_repository.MockPaymentRepository,
				itemRepo *mock_repository.MockItemRepository,
				txManager *mock_transaction.MockManager,
				logger *mock_logger.MockLogger,
				cache *mock_cache.MockCache,
				order *entity.Order,
			) {
				logger.EXPECT().Ctx(gomock.Any()).Return(logger).AnyTimes()

				logger.EXPECT().
					LogAttrs(gomock.Any(), gomock.Any(), "get order requested", gomock.Any()).
					Times(1)

				cache.EXPECT().Get(order.OrderUID).
					Return(nil, false).Times(1)

				logger.EXPECT().
					LogAttrs(gomock.Any(), gomock.Any(), "cache miss", gomock.Any()).
					Times(1)

				orderRepo.EXPECT().GetByOrderUID(gomock.Any(), order.OrderUID).
					Return(order, nil).Times(1)

				deliveryRepo.EXPECT().GetByOrderUID(gomock.Any(), order.OrderUID).
					DoAndReturn(func(ctx context.Context, orderUID uuid.UUID) (*entity.Delivery, error) {
						time.Sleep(600 * time.Millisecond)
						return order.Delivery, nil
					}).
					Times(1)

				paymentRepo.EXPECT().GetByOrderUID(gomock.Any(), order.OrderUID).
					Return(order.Payment, nil).Times(1)

				itemRepo.EXPECT().GetListByOrderUID(gomock.Any(), order.OrderUID).
					Return(order.Items, nil).Times(1)

				cache.EXPECT().Put(order.OrderUID, gomock.Eq(order), gomock.Any()).Times(1)

				logger.EXPECT().
					LogAttrs(gomock.Any(), gomock.Any(), "slow service operation", gomock.Any()).
					Times(1)

				logger.EXPECT().
					LogAttrs(gomock.Any(), gomock.Any(), "order served from database", gomock.Any()).
					Times(1)
			},
			input: func() getOrderTestInput {
				return getOrderTestInput{orderUID: uuid.Nil}
			}(),
			expected: getOrderTestExpected{
				order: nil,
				err:   nil,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			order := tc.setup()
			tc.input.orderUID = order.OrderUID

			orderRepo := mock_repository.NewMockOrderRepository(ctrl)
			deliveryRepo := mock_repository.NewMockDeliveryRepository(ctrl)
			paymentRepo := mock_repository.NewMockPaymentRepository(ctrl)
			itemRepo := mock_repository.NewMockItemRepository(ctrl)
			txManager := mock_transaction.NewMockManager(ctrl)
			logger := mock_logger.NewMockLogger(ctrl)
			cache := mock_cache.NewMockCache(ctrl)

			cache.EXPECT().SetOnEvicted(gomock.Any()).AnyTimes()

			tc.mocks(
				orderRepo,
				deliveryRepo,
				paymentRepo,
				itemRepo,
				txManager,
				logger,
				cache,
				order,
			)

			s := service.NewOrderService(
				deliveryRepo,
				itemRepo,
				orderRepo,
				paymentRepo,
				txManager,
				logger,
				cache,
				time.Minute*5,
			)

			resultOrder, err := s.GetOrder(context.Background(), tc.input.orderUID)

			if tc.expected.err != nil {
				if err == nil {
					t.Fatalf("expected error %v, got nil", tc.expected.err)
				}
				if err.Error() != tc.expected.err.Error() {
					t.Fatalf("expected error %q, got %q", tc.expected.err.Error(), err.Error())
				}
				if resultOrder != nil {
					t.Error("expected nil order on error, got non-nil")
				}
			} else {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if resultOrder == nil {
					t.Fatal("expected non-nil order on success")
				}
			}
		})
	}
}
