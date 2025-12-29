package postgres

import (
	"context"
	"fmt"

	"github.com/simaogato/wealthflow-backend/internal/domain"
)

// transactionRepository implements domain.TransactionRepository
type transactionRepository struct {
	db *DB
}

// NewTransactionRepository creates a new transaction repository
func NewTransactionRepository(db *DB) domain.TransactionRepository {
	return &transactionRepository{db: db}
}

// Create creates a new transaction with all its entries in a database transaction
func (r *transactionRepository) Create(ctx context.Context, tx *domain.Transaction) error {
	// Start a database transaction
	dbTx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer dbTx.Rollback()

	// Insert the transaction header
	insertTxQuery := `
		INSERT INTO transactions (id, description, date, is_internal_transfer, is_external_inflow)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err = dbTx.ExecContext(ctx, insertTxQuery,
		tx.ID,
		tx.Description,
		tx.Date,
		tx.IsInternalTransfer,
		tx.IsExternalInflow,
	)
	if err != nil {
		return fmt.Errorf("failed to insert transaction: %w", err)
	}

	// Insert all transaction entries
	insertEntryQuery := `
		INSERT INTO transaction_entries (id, transaction_id, bucket_id, amount, type, layer)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	for _, entry := range tx.Entries {
		_, err = dbTx.ExecContext(ctx, insertEntryQuery,
			entry.ID,
			entry.TransactionID,
			entry.BucketID,
			entry.Amount.String(),
			string(entry.Type),
			string(entry.Layer),
		)
		if err != nil {
			return fmt.Errorf("failed to insert transaction entry: %w", err)
		}
	}

	// Commit the transaction
	if err := dbTx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
