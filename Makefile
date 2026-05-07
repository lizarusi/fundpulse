BINARY := healthcheck
BIN_DIR := bin
INSTALL_DIR := /usr/local/bin

.PHONY: build test install uninstall clean run show dryrun

build:
	mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(BINARY) ./cmd/healthcheck

test:
	go test ./...

install: build
	cp $(BIN_DIR)/$(BINARY) $(INSTALL_DIR)/$(BINARY)
	@echo "Installed to $(INSTALL_DIR)/$(BINARY). Run \`$(BINARY) init\` to set up."

uninstall:
	@if [ -x $(INSTALL_DIR)/$(BINARY) ]; then $(INSTALL_DIR)/$(BINARY) uninstall || true; fi
	rm -f $(INSTALL_DIR)/$(BINARY)
	@echo "Removed $(INSTALL_DIR)/$(BINARY) and launchd schedule. Config and DB preserved."

clean:
	rm -rf $(BIN_DIR)

show: build
	$(BIN_DIR)/$(BINARY) show

dryrun: build
	$(BIN_DIR)/$(BINARY) run --dry-run

run: build
	$(BIN_DIR)/$(BINARY) run
