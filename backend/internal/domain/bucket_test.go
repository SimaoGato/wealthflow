package domain

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestBucket_Validate(t *testing.T) {
	tests := []struct {
		name    string
		bucket  Bucket
		wantErr bool
		errMsg  string
	}{
		{
			name: "Virtual Bucket without Parent ID should fail",
			bucket: Bucket{
				ID:         uuid.New(),
				Name:       "Test Virtual Bucket",
				BucketType: BucketTypeVirtual,
				// ParentPhysicalBucketID is nil
				CurrentBalance: decimal.Zero,
			},
			wantErr: true,
			errMsg:  "virtual bucket must have a parent physical bucket ID",
		},
		{
			name: "Virtual Bucket with Parent ID should pass",
			bucket: Bucket{
				ID:         uuid.New(),
				Name:       "Test Virtual Bucket",
				BucketType: BucketTypeVirtual,
				ParentPhysicalBucketID: func() *uuid.UUID {
					id := uuid.New()
					return &id
				}(),
				CurrentBalance: decimal.Zero,
			},
			wantErr: false,
		},
		{
			name: "Physical Bucket without Parent ID should pass",
			bucket: Bucket{
				ID:         uuid.New(),
				Name:       "Test Physical Bucket",
				BucketType: BucketTypePhysical,
				// ParentPhysicalBucketID is nil (allowed for Physical)
				CurrentBalance: decimal.Zero,
			},
			wantErr: false,
		},
		{
			name: "Income Bucket without Parent ID should pass",
			bucket: Bucket{
				ID:         uuid.New(),
				Name:       "Test Income Bucket",
				BucketType: BucketTypeIncome,
				// ParentPhysicalBucketID is nil (allowed for Income)
				CurrentBalance: decimal.Zero,
			},
			wantErr: false,
		},
		{
			name: "Expense Bucket without Parent ID should pass",
			bucket: Bucket{
				ID:         uuid.New(),
				Name:       "Test Expense Bucket",
				BucketType: BucketTypeExpense,
				// ParentPhysicalBucketID is nil (allowed for Expense)
				CurrentBalance: decimal.Zero,
			},
			wantErr: false,
		},
		{
			name: "Equity Bucket without Parent ID should pass",
			bucket: Bucket{
				ID:         uuid.New(),
				Name:       "Test Equity Bucket",
				BucketType: BucketTypeEquity,
				// ParentPhysicalBucketID is nil (allowed for Equity)
				CurrentBalance: decimal.Zero,
			},
			wantErr: false,
		},
		{
			name: "System Bucket without Parent ID should pass",
			bucket: Bucket{
				ID:         uuid.New(),
				Name:       "Test System Bucket",
				BucketType: BucketTypeSystem,
				// ParentPhysicalBucketID is nil (allowed for System)
				CurrentBalance: decimal.Zero,
			},
			wantErr: false,
		},
		{
			name: "Bucket with empty name should fail",
			bucket: Bucket{
				ID:             uuid.New(),
				Name:           "",
				BucketType:     BucketTypePhysical,
				CurrentBalance: decimal.Zero,
			},
			wantErr: true,
			errMsg:  "bucket name cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.bucket.Validate()
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
