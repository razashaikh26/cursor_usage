# Changelog

## [Unreleased] - Implementation Completion

### Added

#### Core Features
- **Main Command Interface** (`cmd/cursor-monitor/main.go`)
  - Complete CLI with all commands: start, status, summary, costs, refresh, import, stop
  - Cobra-based command structure
  - Configuration loading and management

#### Daemon Mode
- **Background Process Support**
  - Double-fork daemonization for proper process detachment
  - PID file management (`~/.cursor_monitor/cursor-monitor.pid`)
  - Graceful shutdown with signal handling (SIGTERM, SIGINT)
  - Automatic cleanup of stale PID files
  - Process existence verification before starting

#### Stop Command
- **Service Management**
  - Reads PID file to find running daemon
  - Sends SIGTERM for graceful shutdown
  - Falls back to SIGKILL if process doesn't terminate
  - Removes PID file after successful stop
  - Handles missing or stale PID files gracefully

#### Pagination Support
- **Complete Data Fetching**
  - Automatic pagination in `getFilteredUsageEvents`
  - Fetches all usage events across multiple pages
  - Handles `totalUsageEventsCount` from API response
  - Continues fetching until all events retrieved
  - Logs progress for debugging

#### BYOK Cost Comparison
- **Enhanced Costs Command**
  - Extracts token usage from usage events
  - Calculates BYOK costs per model (Anthropic, OpenAI, Google)
  - Displays comparison table with potential savings
  - Can be enabled via `--byok` flag or `byok.show_comparison` config
  - Shows breakdown by provider

### Enhanced

#### API Client
- **Pagination Implementation** (`internal/api/invoice.go`)
  - Loops through all pages automatically
  - Handles variable page sizes
  - Merges events from all pages
  - Improved error handling

#### Cost Calculator
- **Integration with Commands** (`internal/costs/calculator.go`)
  - Full integration with costs command
  - Token usage extraction from events
  - Provider-specific cost calculations

### Testing

#### New Test Files
- `internal/api/invoice_pagination_test.go`
  - Tests pagination logic
  - Tests response parsing
  - Edge case handling

- `cmd/cursor-monitor/main_test.go`
  - PID file management tests
  - PID file format validation

#### Test Coverage
- All existing tests continue to pass
- New tests for pagination logic
- New tests for PID file operations

### Documentation

#### Updated Files
- **README.md**
  - Updated usage examples
  - Added command details
  - Enhanced feature descriptions
  - Added pagination and daemon mode information

- **IMPLEMENTATION_PLAN.md**
  - Marked all Phase 1-3 features as completed
  - Updated status of all commands
  - Added testing and documentation notes

#### New Files
- **TESTING.md**
  - Comprehensive testing guide
  - Test structure documentation
  - Integration testing checklist
  - Test data examples

- **CHANGELOG.md** (this file)
  - Complete change log
  - Feature additions and enhancements

### Technical Details

#### Process Management
- Uses `syscall.Setsid` for proper daemonization
- Implements double-fork pattern for Unix systems
- Signal handling with context cancellation
- PID file atomic operations

#### Error Handling
- Graceful handling of missing PID files
- Stale PID file detection and cleanup
- Process existence verification
- Improved API error messages

#### Code Quality
- Comprehensive error handling
- Proper resource cleanup (defer statements)
- Context-based cancellation
- Logging for debugging

### Breaking Changes
None - all changes are additive.

### Migration Notes
- No migration required
- Existing configuration files remain compatible
- Database schema unchanged
- All existing commands continue to work

### Future Enhancements (Not Implemented)
- LaunchAgent/LaunchDaemon support for auto-start
- Enhanced error recovery with retry logic
- Data validation and sanity checks
- Advanced cost analysis features
