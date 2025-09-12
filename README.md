# High Load Test

GPU load testing with vast.ai instances.

## Usage

```bash
# Build binaries for Linux
make build

# Create 50 instances, wait 3 minutes  
go run createInstance/main.go -count=50 -wait=3
```