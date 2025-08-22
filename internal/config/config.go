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