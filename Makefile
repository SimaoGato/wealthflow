# Makefile for WealthFlow

# Variables
PROTO_DIR := proto
BACKEND_OUT := backend/internal/adapter/grpc
FRONTEND_OUT := frontend/lib/generated

# Install dependencies (Run once)
install-deps:
	# Install Go plugins
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	# Note: For Dart, ensure 'pub global activate protoc_plugin' is run
	# and your PATH includes the pub cache.

# Generate Code
gen: gen-proto gen-riverpod

# Generate both Go and Dart proto code
gen-proto: gen-proto-go gen-proto-dart

# Generate only Go proto code
gen-proto-go:
	@echo "Generating Go code..."
	mkdir -p $(BACKEND_OUT)
	protoc --proto_path=$(PROTO_DIR) \
		--go_out=$(BACKEND_OUT) --go_opt=paths=source_relative \
		--go-grpc_out=$(BACKEND_OUT) --go-grpc_opt=paths=source_relative \
		$(PROTO_DIR)/wealthflow/v1/*.proto

# Generate only Dart proto code
gen-proto-dart:
	@echo "Generating Dart proto code..."
	mkdir -p $(FRONTEND_OUT)
	protoc --proto_path=$(PROTO_DIR) \
		--dart_out=grpc:$(FRONTEND_OUT) \
		$(PROTO_DIR)/wealthflow/v1/*.proto

gen-riverpod:
	@echo "Generating Riverpod code..."
	cd frontend && flutter pub run build_runner build --delete-conflicting-outputs

	@echo "Done!"

# Clean generated files
clean:
	rm -rf $(BACKEND_OUT)/*.pb.go
	rm -rf $(FRONTEND_OUT)/*.dart
	find frontend/lib -name "*.g.dart" -type f -delete
	find frontend/lib -name "*.freezed.dart" -type f -delete

# Docker commands
docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

# Database migrations
migrate-up:
	docker run --rm -v $(PWD)/backend/db/migrations:/migrations --network host migrate/migrate -path=/migrations/ -database "postgres://postgres:postgres@localhost:5432/wealthflow?sslmode=disable" up

# Integration tests
test-integration:
	cd backend && go test -v -tags=integration ./tests/integration/...
