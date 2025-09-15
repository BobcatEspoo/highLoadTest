#!/bin/bash

# Portal configuration setup script for vast.ai instances

echo "Setting up portal configuration..."

# Check if PORTAL_CONFIG environment variable exists
if [ -z "$PORTAL_CONFIG" ]; then
    echo "Warning: PORTAL_CONFIG environment variable not found"
    exit 1
fi

# Create /etc/portal.yaml from PORTAL_CONFIG
echo "Converting PORTAL_CONFIG to /etc/portal.yaml..."
echo "$PORTAL_CONFIG" | tr '|' '\n' > /etc/portal.yaml

echo "Portal configuration written to /etc/portal.yaml:"
cat /etc/portal.yaml

# Restart portal-related services if they exist
echo "Checking for portal services..."
if systemctl list-units --type=service | grep -q portal; then
    echo "Restarting portal services..."
    systemctl restart portal* 2>/dev/null || true
fi

# Restart caddy if it exists
if systemctl list-units --type=service | grep -q caddy; then
    echo "Restarting caddy service..."
    systemctl restart caddy 2>/dev/null || true
fi

echo "Portal setup complete!"