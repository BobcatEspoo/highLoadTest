#!/bin/bash
set -e  # Exit on any error

# Logging function
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a /tmp/test_startup.log
}

log "Starting test setup..."

# Fix locale warnings
export LANG=en_US.UTF-8
export LC_ALL=en_US.UTF-8
export LANGUAGE=en_US.UTF-8

# Get instance ID from argument (passed from createInstance)
INSTANCE_ID=${1:-"unknown"}
log "Instance ID: $INSTANCE_ID"

# Install required packages with detailed logging
log "Updating package lists..."
sudo apt update 2>&1 | tee -a /tmp/test_startup.log

log "Installing Chrome, SSHFS, and LFTP..."
if sudo apt install -y google-chrome-stable sshfs lftp 2>&1 | tee -a /tmp/test_startup.log; then
    log "Dependencies installed successfully"
else
    log "ERROR: Failed to install dependencies"
    exit 1
fi

# Verify Chrome installation
if which google-chrome-stable >/dev/null 2>&1; then
    log "Chrome installed successfully: $(google-chrome-stable --version)"
else
    log "ERROR: Chrome installation failed"
    exit 1
fi

# Enable fuse module for SSHFS
log "Enabling FUSE module..."
if sudo modprobe fuse 2>&1 | tee -a /tmp/test_startup.log; then
    log "FUSE module enabled"
else
    log "WARNING: Failed to enable FUSE module"
fi

# Verify binary exists
if [ ! -f "./highLoadTest" ]; then
    log "ERROR: highLoadTest binary not found"
    exit 1
fi

# Make binary executable
chmod +x ./highLoadTest
log "Binary permissions set"

# Start the test with logging
log "Starting highLoadTest process..."
nohup ./highLoadTest -instance="$INSTANCE_ID" > /tmp/test_output.log 2>&1 &
TEST_PID=$!

log "Test started with PID: $TEST_PID"

# Wait a moment and verify process is still running
sleep 2
if kill -0 $TEST_PID 2>/dev/null; then
    log "Process confirmed running after 2 seconds"
    echo $TEST_PID > /tmp/test_pid
else
    log "ERROR: Process died immediately after start"
    log "Last output:"
    tail -20 /tmp/test_output.log | tee -a /tmp/test_startup.log
    exit 1
fi

log "Setup completed successfully"