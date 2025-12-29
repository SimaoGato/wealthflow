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

// bucketRepository implements domain.BucketRepository
type bucketRepository struct {
	db *DB
}

// NewBucketRepository creates a new bucket repository
func NewBucketRepository(db *DB) domain.BucketRepository {
	return &bucketRepository{db: db}
}

// GetByID retrieves a bucket by its ID
func (r *bucketRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Bucket, error) {
	query := `
		SELECT id, name, bucket_type, parent_physical_bucket_id, current_balance
		FROM buckets
		WHERE id = $1
	`

	var bucket domain.Bucket
	var parentID sql.NullString
	var balanceStr string

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&bucket.ID,
		&bucket.Name,
		&bucket.BucketType,
		&parentID,
		&balanceStr,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("bucket not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get bucket by ID: %w", err)
	}

	// Parse parent_physical_bucket_id (nullable)
	if parentID.Valid {
		parentUUID, err := uuid.Parse(parentID.String)
		if err != nil {
			return nil, fmt.Errorf("failed to parse parent_physical_bucket_id: %w", err)
		}
		bucket.ParentPhysicalBucketID = &parentUUID
	}

	// Parse current_balance (DECIMAL)
	balance, err := decimal.NewFromString(balanceStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse current_balance: %w", err)
	}
	bucket.CurrentBalance = balance

	return &bucket, nil
}

// Create creates a new bucket
func (r *bucketRepository) Create(ctx context.Context, bucket *domain.Bucket) error {
	query := `
		INSERT INTO buckets (id, name, bucket_type, parent_physical_bucket_id, current_balance)
		VALUES ($1, $2, $3, $4, $5)
	`

	var parentID interface{}
	if bucket.ParentPhysicalBucketID != nil {
		parentID = bucket.ParentPhysicalBucketID
	}

	_, err := r.db.ExecContext(ctx, query,
		bucket.ID,
		bucket.Name,
		string(bucket.BucketType),
		parentID,
		bucket.CurrentBalance.String(),
	)
	if err != nil {
		return fmt.Errorf("failed to create bucket: %w", err)
	}

	return nil
}

// GetSystemBucket retrieves a system bucket by its type
func (r *bucketRepository) GetSystemBucket(ctx context.Context, bucketType domain.BucketType) (*domain.Bucket, error) {
	query := `
		SELECT id, name, bucket_type, parent_physical_bucket_id, current_balance
		FROM buckets
		WHERE bucket_type = $1
	`

	var bucket domain.Bucket
	var parentID sql.NullString
	var balanceStr string

	err := r.db.QueryRowContext(ctx, query, string(bucketType)).Scan(
		&bucket.ID,
		&bucket.Name,
		&bucket.BucketType,
		&parentID,
		&balanceStr,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("system bucket not found for type %s: %w", bucketType, err)
		}
		return nil, fmt.Errorf("failed to get system bucket: %w", err)
	}

	// Parse parent_physical_bucket_id (nullable)
	if parentID.Valid {
		parentUUID, err := uuid.Parse(parentID.String)
		if err != nil {
			return nil, fmt.Errorf("failed to parse parent_physical_bucket_id: %w", err)
		}
		bucket.ParentPhysicalBucketID = &parentUUID
	}

	// Parse current_balance (DECIMAL)
	balance, err := decimal.NewFromString(balanceStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse current_balance: %w", err)
	}
	bucket.CurrentBalance = balance

	return &bucket, nil
}
