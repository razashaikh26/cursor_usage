# Cursor Usage Monitor

A local-only Go service that monitors Cursor Pro+ usage via internal APIs, stores metrics in SQLite, and sends native macOS notifications when usage exceeds thresholds or switches from Included to On-Demand billing.

## What This Application Does

This application provides **local monitoring and alerting** for your Cursor IDE usage. It:

1. **Monitors Usage in Real-Time**: Polls Cursor's internal API every 15 minutes to track your usage
2. **Stores Historical Data**: Saves all usage metrics, events, and costs in a local SQLite database
3. **Sends Alerts**: Notifies you via macOS notifications when:
   - Usage exceeds 75%, 90%, or 100% of included credits
   - Billing switches from "Included" to "On-Demand"
4. **Provides Analytics**: Shows detailed summaries, cost breakdowns, and usage statistics
5. **Runs in Background**: Can run as a daemon process, detached from your terminal

## How It Works

### Architecture Overview

```
┌─────────────────┐
│  Cursor IDE     │
│  (Local DB)     │
└────────┬────────┘
         │ JWT Token Extraction
         ▼
┌─────────────────┐
│  cursor-monitor │
│  (This App)     │
└────────┬────────┘
         │ API Calls
         ▼
┌─────────────────┐      ┌──────────────┐
│  Cursor API     │◄─────┤  SQLite DB   │
│  (Internal)     │      │  (Local)     │
└─────────────────┘      └──────────────┘
         │                       │
         │                       ▼
         │              ┌──────────────┐
         └──────────────►  macOS       │
                        │  Notifications│
                        └──────────────┘
```

### Detailed Workflow

1. **Token Extraction** (`internal/auth/auth.go`)
   - Reads JWT session token from Cursor's local SQLite database
   - Location: `~/Library/Application Support/Cursor/User/globalStorage/state.vscdb`
   - Extracts `cursor.sessionToken` from the key-value store
   - Uses this token to authenticate API requests

2. **API Polling** (`internal/monitor/monitor.go`)
   - Every 15 minutes (configurable), makes authenticated requests to:
     - `/api/dashboard/get-usage` - Current usage snapshot
     - `/api/dashboard/get-monthly-invoice` - Invoice items and billing cycle
     - `/api/dashboard/get-filtered-usage-events` - Detailed usage events
   - Handles pagination automatically to fetch all events
   - Parses JSON responses into structured Go types

3. **Data Storage** (`internal/storage/storage.go`)
   - Stores data in SQLite database (default: `~/.cursor_monitor/metrics.db`)
   - Three main tables:
     - `usage_snapshots` - Periodic usage snapshots
     - `invoice_items` - Billing items from invoices
     - `usage_events` - Individual API call events with token breakdowns
   - Uses UPSERT logic to update existing events with latest cost data

4. **Alert Detection** (`internal/alerts/alerts.go`)
   - Compares current usage against configured thresholds (75%, 90%, 100%)
   - Detects transitions from "Included" to "On-Demand" billing
   - Tracks alert history to prevent duplicate notifications

5. **Notifications** (`internal/alerts/alerts.go`)
   - Uses macOS `osascript` to send native notifications
   - Shows usage percentage, request counts, and on-demand spending
   - Plays system sound (configurable)

6. **Daemon Mode** (`cmd/cursor-monitor/main.go`)
   - Double-fork daemonization for proper background operation
   - Creates PID file for process management
   - Handles graceful shutdown with signal handling

### Data Flow Example

```
1. Poll starts
   ↓
2. Extract JWT token from Cursor's DB
   ↓
3. Call /api/dashboard/get-usage
   Response: {"premiumRequestsUsed": 250, "premiumRequestsLimit": 500, ...}
   ↓
4. Call /api/dashboard/get-filtered-usage-events (with pagination)
   Response: [{"eventDate": "...", "model": "claude-4.5-sonnet", "totalTokens": 50000, ...}, ...]
   ↓
5. Store snapshot and events in SQLite
   ↓
6. Calculate usage percentage: 250/500 = 50%
   ↓
7. Check alert thresholds (50% < 75%, no alert)
   ↓
8. Wait 15 minutes, repeat
```

## Features

- **Real-time Monitoring**: Polls Cursor's internal API every 15 minutes
- **Usage Alerts**: macOS notifications when usage exceeds 75%, 90%, or 100%
- **Critical Alerts**: Immediate notification when billing switches from Included to On-Demand
- **Historical Tracking**: Stores all usage metrics in SQLite for analysis
- **Cost Analysis**: View detailed cost breakdowns and usage statistics
- **Single Binary**: Zero runtime dependencies - just copy and run
- **Background Operation**: Run as daemon process, detached from terminal

## Installation

1. Install Go 1.22 or later
2. Build the binary:
   ```bash
   make build
   ```
3. Install to PATH (optional):
   ```bash
   make install
   ```

## Configuration

Copy `config.yaml.example` to `~/.cursor_monitor/config.yaml` and customize:

```yaml
polling:
  interval_minutes: 15
  
alerts:
  thresholds: [75, 90, 100]
  on_demand_critical: true
  sound: "default"
  
database:
  path: "~/.cursor_monitor/metrics.db"
  retention_days: 90
  
byok:
  enabled: false
  show_comparison: true

plan:
  included_usage_usd: 63.60
  plan_name: "Pro+"
```

**Important:** The config file must be at `~/.cursor_monitor/config.yaml` (the default location) or you must pass `--config` flag to all commands.

## Usage

```bash
# Start the monitoring daemon (foreground)
cursor-monitor start

# Start as background daemon (detaches from terminal, writes PID file)
cursor-monitor start --daemon

# Check current usage status
cursor-monitor status

# View billing cycle summary with invoice items and usage events
cursor-monitor summary

# Show cost analysis (add --byok flag for BYOK comparison)
cursor-monitor costs
cursor-monitor costs --byok

# Manually trigger a poll now (useful for testing)
cursor-monitor refresh

# Import usage events from CSV file
cursor-monitor import usage_events.csv

# Stop background daemon (sends SIGTERM, cleans up PID file)
cursor-monitor stop
```

### Command Details

- **start**: Begins monitoring with configurable polling interval. Use `--daemon` flag to run in background.
- **status**: Shows current usage percentage, request counts, and on-demand spending.
- **summary**: Displays comprehensive billing cycle information including invoice items and usage event counts with 24-hour time format.
- **costs**: Shows cost analysis. Use `--byok` flag to compare Cursor costs with direct API pricing (see `docs/BYOK_ANALYSIS.md` for why this is not recommended).
- **refresh**: Immediately triggers a single poll without starting the full daemon.
- **import**: Imports historical usage events from a CSV file (useful for backfilling data).
- **stop**: Gracefully stops a running daemon process by reading PID file and sending termination signal.

## Technical Details

### API Endpoints Used

The application accesses Cursor's internal API endpoints (not publicly documented):

- `POST /api/dashboard/get-usage` - Current usage snapshot
- `POST /api/dashboard/get-monthly-invoice` - Invoice items for billing cycle
- `POST /api/dashboard/get-filtered-usage-events` - Detailed usage events (paginated)

All requests require authentication via JWT session token extracted from Cursor's local database.

### Database Schema

**usage_snapshots:**
- Periodic snapshots of usage metrics
- Includes: request counts, usage percentage, on-demand spending, billing cycle info

**invoice_items:**
- Aggregated billing items from invoices
- Includes: model name, request count, cost, billing cycle

**usage_events:**
- Individual API call events
- Includes: timestamp, model, token breakdowns (input, output, cache), cost, event type

### Data Retention

By default, data is retained for 90 days. This can be configured in `config.yaml`:

```yaml
database:
  retention_days: 90
```

## Requirements

- macOS (for native notifications)
- Cursor IDE installed and logged in
- Go 1.22+ (for building)

## Documentation

- **`docs/BYOK_ANALYSIS.md`** - Comprehensive analysis of BYOK vs Cursor billing (conclusion: don't use BYOK)
- **`docs/TESTING.md`** - Testing guide and test structure
- **`docs/CHANGELOG.md`** - Complete change log
- **`docs/COMPLETION_SUMMARY.md`** - Implementation completion summary
- **`plan/IMPLEMENTATION_PLAN.md`** - Detailed implementation plan and status

## License

MIT
