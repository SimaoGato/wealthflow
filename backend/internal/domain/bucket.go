package domain

import (
	"errors"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// BucketType represents the type of bucket in the system
type BucketType string

const (
	BucketTypePhysical BucketType = "PHYSICAL"
	BucketTypeVirtual  BucketType = "VIRTUAL"
	BucketTypeIncome   BucketType = "INCOME"
	BucketTypeExpense  BucketType = "EXPENSE"
	BucketTypeEquity   BucketType = "EQUITY"
	BucketTypeSystem   BucketType = "SYSTEM"
)

// Bucket represents a bucket entity in the domain layer
// Adheres to the data model defined in specs.md
type Bucket struct {
	ID                     uuid.UUID
	Name                   string
	BucketType             BucketType
	ParentPhysicalBucketID *uuid.UUID      // NULL if PHYSICAL/INCOME/EXPENSE. NOT NULL if VIRTUAL.
	CurrentBalance         decimal.Decimal // Represents BOOK VALUE (Cash in/out)
}

// Validate ensures the bucket adheres to domain rules
// Returns an error if validation fails
func (b *Bucket) Validate() error {
	if b.Name == "" {
		return errors.New("bucket name cannot be empty")
	}

	// Virtual Buckets MUST have a Parent Physical Bucket ID
	if b.BucketType == BucketTypeVirtual {
		if b.ParentPhysicalBucketID == nil {
			return errors.New("virtual bucket must have a parent physical bucket ID")
		}
	}

	// Physical, Income, Expense, Equity, and System buckets do NOT need a Parent ID
	// (This is implicit - we only validate the constraint for Virtual buckets)

	return nil
}
