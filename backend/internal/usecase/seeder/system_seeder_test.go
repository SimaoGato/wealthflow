package seeder

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

// MockBucketRepository is a mock implementation of BucketRepository
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

func TestSystemSeeder_Seed_BucketsMissing(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockBucketRepository)
	seeder := NewSystemSeeder(mockRepo)

	// Mock GetByID to return "not found" errors for all system buckets
	mockRepo.On("GetByID", ctx, SYS_VIRTUAL_CLEARING).Return(nil, errors.New("not found"))
	mockRepo.On("GetByID", ctx, SYS_LOST_MISC).Return(nil, errors.New("not found"))
	mockRepo.On("GetByID", ctx, SYS_EXTRA_INCOME).Return(nil, errors.New("not found"))

	// Mock Create to succeed for all buckets
	mockRepo.On("Create", ctx, mock.MatchedBy(func(bucket *domain.Bucket) bool {
		return bucket.ID == SYS_VIRTUAL_CLEARING &&
			bucket.Name == "System Virtual Clearing" &&
			bucket.BucketType == domain.BucketTypeSystem &&
			bucket.CurrentBalance.Equal(decimal.Zero)
	})).Return(nil)

	mockRepo.On("Create", ctx, mock.MatchedBy(func(bucket *domain.Bucket) bool {
		return bucket.ID == SYS_LOST_MISC &&
			bucket.Name == "System Lost/Misc" &&
			bucket.BucketType == domain.BucketTypeSystem &&
			bucket.CurrentBalance.Equal(decimal.Zero)
	})).Return(nil)

	mockRepo.On("Create", ctx, mock.MatchedBy(func(bucket *domain.Bucket) bool {
		return bucket.ID == SYS_EXTRA_INCOME &&
			bucket.Name == "System Extra Income" &&
			bucket.BucketType == domain.BucketTypeSystem &&
			bucket.CurrentBalance.Equal(decimal.Zero)
	})).Return(nil)

	// Execute
	err := seeder.Seed(ctx)

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	// Verify Create was called 3 times (once for each system bucket)
	mockRepo.AssertNumberOfCalls(t, "Create", 3)
}

func TestSystemSeeder_Seed_BucketsExist(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockBucketRepository)
	seeder := NewSystemSeeder(mockRepo)

	// Mock GetByID to return existing buckets for all system buckets
	mockRepo.On("GetByID", ctx, SYS_VIRTUAL_CLEARING).Return(&domain.Bucket{
		ID:             SYS_VIRTUAL_CLEARING,
		Name:           "System Virtual Clearing",
		BucketType:     domain.BucketTypeSystem,
		CurrentBalance: decimal.Zero,
	}, nil)

	mockRepo.On("GetByID", ctx, SYS_LOST_MISC).Return(&domain.Bucket{
		ID:             SYS_LOST_MISC,
		Name:           "System Lost/Misc",
		BucketType:     domain.BucketTypeSystem,
		CurrentBalance: decimal.Zero,
	}, nil)

	mockRepo.On("GetByID", ctx, SYS_EXTRA_INCOME).Return(&domain.Bucket{
		ID:             SYS_EXTRA_INCOME,
		Name:           "System Extra Income",
		BucketType:     domain.BucketTypeSystem,
		CurrentBalance: decimal.Zero,
	}, nil)

	// Execute
	err := seeder.Seed(ctx)

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	// Verify Create was NOT called (buckets already exist)
	mockRepo.AssertNotCalled(t, "Create")
}

func TestSystemSeeder_Seed_PartialBucketsExist(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockBucketRepository)
	seeder := NewSystemSeeder(mockRepo)

	// Mock: First bucket exists, second and third are missing
	mockRepo.On("GetByID", ctx, SYS_VIRTUAL_CLEARING).Return(&domain.Bucket{
		ID:             SYS_VIRTUAL_CLEARING,
		Name:           "System Virtual Clearing",
		BucketType:     domain.BucketTypeSystem,
		CurrentBalance: decimal.Zero,
	}, nil)

	mockRepo.On("GetByID", ctx, SYS_LOST_MISC).Return(nil, errors.New("not found"))
	mockRepo.On("GetByID", ctx, SYS_EXTRA_INCOME).Return(nil, errors.New("not found"))

	// Mock Create for missing buckets
	mockRepo.On("Create", ctx, mock.MatchedBy(func(bucket *domain.Bucket) bool {
		return bucket.ID == SYS_LOST_MISC
	})).Return(nil)

	mockRepo.On("Create", ctx, mock.MatchedBy(func(bucket *domain.Bucket) bool {
		return bucket.ID == SYS_EXTRA_INCOME
	})).Return(nil)

	// Execute
	err := seeder.Seed(ctx)

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	// Verify Create was called 2 times (for the 2 missing buckets)
	mockRepo.AssertNumberOfCalls(t, "Create", 2)
}
