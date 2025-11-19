/*
 * Copyright 2018-present HiveMQ and the HiveMQ Community
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/hivemq/businessmap-mcp/internal/config"
	"github.com/hivemq/businessmap-mcp/internal/kanbanize"
)

//go:embed VERSION
var versionFile string

// BuildVersion can be set at build time via ldflags
var BuildVersion = "dev"

// getVersion returns the application version, preferring embedded VERSION file over build version
func getVersion() string {
	if versionFile != "" {
		return strings.TrimSpace(versionFile)
	}
	return BuildVersion
}


func main() {
	var showVersion = flag.Bool("version", false, "show version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("businessmap-mcp version %s\n", getVersion())
		os.Exit(0)
	}

	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	client := kanbanize.NewClient(cfg.KanbanizeBaseURL, cfg.KanbanizeAPIKey)

	mcpServer := server.NewMCPServer("kanbanize-mcp-server", getVersion())

	readCardTool := mcp.NewTool("read_card",
		mcp.WithDescription("Read comprehensive card information including title, description, subtasks, and comments"),
		mcp.WithString("card_id",
			mcp.Required(),
			mcp.Description("The ID of the Kanbanize card to read or full card URL"),
		),
	)

	mcpServer.AddTool(readCardTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		cardID := mcp.ParseString(request, "card_id", "")
		if cardID == "" {
			return mcp.NewToolResultError("card_id parameter is required"), nil
		}

		cardData, err := client.ReadCard(cardID)
		if err != nil {
			return mcp.NewToolResultError("Failed to read card: "+err.Error()), nil
		}

		cardJSON, err := json.Marshal(cardData)
		if err != nil {
			return mcp.NewToolResultError("Failed to serialize card data: "+err.Error()), nil
		}

		return mcp.NewToolResultText(string(cardJSON)), nil
	})

	addCommentTool := mcp.NewTool("add_card_comment",
		mcp.WithDescription("Add a comment to a card"),
		mcp.WithString("card_id",
			mcp.Required(),
			mcp.Description("The ID of the Kanbanize card to add comment to or full card URL"),
		),
		mcp.WithString("comment_text",
			mcp.Required(),
			mcp.Description("The text of the comment to add"),
		),
	)

	mcpServer.AddTool(addCommentTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		cardID := mcp.ParseString(request, "card_id", "")
		if cardID == "" {
			return mcp.NewToolResultError("card_id parameter is required"), nil
		}

		commentText := mcp.ParseString(request, "comment_text", "")
		if commentText == "" {
			return mcp.NewToolResultError("comment_text parameter is required"), nil
		}

		err := client.AddCardComment(cardID, commentText)
		if err != nil {
			return mcp.NewToolResultError("Failed to add comment: "+err.Error()), nil
		}

		return mcp.NewToolResultText("Comment added successfully"), nil
	})

	readCardWithRetryTool := mcp.NewTool("read_card_with_retry",
		mcp.WithDescription("Fetches a card and optionally comments/subtasks using exponential full-jitter backoff with respect for Retry-After headers. Returns structured envelope including attempts, wait time, and partial errors when enabled."),
		mcp.WithString("card_id",
			mcp.Required(),
			mcp.Description("The ID of the Kanbanize card to read or full card URL"),
		),
		mcp.WithNumber("max_attempts",
			mcp.Description("Upper bound attempts per endpoint (default: 10)"),
		),
		mcp.WithNumber("initial_delay_ms",
			mcp.Description("Initial backoff in milliseconds (default: 5000)"),
		),
		mcp.WithNumber("max_delay_ms",
			mcp.Description("Max single delay in milliseconds (default: 300000 = 5 min)"),
		),
		mcp.WithNumber("multiplier",
			mcp.Description("Exponential growth factor (default: 2.0)"),
		),
		mcp.WithBoolean("respect_retry_after",
			mcp.Description("Honor server Retry-After header if present (default: true)"),
		),
		mcp.WithNumber("total_wait_cap_ms",
			mcp.Description("Global time cap in milliseconds (default: 1200000 = 20 min)"),
		),
		mcp.WithBoolean("fail_on_partial",
			mcp.Description("If true, abort when secondary endpoints fail (default: false)"),
		),
	)

	mcpServer.AddTool(readCardWithRetryTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		cardID := mcp.ParseString(request, "card_id", "")
		if cardID == "" {
			return mcp.NewToolResultError("card_id parameter is required"), nil
		}

		// Build retry config with defaults
		retryConfig := kanbanize.DefaultRetryConfig()

		// Override with provided parameters
		if maxAttempts := mcp.ParseFloat64(request, "max_attempts", 0); maxAttempts > 0 {
			retryConfig.MaxAttempts = int(maxAttempts)
		}
		if initialDelayMs := mcp.ParseFloat64(request, "initial_delay_ms", 0); initialDelayMs > 0 {
			retryConfig.InitialDelay = time.Duration(initialDelayMs) * time.Millisecond
		}
		if maxDelayMs := mcp.ParseFloat64(request, "max_delay_ms", 0); maxDelayMs > 0 {
			retryConfig.MaxDelay = time.Duration(maxDelayMs) * time.Millisecond
		}
		if multiplier := mcp.ParseFloat64(request, "multiplier", 0); multiplier > 0 {
			retryConfig.Multiplier = multiplier
		}
		if totalWaitCapMs := mcp.ParseFloat64(request, "total_wait_cap_ms", 0); totalWaitCapMs > 0 {
			retryConfig.TotalWaitCap = time.Duration(totalWaitCapMs) * time.Millisecond
		}

		// Parse boolean parameters
		retryConfig.RespectRetryAfter = mcp.ParseBoolean(request, "respect_retry_after", true)
		failOnPartial := mcp.ParseBoolean(request, "fail_on_partial", false)

		// Execute with retry
		cardData, err := client.ReadCardWithRetry(ctx, cardID, retryConfig, failOnPartial)
		if err != nil {
			// Return partial results if available
			if cardData != nil {
				cardJSON, _ := json.Marshal(cardData)
				return mcp.NewToolResultError(fmt.Sprintf("Partial failure: %s\n\nPartial data:\n%s", err.Error(), string(cardJSON))), nil
			}
			return mcp.NewToolResultError("Failed to read card: "+err.Error()), nil
		}

		cardJSON, err := json.Marshal(cardData)
		if err != nil {
			return mcp.NewToolResultError("Failed to serialize card data: "+err.Error()), nil
		}

		return mcp.NewToolResultText(string(cardJSON)), nil
	})

	getCardsWithRetryTool := mcp.NewTool("get_cards_with_retry",
		mcp.WithDescription("Query multiple cards using filter criteria with exponential backoff retry logic. Returns cards matching the specified filters (board_ids, lane_ids, workflow_ids, or card_ids)."),
		mcp.WithString("board_ids",
			mcp.Description("Comma-separated board IDs to filter by (e.g., \"1,2,3\")"),
		),
		mcp.WithString("lane_ids",
			mcp.Description("Comma-separated lane IDs to filter by (e.g., \"4,5,6\")"),
		),
		mcp.WithString("workflow_ids",
			mcp.Description("Comma-separated workflow IDs to filter by (e.g., \"7,8,9\")"),
		),
		mcp.WithString("card_ids",
			mcp.Description("Comma-separated card IDs to filter by (e.g., \"10,11,12\")"),
		),
		mcp.WithNumber("max_attempts",
			mcp.Description("Upper bound attempts per endpoint (default: 10)"),
		),
		mcp.WithNumber("initial_delay_ms",
			mcp.Description("Initial backoff in milliseconds (default: 5000)"),
		),
		mcp.WithNumber("max_delay_ms",
			mcp.Description("Max single delay in milliseconds (default: 300000 = 5 min)"),
		),
		mcp.WithNumber("multiplier",
			mcp.Description("Exponential growth factor (default: 2.0)"),
		),
		mcp.WithBoolean("respect_retry_after",
			mcp.Description("Honor server Retry-After header if present (default: true)"),
		),
		mcp.WithNumber("total_wait_cap_ms",
			mcp.Description("Global time cap in milliseconds (default: 1200000 = 20 min)"),
		),
		mcp.WithBoolean("fail_on_partial",
			mcp.Description("If true, abort when secondary endpoints fail (default: false)"),
		),
	)

	mcpServer.AddTool(getCardsWithRetryTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Parse filter parameters
		filter := kanbanize.GetCardsRequest{}

		boardIDsStr := mcp.ParseString(request, "board_ids", "")
		if boardIDsStr != "" {
			ids, err := parseIntArray(boardIDsStr)
			if err != nil {
				return mcp.NewToolResultError("Invalid board_ids format: " + err.Error()), nil
			}
			filter.BoardIDs = ids
		}

		laneIDsStr := mcp.ParseString(request, "lane_ids", "")
		if laneIDsStr != "" {
			ids, err := parseIntArray(laneIDsStr)
			if err != nil {
				return mcp.NewToolResultError("Invalid lane_ids format: " + err.Error()), nil
			}
			filter.LaneIDs = ids
		}

		workflowIDsStr := mcp.ParseString(request, "workflow_ids", "")
		if workflowIDsStr != "" {
			ids, err := parseIntArray(workflowIDsStr)
			if err != nil {
				return mcp.NewToolResultError("Invalid workflow_ids format: " + err.Error()), nil
			}
			filter.WorkflowIDs = ids
		}

		cardIDsStr := mcp.ParseString(request, "card_ids", "")
		if cardIDsStr != "" {
			ids, err := parseIntArray(cardIDsStr)
			if err != nil {
				return mcp.NewToolResultError("Invalid card_ids format: " + err.Error()), nil
			}
			filter.CardIDs = ids
		}

		// Validate at least one filter is provided
		if len(filter.BoardIDs) == 0 && len(filter.LaneIDs) == 0 &&
		   len(filter.WorkflowIDs) == 0 && len(filter.CardIDs) == 0 {
			return mcp.NewToolResultError("At least one filter parameter (board_ids, lane_ids, workflow_ids, or card_ids) must be provided"), nil
		}

		// Build retry config with defaults
		retryConfig := kanbanize.DefaultRetryConfig()

		// Override with provided parameters
		if maxAttempts := mcp.ParseFloat64(request, "max_attempts", 0); maxAttempts > 0 {
			retryConfig.MaxAttempts = int(maxAttempts)
		}
		if initialDelayMs := mcp.ParseFloat64(request, "initial_delay_ms", 0); initialDelayMs > 0 {
			retryConfig.InitialDelay = time.Duration(initialDelayMs) * time.Millisecond
		}
		if maxDelayMs := mcp.ParseFloat64(request, "max_delay_ms", 0); maxDelayMs > 0 {
			retryConfig.MaxDelay = time.Duration(maxDelayMs) * time.Millisecond
		}
		if multiplier := mcp.ParseFloat64(request, "multiplier", 0); multiplier > 0 {
			retryConfig.Multiplier = multiplier
		}
		if totalWaitCapMs := mcp.ParseFloat64(request, "total_wait_cap_ms", 0); totalWaitCapMs > 0 {
			retryConfig.TotalWaitCap = time.Duration(totalWaitCapMs) * time.Millisecond
		}

		// Parse boolean parameters
		retryConfig.RespectRetryAfter = mcp.ParseBoolean(request, "respect_retry_after", true)
		failOnPartial := mcp.ParseBoolean(request, "fail_on_partial", false)

		// Execute with retry
		cardsData, err := client.GetCardsWithRetry(ctx, filter, retryConfig, failOnPartial)
		if err != nil {
			// Return partial results if available
			if cardsData != nil {
				cardsJSON, _ := json.Marshal(cardsData)
				return mcp.NewToolResultError(fmt.Sprintf("Partial failure: %s\n\nPartial data:\n%s", err.Error(), string(cardsJSON))), nil
			}
			return mcp.NewToolResultError("Failed to get cards: "+err.Error()), nil
		}

		cardsJSON, err := json.Marshal(cardsData)
		if err != nil {
			return mcp.NewToolResultError("Failed to serialize cards data: "+err.Error()), nil
		}

		return mcp.NewToolResultText(string(cardsJSON)), nil
	})

	if err := server.ServeStdio(mcpServer); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// parseIntArray parses a comma-separated string of integers into a slice
func parseIntArray(s string) ([]int, error) {
	if s == "" {
		return []int{}, nil
	}

	parts := strings.Split(s, ",")
	result := make([]int, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		num, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("invalid integer: %s", part)
		}
		result = append(result, num)
	}

	return result, nil
}