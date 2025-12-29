package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// EntryType represents the type of transaction entry
type EntryType string

const (
	EntryTypeDebit  EntryType = "DEBIT"
	EntryTypeCredit EntryType = "CREDIT"
)

// Layer represents the accounting layer
type Layer string

const (
	LayerPhysical Layer = "PHYSICAL"
	LayerVirtual  Layer = "VIRTUAL"
)

// Transaction represents a transaction entity in the domain layer
// Adheres to the data model defined in specs.md
type Transaction struct {
	ID                 uuid.UUID
	Description        string
	Date               time.Time
	IsInternalTransfer bool
	IsExternalInflow   bool
	Entries            []TransactionEntry
}

// TransactionEntry represents a single entry in a transaction
// Adheres to the data model defined in specs.md
type TransactionEntry struct {
	ID            uuid.UUID
	TransactionID uuid.UUID
	BucketID      uuid.UUID
	Amount        decimal.Decimal // ABSOLUTE VALUE (Always Positive)
	Type          EntryType       // 'DEBIT' or 'CREDIT'
	Layer         Layer           // 'PHYSICAL' or 'VIRTUAL'
}

// Validate ensures the transaction adheres to domain rules
// Returns an error if validation fails
// CRITICAL: Ensures sum of debits equals sum of credits for Physical Layer AND Virtual Layer separately
func (t *Transaction) Validate() error {
	if len(t.Entries) == 0 {
		return errors.New("transaction must have at least one entry")
	}

	// Separate entries by layer
	physicalEntries := make([]TransactionEntry, 0)
	virtualEntries := make([]TransactionEntry, 0)

	for _, entry := range t.Entries {
		if entry.Layer == LayerPhysical {
			physicalEntries = append(physicalEntries, entry)
		} else if entry.Layer == LayerVirtual {
			virtualEntries = append(virtualEntries, entry)
		} else {
			return errors.New("entry layer must be PHYSICAL or VIRTUAL")
		}

		// Validate entry amount is positive (absolute value)
		if entry.Amount.LessThanOrEqual(decimal.Zero) {
			return errors.New("entry amount must be positive (absolute value)")
		}

		// Validate entry type
		if entry.Type != EntryTypeDebit && entry.Type != EntryTypeCredit {
			return errors.New("entry type must be DEBIT or CREDIT")
		}
	}

	// Validate Physical Layer: Sum(Debits) must equal Sum(Credits)
	if err := validateLayerBalance(physicalEntries, LayerPhysical); err != nil {
		return err
	}

	// Validate Virtual Layer: Sum(Debits) must equal Sum(Credits)
	if err := validateLayerBalance(virtualEntries, LayerVirtual); err != nil {
		return err
	}

	return nil
}

// validateLayerBalance ensures that the sum of debits equals the sum of credits for a given layer
func validateLayerBalance(entries []TransactionEntry, layer Layer) error {
	if len(entries) == 0 {
		// Empty layer is valid (transaction might only have entries in one layer)
		return nil
	}

	var totalDebits decimal.Decimal
	var totalCredits decimal.Decimal

	for _, entry := range entries {
		if entry.Type == EntryTypeDebit {
			totalDebits = totalDebits.Add(entry.Amount)
		} else {
			totalCredits = totalCredits.Add(entry.Amount)
		}
	}

	if !totalDebits.Equal(totalCredits) {
		return errors.New("sum of debits must equal sum of credits for " + string(layer) + " layer")
	}

	return nil
}
