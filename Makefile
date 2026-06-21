# Build configuration
GO := go
BINARY_NAME := restreminder
SRC_DIR := ./src

# Default target
all: dev

# Local development build
dev:
	CGO_ENABLED=1 $(GO) build -o $(BINARY_NAME) $(SRC_DIR)

# Optimized production build
prod:
	CGO_ENABLED=1 $(GO) build -ldflags="-s -w" -o $(BINARY_NAME) $(SRC_DIR)

# Cleanup build outputs
clean:
	rm -f $(BINARY_NAME)

.PHONY: all dev prod clean
