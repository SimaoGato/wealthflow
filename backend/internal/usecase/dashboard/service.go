package dashboard

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/simaogato/wealthflow-backend/internal/domain"
)

// NetWorthResult represents the calculated net worth
type NetWorthResult struct {
	Total     decimal.Decimal
	Liquidity decimal.Decimal
	Equity    decimal.Decimal
}

// DashboardService handles dashboard-related operations
type DashboardService struct {
	BucketRepo      domain.BucketRepository
	TransactionRepo domain.TransactionRepository
	MarketValueRepo domain.MarketValueRepository
}

// NewDashboardService creates a new DashboardService instance
func NewDashboardService(
	bucketRepo domain.BucketRepository,
	transactionRepo domain.TransactionRepository,
	marketValueRepo domain.MarketValueRepository,
) *DashboardService {
	return &DashboardService{
		BucketRepo:      bucketRepo,
		TransactionRepo: transactionRepo,
		MarketValueRepo: marketValueRepo,
	}
}

// GetNetWorth calculates the total net worth
// Logic:
//   - Liquidity: Sum of all PHYSICAL bucket balances
//   - Equity: Sum of all EQUITY bucket market values (using latest market_value from market_value_history)
//   - Total: Liquidity + Equity
func (s *DashboardService) GetNetWorth(ctx context.Context) (*NetWorthResult, error) {
	// 1. Get all PHYSICAL buckets and sum their balances
	physicalBuckets, err := s.BucketRepo.List(ctx, domain.BucketTypePhysical)
	if err != nil {
		return nil, fmt.Errorf("failed to list physical buckets: %w", err)
	}

	liquidity := decimal.Zero
	for _, bucket := range physicalBuckets {
		liquidity = liquidity.Add(bucket.CurrentBalance)
	}

	// 2. Get all EQUITY buckets and sum their latest market values
	equityBuckets, err := s.BucketRepo.List(ctx, domain.BucketTypeEquity)
	if err != nil {
		return nil, fmt.Errorf("failed to list equity buckets: %w", err)
	}

	equity := decimal.Zero
	for _, bucket := range equityBuckets {
		// Get latest market value for this bucket
		marketValueEntry, err := s.MarketValueRepo.GetLatest(ctx, bucket.ID)
		if err != nil {
			// If no market value history exists, skip this bucket (or use book value?)
			// Per requirements: use latest market_value, so if none exists, we skip it
			continue
		}
		equity = equity.Add(marketValueEntry.MarketValue)
	}

	// 3. Calculate total
	total := liquidity.Add(equity)

	return &NetWorthResult{
		Total:     total,
		Liquidity: liquidity,
		Equity:    equity,
	}, nil
}
