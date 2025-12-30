package grpc

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	wealthflowv1 "github.com/simaogato/wealthflow-backend/internal/adapter/grpc/wealthflow/v1"
	"github.com/simaogato/wealthflow-backend/internal/domain"
	"github.com/simaogato/wealthflow-backend/internal/usecase/dashboard"
	"github.com/simaogato/wealthflow-backend/internal/usecase/expense"
	"github.com/simaogato/wealthflow-backend/internal/usecase/inflow"
	"github.com/simaogato/wealthflow-backend/internal/usecase/investment"
)

// Server implements the WealthFlowService gRPC server
type Server struct {
	wealthflowv1.UnimplementedWealthFlowServiceServer

	ExpenseService    *expense.ExpenseService
	InflowService     *inflow.InflowService
	InvestmentService *investment.InvestmentService
	DashboardService  *dashboard.DashboardService
}

// NewServer creates a new gRPC server instance
func NewServer(
	expenseService *expense.ExpenseService,
	inflowService *inflow.InflowService,
	investmentService *investment.InvestmentService,
	dashboardService *dashboard.DashboardService,
) *Server {
	return &Server{
		ExpenseService:    expenseService,
		InflowService:     inflowService,
		InvestmentService: investmentService,
		DashboardService:  dashboardService,
	}
}

// RecordInflow handles the RecordInflow RPC
func (s *Server) RecordInflow(ctx context.Context, req *wealthflowv1.RecordInflowRequest) (*wealthflowv1.RecordInflowResponse, error) {
	// Parse amount from string to decimal
	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid amount format: %v", err)
	}

	// Parse source bucket ID
	sourceBucketID, err := uuid.Parse(req.SourceBucketId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid source_bucket_id format: %v", err)
	}

	// Build input for usecase
	// Note: Date is handled internally by the service (uses time.Now())
	// If we need to support custom dates in the future, we'll need to modify the service
	input := inflow.RecordInflowInput{
		Amount:         amount,
		Description:    req.Description,
		SourceBucketID: sourceBucketID,
		IsExternal:     req.IsExternal,
	}

	// Call usecase service
	tx, err := s.InflowService.RecordInflow(ctx, input)
	if err != nil {
		return nil, mapError(err)
	}

	// Build response
	return &wealthflowv1.RecordInflowResponse{
		TransactionId: tx.ID.String(),
		CreatedAt:     timestamppb.New(tx.Date),
	}, nil
}

// LogExpense handles the LogExpense RPC
func (s *Server) LogExpense(ctx context.Context, req *wealthflowv1.LogExpenseRequest) (*wealthflowv1.LogExpenseResponse, error) {
	// Parse amount from string to decimal
	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid amount format: %v", err)
	}

	// Parse virtual bucket ID
	virtualBucketID, err := uuid.Parse(req.VirtualBucketId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid virtual_bucket_id format: %v", err)
	}

	// Parse category bucket ID
	categoryBucketID, err := uuid.Parse(req.CategoryBucketId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid category_bucket_id format: %v", err)
	}

	// Parse optional physical override ID
	var physicalOverrideID *uuid.UUID
	if req.PhysicalBucketOverrideId != "" {
		overrideID, err := uuid.Parse(req.PhysicalBucketOverrideId)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid physical_bucket_override_id format: %v", err)
		}
		physicalOverrideID = &overrideID
	}

	// Build input for usecase
	input := expense.LogExpenseInput{
		Amount:             amount,
		Description:        req.Description,
		VirtualBucketID:    virtualBucketID,
		CategoryBucketID:   categoryBucketID,
		PhysicalOverrideID: physicalOverrideID,
	}

	// Call usecase service
	tx, err := s.ExpenseService.LogExpense(ctx, input)
	if err != nil {
		return nil, mapError(err)
	}

	// Determine which physical bucket was actually credited
	// We need to find it from the transaction entries
	var physicalBucketID string
	for _, entry := range tx.Entries {
		if entry.Layer == domain.LayerPhysical && entry.Type == domain.EntryTypeCredit {
			physicalBucketID = entry.BucketID.String()
			break
		}
	}

	// Build response
	return &wealthflowv1.LogExpenseResponse{
		TransactionId:    tx.ID.String(),
		CreatedAt:        timestamppb.New(tx.Date),
		PhysicalBucketId: physicalBucketID,
	}, nil
}

// UpdateInvestment handles the UpdateInvestment RPC
func (s *Server) UpdateInvestment(ctx context.Context, req *wealthflowv1.UpdateInvestmentRequest) (*wealthflowv1.UpdateInvestmentResponse, error) {
	// Parse bucket ID
	bucketID, err := uuid.Parse(req.BucketId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid bucket_id format: %v", err)
	}

	// Parse market value from string to decimal
	marketValue, err := decimal.NewFromString(req.MarketValue)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid market_value format: %v", err)
	}

	// Call usecase service
	entry, err := s.InvestmentService.UpdateMarketValue(ctx, bucketID, marketValue)
	if err != nil {
		return nil, mapError(err)
	}

	// Build response with the created entry
	return &wealthflowv1.UpdateInvestmentResponse{
		EntryId:   entry.ID.String(),
		CreatedAt: timestamppb.New(entry.Date),
	}, nil
}

// ListBuckets handles the ListBuckets RPC
func (s *Server) ListBuckets(ctx context.Context, req *wealthflowv1.ListBucketsRequest) (*wealthflowv1.ListBucketsResponse, error) {
	// Parse bucket type filter (optional)
	// If bucket_type is UNSPECIFIED (0), treat it as no filter
	var typeFilter domain.BucketType
	if req.BucketType != wealthflowv1.BucketType_BUCKET_TYPE_UNSPECIFIED {
		typeFilter = protoBucketTypeToDomain(req.BucketType)
	}

	// Get buckets from repository
	buckets, err := s.DashboardService.BucketRepo.List(ctx, typeFilter)
	if err != nil {
		return nil, mapError(err)
	}

	// Convert domain buckets to proto buckets
	protoBuckets := make([]*wealthflowv1.Bucket, 0, len(buckets))
	for _, bucket := range buckets {
		protoBuckets = append(protoBuckets, domainBucketToProto(bucket))
	}

	return &wealthflowv1.ListBucketsResponse{
		Buckets: protoBuckets,
	}, nil
}

// ListTransactions handles the ListTransactions RPC
func (s *Server) ListTransactions(ctx context.Context, req *wealthflowv1.ListTransactionsRequest) (*wealthflowv1.ListTransactionsResponse, error) {
	// Validate limit (must be positive)
	if req.Limit <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "limit must be positive")
	}

	// Validate offset (must be non-negative)
	if req.Offset < 0 {
		return nil, status.Errorf(codes.InvalidArgument, "offset must be non-negative")
	}

	// Parse optional bucket ID filter
	var bucketID *uuid.UUID
	if req.BucketId != "" {
		parsedID, err := uuid.Parse(req.BucketId)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid bucket_id format: %v", err)
		}
		bucketID = &parsedID
	}

	// Get total count for accurate pagination
	totalCount, err := s.DashboardService.TransactionRepo.Count(ctx, bucketID)
	if err != nil {
		return nil, mapError(err)
	}

	// Get transactions from repository
	transactions, err := s.DashboardService.TransactionRepo.List(ctx, int(req.Limit), int(req.Offset), bucketID)
	if err != nil {
		return nil, mapError(err)
	}

	// Collect all unique bucket IDs from the transactions
	bucketIDSet := make(map[uuid.UUID]bool)
	for _, tx := range transactions {
		for _, entry := range tx.Entries {
			bucketIDSet[entry.BucketID] = true
		}
	}

	// Fetch bucket names for all unique bucket IDs
	// Always initialize the map (even if empty) to ensure it's never nil
	bucketNames := make(map[string]string)
	if len(bucketIDSet) > 0 {
		bucketIDList := make([]uuid.UUID, 0, len(bucketIDSet))
		for id := range bucketIDSet {
			bucketIDList = append(bucketIDList, id)
		}

		// Fetch each bucket to get its name
		for _, id := range bucketIDList {
			bucket, err := s.DashboardService.BucketRepo.GetByID(ctx, id)
			if err != nil {
				// If bucket not found, skip it (shouldn't happen, but handle gracefully)
				continue
			}
			bucketNames[id.String()] = bucket.Name
		}
	}

	// Convert domain transactions to proto transactions
	protoTransactions := make([]*wealthflowv1.Transaction, 0, len(transactions))
	for _, tx := range transactions {
		// Calculate transaction amount from entries
		// For simplicity, we'll sum all credit amounts in the physical layer
		var amount decimal.Decimal
		for _, entry := range tx.Entries {
			if entry.Layer == domain.LayerPhysical && entry.Type == domain.EntryTypeCredit {
				amount = amount.Add(entry.Amount)
			}
		}

		// Determine if this is an internal transfer
		// It's internal if it's not an external inflow
		isInternalTransfer := !tx.IsExternalInflow

		protoTx := &wealthflowv1.Transaction{
			Id:                 tx.ID.String(),
			Description:        tx.Description,
			Amount:             amount.String(),
			Date:               timestamppb.New(tx.Date),
			IsExternal:         tx.IsExternalInflow,
			IsInternalTransfer: isInternalTransfer,
		}

		protoTransactions = append(protoTransactions, protoTx)
	}

	return &wealthflowv1.ListTransactionsResponse{
		Transactions: protoTransactions,
		TotalCount:   int32(totalCount),
		BucketNames:  bucketNames,
	}, nil
}

// GetNetWorth handles the GetNetWorth RPC
func (s *Server) GetNetWorth(ctx context.Context, req *wealthflowv1.GetNetWorthRequest) (*wealthflowv1.GetNetWorthResponse, error) {
	// Call dashboard service
	result, err := s.DashboardService.GetNetWorth(ctx)
	if err != nil {
		return nil, mapError(err)
	}

	// Build response
	return &wealthflowv1.GetNetWorthResponse{
		TotalNetWorth: result.Total.String(),
		Liquidity:     result.Liquidity.String(),
		Equity:        result.Equity.String(),
	}, nil
}

// GetBucket handles the GetBucket RPC
func (s *Server) GetBucket(ctx context.Context, req *wealthflowv1.GetBucketRequest) (*wealthflowv1.GetBucketResponse, error) {
	// Parse bucket ID
	bucketID, err := uuid.Parse(req.BucketId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid bucket_id format: %v", err)
	}

	// Get bucket from repository
	bucket, err := s.DashboardService.BucketRepo.GetByID(ctx, bucketID)
	if err != nil {
		// Check if it's a "not found" error
		if strings.Contains(err.Error(), "not found") {
			return nil, status.Errorf(codes.NotFound, "bucket not found: %v", err)
		}
		return nil, mapError(err)
	}

	// Convert domain bucket to proto bucket
	protoBucket := domainBucketToProto(bucket)

	// Build response
	return &wealthflowv1.GetBucketResponse{
		Bucket: protoBucket,
	}, nil
}

// domainBucketTypeToProto converts a domain BucketType to a proto BucketType enum
func domainBucketTypeToProto(domainType domain.BucketType) wealthflowv1.BucketType {
	switch domainType {
	case domain.BucketTypePhysical:
		return wealthflowv1.BucketType_BUCKET_TYPE_PHYSICAL
	case domain.BucketTypeVirtual:
		return wealthflowv1.BucketType_BUCKET_TYPE_VIRTUAL
	case domain.BucketTypeIncome:
		return wealthflowv1.BucketType_BUCKET_TYPE_INCOME
	case domain.BucketTypeExpense:
		return wealthflowv1.BucketType_BUCKET_TYPE_EXPENSE
	case domain.BucketTypeEquity:
		return wealthflowv1.BucketType_BUCKET_TYPE_EQUITY
	default:
		return wealthflowv1.BucketType_BUCKET_TYPE_UNSPECIFIED
	}
}

// protoBucketTypeToDomain converts a proto BucketType enum to a domain BucketType
func protoBucketTypeToDomain(protoType wealthflowv1.BucketType) domain.BucketType {
	switch protoType {
	case wealthflowv1.BucketType_BUCKET_TYPE_PHYSICAL:
		return domain.BucketTypePhysical
	case wealthflowv1.BucketType_BUCKET_TYPE_VIRTUAL:
		return domain.BucketTypeVirtual
	case wealthflowv1.BucketType_BUCKET_TYPE_INCOME:
		return domain.BucketTypeIncome
	case wealthflowv1.BucketType_BUCKET_TYPE_EXPENSE:
		return domain.BucketTypeExpense
	case wealthflowv1.BucketType_BUCKET_TYPE_EQUITY:
		return domain.BucketTypeEquity
	default:
		return ""
	}
}

// domainBucketToProto converts a domain Bucket to a proto Bucket message
func domainBucketToProto(bucket *domain.Bucket) *wealthflowv1.Bucket {
	protoBucket := &wealthflowv1.Bucket{
		Id:             bucket.ID.String(),
		Name:           bucket.Name,
		Type:           domainBucketTypeToProto(bucket.BucketType),
		CurrentBalance: bucket.CurrentBalance.String(),
	}

	// Set parent_id if it exists
	if bucket.ParentPhysicalBucketID != nil {
		protoBucket.ParentId = bucket.ParentPhysicalBucketID.String()
	}

	return protoBucket
}

// mapError converts domain errors to gRPC status errors
func mapError(err error) error {
	if err == nil {
		return nil
	}

	errorMsg := err.Error()

	// Map common validation errors to InvalidArgument
	if strings.Contains(errorMsg, "must be positive") ||
		strings.Contains(errorMsg, "invalid") ||
		strings.Contains(errorMsg, "must reference") ||
		strings.Contains(errorMsg, "must have") {
		return status.Errorf(codes.InvalidArgument, "%s", errorMsg)
	}

	// Map "not found" errors to NotFound
	if strings.Contains(errorMsg, "not found") {
		return status.Errorf(codes.NotFound, "%s", errorMsg)
	}

	// Default to Internal error for unknown errors
	return status.Errorf(codes.Internal, "%s", errorMsg)
}
