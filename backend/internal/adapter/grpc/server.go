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
}

// NewServer creates a new gRPC server instance
func NewServer(
	expenseService *expense.ExpenseService,
	inflowService *inflow.InflowService,
	investmentService *investment.InvestmentService,
) *Server {
	return &Server{
		ExpenseService:    expenseService,
		InflowService:     inflowService,
		InvestmentService: investmentService,
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
		return status.Errorf(codes.InvalidArgument, errorMsg)
	}

	// Map "not found" errors to NotFound
	if strings.Contains(errorMsg, "not found") {
		return status.Errorf(codes.NotFound, errorMsg)
	}

	// Default to Internal error for unknown errors
	return status.Errorf(codes.Internal, errorMsg)
}
