# Testing Guide

This document describes the test suite for cursor-monitor and how to run tests.

## Test Structure

Tests are organized by package, mirroring the source code structure:

- `internal/api/` - API client tests including pagination
- `internal/costs/` - Cost calculation and BYOK comparison tests
- `internal/storage/` - Database operations and data persistence tests
- `internal/alerts/` - Alert system tests
- `internal/config/` - Configuration loading tests
- `internal/monitor/` - Monitor polling and coordination tests
- `cmd/cursor-monitor/` - Command-line interface tests

## Running Tests

### Run All Tests
```bash
go test ./...
```

### Run Tests for Specific Package
```bash
go test ./internal/api
go test ./internal/costs
go test ./internal/storage
```

### Run Tests with Verbose Output
```bash
go test -v ./...
```

### Run Specific Test
```bash
go test -v ./internal/api -run TestPaginationLogic
```

## Test Coverage

### API Client Tests (`internal/api/`)

- **Invoice Parsing**: Tests parsing of invoice items with various formats
- **Pagination**: Tests pagination logic for fetching all usage events
  - Exact page boundaries
  - Multiple pages with remainder
  - Single page scenarios
- **Response Parsing**: Tests parsing of API responses with different field formats

### Cost Calculation Tests (`internal/costs/`)

- **BYOK Cost Calculation**: Tests calculation of direct API costs for different models
- **Model Provider Detection**: Tests identification of Anthropic, OpenAI, and Google models
- **Cost Comparison**: Tests comparison between Cursor costs and BYOK costs
- **Cursor Markup**: Tests application of estimated markup

### Storage Tests (`internal/storage/`)

- **Snapshot Management**: Tests saving and retrieving usage snapshots
- **Invoice Items**: Tests storing and querying invoice items by billing cycle
- **Usage Events**: Tests storing and querying usage events
- **Alert Tracking**: Tests recording and checking alert history
- **Data Cleanup**: Tests retention policy and old data removal

### Monitor Tests (`internal/monitor/`)

- **Polling Logic**: Tests single poll cycle execution
- **Historical Data Fetching**: Tests backfilling historical invoice data
- **Error Handling**: Tests handling of API failures and token refresh

### Command Tests (`cmd/cursor-monitor/`)

- **PID File Management**: Tests reading, writing, and removing PID files
- **PID File Format**: Tests correct format of PID files

## Integration Testing

For integration testing, you'll need:

1. A valid Cursor session token (extracted from Cursor's database)
2. Access to Cursor's API endpoints
3. A test database location

### Manual Integration Test Checklist

- [ ] Start daemon in foreground mode
- [ ] Verify polling occurs at configured interval
- [ ] Check that snapshots are saved to database
- [ ] Verify alerts are sent at thresholds
- [ ] Test daemon mode (start with --daemon)
- [ ] Verify PID file is created
- [ ] Test stop command
- [ ] Verify PID file is removed
- [ ] Test status command
- [ ] Test summary command
- [ ] Test costs command with --byok flag
- [ ] Test refresh command
- [ ] Test import command with sample CSV

## Test Data

### Sample CSV Format for Import

```csv
Date,Kind,Model,Max Mode,Input (w/ Cache Write),Input (w/o Cache Write),Cache Read,Output Tokens,Total Tokens,Cost
2026-01-15T10:30:00Z,On-Demand,claude-4-sonnet,,50000,0,10000,25000,85000,0.43
2026-01-15T11:00:00Z,Included,claude-4-sonnet,,30000,0,5000,15000,50000,0.00
```

## Continuous Integration

Tests should pass before merging any changes. Key test scenarios:

1. All unit tests pass
2. No race conditions (run with `go test -race`)
3. Code coverage maintained (run with `go test -cover`)

## Known Limitations

- Some tests require file system access (PID file management)
- Integration tests require valid API credentials
- Daemon mode tests require process management capabilities
- Build cache permission issues may occur on some systems (system-level, not code issue)

## Adding New Tests

When adding new features:

1. Add unit tests for core logic
2. Add integration tests for CLI commands
3. Update this document with new test scenarios
4. Ensure tests are deterministic and don't depend on external state
