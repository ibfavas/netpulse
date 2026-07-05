.PHONY: all build install clean format

# Variables
BINARY_NAME=netpulse
CMD_DIR=./cmd/netpulse

all: format build

format:
	@echo "🎨 Formatting Go source code..."
	go fmt ./...

build:
	@echo "🛠️ Compiling local binary..."
	go build -o $(BINARY_NAME) $(CMD_DIR)

install: build
	@echo "🚀 Installing binary globally to /usr/local/bin (requires sudo)..."
	sudo cp $(BINARY_NAME) /usr/local/bin/
	@echo "🔧 Applying CAP_NET_RAW privileges..."
	sudo setcap cap_net_raw+ep /usr/local/bin/$(BINARY_NAME)
	@echo "✅ Installed successfully! You can now run 'netpulse' or 'sudo netpulse' anywhere."

clean:
	@echo "🧹 Cleaning build artifacts..."
	rm -f $(BINARY_NAME)
