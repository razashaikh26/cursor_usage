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
- Summary command with 24-hour time format
- Status command
- Refresh command
- Cost calculation framework (BYOK pricing data - see docs/BYOK_ANALYSIS.md)
- Historical data fetching
- Daemon mode with PID file management
- Stop command for graceful shutdown
- Pagination for complete data fetching
- BYOK cost comparison (deprecated - see analysis)

### ✅ Recently Completed Features

#### 1. **Stop Command** ✅

- **Status**: Implemented
- **Location**: `cmd/cursor-monitor/main.go:runStop`
- **Features**:
  - Reads PID file from `~/.cursor_monitor/cursor-monitor.pid`
  - Sends SIGTERM to running process
  - Verifies process stopped
  - Cleans up PID file
  - Handles stale PID files gracefully

#### 2. **Daemon Mode Implementation** ✅

- **Status**: Implemented
- **Location**: `cmd/cursor-monitor/main.go:daemonize` and `runStart`
- **Features**:
  - Double-fork to detach from terminal
  - Uses `syscall.Setsid` to create new session
  - Redirects stdin/stdout/stderr
  - Writes PID file to `~/.cursor_monitor/cursor-monitor.pid`
  - Checks for existing daemon before starting
  - Handles graceful shutdown with signal handling

#### 3. **Pagination for Usage Events** ✅

- **Status**: Implemented
- **Location**: `internal/api/invoice.go:getFilteredUsageEvents`
- **Features**:
  - Loops through pages until all events are fetched
  - Handles `totalUsageEventsCount` from API response
  - Merges events from all pages
  - Continues fetching until `len(usageEventsDisplay) < pageSize` or all events retrieved
  - Logs progress for debugging

#### 4. **BYOK Cost Comparison** ✅ (Deprecated)

- **Status**: Implemented but deprecated
- **Location**: `cmd/cursor-monitor/main.go:runCosts`
- **Features**:
  - Parses token usage from usage events
  - Calculates BYOK costs per model using `costs.CompareCosts`
  - Shows comparison table (Cursor vs BYOK by provider)
  - Displays potential savings for Anthropic, OpenAI, and Google
  - Can be enabled with `--byok` flag or `byok.show_comparison` config
- **Note**: Analysis shows BYOK is NOT cost-effective. See `docs/BYOK_ANALYSIS.md` for details.

#### 5. **24-Hour Time Format** ✅

- **Status**: Implemented
- **Location**: `cmd/cursor-monitor/main.go:runSummary`
- **Features**:
  - Changed time display from "Jan 2 at 03:04 PM" to "Jan 02 15:04"
  - Uses local timezone for display
  - More compact and internationally friendly format

### ❌ Missing Features (Lower Priority)

#### 6. **LaunchAgent/LaunchDaemon Support** (Low Priority)

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

#### 7. **Error Recovery & Retry Logic** (Medium Priority)

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

#### 8. **Data Validation & Sanity Checks** (Low Priority)

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

#### 9. **User API Key Event Tracking** (Medium Priority)

- **Status**: Partially implemented
- **Issue**: Events with `kind="User API Key"` are not being properly captured
- **Requirements**:
  - Properly identify and store "User API Key" events
  - Display them separately in summary
  - Track costs for BYOK usage (even though not recommended)
- **Implementation**:
  - Update API parsing to handle "User API Key" kind
  - Store with correct type in database
  - Display in summary with appropriate labeling

## Implementation Priority

### Phase 1: Critical Missing Features ✅ COMPLETED

1. ✅ **Stop Command** - Required for basic service management
2. ✅ **Daemon Mode** - Required for background operation

### Phase 2: Data Completeness ✅ COMPLETED

3. ✅ **Pagination** - Ensure all usage events are fetched

### Phase 3: Enhanced Features ✅ COMPLETED

4. ✅ **BYOK Cost Comparison** - Complete the cost analysis feature (deprecated)
2. ✅ **24-Hour Time Format** - Better time display

### Phase 4: Polish & Automation (Future)

6. **User API Key Tracking** - Properly capture BYOK events
2. **Error Recovery** - Improve reliability with retry logic
3. **LaunchAgent Support** - Auto-start on login
4. **Data Validation** - Ensure data integrity

## Code Locations

### Commands (cmd/cursor-monitor/main.go)

- `runStart` - ✅ Complete with daemon mode support
- `runStatus` - ✅ Complete
- `runSummary` - ✅ Complete with 24-hour time format
- `runCosts` - ✅ Complete with BYOK integration (deprecated)
- `runRefresh` - ✅ Complete
- `runImport` - ✅ Complete
- `runStop` - ✅ Complete

### API Client (internal/api/)

- `getFilteredUsageEvents` - ✅ Complete with pagination support
- `GetMonthlyInvoiceWithCycle` - ✅ Complete

### Costs (internal/costs/)

- `calculator.go` - ✅ Framework complete, BYOK analysis documented

### Monitor (internal/monitor/)

- `monitor.go` - ✅ Core logic complete
- `historical.go` - ✅ Historical fetching complete

## Notes

- ✅ All critical features have been implemented
- ✅ Daemon mode fully functional with PID file management
- ✅ Stop command implemented with graceful shutdown
- ✅ Pagination implemented to fetch all usage events
- ✅ BYOK cost comparison implemented but analysis shows it's not cost-effective
- ✅ 24-hour time format implemented for better readability
- ⚠️ "User API Key" events need proper tracking (see Phase 4)
- All core functionality works, including service management features

## Testing

- Unit tests exist for core functionality (api, costs, storage, alerts)
- Integration tests recommended for:
  - Daemon mode start/stop cycle
  - Pagination with mock API responses
  - PID file management edge cases
  - BYOK cost calculation accuracy (for reference)

## Documentation

- README.md updated with all commands
- IMPLEMENTATION_PLAN.md (this file) reflects current status
- BYOK_ANALYSIS.md contains comprehensive cost analysis
- Code comments added for complex logic (pagination, daemon mode)
