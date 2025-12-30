package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	grpclib "google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	grpcadapter "github.com/simaogato/wealthflow-backend/internal/adapter/grpc"
	wealthflowv1 "github.com/simaogato/wealthflow-backend/internal/adapter/grpc/wealthflow/v1"
	"github.com/simaogato/wealthflow-backend/internal/adapter/repository/postgres"
	"github.com/simaogato/wealthflow-backend/internal/usecase/dashboard"
	"github.com/simaogato/wealthflow-backend/internal/usecase/expense"
	"github.com/simaogato/wealthflow-backend/internal/usecase/inflow"
	"github.com/simaogato/wealthflow-backend/internal/usecase/investment"
	"github.com/simaogato/wealthflow-backend/internal/usecase/seeder"
)

const (
	defaultAPIToken = "dev-token"
	grpcPort        = ":8080"
)

func main() {
	// 1. Setup Database
	dbConnStr := os.Getenv("DB_CONN_STR")
	if dbConnStr == "" {
		// If explicit string is missing, build it from individual vars (Docker friendly)
		host := os.Getenv("DB_HOST")
		if host == "" {
			host = "localhost" // Default for local run without docker
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

		dbConnStr = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			host, port, user, password, dbname)
	}

	// Add 2-second delay to ensure Postgres is up (Simple retry)
	time.Sleep(2 * time.Second)

	db, err := postgres.NewDB(dbConnStr)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// 2. Initialize Repositories (Postgres)
	bucketRepo := postgres.NewBucketRepository(db)
	transactionRepo := postgres.NewTransactionRepository(db)
	splitRuleRepo := postgres.NewSplitRuleRepository(db)
	marketValueRepo := postgres.NewMarketValueRepository(db)

	// 3. Initialize Services (Use Cases)
	inflowService := inflow.NewInflowService(bucketRepo, transactionRepo, splitRuleRepo)
	expenseService := expense.NewExpenseService(bucketRepo, transactionRepo)
	investmentService := investment.NewInvestmentService(bucketRepo, marketValueRepo)
	dashboardService := dashboard.NewDashboardService(bucketRepo, transactionRepo, marketValueRepo)

	// Initialize System Seeder and run it
	systemSeeder := seeder.NewSystemSeeder(bucketRepo)
	ctx := context.Background()
	if err := systemSeeder.Seed(ctx); err != nil {
		log.Fatalf("Failed to seed system buckets: %v", err)
	}
	log.Println("System buckets seeded successfully")

	// 4. Start gRPC Server
	// Get API token from environment or use default
	apiToken := os.Getenv("API_TOKEN")
	if apiToken == "" {
		apiToken = defaultAPIToken
	}

	// Create gRPC server with AuthInterceptor
	grpcServer := grpclib.NewServer(
		grpclib.UnaryInterceptor(grpcadapter.AuthInterceptor(apiToken)),
	)

	// Register WealthFlowServiceServer
	grpcAdapter := grpcadapter.NewServer(expenseService, inflowService, investmentService, dashboardService)
	wealthflowv1.RegisterWealthFlowServiceServer(grpcServer, grpcAdapter)

	reflection.Register(grpcServer)

	// Listen on TCP port 8080
	lis, err := net.Listen("tcp", grpcPort)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", grpcPort, err)
	}

	// Start server in a goroutine
	go func() {
		log.Printf("gRPC server listening on %s", grpcPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve gRPC server: %v", err)
		}
	}()

	// Graceful shutdown
	waitForShutdown(grpcServer)
}

// waitForShutdown waits for SIGTERM or SIGINT and gracefully shuts down the server
func waitForShutdown(grpcServer *grpclib.Server) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	sig := <-sigChan
	log.Printf("Received signal: %v. Shutting down gracefully...", sig)

	grpcServer.GracefulStop()
	log.Println("gRPC server stopped")
}
