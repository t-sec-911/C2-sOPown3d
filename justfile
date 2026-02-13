# justfile for sOPown3d C2 Agent
# Run 'just' or 'just --list' to see available recipes

# Set shell for all recipes
set shell := ["bash", "-c"]

# Variables
BUILD_DIR := "build"
MODULE := "sOPown3d"
AGENT_MAIN := "cmd/agent/main.go"
SERVER_MAIN := "cmd/server/main.go"

# Default recipe - show help
default:
    @just --list

# ========================================
# Building
# ========================================

# Build both agent and server for current platform
build:
    @just build-agent `go env GOOS` `go env GOARCH`
    @just build-server `go env GOOS` `go env GOARCH`
    @echo "✓ Build complete! Binaries in {{BUILD_DIR}}/`go env GOOS`/"

# Build agent for specified platform
build-agent os arch:
    @echo "Building agent for {{os}}/{{arch}}..."
    @mkdir -p {{BUILD_DIR}}/{{os}}
    @if [ "{{os}}" = "windows" ]; then \
        GOOS={{os}} GOARCH={{arch}} go build -o {{BUILD_DIR}}/{{os}}/agent.exe {{AGENT_MAIN}}; \
    else \
        GOOS={{os}} GOARCH={{arch}} go build -o {{BUILD_DIR}}/{{os}}/agent {{AGENT_MAIN}}; \
    fi
    @echo "✓ Agent built successfully"

# Build server for specified platform
build-server os arch:
    @echo "Building server for {{os}}/{{arch}}..."
    @mkdir -p {{BUILD_DIR}}/{{os}}
    @if [ "{{os}}" = "windows" ]; then \
        GOOS={{os}} GOARCH={{arch}} go build -o {{BUILD_DIR}}/{{os}}/server.exe {{SERVER_MAIN}}; \
    else \
        GOOS={{os}} GOARCH={{arch}} go build -o {{BUILD_DIR}}/{{os}}/server {{SERVER_MAIN}}; \
    fi
    @echo "✓ Server built successfully"

# Build for all major platforms (Linux, Windows, macOS)
build-all:
    @echo "Building for all platforms..."
    @just build-agent linux amd64
    @just build-server linux amd64
    @just build-agent windows amd64
    @just build-server windows amd64
    @just build-agent darwin amd64
    @just build-server darwin amd64
    @just build-agent darwin arm64
    @just build-server darwin arm64
    @echo ""
    @echo "✓ All builds complete!"
    @echo "Platforms: Linux (amd64), Windows (amd64), macOS (amd64, arm64)"

# Build with optimizations (smaller binaries)
build-release os arch:
    @echo "Building optimized release for {{os}}/{{arch}}..."
    @mkdir -p {{BUILD_DIR}}/{{os}}
    @if [ "{{os}}" = "windows" ]; then \
        GOOS={{os}} GOARCH={{arch}} go build -ldflags="-s -w" -o {{BUILD_DIR}}/{{os}}/agent.exe {{AGENT_MAIN}}; \
        GOOS={{os}} GOARCH={{arch}} go build -ldflags="-s -w" -o {{BUILD_DIR}}/{{os}}/server.exe {{SERVER_MAIN}}; \
    else \
        GOOS={{os}} GOARCH={{arch}} go build -ldflags="-s -w" -o {{BUILD_DIR}}/{{os}}/agent {{AGENT_MAIN}}; \
        GOOS={{os}} GOARCH={{arch}} go build -ldflags="-s -w" -o {{BUILD_DIR}}/{{os}}/server {{SERVER_MAIN}}; \
    fi
    @echo "✓ Release build complete (optimized)"

# Clean build artifacts
clean:
    @echo "Cleaning build artifacts..."
    @rm -rf {{BUILD_DIR}}
    @rm -f agent/jitter/coverage.out
    @rm -f coverage.out
    @echo "✓ Clean complete"

# ========================================
# Testing
# ========================================

# Run all tests with coverage
test:
    @echo "=========================================="
    @echo "   sOPown3d C2 Agent - Test Suite"
    @echo "=========================================="
    @echo ""
    @echo ">>> Running all tests..."
    @go test -v -cover ./...
    @echo ""
    @echo "✓ All tests passed!"

# Run tests for jitter package only
test-jitter:
    @echo ">>> Testing Jitter Package"
    @echo "=========================================="
    @cd agent/jitter && go test -v -cover -coverprofile=coverage.out
    @echo ""
    @echo "✓ Jitter tests complete"

# Generate and display coverage report
test-coverage:
    @echo ">>> Generating coverage report..."
    @cd agent/jitter && go test -cover -coverprofile=coverage.out
    @echo ""
    @echo ">>> Coverage Report"
    @echo "=========================================="
    @cd agent/jitter && go tool cover -func=coverage.out
    @echo ""
    @echo "✓ Coverage analysis complete"

# Open coverage report in browser
test-coverage-html:
    @echo ">>> Generating HTML coverage report..."
    @cd agent/jitter && go test -coverprofile=coverage.out
    @cd agent/jitter && go tool cover -html=coverage.out
    @echo "✓ Coverage report opened in browser"

# Run performance benchmarks
test-bench:
    @echo ">>> Performance Benchmarks"
    @echo "=========================================="
    @cd agent/jitter && go test -bench=. -benchmem
    @echo ""
    @echo "✓ Benchmarks complete"

# Run tests with race detector
test-race:
    @echo ">>> Running tests with race detector..."
    @go test -race ./...
    @echo "✓ Race detection complete"

# Run complete test suite (like test.sh)
test-full:
    @echo "=========================================="
    @echo "   sOPown3d C2 Agent - Full Test Suite"
    @echo "=========================================="
    @echo ""
    @just test-jitter
    @just test-coverage
    @just test-bench
    @echo ""
    @echo ">>> Building Agent"
    @echo "=========================================="
    @go build -o /tmp/test_agent {{AGENT_MAIN}}
    @echo "✓ Agent builds successfully"
    @echo ""
    @echo ">>> Building Server"
    @echo "=========================================="
    @go build -o /tmp/test_server {{SERVER_MAIN}}
    @echo "✓ Server builds successfully"
    @echo ""
    @echo ">>> Cross-Compilation Test (Windows)"
    @echo "=========================================="
    @GOOS=windows GOARCH=amd64 go build -o /tmp/test_agent.exe {{AGENT_MAIN}}
    @echo "✓ Windows build successful"
    @echo ""
    @echo "=========================================="
    @echo "   All systems operational!"
    @echo "=========================================="

# ========================================
# Development
# ========================================

# Run server in development mode
dev-server port="8080":
    @echo "Starting server on port {{port}}..."
    @echo "Press Ctrl+C to stop"
    @go run {{SERVER_MAIN}}

# Run agent in development mode with custom jitter
dev-agent jitter-min="1" jitter-max="2":
    @echo "Starting agent with jitter range: {{jitter-min}}-{{jitter-max}}s"
    @echo "Press Ctrl+C to stop"
    @go run {{AGENT_MAIN}} -jitter-min={{jitter-min}} -jitter-max={{jitter-max}}

# Run integration test (server + agent)
dev-integration:
    @echo "=========================================="
    @echo "   Integration Test"
    @echo "=========================================="
    @echo ""
    @echo "Starting server in background..."
    @go run {{SERVER_MAIN}} & \
    SERVER_PID=$$! && \
    sleep 2 && \
    echo "Starting agent (will run 3 heartbeats)..." && \
    timeout 10 go run {{AGENT_MAIN}} -jitter-min=1 -jitter-max=2 || true && \
    echo "" && \
    echo "Stopping server..." && \
    kill $$SERVER_PID 2>/dev/null || true && \
    echo "✓ Integration test complete"

# Format all Go code
fmt:
    @echo "Formatting Go code..."
    @go fmt ./...
    @echo "✓ Code formatted"

# Run go vet (linter)
lint:
    @echo "Running go vet..."
    @go vet ./...
    @echo "✓ Linting complete"

# Run pre-commit checks (format, lint, test)
check:
    @echo "=========================================="
    @echo "   Pre-Commit Checks"
    @echo "=========================================="
    @echo ""
    @just fmt
    @echo ""
    @just lint
    @echo ""
    @just test
    @echo ""
    @echo "=========================================="
    @echo "   ✓ All checks passed!"
    @echo "=========================================="

# ========================================
# Network Utilities
# ========================================

# Get local IP address for network deployment
get-local-ip:
    @echo "Detecting local IP address..."
    @echo ""
    @if command -v ipconfig >/dev/null 2>&1; then \
        echo "Local IP (WiFi):"; \
        ipconfig getifaddr en0 2>/dev/null || echo "  Not connected"; \
        echo "Local IP (Ethernet):"; \
        ipconfig getifaddr en1 2>/dev/null || echo "  Not connected"; \
    elif command -v hostname >/dev/null 2>&1; then \
        echo "Local IP:"; \
        hostname -I 2>/dev/null | awk '{print "  " $$1}' || echo "  Unable to detect"; \
    else \
        echo "Local IP:"; \
        ifconfig 2>/dev/null | grep "inet " | grep -v 127.0.0.1 | awk '{print "  " $$2}' || echo "  Unable to detect"; \
    fi
    @echo ""
    @echo "Use this IP when running agent on Windows VM:"
    @echo "  agent.exe -server http://YOUR_IP:8080"

# Start server in network mode (accessible from VMs/other machines)
dev-server-network:
    @echo "Starting server in NETWORK mode..."
    @echo "⚠️  WARNING: Server will be accessible from your network!"
    @echo ""
    @just get-local-ip
    @echo ""
    @SERVER_HOST=0.0.0.0 DATABASE_URL="postgres://c2user:c2pass@localhost:5433/c2_db?sslmode=disable" go run cmd/server/main.go

# ========================================
# Docker & Database
# ========================================

# Start PostgreSQL Docker container
docker-up:
    @echo "Starting PostgreSQL container..."
    @docker-compose up -d
    @echo "Waiting for PostgreSQL to be ready..."
    @sleep 3
    @echo "✓ PostgreSQL is running on localhost:5432"
    @echo ""
    @echo "Database credentials:"
    @echo "  Host: localhost"
    @echo "  Port: 5432"
    @echo "  User: c2user"
    @echo "  Password: c2pass"
    @echo "  Database: c2_db"

# Stop PostgreSQL container
docker-down:
    @echo "Stopping PostgreSQL container..."
    @docker-compose down
    @echo "✓ PostgreSQL stopped"

# View PostgreSQL logs
docker-logs:
    @docker-compose logs -f postgres

# Reset database (delete all data)
docker-reset:
    @echo "⚠️  Warning: This will delete all database data!"
    @echo "Resetting database..."
    @docker-compose down -v
    @docker-compose up -d
    @sleep 3
    @echo "✓ Database reset complete"

# Connect to PostgreSQL CLI
docker-psql:
    @docker-compose exec postgres psql -U c2user -d c2_db

# Check Docker container status
docker-status:
    @echo "Docker container status:"
    @docker-compose ps

# Run tests with Docker database
test-integration:
    @echo "Running integration tests with PostgreSQL..."
    @echo "Make sure Docker is running: just docker-up"
    @go test -v -tags=integration ./server/...

# ========================================
# Dependencies & Utilities
# ========================================

# Download and install dependencies
deps:
    @echo "Installing dependencies..."
    @go mod download
    @echo "✓ Dependencies installed"

# Clean up and optimize go.mod and go.sum
tidy:
    @echo "Tidying modules..."
    @go mod tidy
    @echo "✓ Modules tidied"

# Install binaries to $GOPATH/bin
install: build
    @echo "Installing binaries to $GOPATH/bin..."
    @cp {{BUILD_DIR}}/$(go env GOOS)/agent $(go env GOPATH)/bin/sopown3d-agent
    @cp {{BUILD_DIR}}/$(go env GOOS)/server $(go env GOPATH)/bin/sopown3d-server
    @echo "✓ Installed: sopown3d-agent, sopown3d-server"

# Show project information
info:
    @echo "=========================================="
    @echo "   sOPown3d C2 Agent - Project Info"
    @echo "=========================================="
    @echo "Module:      {{MODULE}}"
    @echo "Go Version:  $(go version | awk '{print $3}')"
    @echo "Build Dir:   {{BUILD_DIR}}"
    @echo "Platform:    $(go env GOOS)/$(go env GOARCH)"
    @echo ""
    @echo "Source Files:"
    @echo "  Agent:     {{AGENT_MAIN}}"
    @echo "  Server:    {{SERVER_MAIN}}"
    @echo ""
    @echo "Test Coverage:"
    @go test -cover ./agent/jitter/ 2>/dev/null | grep coverage || echo "  Run 'just test-coverage' to see coverage"
    @echo "=========================================="

# Show common usage examples
help:
    @echo "=========================================="
    @echo "   sOPown3d C2 Agent - Just Commands"
    @echo "=========================================="
    @echo ""
    @echo "BUILDING:"
    @echo "  just build                    Build agent and server for current platform"
    @echo "  just build-agent linux amd64  Build agent for Linux"
    @echo "  just build-agent windows amd64 Build agent for Windows"
    @echo "  just build-all                Build for all platforms"
    @echo "  just clean                    Remove build artifacts"
    @echo ""
    @echo "TESTING:"
    @echo "  just test                     Run all tests"
    @echo "  just test-jitter              Test jitter package only"
    @echo "  just test-coverage            Show coverage report"
    @echo "  just test-bench               Run benchmarks"
    @echo "  just test-full                Complete test suite (like test.sh)"
    @echo ""
    @echo "DEVELOPMENT:"
    @echo "  just dev-server               Run server (localhost only)"
    @echo "  just dev-server-network       Run server (network accessible)"
    @echo "  just dev-agent                Run agent with 1-2s jitter"
    @echo "  just dev-agent 5 15           Run agent with 5-15s jitter"
    @echo "  just dev-integration          Test server + agent together"
    @echo "  just check                    Run pre-commit checks"
    @echo ""
    @echo "NETWORK:"
    @echo "  just get-local-ip             Show your local IP address"
    @echo "  just dev-server-network       Start server (network mode)"
    @echo ""
    @echo "DOCKER & DATABASE:"
    @echo "  just docker-up                Start PostgreSQL container"
    @echo "  just docker-down              Stop PostgreSQL container"
    @echo "  just docker-logs              View PostgreSQL logs"
    @echo "  just docker-reset             Reset database (delete all data)"
    @echo "  just docker-psql              Connect to PostgreSQL CLI"
    @echo "  just docker-status            Check Docker status"
    @echo ""
    @echo "UTILITIES:"
    @echo "  just fmt                      Format code"
    @echo "  just lint                     Run linter"
    @echo "  just info                     Show project info"
    @echo "  just help                     Show this help message"
    @echo ""
    @echo "DOCUMENTATION:"
    @echo "  DATABASE_SETUP.md             PostgreSQL setup and configuration"
    @echo "  USAGE.md                      Quick usage guide and examples"
    @echo "  NETWORK_SETUP.md              Windows VM network deployment guide"
    @echo ""
    @echo "Examples:"
    @echo "  just docker-up && DATABASE_URL=\"postgres://c2user:c2pass@localhost:5433/c2_db?sslmode=disable\" just dev-server"
    @echo "  just check                    (before committing)"
    @echo "  just build-all                (before release)"
    @echo "=========================================="
