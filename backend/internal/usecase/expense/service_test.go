package expense

import (
	"context"
	"testing"

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

// MockTransactionRepository is a mock implementation of TransactionRepository for testing
type MockTransactionRepository struct {
	mock.Mock
}

func (m *MockTransactionRepository) Create(ctx context.Context, tx *domain.Transaction) error {
	args := m.Called(ctx, tx)
	return args.Error(0)
}

func TestLogExpense_StandardFlow(t *testing.T) {
	ctx := context.Background()
	mockBucketRepo := new(MockBucketRepository)
	mockTxRepo := new(MockTransactionRepository)

	service := NewExpenseService(mockBucketRepo, mockTxRepo)

	// Setup: Physical Bucket (Checking Account)
	physicalBucketID := uuid.New()

	// Setup: Virtual Bucket (Free Cash)
	virtualBucketID := uuid.New()
	virtualBucket := &domain.Bucket{
		ID:                     virtualBucketID,
		Name:                   "Free Cash",
		BucketType:             domain.BucketTypeVirtual,
		ParentPhysicalBucketID: &physicalBucketID,
		CurrentBalance:         decimal.NewFromInt(500),
	}

	// Setup: Category Bucket (Groceries)
	categoryBucketID := uuid.New()
	categoryBucket := &domain.Bucket{
		ID:                     categoryBucketID,
		Name:                   "Groceries",
		BucketType:             domain.BucketTypeExpense,
		ParentPhysicalBucketID: nil,
		CurrentBalance:         decimal.Zero,
	}

	// Mock repository calls
	mockBucketRepo.On("GetByID", ctx, virtualBucketID).Return(virtualBucket, nil)
	mockBucketRepo.On("GetByID", ctx, categoryBucketID).Return(categoryBucket, nil)

	// Input
	input := LogExpenseInput{
		Amount:             decimal.NewFromInt(50),
		Description:        "Weekly groceries",
		VirtualBucketID:    virtualBucketID,
		CategoryBucketID:   categoryBucketID,
		PhysicalOverrideID: nil, // Use virtual bucket's parent
	}

	// Mock transaction creation
	mockTxRepo.On("Create", ctx, mock.MatchedBy(func(tx *domain.Transaction) bool {
		// Verify transaction structure
		if len(tx.Entries) != 4 {
			return false
		}

		// Verify Physical Layer entries
		physicalCreditFound := false
		physicalDebitFound := false
		virtualCreditFound := false
		virtualDebitFound := false

		for _, entry := range tx.Entries {
			if entry.Layer == domain.LayerPhysical {
				if entry.BucketID == physicalBucketID && entry.Type == domain.EntryTypeCredit {
					physicalCreditFound = true
					assert.Equal(t, decimal.NewFromInt(50), entry.Amount)
				}
				if entry.BucketID == categoryBucketID && entry.Type == domain.EntryTypeDebit {
					physicalDebitFound = true
					assert.Equal(t, decimal.NewFromInt(50), entry.Amount)
				}
			}
			if entry.Layer == domain.LayerVirtual {
				if entry.BucketID == virtualBucketID && entry.Type == domain.EntryTypeCredit {
					virtualCreditFound = true
					assert.Equal(t, decimal.NewFromInt(50), entry.Amount)
				}
				if entry.BucketID == categoryBucketID && entry.Type == domain.EntryTypeDebit {
					virtualDebitFound = true
					assert.Equal(t, decimal.NewFromInt(50), entry.Amount)
				}
			}
		}

		return physicalCreditFound && physicalDebitFound && virtualCreditFound && virtualDebitFound
	})).Return(nil)

	// Execute
	result, err := service.LogExpense(ctx, input)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "Weekly groceries", result.Description)
	assert.Equal(t, 4, len(result.Entries))
	assert.False(t, result.IsInternalTransfer)
	assert.False(t, result.IsExternalInflow)

	// Verify all mocks were called
	mockBucketRepo.AssertExpectations(t)
	mockTxRepo.AssertExpectations(t)
}

func TestLogExpense_WrongCardOverride(t *testing.T) {
	ctx := context.Background()
	mockBucketRepo := new(MockBucketRepository)
	mockTxRepo := new(MockTransactionRepository)

	service := NewExpenseService(mockBucketRepo, mockTxRepo)

	// Setup: Physical Bucket 1 (Checking Account - Virtual's parent)
	physicalBucket1ID := uuid.New()

	// Setup: Physical Bucket 2 (Credit Card - Override)
	physicalBucket2ID := uuid.New()
	physicalBucket2 := &domain.Bucket{
		ID:                     physicalBucket2ID,
		Name:                   "Credit Card",
		BucketType:             domain.BucketTypePhysical,
		ParentPhysicalBucketID: nil,
		CurrentBalance:         decimal.NewFromInt(500),
	}

	// Setup: Virtual Bucket (Free Cash) - belongs to Checking
	virtualBucketID := uuid.New()
	virtualBucket := &domain.Bucket{
		ID:                     virtualBucketID,
		Name:                   "Free Cash",
		BucketType:             domain.BucketTypeVirtual,
		ParentPhysicalBucketID: &physicalBucket1ID,
		CurrentBalance:         decimal.NewFromInt(500),
	}

	// Setup: Category Bucket (Groceries)
	categoryBucketID := uuid.New()
	categoryBucket := &domain.Bucket{
		ID:                     categoryBucketID,
		Name:                   "Groceries",
		BucketType:             domain.BucketTypeExpense,
		ParentPhysicalBucketID: nil,
		CurrentBalance:         decimal.Zero,
	}

	// Mock repository calls
	mockBucketRepo.On("GetByID", ctx, virtualBucketID).Return(virtualBucket, nil)
	mockBucketRepo.On("GetByID", ctx, categoryBucketID).Return(categoryBucket, nil)
	mockBucketRepo.On("GetByID", ctx, physicalBucket2ID).Return(physicalBucket2, nil)

	// Input with override
	input := LogExpenseInput{
		Amount:             decimal.NewFromInt(75),
		Description:        "Groceries (wrong card)",
		VirtualBucketID:    virtualBucketID,
		CategoryBucketID:   categoryBucketID,
		PhysicalOverrideID: &physicalBucket2ID, // Override: Use Credit Card instead of Checking
	}

	// Mock transaction creation
	mockTxRepo.On("Create", ctx, mock.MatchedBy(func(tx *domain.Transaction) bool {
		// Verify transaction structure
		if len(tx.Entries) != 4 {
			return false
		}

		// Verify Physical Layer: Credit should point to override (Credit Card), not Checking
		physicalCreditFound := false
		physicalDebitFound := false
		virtualCreditFound := false
		virtualDebitFound := false

		for _, entry := range tx.Entries {
			if entry.Layer == domain.LayerPhysical {
				// Physical Credit should be Credit Card (override), not Checking
				if entry.BucketID == physicalBucket2ID && entry.Type == domain.EntryTypeCredit {
					physicalCreditFound = true
					assert.Equal(t, decimal.NewFromInt(75), entry.Amount)
				}
				if entry.BucketID == categoryBucketID && entry.Type == domain.EntryTypeDebit {
					physicalDebitFound = true
					assert.Equal(t, decimal.NewFromInt(75), entry.Amount)
				}
			}
			if entry.Layer == domain.LayerVirtual {
				// Virtual Credit should still be the Virtual Bucket (Free Cash)
				if entry.BucketID == virtualBucketID && entry.Type == domain.EntryTypeCredit {
					virtualCreditFound = true
					assert.Equal(t, decimal.NewFromInt(75), entry.Amount)
				}
				if entry.BucketID == categoryBucketID && entry.Type == domain.EntryTypeDebit {
					virtualDebitFound = true
					assert.Equal(t, decimal.NewFromInt(75), entry.Amount)
				}
			}
		}

		return physicalCreditFound && physicalDebitFound && virtualCreditFound && virtualDebitFound
	})).Return(nil)

	// Execute
	result, err := service.LogExpense(ctx, input)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "Groceries (wrong card)", result.Description)
	assert.Equal(t, 4, len(result.Entries))

	// Verify all mocks were called
	mockBucketRepo.AssertExpectations(t)
	mockTxRepo.AssertExpectations(t)
}

func TestLogExpense_ValidationFail(t *testing.T) {
	ctx := context.Background()
	mockBucketRepo := new(MockBucketRepository)
	mockTxRepo := new(MockTransactionRepository)

	service := NewExpenseService(mockBucketRepo, mockTxRepo)

	// Input with zero amount
	input := LogExpenseInput{
		Amount:             decimal.Zero,
		Description:        "Invalid expense",
		VirtualBucketID:    uuid.New(),
		CategoryBucketID:   uuid.New(),
		PhysicalOverrideID: nil,
	}

	// Execute
	result, err := service.LogExpense(ctx, input)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "expense amount must be positive")

	// Verify no repository calls were made
	mockBucketRepo.AssertNotCalled(t, "GetByID")
	mockTxRepo.AssertNotCalled(t, "Create")
}

func TestLogExpense_InvalidVirtualBucketType(t *testing.T) {
	ctx := context.Background()
	mockBucketRepo := new(MockBucketRepository)
	mockTxRepo := new(MockTransactionRepository)

	service := NewExpenseService(mockBucketRepo, mockTxRepo)

	// Setup: Physical Bucket (not Virtual)
	physicalBucketID := uuid.New()
	physicalBucket := &domain.Bucket{
		ID:                     physicalBucketID,
		Name:                   "CGD Checking",
		BucketType:             domain.BucketTypePhysical,
		ParentPhysicalBucketID: nil,
		CurrentBalance:         decimal.NewFromInt(1000),
	}

	// Mock repository call
	mockBucketRepo.On("GetByID", ctx, physicalBucketID).Return(physicalBucket, nil)

	// Input with physical bucket ID instead of virtual
	input := LogExpenseInput{
		Amount:             decimal.NewFromInt(50),
		Description:        "Invalid",
		VirtualBucketID:    physicalBucketID, // Wrong: This is a physical bucket
		CategoryBucketID:   uuid.New(),
		PhysicalOverrideID: nil,
	}

	// Execute
	result, err := service.LogExpense(ctx, input)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "virtual bucket ID must reference a virtual bucket")

	// Verify transaction repo was not called
	mockTxRepo.AssertNotCalled(t, "Create")
}

func TestLogExpense_InvalidCategoryBucketType(t *testing.T) {
	ctx := context.Background()
	mockBucketRepo := new(MockBucketRepository)
	mockTxRepo := new(MockTransactionRepository)

	service := NewExpenseService(mockBucketRepo, mockTxRepo)

	// Setup: Virtual Bucket
	virtualBucketID := uuid.New()
	physicalBucketID := uuid.New()
	virtualBucket := &domain.Bucket{
		ID:                     virtualBucketID,
		Name:                   "Free Cash",
		BucketType:             domain.BucketTypeVirtual,
		ParentPhysicalBucketID: &physicalBucketID,
		CurrentBalance:         decimal.NewFromInt(500),
	}

	// Setup: Income Bucket (not Expense)
	incomeBucketID := uuid.New()
	incomeBucket := &domain.Bucket{
		ID:                     incomeBucketID,
		Name:                   "Employer",
		BucketType:             domain.BucketTypeIncome,
		ParentPhysicalBucketID: nil,
		CurrentBalance:         decimal.Zero,
	}

	// Mock repository calls
	mockBucketRepo.On("GetByID", ctx, virtualBucketID).Return(virtualBucket, nil)
	mockBucketRepo.On("GetByID", ctx, incomeBucketID).Return(incomeBucket, nil)

	// Input with income bucket ID instead of expense
	input := LogExpenseInput{
		Amount:             decimal.NewFromInt(50),
		Description:        "Invalid",
		VirtualBucketID:    virtualBucketID,
		CategoryBucketID:   incomeBucketID, // Wrong: This is an income bucket
		PhysicalOverrideID: nil,
	}

	// Execute
	result, err := service.LogExpense(ctx, input)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "category bucket ID must reference an expense bucket")

	// Verify transaction repo was not called
	mockTxRepo.AssertNotCalled(t, "Create")
}

func TestLogExpense_InvalidPhysicalOverrideType(t *testing.T) {
	ctx := context.Background()
	mockBucketRepo := new(MockBucketRepository)
	mockTxRepo := new(MockTransactionRepository)

	service := NewExpenseService(mockBucketRepo, mockTxRepo)

	// Setup: Virtual Bucket
	virtualBucketID := uuid.New()
	physicalBucketID := uuid.New()
	virtualBucket := &domain.Bucket{
		ID:                     virtualBucketID,
		Name:                   "Free Cash",
		BucketType:             domain.BucketTypeVirtual,
		ParentPhysicalBucketID: &physicalBucketID,
		CurrentBalance:         decimal.NewFromInt(500),
	}

	// Setup: Category Bucket
	categoryBucketID := uuid.New()
	categoryBucket := &domain.Bucket{
		ID:                     categoryBucketID,
		Name:                   "Groceries",
		BucketType:             domain.BucketTypeExpense,
		ParentPhysicalBucketID: nil,
		CurrentBalance:         decimal.Zero,
	}

	// Setup: Virtual Bucket as override (wrong type)
	overrideVirtualBucketID := uuid.New()
	overrideVirtualBucket := &domain.Bucket{
		ID:                     overrideVirtualBucketID,
		Name:                   "Another Virtual",
		BucketType:             domain.BucketTypeVirtual,
		ParentPhysicalBucketID: &physicalBucketID,
		CurrentBalance:         decimal.NewFromInt(200),
	}

	// Mock repository calls
	mockBucketRepo.On("GetByID", ctx, virtualBucketID).Return(virtualBucket, nil)
	mockBucketRepo.On("GetByID", ctx, categoryBucketID).Return(categoryBucket, nil)
	mockBucketRepo.On("GetByID", ctx, overrideVirtualBucketID).Return(overrideVirtualBucket, nil)

	// Input with virtual bucket as override (should fail)
	input := LogExpenseInput{
		Amount:             decimal.NewFromInt(50),
		Description:        "Invalid override",
		VirtualBucketID:    virtualBucketID,
		CategoryBucketID:   categoryBucketID,
		PhysicalOverrideID: &overrideVirtualBucketID, // Wrong: This is a virtual bucket
	}

	// Execute
	result, err := service.LogExpense(ctx, input)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "physical override bucket must be a physical bucket")

	// Verify transaction repo was not called
	mockTxRepo.AssertNotCalled(t, "Create")
}
