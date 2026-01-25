# Cursor Usage Monitor

A local-only Go service that monitors Cursor Pro+ usage via internal APIs, stores metrics in SQLite, and sends native macOS notifications when usage exceeds thresholds or switches from Included to On-Demand billing.

## Features

- **Real-time Monitoring**: Polls Cursor's internal API every 15 minutes
- **Usage Alerts**: macOS notifications when usage exceeds 75%, 90%, or 100%
- **Critical Alerts**: Immediate notification when billing switches from Included to On-Demand
- **Historical Tracking**: Stores all usage metrics in SQLite for analysis
- **BYOK Cost Comparison**: Optional feature to compare costs with/without BYOK
- **Single Binary**: Zero runtime dependencies - just copy and run

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
```

## Usage

```bash
# Start the monitoring daemon (foreground)
cursor-monitor start

# Start as background daemon
cursor-monitor start --daemon

# Check current status
cursor-monitor status

# View billing cycle summary
cursor-monitor summary

# Show BYOK cost comparison
cursor-monitor costs --byok

# Manual refresh (poll now)
cursor-monitor refresh

# Stop daemon
cursor-monitor stop
```

## How It Works

1. **Token Extraction**: Reads JWT token from Cursor's local SQLite database at `~/Library/Application Support/Cursor/User/globalStorage/state.vscdb`
2. **API Polling**: Makes authenticated requests to Cursor's internal API endpoints
3. **Data Storage**: Stores usage snapshots in SQLite database
4. **Alert Detection**: Monitors for threshold crossings and billing type changes
5. **Notifications**: Sends native macOS notifications via `osascript`

## Requirements

- macOS (for native notifications)
- Cursor IDE installed and logged in
- Go 1.22+ (for building)

## License

MIT
