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

type ItemRepository struct {
	db *postgres.Postgres
}

func NewItemRepository(db *postgres.Postgres) *ItemRepository {
	return &ItemRepository{db}
}

func (dr *ItemRepository) Create(
	ctx context.Context,
	queryExecuter postgres.QueryExecuter,
	orderUID uuid.UUID,
	items []*entity.Item,
) error {
	const op = "repository.item.Create"

	if len(items) == 0 {
		return nil
	}

	rows := make([][]interface{}, 0, len(items))
	for _, item := range items {
		rows = append(rows, []interface{}{
			uuid.New(),
			orderUID,
			item.ChrtID,
			item.TrackNumber,
			item.Price,
			item.Rid,
			item.Name,
			item.Sale,
			item.Size,
			item.TotalPrice,
			item.NMID,
			item.Brand,
			item.Status,
		})
	}

	tx, ok := queryExecuter.(*postgres.TxQueryExecuter)
	if !ok {
		return fmt.Errorf("%s: queryExecuter is not a transaction", op)
	}

	columnNames := []string{
		"items_id", "order_uid", "chrt_id", "track_number", "price", "rid",
		"name", "sale", "size", "total_price", "nm_id", "brand", "status",
	}

	_, err := tx.Tx.CopyFrom(
		ctx,
		pgx.Identifier{"items"},
		columnNames,
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return entity.ErrConflictingData
		}
		return fmt.Errorf("%s: copy from: %w", op, err)
	}

	return nil
}

func (dr *ItemRepository) GetListByOrderUID(
	ctx context.Context,
	orderUID uuid.UUID,
) ([]*entity.Item, error) {
	const op = "repository.item.GetListByOrderUID"

	var itemID uuid.UUID

	query := dr.db.Builder.Select("*").
		From("items").
		Where(squirrel.Eq{"order_uid": orderUID})

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s: building query: %w", op, err)
	}

	result := make([]*entity.Item, 0)
	rows, err := dr.db.Pool.Query(ctx, sql, args...)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entity.ErrDataNotFound
		}
		return nil, fmt.Errorf("%s: query: %w", op, err)
	}
	defer rows.Close()

	for rows.Next() {
		item := &entity.Item{}
		err = rows.Scan(
			&itemID,
			&orderUID,
			&item.ChrtID,
			&item.TrackNumber,
			&item.Price,
			&item.Rid,
			&item.Name,
			&item.Sale,
			&item.Size,
			&item.TotalPrice,
			&item.NMID,
			&item.Brand,
			&item.Status,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: rows scan: %w", op, err)
		}

		result = append(result, item)
	}

	return result, nil
}
