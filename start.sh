#!/bin/bash
# Single-command startup script for Pasteguard
# Starts Python screen guard API server, then Go server

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration (can be overridden by env vars)
PY_HOST="${SCREEN_GUARD_API_HOST:-127.0.0.1}"
PY_PORT="${SCREEN_GUARD_API_PORT:-8081}"
GO_HOST="${GO_HOST:-}"
GO_PORT="${GO_PORT:-8080}"
PYTHON_CMD="${PYTHON:-python3}"
GO_CMD="${GO_CMD:-go}"

# Calculate full addresses
PY_URL="http://${PY_HOST}:${PY_PORT}"
SCREEN_GUARD_BASE_URL="${SCREEN_GUARD_BASE_URL:-${PY_URL}}"

# Create logs directory
mkdir -p logs

echo -e "${GREEN}Starting Pasteguard services...${NC}"

# Find Python executable
if ! command -v "$PYTHON_CMD" &> /dev/null; then
    echo -e "${RED}ERROR: $PYTHON_CMD not found. Please install Python 3 or set PYTHON env var.${NC}"
    exit 1
fi

# Check for virtual environment
if [ -d ".venv" ]; then
    echo -e "${YELLOW}Found .venv, activating...${NC}"
    source .venv/bin/activate
    PYTHON_CMD="python"
elif [ -d "venv" ]; then
    echo -e "${YELLOW}Found venv, activating...${NC}"
    source venv/bin/activate
    PYTHON_CMD="python"
fi

# Start Python API server in background
echo -e "${GREEN}Starting Python Screen Guard API server on ${PY_HOST}:${PY_PORT}...${NC}"
# Run api_server.py directly from screen_guard_service directory
cd screen_guard_service
$PYTHON_CMD api_server.py > ../logs/screen_guard_service.log 2>&1 &
PY_PID=$!
cd ..

# Function to cleanup on exit
cleanup() {
    echo -e "\n${YELLOW}Shutting down...${NC}"
    if [ ! -z "$PY_PID" ]; then
        echo -e "${YELLOW}Stopping Python service (PID: $PY_PID)...${NC}"
        kill $PY_PID 2>/dev/null || true
        wait $PY_PID 2>/dev/null || true
    fi
    echo -e "${GREEN}Shutdown complete.${NC}"
    exit 0
}

trap cleanup SIGINT SIGTERM

# Wait for Python API to be ready
echo -e "${YELLOW}Waiting for Python API to be ready...${NC}"
MAX_WAIT=30
WAIT_COUNT=0
while [ $WAIT_COUNT -lt $MAX_WAIT ]; do
    if curl -s -f "${PY_URL}/health" > /dev/null 2>&1; then
        echo -e "${GREEN}Python API is ready!${NC}"
        break
    fi
    if ! kill -0 $PY_PID 2>/dev/null; then
        echo -e "${RED}ERROR: Python API server failed to start. Check logs/screen_guard_service.log${NC}"
        exit 1
    fi
    sleep 1
    WAIT_COUNT=$((WAIT_COUNT + 1))
done

if [ $WAIT_COUNT -ge $MAX_WAIT ]; then
    echo -e "${RED}ERROR: Python API server did not become ready within ${MAX_WAIT} seconds.${NC}"
    kill $PY_PID 2>/dev/null || true
    exit 1
fi

# Export for Go server
export SCREEN_GUARD_BASE_URL

# Start Go server in foreground
echo -e "${GREEN}Starting Go server on :${GO_PORT}...${NC}"
echo -e "${GREEN}Python API: ${PY_URL}${NC}"
echo -e "${GREEN}Go Server: http://localhost:${GO_PORT}${NC}"
echo -e "${YELLOW}Press Ctrl+C to stop all services${NC}\n"

if [ -n "$GO_HOST" ]; then
    $GO_CMD run . serve --addr "${GO_HOST}:${GO_PORT}" 2>&1 | tee logs/go_server.log
else
    $GO_CMD run . serve --addr ":${GO_PORT}" 2>&1 | tee logs/go_server.log
fi

