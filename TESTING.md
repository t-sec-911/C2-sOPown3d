# Testing Guide for sOPown3d C2 Agent

## Quick Start

### Run all tests:
```bash
./test.sh
```

### Run specific package tests:
```bash
# Jitter package only
cd agent/jitter && go test -v

# All agent tests
go test ./agent/...

# All project tests
go test ./...
```

## Test Categories

### 1. Unit Tests
Located in: `agent/jitter/jitter_test.go`

**What they test:**
- [x] Configuration validation
- [x] Gaussian distribution properties
- [x] Boundary conditions
- [x] Edge cases (small/large ranges)
- [x] Randomization quality

**Coverage:** 100% of jitter package code

### 2. Build Tests
Validates that the project compiles for different platforms:
- [x] macOS (darwin/arm64)
- [x] Windows (windows/amd64)
- [x] Linux (linux/amd64)

### 3. Integration Tests
Manual testing of the full system:
```bash
# Terminal 1: Start server
go run server/main.go

# Terminal 2: Start agent with test jitter
go run agent/main.go -jitter-min=1 -jitter-max=2

# Observe heartbeat logs
```

## Test Commands Reference

### Basic Testing
```bash
# Run tests
go test ./agent/jitter/

# Run with verbose output
go test -v ./agent/jitter/

# Run specific test
go test -v ./agent/jitter/ -run TestNext_GaussianDistribution
```

### Coverage Analysis
```bash
# Show coverage percentage
go test -cover ./agent/jitter/

# Generate coverage profile
go test -coverprofile=coverage.out ./agent/jitter/

# View coverage in terminal
go tool cover -func=coverage.out

# View coverage in browser (HTML)
go tool cover -html=coverage.out
```

### Performance Testing
```bash
# Run benchmarks
go test -bench=. ./agent/jitter/

# Run benchmarks with memory stats
go test -bench=. -benchmem ./agent/jitter/

# Run benchmarks for 10 seconds
go test -bench=. -benchtime=10s ./agent/jitter/
```

### Race Detection
```bash
# Check for race conditions
go test -race ./agent/jitter/
```

### Build Verification
```bash
# Build for current platform
go build -o agent_test agent/main.go

# Build for Windows from macOS/Linux
GOOS=windows GOARCH=amd64 go build -o agent.exe agent/main.go

# Build for Linux from macOS
GOOS=linux GOARCH=amd64 go build -o agent_linux agent/main.go
```

## Test Results Interpretation

### Good Test Output
```
=== RUN   TestNext_GaussianDistribution
    jitter_test.go:198: Distribution stats: mean=9.9911 (expected: 10.0000), 
                        stdDev=1.6458, within2σ=95.26%
--- PASS: TestNext_GaussianDistribution (0.00s)
```

**What this means:**
- [PASS] Mean is very close to expected (9.99 vs 10.00)
- [PASS] 95.26% of values within 2σ (expected ~95%)
- [PASS] Gaussian distribution is working correctly

### Coverage Output
```
total: (statements) 100.0%
```
**What this means:**
- [PASS] All code paths are tested
- [PASS] No untested functions
- [PASS] Complete test coverage

### Benchmark Output
```
BenchmarkNext-8   	100000000	        10.49 ns/op	       0 B/op	       0 allocs/op
```

**What this means:**
- [PASS] Function runs in ~10 nanoseconds (very fast)
- [PASS] Zero memory allocations (efficient)
- [PASS] 100M operations in test (reliable measurement)

## Common Test Scenarios

### Test 1: Verify Jitter Range
```bash
# Test with 1-2 second range
go run agent/main.go -jitter-min=1 -jitter-max=2

# Observe output - should see values between 1.00s and 2.00s
# Example:
# [Heartbeat #1] Next check in: 1.46s
# [Heartbeat #2] Next check in: 1.65s
# [Heartbeat #3] Next check in: 1.32s
```

### Test 2: Verify Gaussian Distribution
Run agent multiple times and observe that values cluster around the mean:

For 5-15s range:
- Mean should be ~10s
- Most values between 8-12s
- Few values near 5s or 15s (boundaries)

### Test 3: Verify Cross-Platform Build
```bash
# Build for Windows
GOOS=windows GOARCH=amd64 go build -o agent.exe agent/main.go

# Verify file was created
ls -lh agent.exe

# Expected: Windows PE executable
file agent.exe
# Output: agent.exe: PE32+ executable (console) x86-64 (stripped to external PDB), for MS Windows
```

## Continuous Integration

### Pre-commit Hook
Create `.git/hooks/pre-commit`:
```bash
#!/bin/bash
echo "Running tests before commit..."
go test ./agent/jitter/ -cover
if [ $? -ne 0 ]; then
    echo "Tests failed. Commit aborted."
    exit 1
fi
```

### CI/CD Pipeline (GitHub Actions example)
```yaml
name: Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      - run: go test -v -race -coverprofile=coverage.out ./...
      - run: go tool cover -func=coverage.out
```

## Troubleshooting

### Problem: Tests fail with "build constraints exclude all Go files"
**Solution:** This is expected on macOS for Windows-specific code. Build tags are working correctly.

### Problem: Gaussian distribution test fails
**Possible causes:**
- Statistical variance (rare, re-run test)
- Random seed issue (check if time.Now() is working)

**Solution:** Tests use tolerances, so occasional variance is acceptable. If it consistently fails, there may be a bug.

### Problem: Coverage shows less than 100%
**Solution:** 
```bash
# See which lines are not covered
go test -coverprofile=coverage.out ./agent/jitter/
go tool cover -html=coverage.out
```

## Best Practices

1. **Run tests before committing:**
   ```bash
   ./test.sh
   ```

2. **Test on target platform when possible:**
   - Build for Windows and test on Windows VM
   - Verify agent connects to server correctly

3. **Monitor test output:**
   - Check that jitter values make sense
   - Verify distribution statistics are reasonable

4. **Keep tests fast:**
   - Current test suite runs in <2 seconds
   - Don't add slow integration tests to unit tests

5. **Maintain high coverage:**
   - Aim for 100% coverage on critical packages
   - Add tests when adding new features

## Adding New Tests

### Template for new feature:
1. Create `feature_test.go` in the same directory as `feature.go`
2. Write table-driven tests:
```go
func TestMyFeature(t *testing.T) {
    tests := []struct {
        name     string
        input    int
        expected int
    }{
        {"case 1", 1, 2},
        {"case 2", 2, 4},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := MyFeature(tt.input)
            if result != tt.expected {
                t.Errorf("got %d, want %d", result, tt.expected)
            }
        })
    }
}
```

3. Run tests: `go test -v`
4. Check coverage: `go test -cover`

## Resources

- [Go Testing Documentation](https://golang.org/pkg/testing/)
- [Table-Driven Tests](https://dave.cheney.net/2019/05/07/prefer-table-driven-tests)
- [Test Coverage](https://go.dev/blog/cover)
- Project-specific test guide: `agent/jitter/README_TESTS.md`
