//go:build integration

package integration

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	wealthflowv1 "github.com/simaogato/wealthflow-backend/internal/adapter/grpc/wealthflow/v1"
	"github.com/simaogato/wealthflow-backend/internal/adapter/repository/postgres"
	"github.com/simaogato/wealthflow-backend/internal/domain"
)

var (
	db          *postgres.DB
	grpcClient  wealthflowv1.WealthFlowServiceClient
	grpcConn    *grpc.ClientConn
	testBuckets map[string]uuid.UUID // Maps bucket name to ID
)

// TestMain sets up the test environment
func TestMain(m *testing.M) {
	ctx := context.Background()

	// 1. Connect to Database
	dbConnStr := getDBConnectionString()
	var err error
	db, err = postgres.NewDB(dbConnStr)
	if err != nil {
		panic(fmt.Sprintf("Failed to connect to database: %v", err))
	}
	defer db.Close()

	// 2. Connect to gRPC Server
	grpcAddr := getGRPCAddress()
	grpcConn, err = grpc.NewClient(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(fmt.Sprintf("Failed to connect to gRPC server: %v", err))
	}
	defer grpcConn.Close()

	grpcClient = wealthflowv1.NewWealthFlowServiceClient(grpcConn)

	// 3. Self-Healing Setup: Create test buckets if they don't exist
	testBuckets = make(map[string]uuid.UUID)
	if err := setupTestBuckets(ctx, db); err != nil {
		panic(fmt.Sprintf("Failed to setup test buckets: %v", err))
	}

	// 4. Fix Split Rules: Ensure "Employer" -> "Unallocated" split rule exists with valid name
	if err := setupSplitRule(ctx, db); err != nil {
		panic(fmt.Sprintf("Failed to setup split rule: %v", err))
	}

	// Run tests
	code := m.Run()

	os.Exit(code)
}

// setupTestBuckets creates the required test buckets if they don't exist
func setupTestBuckets(ctx context.Context, db *postgres.DB) error {
	bucketRepo := postgres.NewBucketRepository(db)

	// Define test buckets
	buckets := []struct {
		name       string
		bucketType domain.BucketType
		parentName string // For virtual buckets
	}{
		{"Main Bank", domain.BucketTypePhysical, ""},
		{"Unallocated", domain.BucketTypeVirtual, "Main Bank"},
		{"Employer", domain.BucketTypeIncome, ""},
		{"Groceries", domain.BucketTypeExpense, ""},
		{"Tesla Stock", domain.BucketTypeEquity, ""},
	}

	// First pass: Create all non-virtual buckets
	for _, b := range buckets {
		if b.bucketType == domain.BucketTypeVirtual {
			continue
		}

		// Check if bucket exists by name
		var existingID uuid.UUID
		query := `SELECT id FROM buckets WHERE name = $1`
		err := db.QueryRowContext(ctx, query, b.name).Scan(&existingID)
		if err == nil {
			// Bucket exists
			testBuckets[b.name] = existingID
			continue
		}
		if err != sql.ErrNoRows {
			return fmt.Errorf("failed to check bucket existence: %w", err)
		}

		// Create bucket
		bucket := &domain.Bucket{
			ID:             uuid.New(),
			Name:           b.name,
			BucketType:     b.bucketType,
			CurrentBalance: decimal.Zero,
		}

		if err := bucket.Validate(); err != nil {
			return fmt.Errorf("bucket validation failed: %w", err)
		}

		if err := bucketRepo.Create(ctx, bucket); err != nil {
			return fmt.Errorf("failed to create bucket %s: %w", b.name, err)
		}

		testBuckets[b.name] = bucket.ID
	}

	// Second pass: Create virtual buckets (they need parent IDs)
	for _, b := range buckets {
		if b.bucketType != domain.BucketTypeVirtual {
			continue
		}

		// Check if bucket exists by name
		var existingID uuid.UUID
		query := `SELECT id FROM buckets WHERE name = $1`
		err := db.QueryRowContext(ctx, query, b.name).Scan(&existingID)
		if err == nil {
			// Bucket exists
			testBuckets[b.name] = existingID
			continue
		}
		if err != sql.ErrNoRows {
			return fmt.Errorf("failed to check bucket existence: %w", err)
		}

		// Get parent ID
		parentID, ok := testBuckets[b.parentName]
		if !ok {
			return fmt.Errorf("parent bucket %s not found", b.parentName)
		}

		// Create virtual bucket
		bucket := &domain.Bucket{
			ID:                     uuid.New(),
			Name:                   b.name,
			BucketType:             b.bucketType,
			ParentPhysicalBucketID: &parentID,
			CurrentBalance:         decimal.Zero,
		}

		if err := bucket.Validate(); err != nil {
			return fmt.Errorf("bucket validation failed: %w", err)
		}

		if err := bucketRepo.Create(ctx, bucket); err != nil {
			return fmt.Errorf("failed to create bucket %s: %w", b.name, err)
		}

		testBuckets[b.name] = bucket.ID
	}

	return nil
}

// setupSplitRule ensures the "Employer" -> "Unallocated" split rule exists with a valid name
func setupSplitRule(ctx context.Context, db *postgres.DB) error {
	employerID, ok := testBuckets["Employer"]
	if !ok {
		return fmt.Errorf("Employer bucket not found")
	}

	unallocatedID, ok := testBuckets["Unallocated"]
	if !ok {
		return fmt.Errorf("Unallocated bucket not found")
	}

	// Check if split rule exists
	var existingRuleID uuid.UUID
	query := `SELECT id FROM split_rules WHERE source_bucket_id = $1`
	err := db.QueryRowContext(ctx, query, employerID).Scan(&existingRuleID)
	if err == nil {
		// Rule exists, check if name is NULL and update if needed
		var name sql.NullString
		checkNameQuery := `SELECT name FROM split_rules WHERE id = $1`
		err = db.QueryRowContext(ctx, checkNameQuery, existingRuleID).Scan(&name)
		if err != nil {
			return fmt.Errorf("failed to check split rule name: %w", err)
		}

		if !name.Valid || name.String == "" {
			// Update name
			updateQuery := `UPDATE split_rules SET name = $1 WHERE id = $2`
			_, err = db.ExecContext(ctx, updateQuery, "Employer Income Split", existingRuleID)
			if err != nil {
				return fmt.Errorf("failed to update split rule name: %w", err)
			}
		}

		// Check if split rule item exists
		var itemID uuid.UUID
		itemQuery := `SELECT id FROM split_rule_items WHERE split_rule_id = $1 AND target_bucket_id = $2`
		err = db.QueryRowContext(ctx, itemQuery, existingRuleID, unallocatedID).Scan(&itemID)
		if err == nil {
			// Item exists, we're done
			return nil
		}
		if err != sql.ErrNoRows {
			return fmt.Errorf("failed to check split rule item: %w", err)
		}

		// Item doesn't exist, create it
		insertItemQuery := `
			INSERT INTO split_rule_items (id, split_rule_id, target_bucket_id, rule_type, value, priority)
			VALUES ($1, $2, $3, $4, $5, $6)
		`
		_, err = db.ExecContext(ctx, insertItemQuery,
			uuid.New(),
			existingRuleID,
			unallocatedID,
			"REMAINDER",
			"0",
			1,
		)
		if err != nil {
			return fmt.Errorf("failed to create split rule item: %w", err)
		}

		return nil
	}

	if err != sql.ErrNoRows {
		return fmt.Errorf("failed to check split rule existence: %w", err)
	}

	// Create new split rule
	ruleID := uuid.New()
	insertRuleQuery := `
		INSERT INTO split_rules (id, name, source_bucket_id)
		VALUES ($1, $2, $3)
	`
	_, err = db.ExecContext(ctx, insertRuleQuery, ruleID, "Employer Income Split", employerID)
	if err != nil {
		return fmt.Errorf("failed to create split rule: %w", err)
	}

	// Create split rule item (REMAINDER type)
	insertItemQuery := `
		INSERT INTO split_rule_items (id, split_rule_id, target_bucket_id, rule_type, value, priority)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err = db.ExecContext(ctx, insertItemQuery,
		uuid.New(),
		ruleID,
		unallocatedID,
		"REMAINDER",
		"0",
		1,
	)
	if err != nil {
		return fmt.Errorf("failed to create split rule item: %w", err)
	}

	return nil
}

// getAuthContext returns a context with authorization metadata
func getAuthContext() context.Context {
	md := metadata.New(map[string]string{
		"authorization": "dev-token",
	})
	return metadata.NewOutgoingContext(context.Background(), md)
}

// getDBConnectionString returns the database connection string from environment or defaults
func getDBConnectionString() string {
	connStr := os.Getenv("DB_CONN_STR")
	if connStr != "" {
		return connStr
	}

	host := os.Getenv("DB_HOST")
	if host == "" {
		host = "localhost"
	}

	port := os.Getenv("DB_PORT")
	if port == "" {
		port = "5432"
	}

	user := os.Getenv("DB_USER")
	if user == "" {
		user = "postgres"
	}

	password := os.Getenv("DB_PASSWORD")
	if password == "" {
		password = "postgres"
	}

	dbname := os.Getenv("DB_NAME")
	if dbname == "" {
		dbname = "wealthflow"
	}

	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
}

// getGRPCAddress returns the gRPC server address from environment or defaults
func getGRPCAddress() string {
	addr := os.Getenv("GRPC_ADDRESS")
	if addr == "" {
		addr = "localhost:8080"
	}
	return addr
}

// TestEndToEndFlow tests the complete flow: Income -> Expense -> Investment
func TestEndToEndFlow(t *testing.T) {
	ctx := getAuthContext()

	// Step A: RecordInflow from "Employer" with is_external: true
	employerID := testBuckets["Employer"]
	inflowAmount := "1000.00"
	inflowReq := &wealthflowv1.RecordInflowRequest{
		Amount:         inflowAmount,
		Description:    "Monthly Salary",
		SourceBucketId: employerID.String(),
		IsExternal:     true,
	}

	inflowResp, err := grpcClient.RecordInflow(ctx, inflowReq)
	require.NoError(t, err, "RecordInflow should succeed")
	assert.NotEmpty(t, inflowResp.TransactionId, "Transaction ID should be returned")

	// Step B: Verify money landed in "Main Bank" (via the Virtual Unallocated parent)
	// Verify transaction entries were created correctly
	mainBankID := testBuckets["Main Bank"]
	unallocatedID := testBuckets["Unallocated"]

	// Check physical layer entry: Main Bank should be debited
	var physicalDebitCount int
	var physicalDebitAmount string
	query := `
		SELECT COUNT(*), COALESCE(SUM(amount), '0')
		FROM transaction_entries
		WHERE transaction_id = $1
			AND bucket_id = $2
			AND type = 'DEBIT'
			AND layer = 'PHYSICAL'
	`
	err = db.QueryRowContext(ctx, query, inflowResp.TransactionId, mainBankID).Scan(&physicalDebitCount, &physicalDebitAmount)
	require.NoError(t, err, "Should be able to query physical debit entry")
	assert.Equal(t, 1, physicalDebitCount, "Main Bank should have one physical debit entry")

	debitedAmount, err := decimal.NewFromString(physicalDebitAmount)
	require.NoError(t, err)
	expectedAmount, err := decimal.NewFromString(inflowAmount)
	require.NoError(t, err)
	assert.True(t, debitedAmount.Equal(expectedAmount), "Debited amount should match inflow amount")

	// Check virtual layer entry: Unallocated should be debited
	var virtualDebitCount int
	var virtualDebitAmount string
	query = `
		SELECT COUNT(*), COALESCE(SUM(amount), '0')
		FROM transaction_entries
		WHERE transaction_id = $1
			AND bucket_id = $2
			AND type = 'DEBIT'
			AND layer = 'VIRTUAL'
	`
	err = db.QueryRowContext(ctx, query, inflowResp.TransactionId, unallocatedID).Scan(&virtualDebitCount, &virtualDebitAmount)
	require.NoError(t, err, "Should be able to query virtual debit entry")
	assert.Equal(t, 1, virtualDebitCount, "Unallocated should have one virtual debit entry")

	virtualDebitedAmount, err := decimal.NewFromString(virtualDebitAmount)
	require.NoError(t, err)
	expectedAmount, err = decimal.NewFromString(inflowAmount)
	require.NoError(t, err)
	assert.True(t, virtualDebitedAmount.Equal(expectedAmount), "Virtual debited amount should match inflow amount")

	// Step C: LogExpense from "Unallocated" to "Groceries"
	groceriesID := testBuckets["Groceries"]
	expenseAmount := "50.00"
	expenseReq := &wealthflowv1.LogExpenseRequest{
		Amount:           expenseAmount,
		Description:      "Weekly Groceries",
		VirtualBucketId:  unallocatedID.String(),
		CategoryBucketId: groceriesID.String(),
	}

	expenseResp, err := grpcClient.LogExpense(ctx, expenseReq)
	require.NoError(t, err, "LogExpense should succeed")
	assert.NotEmpty(t, expenseResp.TransactionId, "Transaction ID should be returned")
	assert.Equal(t, mainBankID.String(), expenseResp.PhysicalBucketId, "Physical bucket should be Main Bank")

	// Step D: UpdateInvestment for "Tesla Stock"
	teslaID := testBuckets["Tesla Stock"]
	marketValue := "650.00"
	investmentReq := &wealthflowv1.UpdateInvestmentRequest{
		BucketId:    teslaID.String(),
		MarketValue: marketValue,
	}

	investmentResp, err := grpcClient.UpdateInvestment(ctx, investmentReq)
	require.NoError(t, err, "UpdateInvestment should succeed")
	assert.NotEmpty(t, investmentResp.EntryId, "Market value entry ID should be returned")

	// Verify market value was recorded
	var recordedValue string
	mvQuery := `SELECT market_value FROM market_value_history WHERE bucket_id = $1 ORDER BY date DESC LIMIT 1`
	err = db.QueryRowContext(ctx, mvQuery, teslaID).Scan(&recordedValue)
	require.NoError(t, err, "Should be able to query market value history")
	recordedDecimal, err := decimal.NewFromString(recordedValue)
	require.NoError(t, err)
	expectedMarketValue, err := decimal.NewFromString(marketValue)
	require.NoError(t, err)
	assert.True(t, recordedDecimal.Equal(expectedMarketValue), "Market value should match")
}

// TestEndToEndFlow_VerifyBalances tests that transaction entries are correctly created
// Note: This test verifies transaction entries rather than current_balance since
// balances may be calculated on-demand rather than stored
func TestEndToEndFlow_VerifyBalances(t *testing.T) {
	ctx := getAuthContext()

	mainBankID := testBuckets["Main Bank"]
	unallocatedID := testBuckets["Unallocated"]
	employerID := testBuckets["Employer"]
	groceriesID := testBuckets["Groceries"]

	// Record inflow
	inflowAmount := decimal.NewFromInt(500)
	inflowReq := &wealthflowv1.RecordInflowRequest{
		Amount:         inflowAmount.String(),
		Description:    "Test Inflow",
		SourceBucketId: employerID.String(),
		IsExternal:     true,
	}

	inflowResp, err := grpcClient.RecordInflow(ctx, inflowReq)
	require.NoError(t, err)

	// Verify transaction entries for inflow
	// Main Bank should have a physical DEBIT entry
	var mainBankDebitCount int
	var mainBankDebitAmount string
	query := `
		SELECT COUNT(*), COALESCE(SUM(amount), '0')
		FROM transaction_entries
		WHERE transaction_id = $1
			AND bucket_id = $2
			AND type = 'DEBIT'
			AND layer = 'PHYSICAL'
	`
	err = db.QueryRowContext(ctx, query, inflowResp.TransactionId, mainBankID).Scan(&mainBankDebitCount, &mainBankDebitAmount)
	require.NoError(t, err)
	assert.Equal(t, 1, mainBankDebitCount, "Main Bank should have one physical debit entry")

	mainBankDebit, err := decimal.NewFromString(mainBankDebitAmount)
	require.NoError(t, err)
	assert.True(t, mainBankDebit.Equal(inflowAmount), "Main Bank debit should equal inflow amount")

	// Unallocated should have a virtual DEBIT entry
	var unallocatedDebitCount int
	var unallocatedDebitAmount string
	query = `
		SELECT COUNT(*), COALESCE(SUM(amount), '0')
		FROM transaction_entries
		WHERE transaction_id = $1
			AND bucket_id = $2
			AND type = 'DEBIT'
			AND layer = 'VIRTUAL'
	`
	err = db.QueryRowContext(ctx, query, inflowResp.TransactionId, unallocatedID).Scan(&unallocatedDebitCount, &unallocatedDebitAmount)
	require.NoError(t, err)
	assert.Equal(t, 1, unallocatedDebitCount, "Unallocated should have one virtual debit entry")

	unallocatedDebit, err := decimal.NewFromString(unallocatedDebitAmount)
	require.NoError(t, err)
	assert.True(t, unallocatedDebit.Equal(inflowAmount), "Unallocated debit should equal inflow amount")

	// Log expense
	expenseAmount := decimal.NewFromInt(25)
	expenseReq := &wealthflowv1.LogExpenseRequest{
		Amount:           expenseAmount.String(),
		Description:      "Test Expense",
		VirtualBucketId:  unallocatedID.String(),
		CategoryBucketId: groceriesID.String(),
	}

	expenseResp, err := grpcClient.LogExpense(ctx, expenseReq)
	require.NoError(t, err)

	// Verify transaction entries for expense
	// Main Bank should have a physical CREDIT entry
	var mainBankCreditCount int
	var mainBankCreditAmount string
	query = `
		SELECT COUNT(*), COALESCE(SUM(amount), '0')
		FROM transaction_entries
		WHERE transaction_id = $1
			AND bucket_id = $2
			AND type = 'CREDIT'
			AND layer = 'PHYSICAL'
	`
	err = db.QueryRowContext(ctx, query, expenseResp.TransactionId, mainBankID).Scan(&mainBankCreditCount, &mainBankCreditAmount)
	require.NoError(t, err)
	assert.Equal(t, 1, mainBankCreditCount, "Main Bank should have one physical credit entry")

	mainBankCredit, err := decimal.NewFromString(mainBankCreditAmount)
	require.NoError(t, err)
	assert.True(t, mainBankCredit.Equal(expenseAmount), "Main Bank credit should equal expense amount")

	// Unallocated should have a virtual CREDIT entry
	var unallocatedCreditCount int
	var unallocatedCreditAmount string
	query = `
		SELECT COUNT(*), COALESCE(SUM(amount), '0')
		FROM transaction_entries
		WHERE transaction_id = $1
			AND bucket_id = $2
			AND type = 'CREDIT'
			AND layer = 'VIRTUAL'
	`
	err = db.QueryRowContext(ctx, query, expenseResp.TransactionId, unallocatedID).Scan(&unallocatedCreditCount, &unallocatedCreditAmount)
	require.NoError(t, err)
	assert.Equal(t, 1, unallocatedCreditCount, "Unallocated should have one virtual credit entry")

	unallocatedCredit, err := decimal.NewFromString(unallocatedCreditAmount)
	require.NoError(t, err)
	assert.True(t, unallocatedCredit.Equal(expenseAmount), "Unallocated credit should equal expense amount")

	// Groceries should have both physical and virtual DEBIT entries
	var groceriesDebitCount int
	var groceriesDebitAmount string
	query = `
		SELECT COUNT(*), COALESCE(SUM(amount), '0')
		FROM transaction_entries
		WHERE transaction_id = $1
			AND bucket_id = $2
			AND type = 'DEBIT'
	`
	err = db.QueryRowContext(ctx, query, expenseResp.TransactionId, groceriesID).Scan(&groceriesDebitCount, &groceriesDebitAmount)
	require.NoError(t, err)
	assert.Equal(t, 2, groceriesDebitCount, "Groceries should have two debit entries (physical and virtual)")

	groceriesDebit, err := decimal.NewFromString(groceriesDebitAmount)
	require.NoError(t, err)
	expectedGroceriesTotal := expenseAmount.Mul(decimal.NewFromInt(2)) // Physical + Virtual
	assert.True(t, groceriesDebit.Equal(expectedGroceriesTotal), "Groceries total debit should equal 2x expense amount")
}

// TestNegativeScenarios tests error handling for invalid inputs
func TestNegativeScenarios(t *testing.T) {
	ctx := getAuthContext()
	employerID := testBuckets["Employer"]
	groceriesID := testBuckets["Groceries"]

	// 1. Invalid Amount: RecordInflow with negative amount
	t.Run("InvalidAmount", func(t *testing.T) {
		inflowReq := &wealthflowv1.RecordInflowRequest{
			Amount:         "-100.00",
			Description:    "Invalid Negative Amount",
			SourceBucketId: employerID.String(),
			IsExternal:     true,
		}

		_, err := grpcClient.RecordInflow(ctx, inflowReq)
		require.Error(t, err, "RecordInflow with negative amount should return an error")
		assert.Equal(t, codes.InvalidArgument, status.Code(err), "Error code should be InvalidArgument")
	})

	// 2. Non-Existent Bucket: LogExpense with random UUID
	t.Run("NonExistentBucket", func(t *testing.T) {
		nonExistentID := uuid.New()
		expenseReq := &wealthflowv1.LogExpenseRequest{
			Amount:           "50.00",
			Description:      "Expense with non-existent bucket",
			VirtualBucketId:  nonExistentID.String(),
			CategoryBucketId: groceriesID.String(),
		}

		_, err := grpcClient.LogExpense(ctx, expenseReq)
		require.Error(t, err, "LogExpense with non-existent bucket should return an error")
		assert.Equal(t, codes.NotFound, status.Code(err), "Error code should be NotFound")
	})

	// 3. Malformed UUID: UpdateInvestment with invalid UUID
	t.Run("MalformedUUID", func(t *testing.T) {
		investmentReq := &wealthflowv1.UpdateInvestmentRequest{
			BucketId:    "not-a-uuid",
			MarketValue: "100.00",
		}

		_, err := grpcClient.UpdateInvestment(ctx, investmentReq)
		require.Error(t, err, "UpdateInvestment with malformed UUID should return an error")
		assert.Equal(t, codes.InvalidArgument, status.Code(err), "Error code should be InvalidArgument")
	})
}

// TestReadFlow tests the Read APIs: ListBuckets, ListTransactions, and GetNetWorth
func TestReadFlow(t *testing.T) {
	ctx := getAuthContext()

	// Setup: Create test data
	employerID := testBuckets["Employer"]
	teslaID := testBuckets["Tesla Stock"]
	mainBankID := testBuckets["Main Bank"]

	// 1. Record inflow with description "Salary" to test ListTransactions
	salaryAmount := "2000.00"
	salaryReq := &wealthflowv1.RecordInflowRequest{
		Amount:         salaryAmount,
		Description:    "Salary",
		SourceBucketId: employerID.String(),
		IsExternal:     true,
	}

	salaryResp, err := grpcClient.RecordInflow(ctx, salaryReq)
	require.NoError(t, err, "RecordInflow should succeed")

	// 2. Update investment for Tesla Stock to $800.00 to test GetNetWorth equity
	teslaMarketValue := "800.00"
	investmentReq := &wealthflowv1.UpdateInvestmentRequest{
		BucketId:    teslaID.String(),
		MarketValue: teslaMarketValue,
	}

	_, err = grpcClient.UpdateInvestment(ctx, investmentReq)
	require.NoError(t, err, "UpdateInvestment should succeed")

	// 3. Test ListBuckets: Verify "Groceries" and "Tesla Stock" appear in the list
	t.Run("ListBuckets", func(t *testing.T) {
		bucketsReq := &wealthflowv1.ListBucketsRequest{}
		bucketsResp, err := grpcClient.ListBuckets(ctx, bucketsReq)
		require.NoError(t, err, "ListBuckets should succeed")
		require.NotNil(t, bucketsResp, "ListBuckets response should not be nil")

		// Find "Groceries" and "Tesla Stock" in the list
		var groceriesFound, teslaFound bool
		for _, bucket := range bucketsResp.Buckets {
			if bucket.Name == "Groceries" {
				groceriesFound = true
				assert.Equal(t, "EXPENSE", bucket.Type, "Groceries should be EXPENSE type")
			}
			if bucket.Name == "Tesla Stock" {
				teslaFound = true
				assert.Equal(t, "EQUITY", bucket.Type, "Tesla Stock should be EQUITY type")
			}
		}

		assert.True(t, groceriesFound, "Groceries bucket should appear in ListBuckets")
		assert.True(t, teslaFound, "Tesla Stock bucket should appear in ListBuckets")
	})

	// 4. Test ListTransactions: Verify the "Salary" transaction appears in the history
	t.Run("ListTransactions", func(t *testing.T) {
		transactionsReq := &wealthflowv1.ListTransactionsRequest{
			Limit:  100,
			Offset: 0,
		}
		transactionsResp, err := grpcClient.ListTransactions(ctx, transactionsReq)
		require.NoError(t, err, "ListTransactions should succeed")
		require.NotNil(t, transactionsResp, "ListTransactions response should not be nil")

		// Find the specific "Salary" transaction by ID (the one we just created)
		var salaryTx *wealthflowv1.Transaction
		for _, tx := range transactionsResp.Transactions {
			if tx.Id == salaryResp.TransactionId {
				salaryTx = tx
				break
			}
		}

		require.NotNil(t, salaryTx, "Salary transaction should appear in ListTransactions")
		assert.Equal(t, "Salary", salaryTx.Description, "Transaction description should match")

		// Compare amounts as decimals to handle formatting differences
		expectedAmount, err := decimal.NewFromString(salaryAmount)
		require.NoError(t, err)
		actualAmount, err := decimal.NewFromString(salaryTx.Amount)
		require.NoError(t, err)
		assert.True(t, actualAmount.Equal(expectedAmount),
			"Salary transaction amount should match: got %s, expected %s",
			salaryTx.Amount, salaryAmount)

		assert.True(t, salaryTx.IsExternal, "Salary transaction should be marked as external")
	})

	// 5. Test GetNetWorth: Verify Liquidity and Equity values
	t.Run("GetNetWorth", func(t *testing.T) {
		netWorthReq := &wealthflowv1.GetNetWorthRequest{}
		netWorthResp, err := grpcClient.GetNetWorth(ctx, netWorthReq)
		require.NoError(t, err, "GetNetWorth should succeed")
		require.NotNil(t, netWorthResp, "GetNetWorth response should not be nil")

		// Verify Liquidity matches the current bank balance
		// Main Bank should have a balance equal to the salary amount (1000.00 from TestEndToEndFlow + 2000.00 from this test)
		// But we'll check the actual balance from the database
		var mainBankBalance string
		balanceQuery := `SELECT current_balance FROM buckets WHERE id = $1`
		err = db.QueryRowContext(ctx, balanceQuery, mainBankID).Scan(&mainBankBalance)
		require.NoError(t, err, "Should be able to query Main Bank balance")

		mainBankBalanceDecimal, err := decimal.NewFromString(mainBankBalance)
		require.NoError(t, err)

		liquidityDecimal, err := decimal.NewFromString(netWorthResp.Liquidity)
		require.NoError(t, err)

		// Liquidity should match the Main Bank balance (sum of all physical bucket balances)
		// Since we only have Main Bank as a physical bucket in tests, they should match
		assert.True(t, liquidityDecimal.Equal(mainBankBalanceDecimal),
			"Liquidity should match Main Bank balance: got %s, expected %s",
			netWorthResp.Liquidity, mainBankBalance)

		// Verify Equity matches the Tesla stock value ($800.00)
		expectedEquity, err := decimal.NewFromString(teslaMarketValue)
		require.NoError(t, err)

		equityDecimal, err := decimal.NewFromString(netWorthResp.Equity)
		require.NoError(t, err)

		assert.True(t, equityDecimal.Equal(expectedEquity),
			"Equity should match Tesla stock value: got %s, expected %s",
			netWorthResp.Equity, teslaMarketValue)

		// Verify total net worth is liquidity + equity
		expectedTotal := liquidityDecimal.Add(expectedEquity)
		totalDecimal, err := decimal.NewFromString(netWorthResp.TotalNetWorth)
		require.NoError(t, err)

		assert.True(t, totalDecimal.Equal(expectedTotal),
			"Total net worth should equal liquidity + equity: got %s, expected %s",
			netWorthResp.TotalNetWorth, expectedTotal.String())
	})
}
