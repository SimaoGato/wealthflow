package investment

import (
	"context"
	"errors"
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

// MockMarketValueRepository is a mock implementation of MarketValueRepository for testing
type MockMarketValueRepository struct {
	mock.Mock
}

func (m *MockMarketValueRepository) Add(ctx context.Context, entry *domain.MarketValueHistory) error {
	args := m.Called(ctx, entry)
	return args.Error(0)
}

func (m *MockMarketValueRepository) GetLatest(ctx context.Context, bucketID uuid.UUID) (*domain.MarketValueHistory, error) {
	args := m.Called(ctx, bucketID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.MarketValueHistory), args.Error(1)
}

func TestCalculateProfit_ProfitScenario(t *testing.T) {
	ctx := context.Background()
	mockBucketRepo := new(MockBucketRepository)
	mockMarketValueRepo := new(MockMarketValueRepository)

	service := NewInvestmentService(mockBucketRepo, mockMarketValueRepo)

	// Setup: Bucket with Book Value = 1000
	bucketID := uuid.New()
	bucket := &domain.Bucket{
		ID:             bucketID,
		Name:           "XTB Portfolio",
		BucketType:     domain.BucketTypeEquity,
		CurrentBalance: decimal.NewFromInt(1000), // Book Value
	}

	// Setup: Latest Market Value = 1200
	marketValueEntry := &domain.MarketValueHistory{
		ID:          uuid.New(),
		BucketID:    bucketID,
		MarketValue: decimal.NewFromInt(1200),
	}

	// Mock repository calls
	mockBucketRepo.On("GetByID", ctx, bucketID).Return(bucket, nil)
	mockMarketValueRepo.On("GetLatest", ctx, bucketID).Return(marketValueEntry, nil)

	// Execute
	profit, err := service.CalculateProfit(ctx, bucketID)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, decimal.NewFromInt(200), profit) // 1200 - 1000 = +200

	// Verify all mocks were called
	mockBucketRepo.AssertExpectations(t)
	mockMarketValueRepo.AssertExpectations(t)
}

func TestCalculateProfit_LossScenario(t *testing.T) {
	ctx := context.Background()
	mockBucketRepo := new(MockBucketRepository)
	mockMarketValueRepo := new(MockMarketValueRepository)

	service := NewInvestmentService(mockBucketRepo, mockMarketValueRepo)

	// Setup: Bucket with Book Value = 1000
	bucketID := uuid.New()
	bucket := &domain.Bucket{
		ID:             bucketID,
		Name:           "XTB Portfolio",
		BucketType:     domain.BucketTypeEquity,
		CurrentBalance: decimal.NewFromInt(1000), // Book Value
	}

	// Setup: Latest Market Value = 900
	marketValueEntry := &domain.MarketValueHistory{
		ID:          uuid.New(),
		BucketID:    bucketID,
		MarketValue: decimal.NewFromInt(900),
	}

	// Mock repository calls
	mockBucketRepo.On("GetByID", ctx, bucketID).Return(bucket, nil)
	mockMarketValueRepo.On("GetLatest", ctx, bucketID).Return(marketValueEntry, nil)

	// Execute
	profit, err := service.CalculateProfit(ctx, bucketID)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, decimal.NewFromInt(-100), profit) // 900 - 1000 = -100

	// Verify all mocks were called
	mockBucketRepo.AssertExpectations(t)
	mockMarketValueRepo.AssertExpectations(t)
}

func TestCalculateProfit_NoHistory(t *testing.T) {
	ctx := context.Background()
	mockBucketRepo := new(MockBucketRepository)
	mockMarketValueRepo := new(MockMarketValueRepository)

	service := NewInvestmentService(mockBucketRepo, mockMarketValueRepo)

	// Setup: Bucket with Book Value = 1000
	bucketID := uuid.New()
	bucket := &domain.Bucket{
		ID:             bucketID,
		Name:           "XTB Portfolio",
		BucketType:     domain.BucketTypeEquity,
		CurrentBalance: decimal.NewFromInt(1000), // Book Value
	}

	// Setup: No market value history exists
	mockBucketRepo.On("GetByID", ctx, bucketID).Return(bucket, nil)
	mockMarketValueRepo.On("GetLatest", ctx, bucketID).Return(nil, errors.New("no market value history found"))

	// Execute
	profit, err := service.CalculateProfit(ctx, bucketID)

	// Assert
	assert.NoError(t, err)                // Should return 0 gracefully, not error
	assert.Equal(t, decimal.Zero, profit) // Profit should be 0 when no history exists

	// Verify all mocks were called
	mockBucketRepo.AssertExpectations(t)
	mockMarketValueRepo.AssertExpectations(t)
}

func TestCalculateProfit_BucketNotFound(t *testing.T) {
	ctx := context.Background()
	mockBucketRepo := new(MockBucketRepository)
	mockMarketValueRepo := new(MockMarketValueRepository)

	service := NewInvestmentService(mockBucketRepo, mockMarketValueRepo)

	// Setup: Bucket does not exist
	bucketID := uuid.New()
	mockBucketRepo.On("GetByID", ctx, bucketID).Return(nil, errors.New("bucket not found"))

	// Execute
	profit, err := service.CalculateProfit(ctx, bucketID)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, decimal.Zero, profit)
	assert.Contains(t, err.Error(), "bucket not found")

	// Verify market value repo was not called
	mockMarketValueRepo.AssertNotCalled(t, "GetLatest")
}

func TestUpdateMarketValue_Success(t *testing.T) {
	ctx := context.Background()
	mockBucketRepo := new(MockBucketRepository)
	mockMarketValueRepo := new(MockMarketValueRepository)

	service := NewInvestmentService(mockBucketRepo, mockMarketValueRepo)

	// Setup: Bucket exists
	bucketID := uuid.New()
	bucket := &domain.Bucket{
		ID:             bucketID,
		Name:           "XTB Portfolio",
		BucketType:     domain.BucketTypeEquity,
		CurrentBalance: decimal.NewFromInt(1000),
	}

	// Mock repository calls
	mockBucketRepo.On("GetByID", ctx, bucketID).Return(bucket, nil)
	mockMarketValueRepo.On("Add", ctx, mock.MatchedBy(func(entry *domain.MarketValueHistory) bool {
		return entry.BucketID == bucketID &&
			entry.MarketValue.Equal(decimal.NewFromInt(1200))
	})).Return(nil)

	// Execute
	err := service.UpdateMarketValue(ctx, bucketID, decimal.NewFromInt(1200))

	// Assert
	assert.NoError(t, err)

	// Verify all mocks were called
	mockBucketRepo.AssertExpectations(t)
	mockMarketValueRepo.AssertExpectations(t)
}

func TestUpdateMarketValue_ZeroAmount(t *testing.T) {
	ctx := context.Background()
	mockBucketRepo := new(MockBucketRepository)
	mockMarketValueRepo := new(MockMarketValueRepository)

	service := NewInvestmentService(mockBucketRepo, mockMarketValueRepo)

	// Execute with zero amount
	bucketID := uuid.New()
	err := service.UpdateMarketValue(ctx, bucketID, decimal.Zero)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "market value must be positive")

	// Verify no repository calls were made
	mockBucketRepo.AssertNotCalled(t, "GetByID")
	mockMarketValueRepo.AssertNotCalled(t, "Add")
}

func TestUpdateMarketValue_NegativeAmount(t *testing.T) {
	ctx := context.Background()
	mockBucketRepo := new(MockBucketRepository)
	mockMarketValueRepo := new(MockMarketValueRepository)

	service := NewInvestmentService(mockBucketRepo, mockMarketValueRepo)

	// Execute with negative amount
	bucketID := uuid.New()
	err := service.UpdateMarketValue(ctx, bucketID, decimal.NewFromInt(-100))

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "market value must be positive")

	// Verify no repository calls were made
	mockBucketRepo.AssertNotCalled(t, "GetByID")
	mockMarketValueRepo.AssertNotCalled(t, "Add")
}

func TestUpdateMarketValue_BucketNotFound(t *testing.T) {
	ctx := context.Background()
	mockBucketRepo := new(MockBucketRepository)
	mockMarketValueRepo := new(MockMarketValueRepository)

	service := NewInvestmentService(mockBucketRepo, mockMarketValueRepo)

	// Setup: Bucket does not exist
	bucketID := uuid.New()
	mockBucketRepo.On("GetByID", ctx, bucketID).Return(nil, errors.New("bucket not found"))

	// Execute
	err := service.UpdateMarketValue(ctx, bucketID, decimal.NewFromInt(1200))

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "bucket not found")

	// Verify market value repo was not called
	mockMarketValueRepo.AssertNotCalled(t, "Add")
}
