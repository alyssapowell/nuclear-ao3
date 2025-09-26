#!/bin/bash

# Nuclear AO3 Auth Service - Stop Development Server

echo "🛑 Stopping Nuclear AO3 Auth Service..."

# Find and kill auth-service processes
PIDS=$(pgrep -f "auth-service" || true)

if [ -z "$PIDS" ]; then
    echo "ℹ️  No auth-service processes found running"
else
    echo "🔍 Found auth-service processes: $PIDS"
    pkill -f auth-service
    echo "✅ Auth service stopped"
fi

# Check if any are still running
sleep 1
REMAINING=$(pgrep -f "auth-service" || true)
if [ ! -z "$REMAINING" ]; then
    echo "⚠️  Some processes still running, forcing kill..."
    pkill -9 -f auth-service
fi

echo "🏁 All auth-service processes stopped"