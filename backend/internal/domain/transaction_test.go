package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestTransaction_Validate(t *testing.T) {
	tests := []struct {
		name    string
		tx      Transaction
		wantErr bool
		errMsg  string
	}{
		{
			name: "Balanced transaction with Physical and Virtual layers should pass",
			tx: Transaction{
				ID:          uuid.New(),
				Description: "Test Transaction",
				Date:        time.Now(),
				Entries: []TransactionEntry{
					// Physical Layer: Debit Physical Bucket 100, Credit Expense 100
					{
						ID:            uuid.New(),
						TransactionID: uuid.New(),
						BucketID:      uuid.New(),
						Amount:        decimal.NewFromInt(100),
						Type:          EntryTypeDebit,
						Layer:         LayerPhysical,
					},
					{
						ID:            uuid.New(),
						TransactionID: uuid.New(),
						BucketID:      uuid.New(),
						Amount:        decimal.NewFromInt(100),
						Type:          EntryTypeCredit,
						Layer:         LayerPhysical,
					},
					// Virtual Layer: Debit Expense 100, Credit Virtual Bucket 100
					{
						ID:            uuid.New(),
						TransactionID: uuid.New(),
						BucketID:      uuid.New(),
						Amount:        decimal.NewFromInt(100),
						Type:          EntryTypeDebit,
						Layer:         LayerVirtual,
					},
					{
						ID:            uuid.New(),
						TransactionID: uuid.New(),
						BucketID:      uuid.New(),
						Amount:        decimal.NewFromInt(100),
						Type:          EntryTypeCredit,
						Layer:         LayerVirtual,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Unbalanced Physical layer should fail",
			tx: Transaction{
				ID:          uuid.New(),
				Description: "Unbalanced Physical",
				Date:        time.Now(),
				Entries: []TransactionEntry{
					// Physical Layer: Debit 100, Credit 50 (unbalanced)
					{
						ID:            uuid.New(),
						TransactionID: uuid.New(),
						BucketID:      uuid.New(),
						Amount:        decimal.NewFromInt(100),
						Type:          EntryTypeDebit,
						Layer:         LayerPhysical,
					},
					{
						ID:            uuid.New(),
						TransactionID: uuid.New(),
						BucketID:      uuid.New(),
						Amount:        decimal.NewFromInt(50),
						Type:          EntryTypeCredit,
						Layer:         LayerPhysical,
					},
				},
			},
			wantErr: true,
			errMsg:  "sum of debits must equal sum of credits for PHYSICAL layer",
		},
		{
			name: "Unbalanced Virtual layer should fail",
			tx: Transaction{
				ID:          uuid.New(),
				Description: "Unbalanced Virtual",
				Date:        time.Now(),
				Entries: []TransactionEntry{
					// Virtual Layer: Debit 75, Credit 100 (unbalanced)
					{
						ID:            uuid.New(),
						TransactionID: uuid.New(),
						BucketID:      uuid.New(),
						Amount:        decimal.NewFromInt(75),
						Type:          EntryTypeDebit,
						Layer:         LayerVirtual,
					},
					{
						ID:            uuid.New(),
						TransactionID: uuid.New(),
						BucketID:      uuid.New(),
						Amount:        decimal.NewFromInt(100),
						Type:          EntryTypeCredit,
						Layer:         LayerVirtual,
					},
				},
			},
			wantErr: true,
			errMsg:  "sum of debits must equal sum of credits for VIRTUAL layer",
		},
		{
			name: "Transaction with no entries should fail",
			tx: Transaction{
				ID:          uuid.New(),
				Description: "Empty Transaction",
				Date:        time.Now(),
				Entries:     []TransactionEntry{},
			},
			wantErr: true,
			errMsg:  "transaction must have at least one entry",
		},
		{
			name: "Entry with zero amount should fail",
			tx: Transaction{
				ID:          uuid.New(),
				Description: "Zero Amount Entry",
				Date:        time.Now(),
				Entries: []TransactionEntry{
					{
						ID:            uuid.New(),
						TransactionID: uuid.New(),
						BucketID:      uuid.New(),
						Amount:        decimal.Zero,
						Type:          EntryTypeDebit,
						Layer:         LayerPhysical,
					},
				},
			},
			wantErr: true,
			errMsg:  "entry amount must be positive (absolute value)",
		},
		{
			name: "Entry with negative amount should fail",
			tx: Transaction{
				ID:          uuid.New(),
				Description: "Negative Amount Entry",
				Date:        time.Now(),
				Entries: []TransactionEntry{
					{
						ID:            uuid.New(),
						TransactionID: uuid.New(),
						BucketID:      uuid.New(),
						Amount:        decimal.NewFromInt(-10),
						Type:          EntryTypeDebit,
						Layer:         LayerPhysical,
					},
				},
			},
			wantErr: true,
			errMsg:  "entry amount must be positive (absolute value)",
		},
		{
			name: "Entry with invalid layer should fail",
			tx: Transaction{
				ID:          uuid.New(),
				Description: "Invalid Layer",
				Date:        time.Now(),
				Entries: []TransactionEntry{
					{
						ID:            uuid.New(),
						TransactionID: uuid.New(),
						BucketID:      uuid.New(),
						Amount:        decimal.NewFromInt(100),
						Type:          EntryTypeDebit,
						Layer:         Layer("INVALID"),
					},
				},
			},
			wantErr: true,
			errMsg:  "entry layer must be PHYSICAL or VIRTUAL",
		},
		{
			name: "Entry with invalid type should fail",
			tx: Transaction{
				ID:          uuid.New(),
				Description: "Invalid Type",
				Date:        time.Now(),
				Entries: []TransactionEntry{
					{
						ID:            uuid.New(),
						TransactionID: uuid.New(),
						BucketID:      uuid.New(),
						Amount:        decimal.NewFromInt(100),
						Type:          EntryType("INVALID"),
						Layer:         LayerPhysical,
					},
				},
			},
			wantErr: true,
			errMsg:  "entry type must be DEBIT or CREDIT",
		},
		{
			name: "Balanced transaction with only Physical layer should pass",
			tx: Transaction{
				ID:          uuid.New(),
				Description: "Physical Only",
				Date:        time.Now(),
				Entries: []TransactionEntry{
					{
						ID:            uuid.New(),
						TransactionID: uuid.New(),
						BucketID:      uuid.New(),
						Amount:        decimal.NewFromInt(200),
						Type:          EntryTypeDebit,
						Layer:         LayerPhysical,
					},
					{
						ID:            uuid.New(),
						TransactionID: uuid.New(),
						BucketID:      uuid.New(),
						Amount:        decimal.NewFromInt(200),
						Type:          EntryTypeCredit,
						Layer:         LayerPhysical,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Balanced transaction with only Virtual layer should pass",
			tx: Transaction{
				ID:          uuid.New(),
				Description: "Virtual Only",
				Date:        time.Now(),
				Entries: []TransactionEntry{
					{
						ID:            uuid.New(),
						TransactionID: uuid.New(),
						BucketID:      uuid.New(),
						Amount:        decimal.NewFromInt(150),
						Type:          EntryTypeDebit,
						Layer:         LayerVirtual,
					},
					{
						ID:            uuid.New(),
						TransactionID: uuid.New(),
						BucketID:      uuid.New(),
						Amount:        decimal.NewFromInt(150),
						Type:          EntryTypeCredit,
						Layer:         LayerVirtual,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Balanced transaction with multiple entries per layer should pass",
			tx: Transaction{
				ID:          uuid.New(),
				Description: "Multiple Entries",
				Date:        time.Now(),
				Entries: []TransactionEntry{
					// Physical Layer: Debit 50 + 50 = 100, Credit 100
					{
						ID:            uuid.New(),
						TransactionID: uuid.New(),
						BucketID:      uuid.New(),
						Amount:        decimal.NewFromInt(50),
						Type:          EntryTypeDebit,
						Layer:         LayerPhysical,
					},
					{
						ID:            uuid.New(),
						TransactionID: uuid.New(),
						BucketID:      uuid.New(),
						Amount:        decimal.NewFromInt(50),
						Type:          EntryTypeDebit,
						Layer:         LayerPhysical,
					},
					{
						ID:            uuid.New(),
						TransactionID: uuid.New(),
						BucketID:      uuid.New(),
						Amount:        decimal.NewFromInt(100),
						Type:          EntryTypeCredit,
						Layer:         LayerPhysical,
					},
					// Virtual Layer: Debit 100, Credit 30 + 70 = 100
					{
						ID:            uuid.New(),
						TransactionID: uuid.New(),
						BucketID:      uuid.New(),
						Amount:        decimal.NewFromInt(100),
						Type:          EntryTypeDebit,
						Layer:         LayerVirtual,
					},
					{
						ID:            uuid.New(),
						TransactionID: uuid.New(),
						BucketID:      uuid.New(),
						Amount:        decimal.NewFromInt(30),
						Type:          EntryTypeCredit,
						Layer:         LayerVirtual,
					},
					{
						ID:            uuid.New(),
						TransactionID: uuid.New(),
						BucketID:      uuid.New(),
						Amount:        decimal.NewFromInt(70),
						Type:          EntryTypeCredit,
						Layer:         LayerVirtual,
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.tx.Validate()
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
