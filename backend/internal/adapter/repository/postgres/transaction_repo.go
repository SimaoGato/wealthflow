package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/shopspring/decimal"
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

// List retrieves a paginated list of transactions
func (r *transactionRepository) List(ctx context.Context, limit, offset int, bucketID *uuid.UUID) ([]*domain.Transaction, error) {
	var query string
	var args []interface{}

	// Build query based on whether bucketID filter is provided
	if bucketID != nil {
		query = `
			SELECT DISTINCT t.id, t.description, t.date, t.is_internal_transfer, t.is_external_inflow
			FROM transactions t
			INNER JOIN transaction_entries te ON t.id = te.transaction_id
			WHERE te.bucket_id = $1
			ORDER BY t.date DESC, t.id
			LIMIT $2 OFFSET $3
		`
		args = []interface{}{*bucketID, limit, offset}
	} else {
		query = `
			SELECT id, description, date, is_internal_transfer, is_external_inflow
			FROM transactions
			ORDER BY date DESC, id
			LIMIT $1 OFFSET $2
		`
		args = []interface{}{limit, offset}
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list transactions: %w", err)
	}
	defer rows.Close()

	var transactions []*domain.Transaction
	var transactionIDs []uuid.UUID

	// First, collect all transaction headers
	for rows.Next() {
		var tx domain.Transaction
		err := rows.Scan(
			&tx.ID,
			&tx.Description,
			&tx.Date,
			&tx.IsInternalTransfer,
			&tx.IsExternalInflow,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transaction: %w", err)
		}
		tx.Entries = []domain.TransactionEntry{} // Initialize empty entries
		transactions = append(transactions, &tx)
		transactionIDs = append(transactionIDs, tx.ID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating transactions: %w", err)
	}

	// If no transactions found, return empty slice
	if len(transactionIDs) == 0 {
		return transactions, nil
	}

	// Now load all entries for these transactions
	// Build a map for quick lookup
	txMap := make(map[uuid.UUID]*domain.Transaction)
	for _, tx := range transactions {
		txMap[tx.ID] = tx
	}

	// Query all entries for the transactions we found
	entriesQuery := `
		SELECT id, transaction_id, bucket_id, amount, type, layer
		FROM transaction_entries
		WHERE transaction_id = ANY($1)
		ORDER BY transaction_id, id
	`
	entriesRows, err := r.db.QueryContext(ctx, entriesQuery, pq.Array(transactionIDs))
	if err != nil {
		return nil, fmt.Errorf("failed to query transaction entries: %w", err)
	}
	defer entriesRows.Close()

	// Load entries into their respective transactions
	for entriesRows.Next() {
		var entry domain.TransactionEntry
		var amountStr string

		err := entriesRows.Scan(
			&entry.ID,
			&entry.TransactionID,
			&entry.BucketID,
			&amountStr,
			&entry.Type,
			&entry.Layer,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transaction entry: %w", err)
		}

		// Parse amount (DECIMAL)
		amount, err := decimal.NewFromString(amountStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse entry amount: %w", err)
		}
		entry.Amount = amount

		// Add entry to the corresponding transaction
		if tx, ok := txMap[entry.TransactionID]; ok {
			tx.Entries = append(tx.Entries, entry)
		}
	}

	if err := entriesRows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating transaction entries: %w", err)
	}

	return transactions, nil
}

// Count returns the total number of transactions
func (r *transactionRepository) Count(ctx context.Context, bucketID *uuid.UUID) (int, error) {
	var query string
	var args []interface{}

	// Build query based on whether bucketID filter is provided
	if bucketID != nil {
		query = `
			SELECT COUNT(DISTINCT t.id)
			FROM transactions t
			INNER JOIN transaction_entries te ON t.id = te.transaction_id
			WHERE te.bucket_id = $1
		`
		args = []interface{}{*bucketID}
	} else {
		query = `
			SELECT COUNT(*)
			FROM transactions
		`
		args = []interface{}{}
	}

	var count int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count transactions: %w", err)
	}

	return count, nil
}
