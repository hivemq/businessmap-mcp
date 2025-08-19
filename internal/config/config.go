package config

import (
	"fmt"
	"os"
)

type Config struct {
	KanbanizeAPIKey string
	KanbanizeBaseURL string
}

func Load() (*Config, error) {
	apiKey := os.Getenv("KANBANIZE_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("KANBANIZE_API_KEY environment variable is required")
	}

	baseURL := os.Getenv("KANBANIZE_BASE_URL")
	if baseURL == "" {
		return nil, fmt.Errorf("KANBANIZE_BASE_URL environment variable is required")
	}

	return &Config{
		KanbanizeAPIKey:  apiKey,
		KanbanizeBaseURL: baseURL,
	}, nil
}