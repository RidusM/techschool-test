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

type PaymentRepository struct {
	db *postgres.Postgres
}

func NewPaymentRepository(db *postgres.Postgres) *PaymentRepository {
	return &PaymentRepository{db}
}

func (dr *PaymentRepository) Create(
	ctx context.Context,
	queryExecuter postgres.QueryExecuter,
	orderUID uuid.UUID,
	payment *entity.Payment,
) (*entity.Payment, error) {
	const op = "repository.payment.Create"

	query := dr.db.Builder.Insert("payment").
		Columns("order_uid", "transaction", "request_id", "currency", "provider", "amount", "payment_dt", "bank", "delivery_cost", "goods_total", "custom_fee").
		Values(
			orderUID,
			payment.Transaction,
			payment.RequestID,
			payment.Currency,
			payment.Provider,
			payment.Amount,
			payment.PaymentDt,
			payment.Bank,
			payment.DeliveryCost,
			payment.GoodsTotal,
			payment.CustomFee,
		).
		Suffix("RETURNING transaction, request_id, currency, provider, amount, payment_dt, bank, delivery_cost, goods_total, custom_fee")

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s: building query: %w", op, err)
	}

	result := &entity.Payment{}
	err = queryExecuter.QueryRow(ctx, sql, args...).Scan(
		&result.Transaction,
		&result.RequestID,
		&result.Currency,
		&result.Provider,
		&result.Amount,
		&result.PaymentDt,
		&result.Bank,
		&result.DeliveryCost,
		&result.GoodsTotal,
		&result.CustomFee,
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

func (dr *PaymentRepository) GetByOrderUID(
	ctx context.Context,
	orderUID uuid.UUID,
) (*entity.Payment, error) {
	const op = "repository.payment.GetByOrderUID"

	query := dr.db.Builder.Select("*").
		From("payment").
		Where(squirrel.Eq{"order_uid": orderUID}).
		Limit(1)

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s: building query: %w", op, err)
	}

	result := &entity.Payment{}
	err = dr.db.Pool.QueryRow(ctx, sql, args...).Scan(
		&orderUID,
		&result.Transaction,
		&result.RequestID,
		&result.Currency,
		&result.Provider,
		&result.Amount,
		&result.PaymentDt,
		&result.Bank,
		&result.DeliveryCost,
		&result.GoodsTotal,
		&result.CustomFee,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entity.ErrDataNotFound
		}
		return nil, fmt.Errorf("%s: query row: %w", op, err)
	}

	return result, nil
}
