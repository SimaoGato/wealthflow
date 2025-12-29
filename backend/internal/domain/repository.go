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
}

// TransactionRepository defines the interface for transaction persistence operations
type TransactionRepository interface {
	// Create creates a new transaction
	Create(ctx context.Context, tx *Transaction) error
}

// SplitRuleRepository defines the interface for split rule persistence operations
type SplitRuleRepository interface {
	// GetBySourceBucketID retrieves a split rule by its source bucket ID
	GetBySourceBucketID(ctx context.Context, bucketID uuid.UUID) (*SplitRule, error)
}
