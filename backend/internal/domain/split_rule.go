package domain

import (
	"errors"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// SplitRuleItemType represents the type of split rule item
type SplitRuleItemType string

const (
	SplitRuleItemTypeFixed     SplitRuleItemType = "FIXED"
	SplitRuleItemTypePercent   SplitRuleItemType = "PERCENT"
	SplitRuleItemTypeRemainder SplitRuleItemType = "REMAINDER"
)

// SplitRule represents a split rule entity in the domain layer
// Adheres to the data model defined in specs.md
type SplitRule struct {
	ID             uuid.UUID
	Name           string
	SourceBucketID uuid.UUID
	Items          []SplitRuleItem
}

// SplitRuleItem represents a single item in a split rule
// Adheres to the data model defined in specs.md
type SplitRuleItem struct {
	ID             uuid.UUID
	SplitRuleID    uuid.UUID
	TargetBucketID uuid.UUID
	Type           SplitRuleItemType // 'FIXED', 'PERCENT' (of Remainder), or 'REMAINDER' (Catch-all)
	Value          decimal.Decimal   // Amount for FIXED, percentage (0-100) for PERCENT, ignored for REMAINDER
	Priority       int               // Lower number = Executed first (Important for Fixed logic)
}

// Validate ensures the split rule adheres to domain rules
// Returns an error if validation fails
// CRITICAL: Ensures exactly one item is type 'REMAINDER'
func (sr *SplitRule) Validate() error {
	if len(sr.Items) == 0 {
		return errors.New("split rule must have at least one item")
	}

	remainderCount := 0
	for _, item := range sr.Items {
		if item.Type == SplitRuleItemTypeRemainder {
			remainderCount++
		}

		// Validate item type
		if item.Type != SplitRuleItemTypeFixed &&
			item.Type != SplitRuleItemTypePercent &&
			item.Type != SplitRuleItemTypeRemainder {
			return errors.New("split rule item type must be FIXED, PERCENT, or REMAINDER")
		}

		// Validate FIXED value is positive
		if item.Type == SplitRuleItemTypeFixed {
			if item.Value.LessThanOrEqual(decimal.Zero) {
				return errors.New("FIXED split rule item value must be positive")
			}
		}

		// Validate PERCENT value is between 0 and 100
		if item.Type == SplitRuleItemTypePercent {
			if item.Value.LessThan(decimal.Zero) || item.Value.GreaterThan(decimal.NewFromInt(100)) {
				return errors.New("PERCENT split rule item value must be between 0 and 100")
			}
		}
	}

	if remainderCount != 1 {
		return errors.New("split rule must have exactly one REMAINDER item")
	}

	return nil
}
