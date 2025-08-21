#!/bin/bash

# Test script for Businessmap MCP Server
# Usage: ./tests/test_mcp.sh <card_id> [comment_text]

if [ $# -eq 0 ]; then
    echo "Usage: $0 <card_id_or_url> [comment_text]"
    echo "Example: $0 12345"
    echo "Example: $0 \"https://yourcompany.kanbanize.com/ctrl_board/123/cards/12345/\""
    echo "Example: $0 12345 \"Test comment from MCP server\""
    echo ""
    echo "Prerequisites:"
    echo "1. Build the server: go build -o businessmap-mcp"
    echo "2. Configure .env with your Businessmap credentials"
    echo "3. Get a valid card ID or URL from your Businessmap board"
    echo ""
    echo "If comment_text is provided, it will test both read_card and add_card_comment tools."
    echo "If only card_id_or_url is provided, it will only test the read_card tool."
    echo "You can use either a card ID (e.g., 12345) or a full URL."
    exit 1
fi

CARD_ID_OR_URL=$1
COMMENT_TEXT=$2

# Check if businessmap-mcp binary exists
if [ ! -f "./businessmap-mcp" ]; then
    echo "Error: businessmap-mcp binary not found. Please run 'go build -o businessmap-mcp' first."
    exit 1
fi

# Check if .env file exists
if [ ! -f ".env" ]; then
    echo "Warning: .env file not found. Make sure to copy .env.example to .env and configure it."
fi

echo "Testing Businessmap MCP Server with card ID/URL: $CARD_ID_OR_URL"
if [ -n "$COMMENT_TEXT" ]; then
    echo "Comment text: $COMMENT_TEXT"
fi
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
    echo "{\"jsonrpc\": \"2.0\", \"id\": 3, \"method\": \"tools/call\", \"params\": {\"name\": \"read_card\", \"arguments\": {\"card_id\": \"$CARD_ID_OR_URL\"}}}"
    
    # If comment text is provided, test the add_card_comment tool
    if [ -n "$COMMENT_TEXT" ]; then
        echo "{\"jsonrpc\": \"2.0\", \"id\": 4, \"method\": \"tools/call\", \"params\": {\"name\": \"add_card_comment\", \"arguments\": {\"card_id\": \"$CARD_ID_OR_URL\", \"comment_text\": \"$COMMENT_TEXT\"}}}"
        
        # Read the card again to see the new comment
        echo "{\"jsonrpc\": \"2.0\", \"id\": 5, \"method\": \"tools/call\", \"params\": {\"name\": \"read_card\", \"arguments\": {\"card_id\": \"$CARD_ID_OR_URL\"}}}"
    fi
    
    # Small delay to let server process
    sleep 1
} | ./businessmap-mcp