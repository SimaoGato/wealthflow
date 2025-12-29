package domain

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// TransferTask represents a task to move money between physical buckets
// Adheres to the data model defined in specs.md Block B (FR-03)
type TransferTask struct {
	ID                     uuid.UUID
	RelatedTransactionID   uuid.UUID
	CompletedTransactionID *uuid.UUID // NULL until task is completed
	FromPhysicalBucketID   uuid.UUID
	ToPhysicalBucketID     uuid.UUID
	Amount                 decimal.Decimal
	IsCompleted            bool
}
