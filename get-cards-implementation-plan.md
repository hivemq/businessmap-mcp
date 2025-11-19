# Implementation Plan: get_cards_with_retry Tool

## Overview

Add support for querying multiple cards using filter criteria (`board_ids`, `lane_ids`, `workflow_ids`, `card_ids`) with exponential backoff retry logic.

## Current State Analysis

### Existing Tools
1. `read_card` (main.go:72-97) - Reads a single card by `card_id`
2. `read_card_with_retry` (main.go:130-206) - Same as above but with retry logic
3. `add_card_comment` (main.go:99-128) - Adds a comment to a card

### Current Implementation
- Both read tools use `GET /api/v2/cards/{card_id}` endpoint
- They require a specific `card_id` parameter
- They fetch a single card with its comments and subtasks

## The getCards Endpoint

The `GET /api/v2/cards` endpoint (without a card_id) is a **different endpoint** that:
- Returns **multiple cards** based on filter criteria
- Accepts query parameters: `board_ids`, `lane_ids`, `workflow_ids`, and `card_ids` (arrays)
- These parameters are **mutually exclusive filters** - you use one OR another, not all together
- Returns a list of cards matching the filter criteria

## Decision: Create One New Tool

**Create `get_cards_with_retry`** because:
1. **Different API Endpoint**: `GET /api/v2/cards` vs `GET /api/v2/cards/{card_id}`
2. **Different Purpose**: Bulk querying vs single card retrieval
3. **Different Response Structure**: Array of cards vs single card object
4. **Consistency**: Maintains pattern with existing `read_card_with_retry` tool
5. **No breaking changes**: Existing tools remain stable

## New Tool Specification

### Tool: `get_cards_with_retry`

**Purpose**: Query multiple cards using filter criteria with exponential backoff retry logic

**Parameters**:

Filter Parameters (at least ONE required):
- `board_ids` (optional, array of integers) - Filter by board IDs
- `lane_ids` (optional, array of integers) - Filter by lane IDs
- `workflow_ids` (optional, array of integers) - Filter by workflow IDs
- `card_ids` (optional, array of integers) - Filter by specific card IDs

Retry Configuration Parameters (all optional):
- `max_attempts` (default: 10) - Upper bound attempts per endpoint
- `initial_delay_ms` (default: 5000) - Initial backoff in milliseconds
- `max_delay_ms` (default: 300000) - Max single delay in milliseconds (5 min)
- `multiplier` (default: 2.0) - Exponential growth factor
- `respect_retry_after` (default: true) - Honor server Retry-After header
- `total_wait_cap_ms` (default: 1200000) - Global time cap in milliseconds (20 min)
- `fail_on_partial` (default: false) - Abort when secondary endpoints fail

**Validation**:
- At least one of: board_ids, lane_ids, workflow_ids, or card_ids must be provided
- Parameters are arrays (can specify multiple IDs in each)
- Example: board_ids=[1,2,3] returns cards from boards 1, 2, and 3

**Returns**: Structured envelope with retry metadata + array of cards

**Response Structure**:
```json
{
  "filter_used": "board_ids",
  "filter_values": [1, 2, 3],
  "attempts": {"cards": 2},
  "wait_seconds": 5.2,
  "rate_limit_hits": 1,
  "completed": {"cards": true},
  "partial_error": {},
  "cards": [
    {
      "card_id": 123,
      "title": "Sample Card",
      "description": "Card description",
      "board_id": 1,
      "lane_id": 5,
      "workflow_id": 10
    }
  ]
}
```

## Implementation Steps

### Step 1: Update `internal/kanbanize/types.go`

Add the following types:

1. **GetCardsRequest** - Query parameters for filtering
```go
type GetCardsRequest struct {
    BoardIDs    []int `json:"board_ids,omitempty"`
    LaneIDs     []int `json:"lane_ids,omitempty"`
    WorkflowIDs []int `json:"workflow_ids,omitempty"`
    CardIDs     []int `json:"card_ids,omitempty"`
}
```

2. **CardSummary** - Lighter card data structure
```go
type CardSummary struct {
    CardID      int    `json:"card_id"`
    Title       string `json:"title"`
    Description string `json:"description"`
    BoardID     int    `json:"board_id"`
    LaneID      int    `json:"lane_id"`
    WorkflowID  int    `json:"workflow_id"`
    // Add other relevant fields from API response
}
```

3. **GetCardsResponse** - API response structure
```go
type GetCardsResponse struct {
    Data []CardSummary `json:"data"`
}
```

4. **GetCardsWithRetryResponse** - Envelope with metadata
```go
type GetCardsWithRetryResponse struct {
    FilterUsed    string            `json:"filter_used"`
    FilterValues  []int             `json:"filter_values"`
    Attempts      map[string]int    `json:"attempts"`
    WaitSeconds   float64           `json:"wait_seconds"`
    RateLimitHits int               `json:"rate_limit_hits"`
    Completed     map[string]bool   `json:"completed"`
    PartialError  map[string]string `json:"partial_error,omitempty"`
    Cards         []CardSummary     `json:"cards"`
}
```

### Step 2: Update `internal/kanbanize/client.go`

Add two methods:

1. **getCards** (private helper)
```go
func (c *Client) getCards(filter GetCardsRequest) ([]CardSummary, error) {
    // Build query parameters from filter
    // Make API request to GET /api/v2/cards
    // Parse response
    // Return cards
}
```

2. **GetCardsWithRetry** (public method)
```go
func (c *Client) GetCardsWithRetry(
    ctx context.Context,
    filter GetCardsRequest,
    retryConfig RetryConfig,
    failOnPartial bool,
) (*GetCardsWithRetryResponse, error) {
    // Validate at least one filter is provided
    // Use retry logic from existing implementation
    // Track attempts, wait time, rate limit hits
    // Return structured response
}
```

### Step 3: Update `internal/kanbanize/retry.go` (if needed)

Minor adjustments to support array responses:
- Ensure retry logic works with the new getCards method
- May not need changes if retry logic is generic enough

### Step 4: Update `main.go`

Register the new tool:

```go
getCardsWithRetryTool := mcp.NewTool("get_cards_with_retry",
    mcp.WithDescription("Query multiple cards using filter criteria with exponential backoff retry logic"),
    // Add parameters for board_ids, lane_ids, workflow_ids, card_ids (arrays)
    // Add retry configuration parameters
)

mcpServer.AddTool(getCardsWithRetryTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    // Parse array parameters
    // Validate at least one filter is provided
    // Build filter request
    // Build retry config
    // Call client.GetCardsWithRetry()
    // Return JSON response
})
```

### Step 5: Add tests in `internal/kanbanize/client_test.go`

Test cases:
1. Query parameter construction for arrays
2. Response parsing
3. Retry logic with rate limiting
4. Error handling when no filters provided
5. Partial failure scenarios

## File Changes Summary

1. **internal/kanbanize/types.go** - Add 4 new types (~80 lines)
2. **internal/kanbanize/client.go** - Add 2 new methods (~120 lines)
3. **internal/kanbanize/retry.go** - Minor adjustments if needed (~20 lines)
4. **main.go** - Add 1 new tool registration (~70 lines)
5. **internal/kanbanize/client_test.go** - Add tests (~100 lines)

**Total**: ~390 lines of new code

## Design Decisions

### Parameter Validation
- At least one of: board_ids, lane_ids, workflow_ids, or card_ids must be provided
- Parameters are arrays to allow multiple IDs
- Empty arrays are treated as not provided

### Keep Existing Tools Unchanged
- `read_card` and `read_card_with_retry` remain focused on single-card retrieval
- They continue to fetch full details including comments and subtasks
- No breaking changes to existing functionality

### Response Structure
- `get_cards_with_retry` returns lighter card data (no comments/subtasks by default)
- Users can then use `read_card_with_retry` on specific card IDs if they need full details
- This follows the common API pattern: list → detail

### Tool Set After Implementation
1. `read_card` - Single card, no retry
2. `read_card_with_retry` - Single card, with retry (existing)
3. `get_cards_with_retry` - Multiple cards, with retry (new)
4. `add_card_comment` - Add comment (existing)

## Benefits

✅ **Separation of concerns**: Different endpoints serve different purposes
✅ **No breaking changes**: Existing tools remain stable
✅ **Clear API contract**: Users know `read_card` = single card, `get_cards` = bulk query
✅ **Flexibility**: Users can combine both (query first, then get details)
✅ **Follows REST conventions**: Collection endpoint vs resource endpoint
✅ **Consistent retry support**: All data-fetching tools have retry logic
