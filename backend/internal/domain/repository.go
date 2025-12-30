package domain

import (
	"context"

	"github.com/google/uuid"
)

// BucketRepository defines the interface for bucket persistence operations
type BucketRepository interface {
	// GetByID retrieves a bucket by its ID
	GetByID(ctx context.Context, id uuid.UUID) (*Bucket, error)

	// Create creates a new bucket
	Create(ctx context.Context, bucket *Bucket) error

	// GetSystemBucket retrieves a system bucket by its type
	// This is a convenience method for finding system buckets
	GetSystemBucket(ctx context.Context, bucketType BucketType) (*Bucket, error)

	// List retrieves a list of buckets, optionally filtered by type
	// If typeFilter is empty, returns all buckets
	List(ctx context.Context, typeFilter BucketType) ([]*Bucket, error)
}

// TransactionRepository defines the interface for transaction persistence operations
type TransactionRepository interface {
	// Create creates a new transaction
	Create(ctx context.Context, tx *Transaction) error

	// List retrieves a paginated list of transactions
	// If bucketID is nil, returns all transactions
	// limit and offset are used for pagination
	List(ctx context.Context, limit, offset int, bucketID *uuid.UUID) ([]*Transaction, error)

	// Count returns the total number of transactions
	// If bucketID is nil, returns count of all transactions
	// If bucketID is provided, returns count of transactions involving that bucket
	Count(ctx context.Context, bucketID *uuid.UUID) (int, error)
}

// SplitRuleRepository defines the interface for split rule persistence operations
type SplitRuleRepository interface {
	// GetBySourceBucketID retrieves a split rule by its source bucket ID
	GetBySourceBucketID(ctx context.Context, bucketID uuid.UUID) (*SplitRule, error)
}

// MarketValueRepository defines the interface for market value history persistence operations
type MarketValueRepository interface {
	// Add creates a new market value history entry
	Add(ctx context.Context, entry *MarketValueHistory) error

	// GetLatest retrieves the most recent market value entry for a given bucket
	GetLatest(ctx context.Context, bucketID uuid.UUID) (*MarketValueHistory, error)
}
