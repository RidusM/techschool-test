package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"wbtest/internal/entity"
	"wbtest/pkg/cache"
	"wbtest/pkg/logger"
	"wbtest/pkg/storage/postgres"
	"wbtest/pkg/storage/postgres/transaction"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

const (
	_defaultContextTimeout = 500 * time.Millisecond
)

type (
	DeliveryRepository interface {
		Create(
			ctx context.Context,
			queryExecuter postgres.QueryExecuter,
			orderUID uuid.UUID,
			delivery *entity.Delivery,
		) (*entity.Delivery, error)
		GetByOrderUID(ctx context.Context, orderUID uuid.UUID) (*entity.Delivery, error)
	}

	ItemRepository interface {
		Create(
			ctx context.Context,
			queryExecuter postgres.QueryExecuter,
			orderUID uuid.UUID,
			items []*entity.Item,
		) error
		GetListByOrderUID(ctx context.Context, orderUID uuid.UUID) ([]*entity.Item, error)
	}

	OrderRepository interface {
		Create(
			ctx context.Context,
			queryExecuter postgres.QueryExecuter,
			order *entity.Order,
		) (*entity.Order, error)
		GetByOrderUID(ctx context.Context, orderUID uuid.UUID) (*entity.Order, error)
		GetAllOrderUIDs(ctx context.Context) ([]uuid.UUID, error)
	}

	PaymentRepository interface {
		Create(
			ctx context.Context,
			queryExecuter postgres.QueryExecuter,
			orderUID uuid.UUID,
			payment *entity.Payment,
		) (*entity.Payment, error)
		GetByOrderUID(ctx context.Context, orderUID uuid.UUID) (*entity.Payment, error)
	}

	OrderService struct {
		deliveryRepo DeliveryRepository
		itemRepo     ItemRepository
		orderRepo    OrderRepository
		paymentRepo  PaymentRepository
		txManager    transaction.Manager
		logger       logger.Logger
		cache        cache.Cache[uuid.UUID, *entity.Order]
		cacheTTL     time.Duration
	}
)

func NewOrderService(
	deliveryRepo DeliveryRepository,
	itemRepo ItemRepository,
	orderRepo OrderRepository,
	paymentRepo PaymentRepository,
	txManager transaction.Manager,
	logger logger.Logger,
	cache cache.Cache[uuid.UUID, *entity.Order],
	cacheTTL time.Duration,
) *OrderService {
	cache.SetOnEvicted(func(key uuid.UUID, value *entity.Order) {
		logger.Infow("cache eviction",
			"key", key.String(),
			"type", fmt.Sprintf("%T", value),
		)
	})

	return &OrderService{
		deliveryRepo: deliveryRepo,
		itemRepo:     itemRepo,
		orderRepo:    orderRepo,
		paymentRepo:  paymentRepo,
		txManager:    txManager,
		logger:       logger,
		cache:        cache,
		cacheTTL:     cacheTTL,
	}
}

func (os *OrderService) RestoreCache(ctx context.Context) error {
	const op = "service.RestoreCache"
	log := os.logger.Ctx(ctx)

	log.LogAttrs(ctx, logger.InfoLevel, "starting cache restoration from database")

	uids, err := os.orderRepo.GetAllOrderUIDs(ctx)
	if err != nil {
		return fmt.Errorf("%s: get all order uids: %w", op, err)
	}

	if len(uids) == 0 {
		log.LogAttrs(ctx, logger.InfoLevel, "no orders in database to restore cache")
		return nil
	}

	var restoredCount int
	for _, uid := range uids {
		order, orderErr := os.fetchOrderFromDB(ctx, uid)
		if orderErr != nil {
			log.LogAttrs(ctx, logger.WarnLevel, "failed to fetch order for cache restoration",
				logger.String("op", op),
				logger.String("order_uid", uid.String()),
				logger.Any("error", orderErr),
			)
			continue
		}
		os.cache.Put(order.OrderUID, order, os.cacheTTL)
		restoredCount++
	}

	log.LogAttrs(ctx, logger.InfoLevel, "cache restoration finished",
		logger.Int("total_orders_in_db", len(uids)),
		logger.Int("restored_to_cache", restoredCount),
	)

	return nil
}

func (os *OrderService) CreateOrder(
	ctx context.Context,
	order *entity.Order,
) (*entity.Order, error) {
	const op = "service.CreateOrder"
	log := os.logger.Ctx(ctx)

	existingOrder, err := os.orderRepo.GetByOrderUID(ctx, order.OrderUID)
	if err == nil {
		return existingOrder, nil
	}
	if !errors.Is(err, entity.ErrDataNotFound) {
		return nil, fmt.Errorf("%s: check duplicate: %w", op, err)
	}

	log.LogAttrs(ctx, logger.InfoLevel, "create order started",
		logger.String("op", op),
		logger.String("order_uid", order.OrderUID.String()),
		logger.Int("items_count", len(order.Items)),
	)

	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		if duration > 200*time.Millisecond {
			log.LogAttrs(ctx, logger.WarnLevel, "slow service operation",
				logger.String("op", op),
				logger.String("order_uid", order.OrderUID.String()),
				logger.String("duration", duration.String()),
			)
		}
	}()

	if err = os.validateOrder(order); err != nil {
		log.LogAttrs(ctx, logger.ErrorLevel, "order validation failed",
			logger.String("op", op),
			logger.Any("error", err),
			logger.String("order_uid", order.OrderUID.String()),
		)
		return nil, fmt.Errorf("%s: validate order: %w", op, err)
	}

	createdOrder, err := os.createOrderWithTransaction(ctx, order)
	if err != nil {
		log.LogAttrs(ctx, logger.ErrorLevel, "order creation failed",
			logger.String("op", op),
			logger.Any("error", err),
			logger.String("order_uid", order.OrderUID.String()),
		)
		return nil, err
	}

	os.cache.Put(createdOrder.OrderUID, createdOrder, os.cacheTTL)

	duration := time.Since(startTime)
	log.LogAttrs(ctx, logger.InfoLevel, "order created successfully",
		logger.String("op", op),
		logger.String("order_uid", createdOrder.OrderUID.String()),
		logger.String("duration", duration.String()),
	)

	return createdOrder, nil
}

func (os *OrderService) createOrderWithTransaction(
	ctx context.Context,
	order *entity.Order,
) (*entity.Order, error) {
	var createdOrder *entity.Order

	err := os.txManager.ExecuteInTransaction(
		ctx,
		"CreateOrder",
		func(tx postgres.QueryExecuter) error {
			var err error
			createdOrder, err = os.createOrderInTx(ctx, tx, order)
			if err != nil {
				return transaction.HandleError("CreateOrder", "create order", err)
			}

			if _, err = os.createDeliveryInTx(ctx, tx, createdOrder.OrderUID, order.Delivery); err != nil {
				return transaction.HandleError("CreateOrder", "create delivery", err)
			}

			if _, err = os.createPaymentInTx(ctx, tx, createdOrder.OrderUID, order.Payment); err != nil {
				return transaction.HandleError("CreateOrder", "create payment", err)
			}

			if err = os.createItemsInTx(ctx, tx, createdOrder.OrderUID, order.Items); err != nil {
				return transaction.HandleError("CreateOrder", "create items", err)
			}

			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	return createdOrder, nil
}

func (os *OrderService) createOrderInTx(
	ctx context.Context,
	tx postgres.QueryExecuter,
	order *entity.Order,
) (*entity.Order, error) {
	order, err := os.orderRepo.Create(ctx, tx, order)
	if err != nil {
		return nil, err
	}
	return order, nil
}

func (os *OrderService) createDeliveryInTx(
	ctx context.Context,
	tx postgres.QueryExecuter,
	orderUID uuid.UUID,
	delivery *entity.Delivery,
) (*entity.Delivery, error) {
	delivery, err := os.deliveryRepo.Create(ctx, tx, orderUID, delivery)
	if err != nil {
		return nil, err
	}
	return delivery, nil
}

func (os *OrderService) createPaymentInTx(
	ctx context.Context,
	tx postgres.QueryExecuter,
	orderUID uuid.UUID,
	payment *entity.Payment,
) (*entity.Payment, error) {
	payment, err := os.paymentRepo.Create(ctx, tx, orderUID, payment)
	if err != nil {
		return nil, err
	}
	return payment, nil
}

func (os *OrderService) createItemsInTx(
	ctx context.Context,
	tx postgres.QueryExecuter,
	orderUID uuid.UUID,
	items []*entity.Item,
) error {
	if err := os.itemRepo.Create(ctx, tx, orderUID, items); err != nil {
		return err
	}
	return nil
}

func (os *OrderService) GetOrder(ctx context.Context, orderUID uuid.UUID) (*entity.Order, error) {
	const op = "service.GetOrder"
	log := os.logger.Ctx(ctx)

	log.LogAttrs(ctx, logger.InfoLevel, "get order requested",
		logger.String("op", op),
		logger.String("order_uid", orderUID.String()),
	)

	startTime := time.Now()

	defer func() {
		duration := time.Since(startTime)
		if duration > 200*time.Millisecond {
			log.LogAttrs(ctx, logger.WarnLevel, "slow service operation",
				logger.String("op", op),
				logger.String("order_uid", orderUID.String()),
				logger.String("duration", duration.String()),
			)
		}
	}()

	if cached, found := os.cache.Get(orderUID); found {
		duration := time.Since(startTime)
		log.LogAttrs(ctx, logger.InfoLevel, "order served from cache",
			logger.String("op", op),
			logger.String("order_uid", orderUID.String()),
			logger.String("duration", duration.String()),
		)
		return cached, nil
	}

	log.LogAttrs(ctx, logger.DebugLevel, "cache miss",
		logger.String("op", op),
		logger.String("order_uid", orderUID.String()),
	)

	order, err := os.fetchOrderFromDB(ctx, orderUID)
	if err != nil {
		log.LogAttrs(ctx, logger.ErrorLevel, "failed to get order from database",
			logger.String("op", op),
			logger.Any("error", err),
			logger.String("order_uid", orderUID.String()),
		)
		return nil, err
	}

	if order.Delivery != nil && order.Payment != nil && len(order.Items) > 0 {
		os.cache.Put(orderUID, order, os.cacheTTL)
	} else {
		os.logger.LogAttrs(ctx, logger.WarnLevel, "skipping cache for incomplete order",
			logger.String("order_uid", orderUID.String()),
			logger.Bool("has_delivery", order.Delivery != nil),
			logger.Bool("has_payment", order.Payment != nil),
			logger.Int("items_count", len(order.Items)),
		)
	}

	duration := time.Since(startTime)
	log.LogAttrs(ctx, logger.InfoLevel, "order served from database",
		logger.String("op", op),
		logger.String("order_uid", orderUID.String()),
		logger.Int("items_count", len(order.Items)),
		logger.String("duration", duration.String()),
	)

	return order, nil
}

func (os *OrderService) fetchOrderFromDB(
	ctx context.Context,
	orderUID uuid.UUID,
) (*entity.Order, error) {
	ctx, cancel := context.WithTimeout(ctx, _defaultContextTimeout)
	defer cancel()

	order, err := os.orderRepo.GetByOrderUID(ctx, orderUID)
	if err != nil {
		return nil, err
	}

	delivery, payment, items, err := os.fetchOrderComponents(ctx, orderUID)
	if err != nil {
		return nil, err
	}

	order.Delivery = delivery
	order.Payment = payment
	order.Items = items

	return order, nil
}

func (os *OrderService) fetchOrderComponents(
	ctx context.Context,
	orderUID uuid.UUID,
) (*entity.Delivery, *entity.Payment, []*entity.Item, error) {
	var delivery *entity.Delivery
	var payment *entity.Payment
	var items []*entity.Item
	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		delivery, err = os.getDelivery(gCtx, orderUID)
		if err != nil && !errors.Is(err, entity.ErrDataNotFound) {
			return err
		}
		return nil
	})

	g.Go(func() error {
		var err error
		payment, err = os.getPayment(gCtx, orderUID)
		if err != nil && !errors.Is(err, entity.ErrDataNotFound) {
			return err
		}
		return nil
	})

	g.Go(func() error {
		var err error
		items, err = os.getItems(gCtx, orderUID)
		if err != nil && !errors.Is(err, entity.ErrDataNotFound) {
			return err
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, nil, nil, err
	}

	if delivery == nil || payment == nil || len(items) == 0 {
		return nil, nil, nil, entity.ErrDataNotFound
	}

	return delivery, payment, items, nil
}

func (os *OrderService) getDelivery(
	ctx context.Context,
	orderUID uuid.UUID,
) (*entity.Delivery, error) {
	ctx, cancel := context.WithTimeout(ctx, _defaultContextTimeout)
	defer cancel()

	delivery, err := os.deliveryRepo.GetByOrderUID(ctx, orderUID)
	if err != nil && !errors.Is(err, entity.ErrDataNotFound) {
		return nil, fmt.Errorf("service.getDelivery: %w", err)
	}
	return delivery, nil
}

func (os *OrderService) getPayment(
	ctx context.Context,
	orderUID uuid.UUID,
) (*entity.Payment, error) {
	ctx, cancel := context.WithTimeout(ctx, _defaultContextTimeout)
	defer cancel()

	payment, err := os.paymentRepo.GetByOrderUID(ctx, orderUID)
	if err != nil && !errors.Is(err, entity.ErrDataNotFound) {
		return nil, fmt.Errorf("service.getPayment: %w", err)
	}
	return payment, nil
}

func (os *OrderService) getItems(ctx context.Context, orderUID uuid.UUID) ([]*entity.Item, error) {
	ctx, cancel := context.WithTimeout(ctx, _defaultContextTimeout)
	defer cancel()

	items, err := os.itemRepo.GetListByOrderUID(ctx, orderUID)
	if err != nil && !errors.Is(err, entity.ErrDataNotFound) {
		return nil, fmt.Errorf("service.getItems: %w", err)
	}
	return items, nil
}

func (os *OrderService) validateOrder(order *entity.Order) error {
	if order.OrderUID == uuid.Nil {
		return entity.ErrInvalidData
	}
	if order.Delivery == nil {
		return entity.ErrInvalidData
	}
	if order.Payment == nil {
		return entity.ErrInvalidData
	}
	if len(order.Items) == 0 {
		return entity.ErrInvalidData
	}
	return nil
}
