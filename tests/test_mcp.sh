#!/bin/bash

# Test script for Businessmap MCP Server
# Usage: ./tests/test_mcp.sh <card_id>

if [ $# -eq 0 ]; then
    echo "Usage: $0 <card_id>"
    echo "Example: $0 12345"
    echo ""
    echo "Prerequisites:"
    echo "1. Build the server: go build -o businessmap-mcp"
    echo "2. Configure .env with your Businessmap credentials"
    echo "3. Get a valid card ID from your Businessmap board"
    exit 1
fi

CARD_ID=$1

# Check if businessmap-mcp binary exists
if [ ! -f "./businessmap-mcp" ]; then
    echo "Error: businessmap-mcp binary not found. Please run 'go build -o businessmap-mcp' first."
    exit 1
fi

# Check if .env file exists
if [ ! -f ".env" ]; then
    echo "Warning: .env file not found. Make sure to copy .env.example to .env and configure it."
fi

echo "Testing Businessmap MCP Server with card ID: $CARD_ID"
echo "======================================================="

# Start the MCP server and send test messages
{
    # Initialize the MCP session
    echo '{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {"protocolVersion": "2025-06-18", "capabilities": {"tools": {}}, "clientInfo": {"name": "test-client", "version": "1.0.0"}}}'
    
    # Send initialized notification
    echo '{"jsonrpc": "2.0", "method": "notifications/initialized"}'
    
    # List available tools
    echo '{"jsonrpc": "2.0", "id": 2, "method": "tools/list"}'
    
    # Call the read_card tool
    echo "{\"jsonrpc\": \"2.0\", \"id\": 3, \"method\": \"tools/call\", \"params\": {\"name\": \"read_card\", \"arguments\": {\"card_id\": \"$CARD_ID\"}}}"
    
    # Small delay to let server process
    sleep 1
} | ./businessmap-mcp