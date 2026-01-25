# Implementation Plan

## Status Overview

### ✅ Completed Features
- Core monitoring service with polling
- SQLite database storage (snapshots, events, invoice items)
- API client for Cursor's internal APIs
- Authentication via JWT token extraction from Cursor's local database
- Usage event fetching and parsing
- Invoice items fetching and parsing
- CSV import functionality
- macOS notifications via osascript
- Alert system (threshold alerts, on-demand switch alerts)
- Summary command
- Status command
- Refresh command
- Cost calculation framework (BYOK pricing data)
- Historical data fetching

### ❌ Missing Features

#### 1. **Stop Command** (High Priority)
- **Status**: Not implemented
- **Location**: README mentions `cursor-monitor stop` but no command exists
- **Requirements**:
  - Read PID file (if daemon mode is implemented)
  - Send SIGTERM to running process
  - Verify process stopped
  - Clean up PID file
- **Implementation**:
  - Add `stopCmd` to `cmd/cursor-monitor/main.go`
  - Create PID file management functions
  - Implement process lookup and termination

#### 2. **Daemon Mode Implementation** (High Priority)
- **Status**: Flag exists but not implemented (TODO at line 127)
- **Location**: `cmd/cursor-monitor/main.go:127`
- **Requirements**:
  - Fork process to background
  - Detach from terminal (setsid)
  - Redirect stdin/stdout/stderr to /dev/null or log file
  - Write PID file to `~/.cursor_monitor/cursor-monitor.pid`
  - Handle double-fork on Unix systems
- **Implementation**:
  - Use `os/exec` or `syscall` for forking
  - Implement PID file management
  - Update `runStart` to handle daemon mode properly
  - Ensure graceful shutdown works in daemon mode

#### 3. **Pagination for Usage Events** (Medium Priority)
- **Status**: Only fetches first 100 events (pageSize: 100)
- **Location**: `internal/api/invoice.go:getFilteredUsageEvents`
- **Issue**: HAR file shows `totalUsageEventsCount: 808` but only 100 are fetched
- **Requirements**:
  - Loop through pages until all events are fetched
  - Handle `totalUsageEventsCount` from API response
  - Merge events from all pages
- **Implementation**:
  - Add pagination loop in `getFilteredUsageEvents`
  - Track `page` and `pageSize` parameters
  - Continue fetching until `len(usageEventsDisplay) < pageSize` or all events retrieved

#### 4. **BYOK Cost Comparison Display** (Medium Priority)
- **Status**: Framework exists but not fully integrated
- **Location**: `internal/costs/calculator.go` exists, but `costs` command doesn't show detailed comparison
- **Requirements**:
  - Parse token usage from invoice items or usage events
  - Calculate BYOK costs per model
  - Show comparison table (Cursor vs BYOK by provider)
  - Display potential savings
- **Implementation**:
  - Enhance `runCosts` in `cmd/cursor-monitor/main.go`
  - Extract token usage from usage events or invoice items
  - Use `costs.CompareCosts` function
  - Format and display comparison results

#### 5. **LaunchAgent/LaunchDaemon Support** (Low Priority)
- **Status**: Not implemented
- **Requirements**:
  - Create macOS LaunchAgent plist file
  - Install/uninstall commands
  - Auto-start on login
  - Proper logging to file
- **Implementation**:
  - Add `install` and `uninstall` commands
  - Generate LaunchAgent plist template
  - Install to `~/Library/LaunchAgents/`
  - Use `launchctl` for management

#### 6. **Error Recovery & Retry Logic** (Medium Priority)
- **Status**: Basic error handling exists, but no retry logic
- **Requirements**:
  - Exponential backoff for API failures
  - Retry failed polls
  - Handle network timeouts gracefully
  - Log retry attempts
- **Implementation**:
  - Add retry logic to API client
  - Implement exponential backoff
  - Track consecutive failures
  - Alert if service is down for extended period

#### 7. **Data Validation & Sanity Checks** (Low Priority)
- **Status**: Basic validation exists
- **Requirements**:
  - Validate cost calculations match invoice totals
  - Check for data inconsistencies
  - Warn if events don't match invoice items
  - Detect and report anomalies
- **Implementation**:
  - Add validation functions
  - Cross-check event totals vs invoice totals
  - Log warnings for discrepancies

## Implementation Priority

### Phase 1: Critical Missing Features
1. **Stop Command** - Required for basic service management
2. **Daemon Mode** - Required for background operation (currently needs `&`)

### Phase 2: Data Completeness
3. **Pagination** - Ensure all usage events are fetched (currently only 100/808)

### Phase 3: Enhanced Features
4. **BYOK Cost Comparison** - Complete the cost analysis feature
5. **Error Recovery** - Improve reliability

### Phase 4: Polish & Automation
6. **LaunchAgent Support** - Auto-start on login
7. **Data Validation** - Ensure data integrity

## Code Locations

### Commands (cmd/cursor-monitor/main.go)
- `runStart` - Lines 97-134 (needs daemon implementation)
- `runStatus` - Lines 137-184 ✅
- `runSummary` - Lines 186-343 ✅
- `runCosts` - Lines 345-380 (needs BYOK integration)
- `runRefresh` - Lines 382-405 ✅
- `runImport` - Lines 407-441 ✅
- **Missing**: `runStop` - Needs to be created

### API Client (internal/api/)
- `getFilteredUsageEvents` - Needs pagination (invoice.go)
- `GetMonthlyInvoiceWithCycle` - ✅ Complete

### Costs (internal/costs/)
- `calculator.go` - ✅ Framework complete, needs integration

### Monitor (internal/monitor/)
- `monitor.go` - ✅ Core logic complete
- `historical.go` - ✅ Historical fetching complete

## Notes

- The `--daemon` flag currently only logs a message but doesn't actually daemonize
- The `stop` command is mentioned in README but doesn't exist
- Pagination is needed to fetch all 808 events (currently only 100)
- BYOK cost comparison framework exists but isn't displayed in `costs` command
- All core functionality works, but service management features are missing
