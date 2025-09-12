# Makefile for building binaries for Linux x86-64 (for vast.ai instances)

.PHONY: build clean

# Build all binaries for Linux x86-64
build:
	@echo "Building binaries for Linux x86-64..."
	GOOS=linux GOARCH=amd64 go build -o test_run/start_game main.go
	GOOS=linux GOARCH=amd64 go build -o test_run/recorder recorder/main.go
	@echo "Build complete! Binaries are ready for vast.ai Linux instances."

# Verify binary architecture
verify:
	@echo "Verifying binary architecture..."
	file test_run/start_game test_run/recorder

# Clean built binaries
clean:
	rm -f test_run/start_game test_run/recorder

# Build and verify
all: build verify