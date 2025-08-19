package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/joho/godotenv"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/zbigniewzabost-hivemq/businessmap-mcp/internal/config"
	"github.com/zbigniewzabost-hivemq/businessmap-mcp/internal/kanbanize"
)

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	client := kanbanize.NewClient(cfg.KanbanizeBaseURL, cfg.KanbanizeAPIKey)

	mcpServer := server.NewMCPServer("kanbanize-mcp-server", "1.0.0")

	readCardTool := mcp.NewTool("read_card",
		mcp.WithDescription("Read comprehensive card information including title, description, subtasks, and comments"),
		mcp.WithString("card_id",
			mcp.Required(),
			mcp.Description("The ID of the Kanbanize card to read"),
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

	if err := server.ServeStdio(mcpServer); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}