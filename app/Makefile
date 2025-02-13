.PHONY: all clean ui-build go-build

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

test-report:
	go test ./... -v --cover -coverprofile=coverage.out
	go tool cover -html=coverage.out
