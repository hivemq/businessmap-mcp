#!/usr/bin/env python3
"""
Interactive test client for Businessmap MCP Server
Usage: python3 test_interactive.py <card_id>
"""

import json
import subprocess
import sys
import time

def send_mcp_request(process, request):
    """Send a JSON-RPC request to the MCP server"""
    request_json = json.dumps(request)
    print(f"→ Sending: {request_json}")
    
    process.stdin.write(request_json + '\n')
    process.stdin.flush()
    
    # Give server time to process
    time.sleep(0.1)
    
    # Try to read response (non-blocking)
    try:
        response = process.stdout.readline()
        if response:
            response_data = json.loads(response.strip())
            print(f"← Received: {json.dumps(response_data, indent=2)}")
            return response_data
    except json.JSONDecodeError as e:
        print(f"← Error parsing response: {e}")
    except Exception as e:
        print(f"← Error reading response: {e}")
    
    return None

def main():
    if len(sys.argv) != 2:
        print("Usage: python3 test_interactive.py <card_id>")
        print("Example: python3 test_interactive.py 12345")
        sys.exit(1)
    
    card_id = sys.argv[1]
    print(f"Testing Businessmap MCP Server with card ID: {card_id}")
    print("=" * 50)
    
    # Start the MCP server process
    try:
        process = subprocess.Popen(
            ['./businessmap-mcp'],
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
            bufsize=0
        )
        
        print("Started MCP server process")
        
        # Initialize the session
        init_request = {
            "jsonrpc": "2.0",
            "id": 1,
            "method": "initialize",
            "params": {
                "protocolVersion": "2025-06-18",
                "capabilities": {"tools": {}},
                "clientInfo": {"name": "test-client", "version": "1.0.0"}
            }
        }
        send_mcp_request(process, init_request)
        
        # Send initialized notification
        initialized_notification = {
            "jsonrpc": "2.0",
            "method": "notifications/initialized"
        }
        send_mcp_request(process, initialized_notification)
        
        # List available tools
        list_tools_request = {
            "jsonrpc": "2.0",
            "id": 2,
            "method": "tools/list"
        }
        send_mcp_request(process, list_tools_request)
        
        # Call read_card tool
        call_tool_request = {
            "jsonrpc": "2.0",
            "id": 3,
            "method": "tools/call",
            "params": {
                "name": "read_card",
                "arguments": {
                    "card_id": card_id
                }
            }
        }
        send_mcp_request(process, call_tool_request)
        
        # Wait for any remaining output
        time.sleep(1)
        
        # Check for any errors
        stderr_output = process.stderr.read()
        if stderr_output:
            print(f"Server stderr: {stderr_output}")
        
        process.terminate()
        process.wait()
        
    except Exception as e:
        print(f"Error running test: {e}")
        sys.exit(1)

if __name__ == "__main__":
    main()