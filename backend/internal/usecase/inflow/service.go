package inflow

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/simaogato/wealthflow-backend/internal/domain"
	"github.com/simaogato/wealthflow-backend/internal/usecase/allocator"
)

// RecordInflowInput represents the input for recording an inflow
type RecordInflowInput struct {
	Amount         decimal.Decimal
	Description    string
	SourceBucketID uuid.UUID
	IsExternal     bool
}

// InflowService handles inflow recording operations
type InflowService struct {
	BucketRepo      domain.BucketRepository
	TransactionRepo domain.TransactionRepository
	SplitRuleRepo   domain.SplitRuleRepository
}

// NewInflowService creates a new InflowService instance
func NewInflowService(
	bucketRepo domain.BucketRepository,
	transactionRepo domain.TransactionRepository,
	splitRuleRepo domain.SplitRuleRepository,
) *InflowService {
	return &InflowService{
		BucketRepo:      bucketRepo,
		TransactionRepo: transactionRepo,
		SplitRuleRepo:   splitRuleRepo,
	}
}

// RecordInflow creates a transaction for an inflow
// Logic:
//  1. Fetch Source Bucket
//  2. If IsExternal is true:
//     - Fetch the Split Rule for this source bucket
//     - Call allocator.CalculateAllocation to get target buckets
//     - Create Transaction:
//     - Physical Layer: Debit Source's Parent Physical (Bank), Credit Source (Income Bucket)
//     - Virtual Layer: Debit Target Buckets (from allocation), Credit Source (Income Bucket)
//  3. If IsExternal is false (Internal Transfer):
//     - For this task, focus on External logic as priority
func (s *InflowService) RecordInflow(ctx context.Context, input RecordInflowInput) (*domain.Transaction, error) {
	// Validate input
	if input.Amount.LessThanOrEqual(decimal.Zero) {
		return nil, errors.New("inflow amount must be positive")
	}

	// 1. Fetch Source Bucket
	sourceBucket, err := s.BucketRepo.GetByID(ctx, input.SourceBucketID)
	if err != nil {
		return nil, err
	}

	// Validate source bucket type
	if sourceBucket.BucketType != domain.BucketTypeIncome {
		return nil, errors.New("source bucket must be an income bucket")
	}

	// 2. Handle External Inflow
	if input.IsExternal {
		return s.recordExternalInflow(ctx, input, sourceBucket)
	}

	// 3. Internal Transfer (simplified for now - focus on external as priority)
	// TODO: Implement internal transfer logic if needed
	return nil, errors.New("internal transfer inflow not yet implemented")
}

// recordExternalInflow handles external inflow with split rule allocation
func (s *InflowService) recordExternalInflow(
	ctx context.Context,
	input RecordInflowInput,
	sourceBucket *domain.Bucket,
) (*domain.Transaction, error) {
	// Fetch Split Rule for this source bucket
	splitRule, err := s.SplitRuleRepo.GetBySourceBucketID(ctx, input.SourceBucketID)
	if err != nil {
		return nil, err
	}

	// Calculate allocation using the allocator
	allocation, err := allocator.CalculateAllocation(input.Amount, splitRule.Items)
	if err != nil {
		return nil, err
	}

	// Determine the parent physical bucket for the source income bucket.
	// Logic: We infer the physical destination bucket from the first virtual target in the split rule.
	// Assumption: All split targets in a single rule belong to the same physical bucket (e.g., Bank Account).

	// Get the first target bucket to determine parent physical
	if len(splitRule.Items) == 0 {
		return nil, errors.New("split rule must have at least one item")
	}

	firstTargetBucketID := splitRule.Items[0].TargetBucketID
	firstTargetBucket, err := s.BucketRepo.GetByID(ctx, firstTargetBucketID)
	if err != nil {
		return nil, err
	}

	// Validate target bucket is virtual
	if firstTargetBucket.BucketType != domain.BucketTypeVirtual {
		return nil, errors.New("split rule target buckets must be virtual buckets")
	}

	// Get parent physical bucket from the first target virtual bucket
	if firstTargetBucket.ParentPhysicalBucketID == nil {
		return nil, errors.New("virtual bucket must have a parent physical bucket")
	}
	parentPhysicalBucketID := *firstTargetBucket.ParentPhysicalBucketID

	// Verify all target buckets belong to the same parent physical bucket
	// (This is a business rule: all split targets should be in the same physical bucket)
	for bucketID := range allocation {
		targetBucket, err := s.BucketRepo.GetByID(ctx, bucketID)
		if err != nil {
			return nil, err
		}
		if targetBucket.BucketType != domain.BucketTypeVirtual {
			return nil, errors.New("all split rule target buckets must be virtual buckets")
		}
		if targetBucket.ParentPhysicalBucketID == nil {
			return nil, errors.New("virtual bucket must have a parent physical bucket")
		}
		if *targetBucket.ParentPhysicalBucketID != parentPhysicalBucketID {
			return nil, errors.New("all split rule target buckets must belong to the same parent physical bucket")
		}
	}

	// Create Transaction
	txID := uuid.New()
	now := time.Now()
	entries := make([]domain.TransactionEntry, 0)

	// Physical Layer: Debit Parent Physical Bucket (Bank - increase asset), Credit Income Source
	physicalDebitEntry := domain.TransactionEntry{
		ID:            uuid.New(),
		TransactionID: txID,
		BucketID:      parentPhysicalBucketID,
		Amount:        input.Amount,
		Type:          domain.EntryTypeDebit,
		Layer:         domain.LayerPhysical,
	}

	physicalCreditEntry := domain.TransactionEntry{
		ID:            uuid.New(),
		TransactionID: txID,
		BucketID:      input.SourceBucketID,
		Amount:        input.Amount,
		Type:          domain.EntryTypeCredit,
		Layer:         domain.LayerPhysical,
	}

	entries = append(entries, physicalDebitEntry, physicalCreditEntry)

	// Virtual Layer: Debit Target Virtual Buckets (from allocation), Credit Income Source
	for targetBucketID, allocatedAmount := range allocation {
		virtualDebitEntry := domain.TransactionEntry{
			ID:            uuid.New(),
			TransactionID: txID,
			BucketID:      targetBucketID,
			Amount:        allocatedAmount,
			Type:          domain.EntryTypeDebit,
			Layer:         domain.LayerVirtual,
		}
		entries = append(entries, virtualDebitEntry)
	}

	// Credit Income Source in Virtual Layer
	virtualCreditEntry := domain.TransactionEntry{
		ID:            uuid.New(),
		TransactionID: txID,
		BucketID:      input.SourceBucketID,
		Amount:        input.Amount,
		Type:          domain.EntryTypeCredit,
		Layer:         domain.LayerVirtual,
	}
	entries = append(entries, virtualCreditEntry)

	tx := &domain.Transaction{
		ID:                 txID,
		Description:        input.Description,
		Date:               now,
		IsInternalTransfer: false,
		IsExternalInflow:   true,
		Entries:            entries,
	}

	// Validate transaction
	if err := tx.Validate(); err != nil {
		return nil, err
	}

	// Save using TransactionRepo.Create
	if err := s.TransactionRepo.Create(ctx, tx); err != nil {
		return nil, err
	}

	return tx, nil
}
