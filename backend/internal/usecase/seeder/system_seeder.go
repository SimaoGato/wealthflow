package seeder

import (
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/simaogato/wealthflow-backend/internal/domain"
)

// Fixed UUIDs for system buckets (immutable as per specs.md FR-09)
var (
	SYS_VIRTUAL_CLEARING = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	SYS_LOST_MISC        = uuid.MustParse("00000000-0000-0000-0000-000000000002")
	SYS_EXTRA_INCOME     = uuid.MustParse("00000000-0000-0000-0000-000000000003")
)

// SystemBucket defines the structure for a system bucket to be seeded
type SystemBucket struct {
	ID         uuid.UUID
	Name       string
	BucketType domain.BucketType
}

// SystemSeeder handles seeding of required system buckets
type SystemSeeder struct {
	repo domain.BucketRepository
}

// NewSystemSeeder creates a new SystemSeeder instance
func NewSystemSeeder(repo domain.BucketRepository) *SystemSeeder {
	return &SystemSeeder{
		repo: repo,
	}
}

// Seed ensures all required system buckets exist in the database
// If a bucket doesn't exist, it creates it
func (s *SystemSeeder) Seed(ctx context.Context) error {
	systemBuckets := []SystemBucket{
		{
			ID:         SYS_VIRTUAL_CLEARING,
			Name:       "System Virtual Clearing",
			BucketType: domain.BucketTypeSystem,
		},
		{
			ID:         SYS_LOST_MISC,
			Name:       "System Lost/Misc",
			BucketType: domain.BucketTypeSystem,
		},
		{
			ID:         SYS_EXTRA_INCOME,
			Name:       "System Extra Income",
			BucketType: domain.BucketTypeSystem,
		},
	}

	for _, sysBucket := range systemBuckets {
		// Try to get the bucket by ID
		_, err := s.repo.GetByID(ctx, sysBucket.ID)
		if err != nil {
			// Bucket doesn't exist, create it
			bucket := &domain.Bucket{
				ID:             sysBucket.ID,
				Name:           sysBucket.Name,
				BucketType:     sysBucket.BucketType,
				CurrentBalance: decimal.Zero,
				// System buckets don't need a parent physical bucket ID
			}

			// Validate before creating
			if err := bucket.Validate(); err != nil {
				return err
			}

			if err := s.repo.Create(ctx, bucket); err != nil {
				return err
			}
		}
		// If bucket exists, no action needed
	}

	return nil
}
