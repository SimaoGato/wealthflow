package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/simaogato/wealthflow-backend/internal/domain"
)

// marketValueRepository implements domain.MarketValueRepository
type marketValueRepository struct {
	db *DB
}

// NewMarketValueRepository creates a new market value repository
func NewMarketValueRepository(db *DB) domain.MarketValueRepository {
	return &marketValueRepository{db: db}
}

// Add creates a new market value history entry
func (r *marketValueRepository) Add(ctx context.Context, entry *domain.MarketValueHistory) error {
	query := `
		INSERT INTO market_value_history (id, bucket_id, date, market_value)
		VALUES ($1, $2, $3, $4)
	`

	_, err := r.db.ExecContext(ctx, query,
		entry.ID,
		entry.BucketID,
		entry.Date,
		entry.MarketValue.String(),
	)
	if err != nil {
		return fmt.Errorf("failed to insert market value history entry: %w", err)
	}

	return nil
}

// GetLatest retrieves the most recent market value entry for a given bucket
func (r *marketValueRepository) GetLatest(ctx context.Context, bucketID uuid.UUID) (*domain.MarketValueHistory, error) {
	query := `
		SELECT id, bucket_id, date, market_value
		FROM market_value_history
		WHERE bucket_id = $1
		ORDER BY date DESC
		LIMIT 1
	`

	var entry domain.MarketValueHistory
	var marketValueStr string

	err := r.db.QueryRowContext(ctx, query, bucketID).Scan(
		&entry.ID,
		&entry.BucketID,
		&entry.Date,
		&marketValueStr,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("no market value history found for bucket %s: %w", bucketID, err)
		}
		return nil, fmt.Errorf("failed to get latest market value: %w", err)
	}

	// Parse market_value (DECIMAL)
	marketValue, err := decimal.NewFromString(marketValueStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse market_value: %w", err)
	}
	entry.MarketValue = marketValue

	return &entry, nil
}
