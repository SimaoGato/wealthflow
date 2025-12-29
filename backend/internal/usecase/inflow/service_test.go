package inflow

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

func (m *MockBucketRepository) List(ctx context.Context, typeFilter domain.BucketType) ([]*domain.Bucket, error) {
	args := m.Called(ctx, typeFilter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Bucket), args.Error(1)
}

// MockTransactionRepository is a mock implementation of TransactionRepository for testing
type MockTransactionRepository struct {
	mock.Mock
}

func (m *MockTransactionRepository) Create(ctx context.Context, tx *domain.Transaction) error {
	args := m.Called(ctx, tx)
	return args.Error(0)
}

func (m *MockTransactionRepository) List(ctx context.Context, limit, offset int, bucketID *uuid.UUID) ([]*domain.Transaction, error) {
	args := m.Called(ctx, limit, offset, bucketID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Transaction), args.Error(1)
}

// MockSplitRuleRepository is a mock implementation of SplitRuleRepository for testing
type MockSplitRuleRepository struct {
	mock.Mock
}

func (m *MockSplitRuleRepository) GetBySourceBucketID(ctx context.Context, bucketID uuid.UUID) (*domain.SplitRule, error) {
	args := m.Called(ctx, bucketID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.SplitRule), args.Error(1)
}

func TestRecordInflow_SalaryInflowWithSplitRule(t *testing.T) {
	ctx := context.Background()
	mockBucketRepo := new(MockBucketRepository)
	mockTxRepo := new(MockTransactionRepository)
	mockSplitRuleRepo := new(MockSplitRuleRepository)

	service := NewInflowService(mockBucketRepo, mockTxRepo, mockSplitRuleRepo)

	// Setup: Physical Bucket (Bank Account)
	physicalBucketID := uuid.New()

	// Setup: Income Source Bucket (Employer)
	incomeBucketID := uuid.New()
	incomeBucket := &domain.Bucket{
		ID:                     incomeBucketID,
		Name:                   "Employer",
		BucketType:             domain.BucketTypeIncome,
		ParentPhysicalBucketID: nil,
		CurrentBalance:         decimal.Zero,
	}

	// Setup: Virtual Buckets (Targets for split)
	vaultBucketID := uuid.New()
	vaultBucket := &domain.Bucket{
		ID:                     vaultBucketID,
		Name:                   "Vault",
		BucketType:             domain.BucketTypeVirtual,
		ParentPhysicalBucketID: &physicalBucketID,
		CurrentBalance:         decimal.NewFromInt(200),
	}

	freeCashBucketID := uuid.New()
	freeCashBucket := &domain.Bucket{
		ID:                     freeCashBucketID,
		Name:                   "Free Cash",
		BucketType:             domain.BucketTypeVirtual,
		ParentPhysicalBucketID: &physicalBucketID,
		CurrentBalance:         decimal.NewFromInt(300),
	}

	// Setup: Split Rule (Fixed 500€ to Vault, 30% of remainder to Free Cash, rest to a third bucket as remainder)
	// Note: Using a different bucket for remainder to avoid allocator overwrite issue
	emergencyBucketID := uuid.New()
	emergencyBucket := &domain.Bucket{
		ID:                     emergencyBucketID,
		Name:                   "Emergency Fund",
		BucketType:             domain.BucketTypeVirtual,
		ParentPhysicalBucketID: &physicalBucketID,
		CurrentBalance:         decimal.NewFromInt(100),
	}

	splitRuleID := uuid.New()
	splitRule := &domain.SplitRule{
		ID:             splitRuleID,
		Name:           "Salary Split",
		SourceBucketID: incomeBucketID,
		Items: []domain.SplitRuleItem{
			{
				ID:             uuid.New(),
				SplitRuleID:    splitRuleID,
				TargetBucketID: vaultBucketID,
				Type:           domain.SplitRuleItemTypeFixed,
				Value:          decimal.NewFromInt(500),
				Priority:       1,
			},
			{
				ID:             uuid.New(),
				SplitRuleID:    splitRuleID,
				TargetBucketID: freeCashBucketID,
				Type:           domain.SplitRuleItemTypePercent,
				Value:          decimal.NewFromInt(30),
				Priority:       2,
			},
			{
				ID:             uuid.New(),
				SplitRuleID:    splitRuleID,
				TargetBucketID: emergencyBucketID,
				Type:           domain.SplitRuleItemTypeRemainder,
				Value:          decimal.Zero,
				Priority:       3,
			},
		},
	}

	// Input: 2000€ salary inflow
	inflowAmount := decimal.NewFromInt(2000)
	input := RecordInflowInput{
		Amount:         inflowAmount,
		Description:    "Monthly Salary",
		SourceBucketID: incomeBucketID,
		IsExternal:     true,
	}

	// Expected allocation:
	// - Fixed: 500€ to Vault
	// - Remainder after fixed: 2000 - 500 = 1500€
	// - Percent (30% of 1500): 450€ to Free Cash
	// - Final remainder: 2000 - 500 - 450 = 1050€ to Emergency Fund
	expectedVaultTotal := decimal.NewFromInt(500)      // Fixed amount
	expectedFreeCashTotal := decimal.NewFromInt(450)   // 30% of 1500
	expectedEmergencyTotal := decimal.NewFromInt(1050) // Remainder

	// Mock repository calls
	mockBucketRepo.On("GetByID", ctx, incomeBucketID).Return(incomeBucket, nil)
	mockSplitRuleRepo.On("GetBySourceBucketID", ctx, incomeBucketID).Return(splitRule, nil)
	// GetByID for vaultBucketID: once for first item check, once in validation loop
	mockBucketRepo.On("GetByID", ctx, vaultBucketID).Return(vaultBucket, nil).Times(2)
	// GetByID for freeCashBucketID: once in validation loop
	mockBucketRepo.On("GetByID", ctx, freeCashBucketID).Return(freeCashBucket, nil).Once()
	// GetByID for emergencyBucketID: once in validation loop
	mockBucketRepo.On("GetByID", ctx, emergencyBucketID).Return(emergencyBucket, nil).Once()

	// Mock transaction creation
	mockTxRepo.On("Create", ctx, mock.MatchedBy(func(tx *domain.Transaction) bool {
		// Verify transaction structure
		if !tx.IsExternalInflow {
			return false
		}
		if tx.IsInternalTransfer {
			return false
		}
		if tx.Description != "Monthly Salary" {
			return false
		}

		// Verify we have the correct number of entries
		// Physical Layer: 2 entries (Debit Physical, Credit Income)
		// Virtual Layer: 4 entries (Debit Vault, Debit Free Cash, Debit Emergency, Credit Income)
		// Total: 6 entries
		if len(tx.Entries) != 6 {
			return false
		}

		// Separate entries by layer
		physicalEntries := make([]domain.TransactionEntry, 0)
		virtualEntries := make([]domain.TransactionEntry, 0)

		for _, entry := range tx.Entries {
			if entry.Layer == domain.LayerPhysical {
				physicalEntries = append(physicalEntries, entry)
			} else if entry.Layer == domain.LayerVirtual {
				virtualEntries = append(virtualEntries, entry)
			}
		}

		// Verify Physical Layer: 2 entries
		if len(physicalEntries) != 2 {
			return false
		}

		// Verify Physical Layer: Debit Physical Bucket, Credit Income Source
		var physicalDebitFound, physicalCreditFound bool
		for _, entry := range physicalEntries {
			if entry.BucketID == physicalBucketID && entry.Type == domain.EntryTypeDebit {
				if !entry.Amount.Equal(inflowAmount) {
					return false
				}
				physicalDebitFound = true
			}
			if entry.BucketID == incomeBucketID && entry.Type == domain.EntryTypeCredit {
				if !entry.Amount.Equal(inflowAmount) {
					return false
				}
				physicalCreditFound = true
			}
		}
		if !physicalDebitFound || !physicalCreditFound {
			return false
		}

		// Verify Virtual Layer: 4 entries (3 debits for targets, 1 credit for income)
		if len(virtualEntries) != 4 {
			return false
		}

		// Verify Virtual Layer: Multiple DEBIT entries for target buckets, one CREDIT for Income
		var vaultDebitFound, freeCashDebitFound, emergencyDebitFound, incomeCreditFound bool
		var vaultDebitAmount, freeCashDebitAmount, emergencyDebitAmount decimal.Decimal

		for _, entry := range virtualEntries {
			if entry.BucketID == vaultBucketID && entry.Type == domain.EntryTypeDebit {
				vaultDebitAmount = vaultDebitAmount.Add(entry.Amount)
				vaultDebitFound = true
			}
			if entry.BucketID == freeCashBucketID && entry.Type == domain.EntryTypeDebit {
				freeCashDebitAmount = freeCashDebitAmount.Add(entry.Amount)
				freeCashDebitFound = true
			}
			if entry.BucketID == emergencyBucketID && entry.Type == domain.EntryTypeDebit {
				emergencyDebitAmount = emergencyDebitAmount.Add(entry.Amount)
				emergencyDebitFound = true
			}
			if entry.BucketID == incomeBucketID && entry.Type == domain.EntryTypeCredit {
				if !entry.Amount.Equal(inflowAmount) {
					return false
				}
				incomeCreditFound = true
			}
		}

		if !vaultDebitFound || !freeCashDebitFound || !emergencyDebitFound || !incomeCreditFound {
			return false
		}

		// Verify allocation amounts
		if !vaultDebitAmount.Equal(expectedVaultTotal) {
			return false
		}
		if !freeCashDebitAmount.Equal(expectedFreeCashTotal) {
			return false
		}
		if !emergencyDebitAmount.Equal(expectedEmergencyTotal) {
			return false
		}

		return true
	})).Return(nil)

	// Execute
	result, err := service.RecordInflow(ctx, input)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsExternalInflow)
	assert.Equal(t, "Monthly Salary", result.Description)
	assert.Len(t, result.Entries, 6)

	// Verify all mocks were called
	mockBucketRepo.AssertExpectations(t)
	mockTxRepo.AssertExpectations(t)
	mockSplitRuleRepo.AssertExpectations(t)
}

func TestRecordInflow_InvalidAmount(t *testing.T) {
	ctx := context.Background()
	mockBucketRepo := new(MockBucketRepository)
	mockTxRepo := new(MockTransactionRepository)
	mockSplitRuleRepo := new(MockSplitRuleRepository)

	service := NewInflowService(mockBucketRepo, mockTxRepo, mockSplitRuleRepo)

	input := RecordInflowInput{
		Amount:         decimal.Zero,
		Description:    "Test",
		SourceBucketID: uuid.New(),
		IsExternal:     true,
	}

	result, err := service.RecordInflow(ctx, input)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "amount must be positive")
}

func TestRecordInflow_InvalidSourceBucketType(t *testing.T) {
	ctx := context.Background()
	mockBucketRepo := new(MockBucketRepository)
	mockTxRepo := new(MockTransactionRepository)
	mockSplitRuleRepo := new(MockSplitRuleRepository)

	service := NewInflowService(mockBucketRepo, mockTxRepo, mockSplitRuleRepo)

	// Setup: Physical Bucket (wrong type)
	physicalBucketID := uuid.New()
	physicalBucket := &domain.Bucket{
		ID:                     physicalBucketID,
		Name:                   "CGD Checking",
		BucketType:             domain.BucketTypePhysical,
		ParentPhysicalBucketID: nil,
		CurrentBalance:         decimal.NewFromInt(1000),
	}

	input := RecordInflowInput{
		Amount:         decimal.NewFromInt(1000),
		Description:    "Test",
		SourceBucketID: physicalBucketID,
		IsExternal:     true,
	}

	mockBucketRepo.On("GetByID", ctx, physicalBucketID).Return(physicalBucket, nil)

	result, err := service.RecordInflow(ctx, input)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "source bucket must be an income bucket")
}

func TestRecordInflow_InternalTransferNotImplemented(t *testing.T) {
	ctx := context.Background()
	mockBucketRepo := new(MockBucketRepository)
	mockTxRepo := new(MockTransactionRepository)
	mockSplitRuleRepo := new(MockSplitRuleRepository)

	service := NewInflowService(mockBucketRepo, mockTxRepo, mockSplitRuleRepo)

	incomeBucketID := uuid.New()
	incomeBucket := &domain.Bucket{
		ID:                     incomeBucketID,
		Name:                   "Employer",
		BucketType:             domain.BucketTypeIncome,
		ParentPhysicalBucketID: nil,
		CurrentBalance:         decimal.Zero,
	}

	input := RecordInflowInput{
		Amount:         decimal.NewFromInt(1000),
		Description:    "Test",
		SourceBucketID: incomeBucketID,
		IsExternal:     false,
	}

	mockBucketRepo.On("GetByID", ctx, incomeBucketID).Return(incomeBucket, nil)

	result, err := service.RecordInflow(ctx, input)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "internal transfer inflow not yet implemented")
}
