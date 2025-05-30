.PHONY: all clean ui-build go-build dev run migrate-up migrate-down migrate-list migrate-up-step migrate-down-step deploy-build test-report

# Default target
all: clean ui-build go-build

# Clean build artifacts for UI static files in pb_public (preserving a placeholder to avoid embed errors)
clean:
	@echo "Cleaning build artifacts..."
	rm -f tybalt
	rm -rf pb_public/*
	mkdir -p pb_public
	# Create a placeholder file to ensure pb_public is never empty (so that the embed directive doesn't fail)
	touch pb_public/.keep

# Build the UI and copy to pb_public
ui-build:
	@echo "Building UI..."
	cd ../ui && npm run build
	@echo "Copying UI build to pb_public..."
	cp -r ../ui/build/* pb_public/

# Build the Go application
go-build:
	@echo "Building Go application..."
	go build -o tybalt main.go

# Development build that includes both UI and Go
dev: all
	@echo "Starting development server..."
	./tybalt serve

# Just run the Go application (without rebuilding UI)
run:
	@echo "Starting server..."
	./tybalt serve

# Migration commands
migrate-up:
	@echo "Applying migrations..."
	./tybalt migrate up

migrate-down:
	@echo "Rolling back last migration..."
	./tybalt migrate down --step=1

migrate-list:
	@echo "Listing migration status..."
	./tybalt migrate list

# Apply specific number of migrations
migrate-up-step:
	@echo "Applying migrations (step by step)..."
	@read -p "Number of steps: " steps; \
	./tybalt migrate up --step=$$steps

# Rollback specific number of migrations  
migrate-down-step:
	@echo "Rolling back migrations (step by step)..."
	@read -p "Number of steps: " steps; \
	./tybalt migrate down --step=$$steps

# Production deployment helper
deploy-build: all
	@echo "Production build complete. Ready for deployment."

test-report:
	go test ./... -v --cover -coverprofile=coverage.out
	go tool cover -html=coverage.out
