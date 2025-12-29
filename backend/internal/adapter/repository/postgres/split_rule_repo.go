package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/simaogato/wealthflow-backend/internal/domain"
)

// splitRuleRepository implements domain.SplitRuleRepository
type splitRuleRepository struct {
	db *DB
}

// NewSplitRuleRepository creates a new split rule repository
func NewSplitRuleRepository(db *DB) domain.SplitRuleRepository {
	return &splitRuleRepository{db: db}
}

// GetBySourceBucketID retrieves a split rule by its source bucket ID
// This method joins split_rules and split_rule_items tables
func (r *splitRuleRepository) GetBySourceBucketID(ctx context.Context, bucketID uuid.UUID) (*domain.SplitRule, error) {
	// First, get the split rule
	ruleQuery := `
		SELECT id, name, source_bucket_id
		FROM split_rules
		WHERE source_bucket_id = $1
	`

	var splitRule domain.SplitRule
	err := r.db.QueryRowContext(ctx, ruleQuery, bucketID).Scan(
		&splitRule.ID,
		&splitRule.Name,
		&splitRule.SourceBucketID,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("split rule not found for source bucket ID %s: %w", bucketID, err)
		}
		return nil, fmt.Errorf("failed to get split rule: %w", err)
	}

	// Then, get all split rule items
	itemsQuery := `
		SELECT id, split_rule_id, target_bucket_id, rule_type, value, priority
		FROM split_rule_items
		WHERE split_rule_id = $1
		ORDER BY priority ASC
	`

	rows, err := r.db.QueryContext(ctx, itemsQuery, splitRule.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to query split rule items: %w", err)
	}
	defer rows.Close()

	var items []domain.SplitRuleItem
	for rows.Next() {
		var item domain.SplitRuleItem
		var valueStr string

		err := rows.Scan(
			&item.ID,
			&item.SplitRuleID,
			&item.TargetBucketID,
			&item.Type,
			&valueStr,
			&item.Priority,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan split rule item: %w", err)
		}

		// Parse value (DECIMAL)
		value, err := decimal.NewFromString(valueStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse split rule item value: %w", err)
		}
		item.Value = value

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating split rule items: %w", err)
	}

	if len(items) == 0 {
		return nil, fmt.Errorf("split rule %s has no items", splitRule.ID)
	}

	// Sort items by priority (lower number = higher priority)
	sort.Slice(items, func(i, j int) bool {
		return items[i].Priority < items[j].Priority
	})

	splitRule.Items = items

	return &splitRule, nil
}
