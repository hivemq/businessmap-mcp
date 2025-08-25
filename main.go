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
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/hivemq/businessmap-mcp/internal/config"
	"github.com/hivemq/businessmap-mcp/internal/kanbanize"
)

// Version can be set at build time via ldflags
var version = "dev"

func main() {
	var showVersion = flag.Bool("version", false, "show version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("businessmap-mcp version %s\n", version)
		os.Exit(0)
	}

	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	client := kanbanize.NewClient(cfg.KanbanizeBaseURL, cfg.KanbanizeAPIKey)

	mcpServer := server.NewMCPServer("kanbanize-mcp-server", version)

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

	if err := server.ServeStdio(mcpServer); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}