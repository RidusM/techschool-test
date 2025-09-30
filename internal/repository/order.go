package repository

import (
	"context"
	"errors"
	"fmt"

	"wbtest/internal/entity"
	"wbtest/pkg/storage/postgres"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v5"
)

type OrderRepository struct {
	db *postgres.Postgres
}

func NewOrderRepository(db *postgres.Postgres) *OrderRepository {
	return &OrderRepository{db}
}

func (dr *OrderRepository) Create(
	ctx context.Context,
	queryExecuter postgres.QueryExecuter,
	order *entity.Order,
) (*entity.Order, error) {
	const op = "repository.order.Create"

	query := dr.db.Builder.Insert(`"orders"`).
		Columns("order_uid", "track_number", "entry", "locale", "internal_signature", "customer_id", "delivery_service", "shardkey", "sm_id", "date_created", "oof_shard").
		Values(
			order.OrderUID,
			order.TrackNumber,
			order.Entry,
			order.Locale,
			order.InternalSignature,
			order.CustomerID,
			order.DeliveryService,
			order.Shardkey,
			order.SmID,
			order.DateCreated,
			order.OofShard,
		).
		Suffix("RETURNING order_uid, track_number, entry, locale, internal_signature, customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard")

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s: building query: %w", op, err)
	}

	result := &entity.Order{}
	err = queryExecuter.QueryRow(ctx, sql, args...).Scan(
		&result.OrderUID,
		&result.TrackNumber,
		&result.Entry,
		&result.Locale,
		&result.InternalSignature,
		&result.CustomerID,
		&result.DeliveryService,
		&result.Shardkey,
		&result.SmID,
		&result.DateCreated,
		&result.OofShard,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, entity.ErrConflictingData
		}
		return nil, fmt.Errorf("%s: query row: %w", op, err)
	}

	return result, nil
}

func (dr *OrderRepository) GetByOrderUID(
	ctx context.Context,
	orderUID uuid.UUID,
) (*entity.Order, error) {
	const op = "repository.order.Get"

	query := dr.db.Builder.Select("*").
		From(`"orders"`).
		Where(squirrel.Eq{"order_uid": orderUID}).
		Limit(1)

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s: building query: %w", op, err)
	}

	result := &entity.Order{}
	err = dr.db.Pool.QueryRow(ctx, sql, args...).Scan(
		&result.OrderUID,
		&result.TrackNumber,
		&result.Entry,
		&result.Locale,
		&result.InternalSignature,
		&result.CustomerID,
		&result.DeliveryService,
		&result.Shardkey,
		&result.SmID,
		&result.DateCreated,
		&result.OofShard,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entity.ErrDataNotFound
		}
		return nil, fmt.Errorf("%s: query row: %w", op, err)
	}

	return result, nil
}

func (dr *OrderRepository) GetAllOrderUIDs(ctx context.Context) ([]uuid.UUID, error) {
	const op = "repository.order.GetAllOrderUIDs"

	query := dr.db.Builder.Select("order_uid").From(`"orders"`)

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s: building query: %w", op, err)
	}

	rows, err := dr.db.Pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: query: %w", op, err)
	}
	defer rows.Close()

	var uids []uuid.UUID
	for rows.Next() {
		var uid uuid.UUID
		if err = rows.Scan(&uid); err != nil {
			return nil, fmt.Errorf("%s: row scan: %w", op, err)
		}
		uids = append(uids, uid)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("%s: rows final error: %w", op, rows.Err())
	}

	return uids, nil
}
