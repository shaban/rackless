#!/bin/bash

# MC-SoFX Controller Development Environment
# This script starts both the CSS watcher and the Go server for development

set -e

echo "🎛️  Starting MC-SoFX Controller Development Environment"
echo "─────────────────────────────────────────────────────────"

# Check if server is already running on port 8080
if lsof -Pi :8080 -sTCP:LISTEN -t >/dev/null 2>&1 ; then
    echo "⚠️  Port 8080 is already in use. Please stop the existing server first."
    exit 1
fi

# Function to handle cleanup on exit
cleanup() {
    echo ""
    echo "🛑 Stopping development environment..."
    # Kill all background jobs
    jobs -p | xargs -r kill
    wait
    echo "✅ Development environment stopped"
    exit 0
}

# Set up trap to handle Ctrl+C
trap cleanup SIGINT SIGTERM

echo "📦 Building initial CSS..."
make css

echo "👀 Starting CSS watcher..."
make css-watch &
CSS_PID=$!

# Give CSS watcher a moment to start
sleep 2

echo "🚀 Starting Go server..."
make start-server &
SERVER_PID=$!

echo ""
echo "✅ Development environment is running!"
echo "   🌐 Web interface: http://localhost:8080"
echo "   👀 CSS watcher: monitoring frontend/src/input.css"
echo "   🎛️  Audio controls: MC-SoFX Controller is ready"
echo ""
echo "Press Ctrl+C to stop all services"

# Wait for background processes
wait
