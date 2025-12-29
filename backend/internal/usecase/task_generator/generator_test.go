package task_generator

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/simaogato/wealthflow-backend/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockBucketRepository is a mock implementation of BucketRepository for testing
type MockBucketRepository struct {
	mock.Mock
}

func (m *MockBucketRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Bucket, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Bucket), args.Error(1)
}

func (m *MockBucketRepository) Create(ctx context.Context, bucket *domain.Bucket) error {
	args := m.Called(ctx, bucket)
	return args.Error(0)
}

func (m *MockBucketRepository) GetSystemBucket(ctx context.Context, bucketType domain.BucketType) (*domain.Bucket, error) {
	args := m.Called(ctx, bucketType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Bucket), args.Error(1)
}

func TestGenerateTasks_PaydayScenario(t *testing.T) {
	// Payday scenario: Money moves from Income (External) to Savings (Physical)
	// This should NOT generate a task because Income is an external bucket
	// and the money is going directly into a physical bucket (not between physical buckets)

	ctx := context.Background()
	mockRepo := new(MockBucketRepository)

	// Setup: Income bucket (External)
	incomeBucketID := uuid.New()
	incomeBucket := &domain.Bucket{
		ID:                     incomeBucketID,
		Name:                   "Employer",
		BucketType:             domain.BucketTypeIncome,
		ParentPhysicalBucketID: nil,
		CurrentBalance:         decimal.Zero,
	}

	// Setup: Physical Savings bucket
	savingsPhysicalID := uuid.New()

	// Setup: Virtual bucket inside Savings
	savingsVirtualID := uuid.New()
	savingsVirtualBucket := &domain.Bucket{
		ID:                     savingsVirtualID,
		Name:                   "Emergency Fund",
		BucketType:             domain.BucketTypeVirtual,
		ParentPhysicalBucketID: &savingsPhysicalID,
		CurrentBalance:         decimal.Zero,
	}

	// Mock repository calls
	mockRepo.On("GetByID", ctx, incomeBucketID).Return(incomeBucket, nil)
	mockRepo.On("GetByID", ctx, savingsVirtualID).Return(savingsVirtualBucket, nil)

	// Transaction: Income -> Virtual Savings
	// Physical Layer: Debit Savings Physical, Credit Income
	// Virtual Layer: Debit Virtual Savings, Credit Income
	tx := domain.Transaction{
		ID:                 uuid.New(),
		Description:        "Payday",
		Date:               time.Now(),
		IsInternalTransfer: false,
		IsExternalInflow:   true,
		Entries: []domain.TransactionEntry{
			// Physical Layer
			{
				ID:            uuid.New(),
				TransactionID: uuid.New(),
				BucketID:      savingsPhysicalID,
				Amount:        decimal.NewFromInt(1000),
				Type:          domain.EntryTypeDebit,
				Layer:         domain.LayerPhysical,
			},
			{
				ID:            uuid.New(),
				TransactionID: uuid.New(),
				BucketID:      incomeBucketID,
				Amount:        decimal.NewFromInt(1000),
				Type:          domain.EntryTypeCredit,
				Layer:         domain.LayerPhysical,
			},
			// Virtual Layer
			{
				ID:            uuid.New(),
				TransactionID: uuid.New(),
				BucketID:      savingsVirtualID,
				Amount:        decimal.NewFromInt(1000),
				Type:          domain.EntryTypeDebit,
				Layer:         domain.LayerVirtual,
			},
			{
				ID:            uuid.New(),
				TransactionID: uuid.New(),
				BucketID:      incomeBucketID,
				Amount:        decimal.NewFromInt(1000),
				Type:          domain.EntryTypeCredit,
				Layer:         domain.LayerVirtual,
			},
		},
	}

	tasks, err := GenerateTasks(ctx, tx, mockRepo)
	assert.NoError(t, err)
	assert.Empty(t, tasks, "Payday scenario should not generate tasks (Income is external bucket)")

	mockRepo.AssertExpectations(t)
}

func TestGenerateTasks_InterBankTransfer(t *testing.T) {
	// Inter-Bank Transfer: Virtual Bucket A (Bank 1) -> Virtual Bucket B (Bank 2)
	// This SHOULD generate a task because money moves between different physical buckets

	ctx := context.Background()
	mockRepo := new(MockBucketRepository)

	// Setup: Physical Bank 1
	bank1PhysicalID := uuid.New()

	// Setup: Virtual bucket in Bank 1
	bank1VirtualID := uuid.New()
	bank1VirtualBucket := &domain.Bucket{
		ID:                     bank1VirtualID,
		Name:                   "Free Cash (CGD)",
		BucketType:             domain.BucketTypeVirtual,
		ParentPhysicalBucketID: &bank1PhysicalID,
		CurrentBalance:         decimal.Zero,
	}

	// Setup: Physical Bank 2
	bank2PhysicalID := uuid.New()

	// Setup: Virtual bucket in Bank 2
	bank2VirtualID := uuid.New()
	bank2VirtualBucket := &domain.Bucket{
		ID:                     bank2VirtualID,
		Name:                   "Investment Fund (XTB)",
		BucketType:             domain.BucketTypeVirtual,
		ParentPhysicalBucketID: &bank2PhysicalID,
		CurrentBalance:         decimal.Zero,
	}

	// Mock repository calls
	mockRepo.On("GetByID", ctx, bank1VirtualID).Return(bank1VirtualBucket, nil)
	mockRepo.On("GetByID", ctx, bank2VirtualID).Return(bank2VirtualBucket, nil)

	// Transaction: Virtual Bank 1 -> Virtual Bank 2
	// Physical Layer: Credit Bank 1 Physical, Debit Bank 2 Physical
	// Virtual Layer: Credit Bank 1 Virtual, Debit Bank 2 Virtual
	transferAmount := decimal.NewFromInt(500)
	tx := domain.Transaction{
		ID:                 uuid.New(),
		Description:        "Transfer to Investment",
		Date:               time.Now(),
		IsInternalTransfer: true,
		IsExternalInflow:   false,
		Entries: []domain.TransactionEntry{
			// Physical Layer
			{
				ID:            uuid.New(),
				TransactionID: uuid.New(),
				BucketID:      bank1PhysicalID,
				Amount:        transferAmount,
				Type:          domain.EntryTypeCredit,
				Layer:         domain.LayerPhysical,
			},
			{
				ID:            uuid.New(),
				TransactionID: uuid.New(),
				BucketID:      bank2PhysicalID,
				Amount:        transferAmount,
				Type:          domain.EntryTypeDebit,
				Layer:         domain.LayerPhysical,
			},
			// Virtual Layer
			{
				ID:            uuid.New(),
				TransactionID: uuid.New(),
				BucketID:      bank1VirtualID,
				Amount:        transferAmount,
				Type:          domain.EntryTypeCredit,
				Layer:         domain.LayerVirtual,
			},
			{
				ID:            uuid.New(),
				TransactionID: uuid.New(),
				BucketID:      bank2VirtualID,
				Amount:        transferAmount,
				Type:          domain.EntryTypeDebit,
				Layer:         domain.LayerVirtual,
			},
		},
	}

	tasks, err := GenerateTasks(ctx, tx, mockRepo)
	assert.NoError(t, err)
	assert.Len(t, tasks, 1, "Inter-bank transfer should generate exactly one task")

	if len(tasks) > 0 {
		task := tasks[0]
		assert.Equal(t, bank1PhysicalID, task.FromPhysicalBucketID, "Task should be from Bank 1")
		assert.Equal(t, bank2PhysicalID, task.ToPhysicalBucketID, "Task should be to Bank 2")
		assert.True(t, task.Amount.Equal(transferAmount), "Task amount should match transfer amount")
		assert.Equal(t, tx.ID, task.RelatedTransactionID, "Task should reference the transaction")
		assert.False(t, task.IsCompleted, "Task should not be completed initially")
	}

	mockRepo.AssertExpectations(t)
}

func TestGenerateTasks_IntraBankTransfer(t *testing.T) {
	// Intra-Bank Transfer: Virtual Bucket A (Bank 1) -> Virtual Bucket B (Bank 1)
	// This should NOT generate a task because money moves within the same physical bucket

	ctx := context.Background()
	mockRepo := new(MockBucketRepository)

	// Setup: Physical Bank 1
	bank1PhysicalID := uuid.New()

	// Setup: Virtual bucket A in Bank 1
	bank1VirtualAID := uuid.New()
	bank1VirtualABucket := &domain.Bucket{
		ID:                     bank1VirtualAID,
		Name:                   "Free Cash (CGD)",
		BucketType:             domain.BucketTypeVirtual,
		ParentPhysicalBucketID: &bank1PhysicalID,
		CurrentBalance:         decimal.Zero,
	}

	// Setup: Virtual bucket B in Bank 1
	bank1VirtualBID := uuid.New()
	bank1VirtualBBucket := &domain.Bucket{
		ID:                     bank1VirtualBID,
		Name:                   "Emergency Fund (CGD)",
		BucketType:             domain.BucketTypeVirtual,
		ParentPhysicalBucketID: &bank1PhysicalID,
		CurrentBalance:         decimal.Zero,
	}

	// Mock repository calls
	mockRepo.On("GetByID", ctx, bank1VirtualAID).Return(bank1VirtualABucket, nil)
	mockRepo.On("GetByID", ctx, bank1VirtualBID).Return(bank1VirtualBBucket, nil)

	// Transaction: Virtual A -> Virtual B (both in same physical bucket)
	// Physical Layer: No change (money stays in same physical bucket)
	// Virtual Layer: Credit Virtual A, Debit Virtual B
	transferAmount := decimal.NewFromInt(200)
	tx := domain.Transaction{
		ID:                 uuid.New(),
		Description:        "Move to Emergency Fund",
		Date:               time.Now(),
		IsInternalTransfer: true,
		IsExternalInflow:   false,
		Entries: []domain.TransactionEntry{
			// Virtual Layer only (no physical movement)
			{
				ID:            uuid.New(),
				TransactionID: uuid.New(),
				BucketID:      bank1VirtualAID,
				Amount:        transferAmount,
				Type:          domain.EntryTypeCredit,
				Layer:         domain.LayerVirtual,
			},
			{
				ID:            uuid.New(),
				TransactionID: uuid.New(),
				BucketID:      bank1VirtualBID,
				Amount:        transferAmount,
				Type:          domain.EntryTypeDebit,
				Layer:         domain.LayerVirtual,
			},
		},
	}

	tasks, err := GenerateTasks(ctx, tx, mockRepo)
	assert.NoError(t, err)
	assert.Empty(t, tasks, "Intra-bank transfer should not generate tasks (same physical bucket)")

	mockRepo.AssertExpectations(t)
}

func TestGenerateTasks_NoVirtualEntries(t *testing.T) {
	// Transaction with only physical layer entries should not generate tasks
	ctx := context.Background()
	mockRepo := new(MockBucketRepository)

	bank1ID := uuid.New()
	bank2ID := uuid.New()

	tx := domain.Transaction{
		ID:          uuid.New(),
		Description: "Physical Transfer",
		Date:        time.Now(),
		Entries: []domain.TransactionEntry{
			{
				ID:            uuid.New(),
				TransactionID: uuid.New(),
				BucketID:      bank1ID,
				Amount:        decimal.NewFromInt(100),
				Type:          domain.EntryTypeCredit,
				Layer:         domain.LayerPhysical,
			},
			{
				ID:            uuid.New(),
				TransactionID: uuid.New(),
				BucketID:      bank2ID,
				Amount:        decimal.NewFromInt(100),
				Type:          domain.EntryTypeDebit,
				Layer:         domain.LayerPhysical,
			},
		},
	}

	tasks, err := GenerateTasks(ctx, tx, mockRepo)
	assert.NoError(t, err)
	assert.Empty(t, tasks, "Transaction with no virtual entries should not generate tasks")
}
