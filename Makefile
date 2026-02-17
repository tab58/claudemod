.PHONY: build test test-race lint vet clean install

BINARY := claudemod
BUILD_DIR := bin
INSTALL_DIR := $(HOME)/.local/bin

build:
	go build -o $(BUILD_DIR)/$(BINARY) ./cmd/claudemod

test:
	go test -cover ./...

test-race:
	go test -race ./...

vet:
	go vet ./...

lint: vet
	@which staticcheck > /dev/null 2>&1 && staticcheck ./... || echo "staticcheck not installed, skipping"

clean:
	rm -f $(BUILD_DIR)/$(BINARY)

install: build
	mkdir -p $(INSTALL_DIR)
	cp $(BUILD_DIR)/$(BINARY) $(INSTALL_DIR)/$(BINARY)
