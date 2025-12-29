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
gen:
	@echo "Generating Go code..."
	mkdir -p $(BACKEND_OUT)
	protoc --proto_path=$(PROTO_DIR) \
		--go_out=$(BACKEND_OUT) --go_opt=paths=source_relative \
		--go-grpc_out=$(BACKEND_OUT) --go-grpc_opt=paths=source_relative \
		$(PROTO_DIR)/wealthflow/v1/*.proto

	@echo "Generating Dart code..."
	mkdir -p $(FRONTEND_OUT)
	protoc --proto_path=$(PROTO_DIR) \
		--dart_out=grpc:$(FRONTEND_OUT) \
		$(PROTO_DIR)/wealthflow/v1/*.proto

	@echo "Done!"

# Clean generated files
clean:
	rm -rf $(BACKEND_OUT)/*.pb.go
	rm -rf $(FRONTEND_OUT)/*.dart

# Docker commands
docker-up:
	docker-compose up -d

docker-down:
	docker-compose down
