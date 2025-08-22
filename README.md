# Businessmap MCP Server

![Go Version](https://img.shields.io/badge/Go-1.25+-blue.svg)
![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)
![MCP Protocol](https://img.shields.io/badge/MCP-Compatible-orange.svg)

A Go-based Model Context Protocol (MCP) server that provides comprehensive access to Businessmap (formerly Kanbanize) cards including reading card information and adding comments.

## Features

- **🎯 Two MCP Tools**: 
  - `read_card` - Comprehensive card information retrieval
  - `add_card_comment` - Add comments to existing cards
- **📊 Structured Data**: Returns clean JSON with title, description, subtasks, and comments
- **✍️ Card Interaction**: Add comments to cards directly through the API
- **🔐 Secure Authentication**: API key and base URL configuration via environment variables
- **🔗 Direct API Integration**: Uses official Businessmap API v2 endpoints
- **⚡ Lightweight**: Minimal dependencies and fast response times
- **🚀 Native Performance**: Direct Go binary execution without containers

## Prerequisites

- Go 1.25 or later
- Kanbanize/Businessmap account with API access

## Quick Start

### 1. Configuration

Copy the environment template and configure your Kanbanize credentials:

```bash
cp .env.example .env
```

Edit `.env` and set your values:

```bash
KANBANIZE_API_KEY=your_actual_api_key
KANBANIZE_BASE_URL=https://your-subdomain.kanbanize.com
```

### 2. Getting Your API Key

1. Log into your Kanbanize/Businessmap account
2. Click on your user dropdown menu (top right)
3. Select "API"
4. Copy your API key

### 3. Build and Run

```bash
# Install dependencies
go mod download

# Build the server
go build -o businessmap-mcp

# Run the server
./businessmap-mcp
```

## Claude Code Integration

This MCP server is designed to work seamlessly with Claude Code. Follow these steps to integrate:

### 1. Install and Build

```bash
# Clone the repository
git clone https://github.com/zbigniewzabost-hivemq/businessmap-mcp.git
cd businessmap-mcp

# Install dependencies
go mod download

# Build the server
go build -o businessmap-mcp
```

### 2. Configure Claude Code

Add the MCP server to your Claude Code configuration. Create or edit your Claude Code configuration file:

**For Claude Code CLI** (`~/.claude/mcp_servers.json`):
```json
{
  "mcpServers": {
    "businessmap": {
      "command": "/path/to/businessmap-mcp/businessmap-mcp",
      "env": {
        "KANBANIZE_API_KEY": "your_api_key_here",
        "KANBANIZE_BASE_URL": "https://your-subdomain.kanbanize.com"
      }
    }
  }
}
```

**For Claude Desktop** (configuration file location varies by OS):
```json
{
  "mcpServers": {
    "businessmap": {
      "command": "/absolute/path/to/businessmap-mcp",
      "env": {
        "KANBANIZE_API_KEY": "your_api_key_here", 
        "KANBANIZE_BASE_URL": "https://your-subdomain.kanbanize.com"
      }
    }
  }
}
```

### 3. Environment Variables (Alternative)

Instead of putting credentials in the config file, you can use a `.env` file:

```bash
# In the businessmap-mcp directory
cp .env.example .env
# Edit .env with your credentials
```

Then update the Claude Code config to use the working directory:
```json
{
  "mcpServers": {
    "businessmap": {
      "command": "/path/to/businessmap-mcp/businessmap-mcp",
      "cwd": "/path/to/businessmap-mcp"
    }
  }
}
```

### 4. Restart Claude Code

After updating the configuration, restart Claude Code to load the new MCP server.

### 5. Using the Tools

Once integrated, you can use the tools in Claude Code:

**Reading card information**:
```
Please read the details of Businessmap card 12345
```

**Reading card from URL** (supports various URL formats):
```
Please read the details from https://yourcompany.kanbanize.com/ctrl_board/123/cards/12345
Please read the details from https://yourcompany.kanbanize.com/ctrl_board/123/cards/12345/
Please read the details from https://yourcompany.kanbanize.com/ctrl_board/123/cards/12345/details/
Please read the details from https://yourcompany.kanbanize.com/ctrl_board/123/cards/12345/comments/
```

**Adding comments to cards**:
```
Please add a comment "Work completed successfully" to Businessmap card 12345
```

**Adding comments using URL** (supports various URL formats):
```
Please add a comment "Task completed" to https://yourcompany.kanbanize.com/ctrl_board/123/cards/12345
Please add a comment "Task completed" to https://yourcompany.kanbanize.com/ctrl_board/123/cards/12345/
Please add a comment "Task completed" to https://yourcompany.kanbanize.com/ctrl_board/123/cards/12345/details/
Please add a comment "Task completed" to https://yourcompany.kanbanize.com/ctrl_board/123/cards/12345/comments/
```

Claude will automatically use the MCP server to fetch comprehensive card information or add comments as requested.

## Usage

The MCP server communicates via stdin/stdout using the JSON-RPC protocol. It provides two tools:

### `read_card`

**Description**: Read comprehensive card information including title, description, subtasks, and comments

**Parameters**:
- `card_id` (string, required): The ID of the Kanbanize card to read or full card URL

**Example Response**: 
```json
{
  "title": "Card Title",
  "description": "Card description text",
  "subtasks": [
    {
      "id": "123",
      "title": "Subtask title",
      "description": "Subtask description",
      "completed": false
    }
  ],
  "comments": [
    {
      "id": "456",
      "text": "Comment text",
      "author": "Author Name",
      "created_at": "2023-12-01T10:00:00Z"
    }
  ]
}
```

### `add_card_comment`

**Description**: Add a comment to a card

**Parameters**:
- `card_id` (string, required): The ID of the Kanbanize card to add comment to or full card URL  
- `comment_text` (string, required): The text of the comment to add

**Example Response**: 
```json
"Comment added successfully"
```

## Testing

### Prerequisites for Testing

1. **Kanbanize Account**: Active Kanbanize/Businessmap account with API access
2. **API Key**: Generated from your account settings (User menu → API)
3. **Card ID**: Valid card ID from your Kanbanize board to test with
4. **Environment Setup**: Configured `.env` file with your credentials

### Quick Test

1. **Setup Environment**:
   ```bash
   cp .env.example .env
   # Edit .env with your actual API key and base URL
   ```

2. **Build and Test**:
   ```bash
   # Build the server
   go build -o businessmap-mcp
   
   # Test read_card only (replace 12345 with your actual card ID)
   ./tests/test_mcp.sh 12345
   
   # Test with full URL (various formats supported)
   ./tests/test_mcp.sh "https://yourcompany.kanbanize.com/ctrl_board/123/cards/12345"
   ./tests/test_mcp.sh "https://yourcompany.kanbanize.com/ctrl_board/123/cards/12345/"
   ./tests/test_mcp.sh "https://yourcompany.kanbanize.com/ctrl_board/123/cards/12345/details/"
   
   # Test both read_card and add_card_comment tools
   ./tests/test_mcp.sh 12345 "Test comment from MCP server"
   ```

### Testing Options

#### Option 1: Shell Script (Recommended)
```bash
# Test read_card only with card ID
./tests/test_mcp.sh YOUR_CARD_ID

# Test read_card with full URL (various formats supported)
./tests/test_mcp.sh "https://yourcompany.kanbanize.com/ctrl_board/123/cards/YOUR_CARD_ID"
./tests/test_mcp.sh "https://yourcompany.kanbanize.com/ctrl_board/123/cards/YOUR_CARD_ID/"
./tests/test_mcp.sh "https://yourcompany.kanbanize.com/ctrl_board/123/cards/YOUR_CARD_ID/details/"
./tests/test_mcp.sh "https://yourcompany.kanbanize.com/ctrl_board/123/cards/YOUR_CARD_ID/comments/"

# Test both tools
./tests/test_mcp.sh YOUR_CARD_ID "Your test comment"
```

#### Option 2: Interactive Python Script
```bash
# Test read_card only with card ID
python3 tests/test_interactive.py YOUR_CARD_ID

# Test read_card with full URL  
python3 tests/test_interactive.py "https://yourcompany.kanbanize.com/ctrl_board/123/cards/YOUR_CARD_ID/"

# Test both tools
python3 tests/test_interactive.py YOUR_CARD_ID "Your test comment"
```

#### Option 3: Manual JSON-RPC Testing
```bash
# Replace YOUR_CARD_ID in the test file (tests both tools)
sed 's/REPLACE_WITH_CARD_ID/YOUR_CARD_ID/g' tests/test_messages.jsonl | ./businessmap-mcp

# Or test with URL (escape slashes properly)
sed 's|REPLACE_WITH_CARD_ID|https://yourcompany.kanbanize.com/ctrl_board/123/cards/YOUR_CARD_ID/|g' tests/test_messages.jsonl | ./businessmap-mcp
```

### Expected Output

The server should return a clean JSON response like:
```json
{
  "title": "Example Card Title",
  "description": "Card description text",
  "subtasks": [
    {
      "id": "123",
      "title": "Subtask title",
      "description": "Optional description",
      "completed": false
    }
  ],
  "comments": [
    {
      "id": "456",
      "text": "Comment text",
      "author": "Author Name",
      "created_at": "2023-12-01T10:00:00Z"
    }
  ]
}
```

## API Integration

The server uses the official Businessmap API v2 endpoints:

**Endpoints**:
```bash
GET  {KANBANIZE_BASE_URL}/api/v2/cards/{card_id}              # Card details
GET  {KANBANIZE_BASE_URL}/api/v2/cards/{card_id}/comments     # Comments
GET  {KANBANIZE_BASE_URL}/api/v2/cards/{card_id}/subtasks     # Subtasks
POST {KANBANIZE_BASE_URL}/api/v2/cards/{card_id}/comments     # Add comment
```

**Authentication**:
```
apikey: {KANBANIZE_API_KEY}
```

## Project Structure

```
businessmap-mcp/
├── README.md                 # This documentation
├── LICENSE                   # Apache License 2.0
├── CONTRIBUTING.md           # Contribution guidelines
├── main.go                   # MCP server entry point
├── go.mod                    # Go module definition
├── go.sum                    # Go dependencies
├── .env.example             # Environment variables template
├── .gitignore               # Git ignore rules
├── internal/
│   ├── config/
│   │   └── config.go        # Configuration management
│   └── kanbanize/
│       ├── client.go        # Kanbanize API client
│       └── types.go         # API response types
└── tests/
    ├── test_mcp.sh          # Shell script testing
    ├── test_interactive.py  # Interactive Python testing
    └── test_messages.jsonl  # Manual JSON-RPC testing
```

## Error Handling

The server provides robust error handling with graceful degradation:

- **🚫 Missing Configuration**: Fails fast on startup if API credentials are missing
- **❌ Invalid Card ID**: Clear error messages for empty or invalid card IDs
- **🌐 API Errors**: Forwards Businessmap API error messages with context
- **📊 Partial Data**: Returns available data with empty arrays for missing comments/subtasks
- **🔗 Network Issues**: Detailed error reporting for connectivity problems

## Security Features

- **🔒 Environment-based Secrets**: API keys stored in environment variables only
- **🚫 No Secret Logging**: Credentials never appear in logs or error messages
- **🔐 HTTPS Only**: All API communications use secure HTTPS connections
- **🛡️ Secure by Default**: No elevated privileges required for execution

## Dependencies

- **[mark3labs/mcp-go](https://github.com/mark3labs/mcp-go)** - Model Context Protocol Go SDK
- **[joho/godotenv](https://github.com/joho/godotenv)** - Environment variable management

## Contributing

Contributions are welcome! Please read [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'feat: add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## Support

- **Issues**: Report bugs or request features via [GitHub Issues](https://github.com/zbigniewzabost-hivemq/businessmap-mcp/issues)
- **Documentation**: Full API documentation available in this README
- **Community**: Contributions and feedback welcome!