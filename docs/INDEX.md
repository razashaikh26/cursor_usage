# Documentation Index

This directory contains all documentation for the Cursor Usage Monitor application.

## Quick Start

- **README.md** (project root) - Start here! Overview of what the application does and how to use it.

## Analysis & Research

- **BYOK_ANALYSIS.md** - Comprehensive analysis of BYOK (Bring Your Own Key) vs Cursor billing. **Conclusion: Do NOT use BYOK** - Cursor is 2-3x cheaper.

## Development & Planning

- **../plan/IMPLEMENTATION_PLAN.md** - Detailed implementation plan, feature status, and future enhancements.

## Testing

- **TESTING.md** - Testing guide, test structure, and integration testing checklist.

## History

- **CHANGELOG.md** - Complete change log of all features and enhancements.
- **COMPLETION_SUMMARY.md** - Summary of implementation completion status.

## Documentation Structure

```
cursor_usage/
├── README.md                    # Main documentation (what/how)
├── docs/
│   ├── INDEX.md                # This file
│   ├── BYOK_ANALYSIS.md        # BYOK cost analysis
│   ├── TESTING.md              # Testing guide
│   ├── CHANGELOG.md            # Change log
│   └── COMPLETION_SUMMARY.md   # Implementation summary
└── plan/
    └── IMPLEMENTATION_PLAN.md   # Implementation plan & status
```

## Key Findings

### BYOK Analysis
After comprehensive testing, we determined that:
- **Cursor billing is 2-3x cheaper** than direct API access
- Using BYOK would cost significantly more for the same usage
- Cursor's bulk pricing and cache optimization provide major cost savings

See `BYOK_ANALYSIS.md` for complete details and test data.

### Application Purpose
This application:
- Monitors Cursor Pro+ usage via internal APIs
- Stores historical data in local SQLite database
- Sends macOS notifications for usage thresholds
- Provides detailed analytics and cost breakdowns
- Runs as background daemon process

See `README.md` for complete usage instructions.
