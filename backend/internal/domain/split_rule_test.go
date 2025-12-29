package domain

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestSplitRule_Validate(t *testing.T) {
	tests := []struct {
		name    string
		rule    SplitRule
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid split rule with one remainder",
			rule: SplitRule{
				ID:             uuid.New(),
				Name:           "Test Rule",
				SourceBucketID: uuid.New(),
				Items: []SplitRuleItem{
					{
						ID:             uuid.New(),
						TargetBucketID: uuid.New(),
						Type:           SplitRuleItemTypeFixed,
						Value:          decimal.NewFromInt(50),
						Priority:       1,
					},
					{
						ID:             uuid.New(),
						TargetBucketID: uuid.New(),
						Type:           SplitRuleItemTypeRemainder,
						Value:          decimal.Zero,
						Priority:       2,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid split rule with fixed, percent, and remainder",
			rule: SplitRule{
				ID:             uuid.New(),
				Name:           "Test Rule",
				SourceBucketID: uuid.New(),
				Items: []SplitRuleItem{
					{
						ID:             uuid.New(),
						TargetBucketID: uuid.New(),
						Type:           SplitRuleItemTypeFixed,
						Value:          decimal.NewFromInt(50),
						Priority:       1,
					},
					{
						ID:             uuid.New(),
						TargetBucketID: uuid.New(),
						Type:           SplitRuleItemTypePercent,
						Value:          decimal.NewFromInt(10),
						Priority:       2,
					},
					{
						ID:             uuid.New(),
						TargetBucketID: uuid.New(),
						Type:           SplitRuleItemTypeRemainder,
						Value:          decimal.Zero,
						Priority:       3,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "empty items list",
			rule: SplitRule{
				ID:             uuid.New(),
				Name:           "Test Rule",
				SourceBucketID: uuid.New(),
				Items:          []SplitRuleItem{},
			},
			wantErr: true,
			errMsg:  "split rule must have at least one item",
		},
		{
			name: "no remainder item",
			rule: SplitRule{
				ID:             uuid.New(),
				Name:           "Test Rule",
				SourceBucketID: uuid.New(),
				Items: []SplitRuleItem{
					{
						ID:             uuid.New(),
						TargetBucketID: uuid.New(),
						Type:           SplitRuleItemTypeFixed,
						Value:          decimal.NewFromInt(50),
						Priority:       1,
					},
				},
			},
			wantErr: true,
			errMsg:  "split rule must have exactly one REMAINDER item",
		},
		{
			name: "multiple remainder items",
			rule: SplitRule{
				ID:             uuid.New(),
				Name:           "Test Rule",
				SourceBucketID: uuid.New(),
				Items: []SplitRuleItem{
					{
						ID:             uuid.New(),
						TargetBucketID: uuid.New(),
						Type:           SplitRuleItemTypeRemainder,
						Value:          decimal.Zero,
						Priority:       1,
					},
					{
						ID:             uuid.New(),
						TargetBucketID: uuid.New(),
						Type:           SplitRuleItemTypeRemainder,
						Value:          decimal.Zero,
						Priority:       2,
					},
				},
			},
			wantErr: true,
			errMsg:  "split rule must have exactly one REMAINDER item",
		},
		{
			name: "invalid item type",
			rule: SplitRule{
				ID:             uuid.New(),
				Name:           "Test Rule",
				SourceBucketID: uuid.New(),
				Items: []SplitRuleItem{
					{
						ID:             uuid.New(),
						TargetBucketID: uuid.New(),
						Type:           SplitRuleItemType("INVALID"),
						Value:          decimal.Zero,
						Priority:       1,
					},
					{
						ID:             uuid.New(),
						TargetBucketID: uuid.New(),
						Type:           SplitRuleItemTypeRemainder,
						Value:          decimal.Zero,
						Priority:       2,
					},
				},
			},
			wantErr: true,
			errMsg:  "split rule item type must be FIXED, PERCENT, or REMAINDER",
		},
		{
			name: "FIXED item with zero value",
			rule: SplitRule{
				ID:             uuid.New(),
				Name:           "Test Rule",
				SourceBucketID: uuid.New(),
				Items: []SplitRuleItem{
					{
						ID:             uuid.New(),
						TargetBucketID: uuid.New(),
						Type:           SplitRuleItemTypeFixed,
						Value:          decimal.Zero,
						Priority:       1,
					},
					{
						ID:             uuid.New(),
						TargetBucketID: uuid.New(),
						Type:           SplitRuleItemTypeRemainder,
						Value:          decimal.Zero,
						Priority:       2,
					},
				},
			},
			wantErr: true,
			errMsg:  "FIXED split rule item value must be positive",
		},
		{
			name: "FIXED item with negative value",
			rule: SplitRule{
				ID:             uuid.New(),
				Name:           "Test Rule",
				SourceBucketID: uuid.New(),
				Items: []SplitRuleItem{
					{
						ID:             uuid.New(),
						TargetBucketID: uuid.New(),
						Type:           SplitRuleItemTypeFixed,
						Value:          decimal.NewFromInt(-10),
						Priority:       1,
					},
					{
						ID:             uuid.New(),
						TargetBucketID: uuid.New(),
						Type:           SplitRuleItemTypeRemainder,
						Value:          decimal.Zero,
						Priority:       2,
					},
				},
			},
			wantErr: true,
			errMsg:  "FIXED split rule item value must be positive",
		},
		{
			name: "PERCENT item with negative value",
			rule: SplitRule{
				ID:             uuid.New(),
				Name:           "Test Rule",
				SourceBucketID: uuid.New(),
				Items: []SplitRuleItem{
					{
						ID:             uuid.New(),
						TargetBucketID: uuid.New(),
						Type:           SplitRuleItemTypePercent,
						Value:          decimal.NewFromInt(-10),
						Priority:       1,
					},
					{
						ID:             uuid.New(),
						TargetBucketID: uuid.New(),
						Type:           SplitRuleItemTypeRemainder,
						Value:          decimal.Zero,
						Priority:       2,
					},
				},
			},
			wantErr: true,
			errMsg:  "PERCENT split rule item value must be between 0 and 100",
		},
		{
			name: "PERCENT item with value over 100",
			rule: SplitRule{
				ID:             uuid.New(),
				Name:           "Test Rule",
				SourceBucketID: uuid.New(),
				Items: []SplitRuleItem{
					{
						ID:             uuid.New(),
						TargetBucketID: uuid.New(),
						Type:           SplitRuleItemTypePercent,
						Value:          decimal.NewFromInt(150),
						Priority:       1,
					},
					{
						ID:             uuid.New(),
						TargetBucketID: uuid.New(),
						Type:           SplitRuleItemTypeRemainder,
						Value:          decimal.Zero,
						Priority:       2,
					},
				},
			},
			wantErr: true,
			errMsg:  "PERCENT split rule item value must be between 0 and 100",
		},
		{
			name: "PERCENT item with value exactly 100",
			rule: SplitRule{
				ID:             uuid.New(),
				Name:           "Test Rule",
				SourceBucketID: uuid.New(),
				Items: []SplitRuleItem{
					{
						ID:             uuid.New(),
						TargetBucketID: uuid.New(),
						Type:           SplitRuleItemTypePercent,
						Value:          decimal.NewFromInt(100),
						Priority:       1,
					},
					{
						ID:             uuid.New(),
						TargetBucketID: uuid.New(),
						Type:           SplitRuleItemTypeRemainder,
						Value:          decimal.Zero,
						Priority:       2,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "PERCENT item with value exactly 0",
			rule: SplitRule{
				ID:             uuid.New(),
				Name:           "Test Rule",
				SourceBucketID: uuid.New(),
				Items: []SplitRuleItem{
					{
						ID:             uuid.New(),
						TargetBucketID: uuid.New(),
						Type:           SplitRuleItemTypePercent,
						Value:          decimal.Zero,
						Priority:       1,
					},
					{
						ID:             uuid.New(),
						TargetBucketID: uuid.New(),
						Type:           SplitRuleItemTypeRemainder,
						Value:          decimal.Zero,
						Priority:       2,
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.rule.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
