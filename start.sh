#!/bin/bash
# Fix locale warnings
export LANG=en_US.UTF-8
export LC_ALL=en_US.UTF-8
export LANGUAGE=en_US.UTF-8

# Get instance ID from argument (passed from createInstance)
INSTANCE_ID=${1:-"unknown"}

# Install required packages
sudo apt update
sudo apt install -y google-chrome-stable sshfs lftp

# Use the Linux binary we built
chmod +x ./highLoadTest
./highLoadTest -instance="$INSTANCE_ID"