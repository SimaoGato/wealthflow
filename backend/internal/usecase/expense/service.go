package expense

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/simaogato/wealthflow-backend/internal/domain"
)

// LogExpenseInput represents the input for logging an expense
type LogExpenseInput struct {
	Amount             decimal.Decimal
	Description        string
	VirtualBucketID    uuid.UUID
	CategoryBucketID   uuid.UUID
	PhysicalOverrideID *uuid.UUID // Optional: Override the physical bucket source
}

// ExpenseService handles expense logging operations
type ExpenseService struct {
	BucketRepo      domain.BucketRepository
	TransactionRepo domain.TransactionRepository
}

// NewExpenseService creates a new ExpenseService instance
func NewExpenseService(bucketRepo domain.BucketRepository, transactionRepo domain.TransactionRepository) *ExpenseService {
	return &ExpenseService{
		BucketRepo:      bucketRepo,
		TransactionRepo: transactionRepo,
	}
}

// LogExpense creates a transaction for an expense with double-layer entries
// Logic:
//  1. Fetch Virtual Bucket and Category Bucket
//  2. Determine Source Physical Bucket (override or parent)
//  3. Create Transaction with 4 entries:
//     - Physical Layer: Credit Source Physical, Debit Category
//     - Virtual Layer: Credit Virtual Bucket, Debit Category
//  4. Validate transaction
//  5. Save using TransactionRepo.Create
func (s *ExpenseService) LogExpense(ctx context.Context, input LogExpenseInput) (*domain.Transaction, error) {
	// Validate input
	if input.Amount.LessThanOrEqual(decimal.Zero) {
		return nil, errors.New("expense amount must be positive")
	}

	// 1. Fetch Virtual Bucket and Category Bucket
	virtualBucket, err := s.BucketRepo.GetByID(ctx, input.VirtualBucketID)
	if err != nil {
		return nil, err
	}

	// Validate virtual bucket type
	if virtualBucket.BucketType != domain.BucketTypeVirtual {
		return nil, errors.New("virtual bucket ID must reference a virtual bucket")
	}

	categoryBucket, err := s.BucketRepo.GetByID(ctx, input.CategoryBucketID)
	if err != nil {
		return nil, err
	}

	// Validate category bucket type
	if categoryBucket.BucketType != domain.BucketTypeExpense {
		return nil, errors.New("category bucket ID must reference an expense bucket")
	}

	// 2. Determine Source Physical Bucket
	var sourcePhysicalBucketID uuid.UUID
	if input.PhysicalOverrideID != nil {
		// Use override if provided
		sourcePhysicalBucketID = *input.PhysicalOverrideID
		// Validate that the override bucket exists and is physical
		overrideBucket, err := s.BucketRepo.GetByID(ctx, sourcePhysicalBucketID)
		if err != nil {
			return nil, err
		}
		if overrideBucket.BucketType != domain.BucketTypePhysical {
			return nil, errors.New("physical override bucket must be a physical bucket")
		}
	} else {
		// Use virtual bucket's parent physical bucket
		if virtualBucket.ParentPhysicalBucketID == nil {
			return nil, errors.New("virtual bucket must have a parent physical bucket ID")
		}
		sourcePhysicalBucketID = *virtualBucket.ParentPhysicalBucketID
	}

	// 3. Create Transaction with 4 entries
	txID := uuid.New()
	now := time.Now()

	// Physical Layer: Credit Source Physical (decrease asset), Debit Category (increase expense)
	physicalCreditEntry := domain.TransactionEntry{
		ID:            uuid.New(),
		TransactionID: txID,
		BucketID:      sourcePhysicalBucketID,
		Amount:        input.Amount,
		Type:          domain.EntryTypeCredit,
		Layer:         domain.LayerPhysical,
	}

	physicalDebitEntry := domain.TransactionEntry{
		ID:            uuid.New(),
		TransactionID: txID,
		BucketID:      input.CategoryBucketID,
		Amount:        input.Amount,
		Type:          domain.EntryTypeDebit,
		Layer:         domain.LayerPhysical,
	}

	// Virtual Layer: Credit Virtual Bucket (decrease available funds), Debit Category (increase expense)
	virtualCreditEntry := domain.TransactionEntry{
		ID:            uuid.New(),
		TransactionID: txID,
		BucketID:      input.VirtualBucketID,
		Amount:        input.Amount,
		Type:          domain.EntryTypeCredit,
		Layer:         domain.LayerVirtual,
	}

	virtualDebitEntry := domain.TransactionEntry{
		ID:            uuid.New(),
		TransactionID: txID,
		BucketID:      input.CategoryBucketID,
		Amount:        input.Amount,
		Type:          domain.EntryTypeDebit,
		Layer:         domain.LayerVirtual,
	}

	tx := &domain.Transaction{
		ID:                 txID,
		Description:        input.Description,
		Date:               now,
		IsInternalTransfer: false,
		IsExternalInflow:   false,
		Entries: []domain.TransactionEntry{
			physicalCreditEntry,
			physicalDebitEntry,
			virtualCreditEntry,
			virtualDebitEntry,
		},
	}

	// 4. Validate transaction
	if err := tx.Validate(); err != nil {
		return nil, err
	}

	// 5. Save using TransactionRepo.Create
	if err := s.TransactionRepo.Create(ctx, tx); err != nil {
		return nil, err
	}

	return tx, nil
}
