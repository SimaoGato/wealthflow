package investment

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/simaogato/wealthflow-backend/internal/domain"
)

// InvestmentService handles investment-related operations
type InvestmentService struct {
	BucketRepo      domain.BucketRepository
	MarketValueRepo domain.MarketValueRepository
}

// NewInvestmentService creates a new InvestmentService instance
func NewInvestmentService(bucketRepo domain.BucketRepository, marketValueRepo domain.MarketValueRepository) *InvestmentService {
	return &InvestmentService{
		BucketRepo:      bucketRepo,
		MarketValueRepo: marketValueRepo,
	}
}

// UpdateMarketValue records a new market value point for a bucket
// Logic: Insert a new row into market_value_history (does NOT create a transaction entry)
// Returns the created market value history entry
func (s *InvestmentService) UpdateMarketValue(ctx context.Context, bucketID uuid.UUID, amount decimal.Decimal) (*domain.MarketValueHistory, error) {
	// Validate amount is positive
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, errors.New("market value must be positive")
	}

	// Verify bucket exists (we don't need to use the bucket, just verify it exists)
	_, err := s.BucketRepo.GetByID(ctx, bucketID)
	if err != nil {
		return nil, err
	}

	// Create market value history entry
	entry := &domain.MarketValueHistory{
		ID:          uuid.New(),
		BucketID:    bucketID,
		Date:        time.Now(),
		MarketValue: amount,
	}

	// Save to repository
	if err := s.MarketValueRepo.Add(ctx, entry); err != nil {
		return nil, err
	}

	return entry, nil
}

// CalculateProfit calculates the profit/loss for a bucket
// Logic: Profit = MarketValue - BookValue
// BookValue = bucket.current_balance
// MarketValue = latest entry in market_value_history
func (s *InvestmentService) CalculateProfit(ctx context.Context, bucketID uuid.UUID) (decimal.Decimal, error) {
	// Fetch bucket to get Book Value (current_balance)
	bucket, err := s.BucketRepo.GetByID(ctx, bucketID)
	if err != nil {
		return decimal.Zero, err
	}

	bookValue := bucket.CurrentBalance

	// Fetch latest market value
	marketValueEntry, err := s.MarketValueRepo.GetLatest(ctx, bucketID)
	if err != nil {
		// If no history exists, return 0 (safe default)
		// This handles the "No History" scenario gracefully
		return decimal.Zero, nil
	}

	// Calculate profit: MarketValue - BookValue
	profit := marketValueEntry.MarketValue.Sub(bookValue)

	return profit, nil
}
