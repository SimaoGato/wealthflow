package allocator

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/simaogato/wealthflow-backend/internal/domain"
)

func TestCalculateAllocation_ChurchFootballScenario(t *testing.T) {
	// Test "The Church/Football Scenario" from product_definition.md
	// Input: 1000€
	// Rule: 50€ Fixed (Coffee)
	// Rule: 10% of Remainder (Missions)
	// Rule: Remainder (Catch-All)
	// Expected: Coffee=50, Missions=95, Catch-All=855

	coffeeBucketID := uuid.New()
	missionsBucketID := uuid.New()
	catchAllBucketID := uuid.New()

	items := []domain.SplitRuleItem{
		{
			ID:             uuid.New(),
			TargetBucketID: coffeeBucketID,
			Type:           domain.SplitRuleItemTypeFixed,
			Value:          decimal.NewFromInt(50),
			Priority:       1, // Lower = First
		},
		{
			ID:             uuid.New(),
			TargetBucketID: missionsBucketID,
			Type:           domain.SplitRuleItemTypePercent,
			Value:          decimal.NewFromInt(10), // 10% of remainder
			Priority:       2,
		},
		{
			ID:             uuid.New(),
			TargetBucketID: catchAllBucketID,
			Type:           domain.SplitRuleItemTypeRemainder,
			Value:          decimal.Zero, // Ignored for REMAINDER
			Priority:       3,
		},
	}

	totalAmount := decimal.NewFromInt(1000)
	allocation, err := CalculateAllocation(totalAmount, items)

	require.NoError(t, err)
	require.NotNil(t, allocation)

	// Expected: Coffee=50, Missions=95, Catch-All=855
	assert.True(t, allocation[coffeeBucketID].Equal(decimal.NewFromInt(50)), "Coffee should be 50€")
	assert.True(t, allocation[missionsBucketID].Equal(decimal.NewFromInt(95)), "Missions should be 95€ (10% of 950)")
	assert.True(t, allocation[catchAllBucketID].Equal(decimal.NewFromInt(855)), "Catch-All should be 855€")

	// Verify total equals input
	totalAllocated := decimal.Zero
	for _, amount := range allocation {
		totalAllocated = totalAllocated.Add(amount)
	}
	assert.True(t, totalAllocated.Equal(totalAmount), "Total allocated should equal total amount")
}

func TestCalculateAllocation_OnlyFixed(t *testing.T) {
	bucket1ID := uuid.New()
	bucket2ID := uuid.New()
	catchAllID := uuid.New()

	items := []domain.SplitRuleItem{
		{
			ID:             uuid.New(),
			TargetBucketID: bucket1ID,
			Type:           domain.SplitRuleItemTypeFixed,
			Value:          decimal.NewFromInt(100),
			Priority:       1,
		},
		{
			ID:             uuid.New(),
			TargetBucketID: bucket2ID,
			Type:           domain.SplitRuleItemTypeFixed,
			Value:          decimal.NewFromInt(200),
			Priority:       2,
		},
		{
			ID:             uuid.New(),
			TargetBucketID: catchAllID,
			Type:           domain.SplitRuleItemTypeRemainder,
			Value:          decimal.Zero,
			Priority:       3,
		},
	}

	totalAmount := decimal.NewFromInt(500)
	allocation, err := CalculateAllocation(totalAmount, items)

	require.NoError(t, err)
	assert.True(t, allocation[bucket1ID].Equal(decimal.NewFromInt(100)))
	assert.True(t, allocation[bucket2ID].Equal(decimal.NewFromInt(200)))
	assert.True(t, allocation[catchAllID].Equal(decimal.NewFromInt(200))) // 500 - 100 - 200 = 200
}

func TestCalculateAllocation_OnlyPercent(t *testing.T) {
	bucket1ID := uuid.New()
	bucket2ID := uuid.New()
	catchAllID := uuid.New()

	items := []domain.SplitRuleItem{
		{
			ID:             uuid.New(),
			TargetBucketID: bucket1ID,
			Type:           domain.SplitRuleItemTypePercent,
			Value:          decimal.NewFromInt(30), // 30% of remainder (which is 100% of total)
			Priority:       1,
		},
		{
			ID:             uuid.New(),
			TargetBucketID: bucket2ID,
			Type:           domain.SplitRuleItemTypePercent,
			Value:          decimal.NewFromInt(40), // 40% of remainder
			Priority:       2,
		},
		{
			ID:             uuid.New(),
			TargetBucketID: catchAllID,
			Type:           domain.SplitRuleItemTypeRemainder,
			Value:          decimal.Zero,
			Priority:       3,
		},
	}

	totalAmount := decimal.NewFromInt(1000)
	allocation, err := CalculateAllocation(totalAmount, items)

	require.NoError(t, err)
	// 30% of 1000 = 300
	assert.True(t, allocation[bucket1ID].Equal(decimal.NewFromInt(300)))
	// 40% of 1000 = 400
	assert.True(t, allocation[bucket2ID].Equal(decimal.NewFromInt(400)))
	// Remainder = 1000 - 300 - 400 = 300
	assert.True(t, allocation[catchAllID].Equal(decimal.NewFromInt(300)))
}

func TestCalculateAllocation_MixedWithPriority(t *testing.T) {
	bucket1ID := uuid.New()
	bucket2ID := uuid.New()
	bucket3ID := uuid.New()
	catchAllID := uuid.New()

	items := []domain.SplitRuleItem{
		{
			ID:             uuid.New(),
			TargetBucketID: bucket2ID,
			Type:           domain.SplitRuleItemTypeFixed,
			Value:          decimal.NewFromInt(100),
			Priority:       2, // Should execute second
		},
		{
			ID:             uuid.New(),
			TargetBucketID: bucket1ID,
			Type:           domain.SplitRuleItemTypeFixed,
			Value:          decimal.NewFromInt(50),
			Priority:       1, // Should execute first
		},
		{
			ID:             uuid.New(),
			TargetBucketID: bucket3ID,
			Type:           domain.SplitRuleItemTypePercent,
			Value:          decimal.NewFromInt(20), // 20% of remainder
			Priority:       3,
		},
		{
			ID:             uuid.New(),
			TargetBucketID: catchAllID,
			Type:           domain.SplitRuleItemTypeRemainder,
			Value:          decimal.Zero,
			Priority:       4,
		},
	}

	totalAmount := decimal.NewFromInt(1000)
	allocation, err := CalculateAllocation(totalAmount, items)

	require.NoError(t, err)
	// Fixed: 50 (priority 1) + 100 (priority 2) = 150
	// Remainder after fixed: 1000 - 150 = 850
	// Percent: 20% of 850 = 170
	// Catch-all: 1000 - 50 - 100 - 170 = 680
	assert.True(t, allocation[bucket1ID].Equal(decimal.NewFromInt(50)))
	assert.True(t, allocation[bucket2ID].Equal(decimal.NewFromInt(100)))
	assert.True(t, allocation[bucket3ID].Equal(decimal.NewFromInt(170)))
	assert.True(t, allocation[catchAllID].Equal(decimal.NewFromInt(680)))
}

func TestCalculateAllocation_FixedExceedsTotal(t *testing.T) {
	bucketID := uuid.New()
	catchAllID := uuid.New()

	items := []domain.SplitRuleItem{
		{
			ID:             uuid.New(),
			TargetBucketID: bucketID,
			Type:           domain.SplitRuleItemTypeFixed,
			Value:          decimal.NewFromInt(1000),
			Priority:       1,
		},
		{
			ID:             uuid.New(),
			TargetBucketID: catchAllID,
			Type:           domain.SplitRuleItemTypeRemainder,
			Value:          decimal.Zero,
			Priority:       2,
		},
	}

	totalAmount := decimal.NewFromInt(500)
	_, err := CalculateAllocation(totalAmount, items)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "FIXED amount exceeds remaining balance")
}

func TestCalculateAllocation_NoRemainderItem(t *testing.T) {
	bucketID := uuid.New()

	items := []domain.SplitRuleItem{
		{
			ID:             uuid.New(),
			TargetBucketID: bucketID,
			Type:           domain.SplitRuleItemTypeFixed,
			Value:          decimal.NewFromInt(100),
			Priority:       1,
		},
		// Missing REMAINDER item
	}

	totalAmount := decimal.NewFromInt(500)
	_, err := CalculateAllocation(totalAmount, items)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no REMAINDER item found")
}

func TestCalculateAllocation_ZeroAmount(t *testing.T) {
	catchAllID := uuid.New()

	items := []domain.SplitRuleItem{
		{
			ID:             uuid.New(),
			TargetBucketID: catchAllID,
			Type:           domain.SplitRuleItemTypeRemainder,
			Value:          decimal.Zero,
			Priority:       1,
		},
	}

	totalAmount := decimal.Zero
	_, err := CalculateAllocation(totalAmount, items)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "total amount must be positive")
}

func TestCalculateAllocation_EmptyItems(t *testing.T) {
	totalAmount := decimal.NewFromInt(1000)
	_, err := CalculateAllocation(totalAmount, []domain.SplitRuleItem{})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "items list cannot be empty")
}

func TestCalculateAllocation_DecimalPrecision(t *testing.T) {
	// Test with decimal amounts to ensure precision is maintained
	bucket1ID := uuid.New()
	bucket2ID := uuid.New()
	catchAllID := uuid.New()

	items := []domain.SplitRuleItem{
		{
			ID:             uuid.New(),
			TargetBucketID: bucket1ID,
			Type:           domain.SplitRuleItemTypeFixed,
			Value:          decimal.RequireFromString("33.33"),
			Priority:       1,
		},
		{
			ID:             uuid.New(),
			TargetBucketID: bucket2ID,
			Type:           domain.SplitRuleItemTypePercent,
			Value:          decimal.RequireFromString("33.33"), // 33.33% of remainder
			Priority:       2,
		},
		{
			ID:             uuid.New(),
			TargetBucketID: catchAllID,
			Type:           domain.SplitRuleItemTypeRemainder,
			Value:          decimal.Zero,
			Priority:       3,
		},
	}

	totalAmount := decimal.RequireFromString("100.00")
	allocation, err := CalculateAllocation(totalAmount, items)

	require.NoError(t, err)
	// Fixed: 33.33
	// Remainder after fixed: 100 - 33.33 = 66.67
	// Percent: 33.33% of 66.67 = 22.222611
	// Catch-all: 100 - 33.33 - 22.222611 = 44.447389
	// Total should still equal 100.00 exactly
	totalAllocated := decimal.Zero
	for _, amount := range allocation {
		totalAllocated = totalAllocated.Add(amount)
	}
	assert.True(t, totalAllocated.Equal(totalAmount), "Total allocated should equal total amount even with decimal precision")
}
