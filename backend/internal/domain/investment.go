package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// MarketValueHistory represents a market value history entry in the domain layer
// Adheres to the data model defined in specs.md
// This struct tracks the real-world value of an asset (e.g., Stock ETF) vs its "Book Value" (what you paid)
type MarketValueHistory struct {
	ID          uuid.UUID
	BucketID    uuid.UUID
	Date        time.Time
	MarketValue decimal.Decimal // The actual value at this point in time
}
