package allocator

import (
	"errors"
	"sort"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/simaogato/wealthflow-backend/internal/domain"
)

// CalculateAllocation calculates the allocation of a total amount across split rule items
// Returns a map of bucket ID to allocated amount
// Logic:
//  1. Sort items by Priority (Lower = First)
//  2. Deduct FIXED amounts first
//  3. Calculate PERCENT amounts based on the *Remainder* (Total - Fixed), NOT the original total
//  4. Assign the final leftover amount to the REMAINDER item
//
// Safety: Ensures total allocation equals total inflow exactly (no penny lost)
func CalculateAllocation(totalAmount decimal.Decimal, items []domain.SplitRuleItem) (map[uuid.UUID]decimal.Decimal, error) {
	if totalAmount.LessThanOrEqual(decimal.Zero) {
		return nil, errors.New("total amount must be positive")
	}

	if len(items) == 0 {
		return nil, errors.New("items list cannot be empty")
	}

	// Create a copy of items to avoid mutating the original slice
	sortedItems := make([]domain.SplitRuleItem, len(items))
	copy(sortedItems, items)

	// Sort items by Priority (Lower = First)
	sort.Slice(sortedItems, func(i, j int) bool {
		return sortedItems[i].Priority < sortedItems[j].Priority
	})

	// Initialize allocation map
	allocation := make(map[uuid.UUID]decimal.Decimal)
	remaining := totalAmount

	// Step 1: Deduct FIXED amounts first
	for _, item := range sortedItems {
		if item.Type == domain.SplitRuleItemTypeFixed {
			if item.Value.GreaterThan(remaining) {
				return nil, errors.New("FIXED amount exceeds remaining balance")
			}
			allocation[item.TargetBucketID] = item.Value
			remaining = remaining.Sub(item.Value)
		}
	}

	// Step 2: Calculate PERCENT amounts based on the Remainder
	percentTotal := decimal.Zero
	for _, item := range sortedItems {
		if item.Type == domain.SplitRuleItemTypePercent {
			// Calculate percentage of the remainder (not the original total)
			percentAmount := remaining.Mul(item.Value).Div(decimal.NewFromInt(100))
			allocation[item.TargetBucketID] = percentAmount
			percentTotal = percentTotal.Add(percentAmount)
		}
	}

	// Step 3: Assign the final leftover amount to the REMAINDER item
	remainderItem := findRemainderItem(sortedItems)
	if remainderItem == nil {
		return nil, errors.New("no REMAINDER item found")
	}

	// Calculate what's left after FIXED and PERCENT allocations
	allocatedSoFar := decimal.Zero
	for _, amount := range allocation {
		allocatedSoFar = allocatedSoFar.Add(amount)
	}
	remainderAmount := totalAmount.Sub(allocatedSoFar)
	allocation[remainderItem.TargetBucketID] = remainderAmount

	// Safety check: Ensure total allocation equals total inflow exactly
	totalAllocated := decimal.Zero
	for _, amount := range allocation {
		totalAllocated = totalAllocated.Add(amount)
	}

	if !totalAllocated.Equal(totalAmount) {
		return nil, errors.New("total allocation does not equal total amount")
	}

	return allocation, nil
}

// findRemainderItem finds the REMAINDER item in the items slice
func findRemainderItem(items []domain.SplitRuleItem) *domain.SplitRuleItem {
	for i := range items {
		if items[i].Type == domain.SplitRuleItemTypeRemainder {
			return &items[i]
		}
	}
	return nil
}
