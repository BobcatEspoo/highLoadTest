# High Load Test - Automated Browser Testing Platform

Automated system for creating and managing browser testing instances on vast.ai with Playwright automation.

## Features

- 🚀 Automated vast.ai GPU instance creation and management
- 🎭 Playwright-based browser automation
- 📸 Periodic screenshot capture (every 15 seconds)
- 💾 Automatic upload to remote storage (SFTP with fallback)
- 🔄 Parallel test execution across multiple instances
- ✅ Instance verification filtering
- 📊 Comprehensive logging and monitoring

## Quick Start

### Create and run test pool

```bash
cd createInstance
go run main.go --count=5 --max-price=0.50 --wait=5 --start-tests
```

This command will:
1. Create 5 vast.ai GPU instances (max $0.50/hour each)
2. Wait 5 minutes for initialization
3. Deploy and start Playwright tests on each ready instance
4. Capture screenshots every 15 seconds
5. Upload results to storage@kryuchenko.org

## Command Line Options

### Instance Creation Options

| Flag | Default | Description |
|------|---------|-------------|
| `--count` | 1 | Number of instances to create |
| `--max-price` | 0.50 | Maximum price per hour in USD |
| `--wait` | 5 | Minutes to wait for instances to be ready |
| `--start-tests` | false | Automatically start tests after creation |
| `--verified` | false | Use only verified instances (more reliable) |

### Examples

**Create verified instances only (recommended for production):**
```bash
go run main.go --count=10 --max-price=1.00 --wait=7 --verified --start-tests
```

**Create pool without starting tests:**
```bash
go run main.go --count=5 --max-price=0.50 --wait=5
```

**Budget-friendly setup:**
```bash
go run main.go --count=3 --max-price=0.30 --wait=6 --start-tests
```

## Instance Verification

- **Verified instances** (`--verified` flag): More reliable, professionally managed, but may be more expensive
- **All instances** (default): Includes both verified and unverified, more options but potentially less reliable

## Test Execution Flow

1. **Pool Creation**: Creates specified number of vast.ai instances
2. **Initialization Wait**: Waits for instances to boot and become SSH-ready
3. **Status Check**: Verifies which instances are ready
4. **Test Deployment**: Downloads test binary and dependencies
5. **Playwright Execution**: 
   - Opens Chrome browser via Playwright
   - Navigates to target URL
   - Handles GDPR consent
   - Takes screenshots every 15 seconds
   - Runs for 5 minutes per session
6. **Storage**: Saves artifacts to remote storage with unique session IDs

## Storage

Results are automatically uploaded to:
- **Host**: storage@kryuchenko.org
- **Path**: ~/files/session_YYYY-MM-DD_HH-MM-SS_instINSTANCE_ID_RANDOM/
- **Method**: SSHFS mount with lftp fallback

## Build

Build Linux binary for deployment:
```bash
make build-linux
```

## Requirements

- Go 1.21+
- vast.ai API key (set as VASTAI_API_KEY environment variable)
- SSH key configured for vast.ai instances

## Architecture

```
createInstance/
├── main.go           # Pool creation and management
└── ssh_*.json        # SSH connection cache

main.go               # Playwright test runner
start.sh             # Instance setup script
highLoadTest         # Compiled Linux binary
```

## Monitoring

The system provides detailed logging including:
- Instance creation status
- SSH connection attempts
- Test deployment progress
- Success/failure statistics
- Real-time progress indicators

## Troubleshooting

- **429 Too Many Requests**: Rate limiting is built-in, but reduce `--count` if needed
- **SSH Connection Failed**: Increase `--wait` time for slower instances
- **Unverified Instance Issues**: Use `--verified` flag for more reliable instances

## License

MIT