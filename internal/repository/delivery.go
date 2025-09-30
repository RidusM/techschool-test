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

type DeliveryRepository struct {
	db *postgres.Postgres
}

func NewDeliveryRepository(db *postgres.Postgres) *DeliveryRepository {
	return &DeliveryRepository{db}
}

func (dr *DeliveryRepository) Create(
	ctx context.Context,
	queryExecuter postgres.QueryExecuter,
	orderUID uuid.UUID,
	delivery *entity.Delivery,
) (*entity.Delivery, error) {
	const op = "repository.delivery.Create"

	query := dr.db.Builder.Insert("delivery").
		Columns("order_uid", "name", "phone", "zip", "city", "address", "region", "email").
		Values(
			orderUID,
			delivery.Name,
			delivery.Phone,
			delivery.Zip,
			delivery.City,
			delivery.Address,
			delivery.Region,
			delivery.Email,
		).
		Suffix("RETURNING name, phone, zip, city, address, region, email")

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s: building query: %w", op, err)
	}

	result := &entity.Delivery{}
	err = queryExecuter.QueryRow(ctx, sql, args...).Scan(
		&result.Name,
		&result.Phone,
		&result.Zip,
		&result.City,
		&result.Address,
		&result.Region,
		&result.Email,
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

func (dr *DeliveryRepository) GetByOrderUID(
	ctx context.Context,
	orderUID uuid.UUID,
) (*entity.Delivery, error) {
	const op = "repository.delivery.Get"

	query := dr.db.Builder.Select("*").
		From("delivery").
		Where(squirrel.Eq{"order_uid": orderUID}).
		Limit(1)

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s: building query: %w", op, err)
	}

	result := &entity.Delivery{}
	err = dr.db.Pool.QueryRow(ctx, sql, args...).Scan(
		&orderUID,
		&result.Name,
		&result.Phone,
		&result.Zip,
		&result.City,
		&result.Address,
		&result.Region,
		&result.Email,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entity.ErrDataNotFound
		}
		return nil, fmt.Errorf("%s: query row: %w", op, err)
	}

	return result, nil
}
