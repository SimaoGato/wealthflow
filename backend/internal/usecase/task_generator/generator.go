package task_generator

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/simaogato/wealthflow-backend/internal/domain"
)

// GenerateTasks analyzes a transaction and generates TransferTasks if money moves
// between different physical buckets in the virtual layer.
//
// Logic:
//   - Iterate through VIRTUAL layer transaction entries
//   - Group credits and debits by their Physical Parent Bucket
//   - If money moves from Physical Bucket A to Physical Bucket B -> Create a Task
//   - If money moves within Bucket A (Virtual A1 -> Virtual A2) -> DO NOT create a Task
//
// Returns an error if bucket lookup fails or if entries reference invalid buckets.
func GenerateTasks(ctx context.Context, tx domain.Transaction, bucketRepo domain.BucketRepository) ([]domain.TransferTask, error) {
	// Only analyze VIRTUAL layer entries (physical transfers are already done)
	virtualEntries := make([]domain.TransactionEntry, 0)
	for _, entry := range tx.Entries {
		if entry.Layer == domain.LayerVirtual {
			virtualEntries = append(virtualEntries, entry)
		}
	}

	// If no virtual entries, no tasks to generate
	if len(virtualEntries) == 0 {
		return []domain.TransferTask{}, nil
	}

	// Map to track net flow per physical bucket
	// Key: Physical Bucket ID, Value: Net flow (positive = money coming in, negative = money going out)
	physicalBucketFlows := make(map[uuid.UUID]decimal.Decimal)

	// Iterate through virtual entries and calculate net flow per physical bucket
	for _, entry := range virtualEntries {
		// Get the bucket to find its physical parent
		bucket, err := bucketRepo.GetByID(ctx, entry.BucketID)
		if err != nil {
			return nil, err
		}

		// Determine the physical parent bucket ID
		var physicalBucketID uuid.UUID
		switch bucket.BucketType {
		case domain.BucketTypePhysical:
			// If it's a physical bucket, use it directly
			physicalBucketID = bucket.ID
		case domain.BucketTypeVirtual:
			// Virtual buckets must have a parent physical bucket
			if bucket.ParentPhysicalBucketID == nil {
				return nil, errors.New("virtual bucket must have a parent physical bucket")
			}
			physicalBucketID = *bucket.ParentPhysicalBucketID
		case domain.BucketTypeIncome, domain.BucketTypeExpense:
			// External buckets (Income/Expense) are layer-agnostic
			// They don't have a physical parent, so skip them for task generation
			continue
		default:
			// System buckets and others don't generate transfer tasks
			continue
		}

		// Calculate net flow: DEBIT increases (money coming in), CREDIT decreases (money going out)
		currentFlow := physicalBucketFlows[physicalBucketID]
		if entry.Type == domain.EntryTypeDebit {
			// Money coming into this physical bucket
			physicalBucketFlows[physicalBucketID] = currentFlow.Add(entry.Amount)
		} else if entry.Type == domain.EntryTypeCredit {
			// Money going out of this physical bucket
			physicalBucketFlows[physicalBucketID] = currentFlow.Sub(entry.Amount)
		}
	}

	// Generate tasks for net flows between different physical buckets
	tasks := make([]domain.TransferTask, 0)

	// Find buckets with positive flow (receiving money) and negative flow (sending money)
	receivingBuckets := make([]uuid.UUID, 0)
	sendingBuckets := make([]uuid.UUID, 0)

	for bucketID, flow := range physicalBucketFlows {
		if flow.GreaterThan(decimal.Zero) {
			receivingBuckets = append(receivingBuckets, bucketID)
		} else if flow.LessThan(decimal.Zero) {
			sendingBuckets = append(sendingBuckets, bucketID)
		}
		// Zero flow means no net movement (intra-bucket transfer), skip
	}

	// Match sending buckets with receiving buckets
	// For simplicity, we'll create a task for each sending bucket to each receiving bucket
	// In practice, there should typically be one sender and one receiver
	for _, fromBucketID := range sendingBuckets {
		fromAmount := physicalBucketFlows[fromBucketID].Abs() // Make positive

		for _, toBucketID := range receivingBuckets {
			toAmount := physicalBucketFlows[toBucketID]

			// Create a task for the amount being transferred
			// Use the minimum of what's being sent and what's being received
			transferAmount := decimal.Min(fromAmount, toAmount)

			if transferAmount.GreaterThan(decimal.Zero) {
				task := domain.TransferTask{
					ID:                     uuid.New(),
					RelatedTransactionID:   tx.ID,
					CompletedTransactionID: nil,
					FromPhysicalBucketID:   fromBucketID,
					ToPhysicalBucketID:     toBucketID,
					Amount:                 transferAmount,
					IsCompleted:            false,
				}
				tasks = append(tasks, task)

				// Reduce the amounts to avoid double-counting
				fromAmount = fromAmount.Sub(transferAmount)
				toAmount = toAmount.Sub(transferAmount)
				physicalBucketFlows[toBucketID] = toAmount

				// If we've exhausted the sending amount, break
				if fromAmount.LessThanOrEqual(decimal.Zero) {
					break
				}
			}
		}
	}

	return tasks, nil
}
