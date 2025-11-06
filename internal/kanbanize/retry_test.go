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

package kanbanize

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestParseRetryAfter_Seconds(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		expected time.Duration
	}{
		{"valid seconds", "60", 60 * time.Second},
		{"zero seconds", "0", 0},
		{"negative seconds", "-10", 0},
		{"empty string", "", 0},
		{"invalid string", "abc", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseRetryAfter(tt.header)
			if result != tt.expected {
				t.Errorf("parseRetryAfter(%q) = %v, want %v", tt.header, result, tt.expected)
			}
		})
	}
}

func TestParseRetryAfter_HTTPDate(t *testing.T) {
	// Test with a future time (5 seconds from now)
	future := time.Now().Add(5 * time.Second)
	httpDate := future.Format(time.RFC1123)

	result := parseRetryAfter(httpDate)

	// Allow 1 second tolerance for test execution time
	if result < 4*time.Second || result > 6*time.Second {
		t.Errorf("parseRetryAfter(%q) = %v, expected ~5s", httpDate, result)
	}
}

func TestRetryConfigValidate(t *testing.T) {
	tests := []struct {
		name      string
		config    RetryConfig
		expectErr bool
	}{
		{
			name:      "valid config",
			config:    DefaultRetryConfig(),
			expectErr: false,
		},
		{
			name: "invalid max attempts",
			config: RetryConfig{
				MaxAttempts:  0,
				InitialDelay: 1 * time.Second,
				MaxDelay:     5 * time.Second,
				Multiplier:   2.0,
				TotalWaitCap: 10 * time.Second,
			},
			expectErr: true,
		},
		{
			name: "invalid multiplier",
			config: RetryConfig{
				MaxAttempts:  3,
				InitialDelay: 1 * time.Second,
				MaxDelay:     5 * time.Second,
				Multiplier:   0.5,
				TotalWaitCap: 10 * time.Second,
			},
			expectErr: true,
		},
		{
			name: "invalid initial delay",
			config: RetryConfig{
				MaxAttempts:  3,
				InitialDelay: 0,
				MaxDelay:     5 * time.Second,
				Multiplier:   2.0,
				TotalWaitCap: 10 * time.Second,
			},
			expectErr: true,
		},
		{
			name: "max delay less than initial",
			config: RetryConfig{
				MaxAttempts:  3,
				InitialDelay: 10 * time.Second,
				MaxDelay:     5 * time.Second,
				Multiplier:   2.0,
				TotalWaitCap: 20 * time.Second,
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.expectErr {
				t.Errorf("Validate() error = %v, expectErr %v", err, tt.expectErr)
			}
		})
	}
}

func TestExponentialBackoffWithJitter(t *testing.T) {
	cfg := RetryConfig{
		InitialDelay: 1 * time.Second,
		MaxDelay:     10 * time.Second,
		Multiplier:   2.0,
	}

	tests := []struct {
		name       string
		attempt    int
		retryAfter time.Duration
		minExpect  time.Duration
		maxExpect  time.Duration
	}{
		{"attempt 0", 0, 0, 0, 1 * time.Second},
		{"attempt 1", 1, 0, 0, 2 * time.Second},
		{"attempt 2", 2, 0, 0, 4 * time.Second},
		{"attempt 3 capped", 3, 0, 0, 8 * time.Second},
		{"attempt 4 capped at max", 4, 0, 0, 10 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Run multiple times due to randomness
			for i := 0; i < 10; i++ {
				result := exponentialBackoffWithJitter(cfg, tt.attempt, tt.retryAfter)
				if result < tt.minExpect || result > tt.maxExpect {
					t.Errorf("exponentialBackoffWithJitter() = %v, want between %v and %v",
						result, tt.minExpect, tt.maxExpect)
				}
			}
		})
	}
}

func TestExponentialBackoffWithJitter_RetryAfterPriority(t *testing.T) {
	cfg := RetryConfig{
		InitialDelay:      1 * time.Second,
		MaxDelay:          10 * time.Second,
		Multiplier:        2.0,
		RespectRetryAfter: true,
	}

	retryAfter := 15 * time.Second
	result := exponentialBackoffWithJitter(cfg, 0, retryAfter)

	if result != retryAfter {
		t.Errorf("exponentialBackoffWithJitter() with Retry-After = %v, want %v", result, retryAfter)
	}
}

func TestReadCardWithRetry_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/cards/1001") && !strings.Contains(r.URL.Path, "/comments") && !strings.Contains(r.URL.Path, "/subtasks") {
			response := CardDataResponse{
				Data: CardData{
					CardID:      1001,
					Title:       "Test Card",
					Description: "Test Description",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		} else if strings.Contains(r.URL.Path, "/comments") {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(CommentsResponse{Data: []CommentData{}})
		} else if strings.Contains(r.URL.Path, "/subtasks") {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(SubtasksResponse{Data: []SubtaskData{}})
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")
	cfg := RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     1 * time.Second,
		Multiplier:   2.0,
		TotalWaitCap: 5 * time.Second,
	}

	ctx := context.Background()
	response, err := client.ReadCardWithRetry(ctx, "1001", cfg, false)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if response.Data.Title != "Test Card" {
		t.Errorf("Expected title 'Test Card', got '%s'", response.Data.Title)
	}

	if response.Attempts["card"] != 1 {
		t.Errorf("Expected 1 attempt for card, got %d", response.Attempts["card"])
	}

	if response.RateLimitHits != 0 {
		t.Errorf("Expected 0 rate limit hits, got %d", response.RateLimitHits)
	}

	if !response.Completed["card"] {
		t.Error("Expected card to be completed")
	}
}

func TestReadCardWithRetry_RateLimitThenSuccess(t *testing.T) {
	attemptCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/cards/1001") && !strings.Contains(r.URL.Path, "/comments") && !strings.Contains(r.URL.Path, "/subtasks") {
			attemptCount++
			if attemptCount < 2 {
				// First attempt: return 429
				w.Header().Set("Retry-After", "1")
				w.WriteHeader(http.StatusTooManyRequests)
				json.NewEncoder(w).Encode(map[string]string{"error": "rate limited"})
			} else {
				// Second attempt: success
				response := CardDataResponse{
					Data: CardData{
						CardID:      1001,
						Title:       "Test Card",
						Description: "Test Description",
					},
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
			}
		} else if strings.Contains(r.URL.Path, "/comments") {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(CommentsResponse{Data: []CommentData{}})
		} else if strings.Contains(r.URL.Path, "/subtasks") {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(SubtasksResponse{Data: []SubtaskData{}})
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")
	cfg := RetryConfig{
		MaxAttempts:       5,
		InitialDelay:      100 * time.Millisecond,
		MaxDelay:          1 * time.Second,
		Multiplier:        2.0,
		RespectRetryAfter: true,
		TotalWaitCap:      10 * time.Second,
	}

	ctx := context.Background()
	response, err := client.ReadCardWithRetry(ctx, "1001", cfg, false)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if response.Data.Title != "Test Card" {
		t.Errorf("Expected title 'Test Card', got '%s'", response.Data.Title)
	}

	if response.Attempts["card"] != 2 {
		t.Errorf("Expected 2 attempts for card, got %d", response.Attempts["card"])
	}

	if response.RateLimitHits != 1 {
		t.Errorf("Expected 1 rate limit hit, got %d", response.RateLimitHits)
	}

	if !response.Completed["card"] {
		t.Error("Expected card to be completed")
	}
}

func TestReadCardWithRetry_MaxRetriesExceeded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Always return 429
		w.Header().Set("Retry-After", "1")
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]string{"error": "rate limited"})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")
	cfg := RetryConfig{
		MaxAttempts:       3,
		InitialDelay:      50 * time.Millisecond,
		MaxDelay:          200 * time.Millisecond,
		Multiplier:        2.0,
		RespectRetryAfter: false, // Don't respect to speed up test
		TotalWaitCap:      5 * time.Second,
	}

	ctx := context.Background()
	response, err := client.ReadCardWithRetry(ctx, "1001", cfg, false)

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if !strings.Contains(err.Error(), "max retries exceeded") {
		t.Errorf("Expected 'max retries exceeded' error, got: %v", err)
	}

	if response.Attempts["card"] != 3 {
		t.Errorf("Expected 3 attempts for card, got %d", response.Attempts["card"])
	}

	if response.RateLimitHits != 3 {
		t.Errorf("Expected 3 rate limit hits, got %d", response.RateLimitHits)
	}

	if response.Completed["card"] {
		t.Error("Expected card not to be completed")
	}
}

func TestReadCardWithRetry_PartialResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/cards/1001") && !strings.Contains(r.URL.Path, "/comments") && !strings.Contains(r.URL.Path, "/subtasks") {
			response := CardDataResponse{
				Data: CardData{
					CardID:      1001,
					Title:       "Test Card",
					Description: "Test Description",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		} else if strings.Contains(r.URL.Path, "/comments") {
			// Comments endpoint always fails with 429
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]string{"error": "rate limited"})
		} else if strings.Contains(r.URL.Path, "/subtasks") {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(SubtasksResponse{Data: []SubtaskData{}})
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")
	cfg := RetryConfig{
		MaxAttempts:  2,
		InitialDelay: 50 * time.Millisecond,
		MaxDelay:     200 * time.Millisecond,
		Multiplier:   2.0,
		TotalWaitCap: 2 * time.Second,
	}

	ctx := context.Background()
	response, err := client.ReadCardWithRetry(ctx, "1001", cfg, false)

	// Should succeed with partial results
	if err != nil {
		t.Fatalf("Expected no error with partial results, got %v", err)
	}

	if response.Data.Title != "Test Card" {
		t.Errorf("Expected title 'Test Card', got '%s'", response.Data.Title)
	}

	if response.Completed["card"] != true {
		t.Error("Expected card to be completed")
	}

	if response.Completed["comments"] != false {
		t.Error("Expected comments not to be completed")
	}

	if response.PartialError["comments"] == "" {
		t.Error("Expected partial error for comments")
	}

	if response.Completed["subtasks"] != true {
		t.Error("Expected subtasks to be completed")
	}
}

func TestReadCardWithRetry_FailOnPartial(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/cards/1001") && !strings.Contains(r.URL.Path, "/comments") && !strings.Contains(r.URL.Path, "/subtasks") {
			response := CardDataResponse{
				Data: CardData{
					CardID:      1001,
					Title:       "Test Card",
					Description: "Test Description",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		} else if strings.Contains(r.URL.Path, "/comments") {
			// Comments endpoint always fails
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]string{"error": "rate limited"})
		} else if strings.Contains(r.URL.Path, "/subtasks") {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(SubtasksResponse{Data: []SubtaskData{}})
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")
	cfg := RetryConfig{
		MaxAttempts:  2,
		InitialDelay: 50 * time.Millisecond,
		MaxDelay:     200 * time.Millisecond,
		Multiplier:   2.0,
		TotalWaitCap: 2 * time.Second,
	}

	ctx := context.Background()
	response, err := client.ReadCardWithRetry(ctx, "1001", cfg, true) // fail_on_partial = true

	// Should fail
	if err == nil {
		t.Fatal("Expected error with fail_on_partial=true, got nil")
	}

	if !strings.Contains(err.Error(), "failed to fetch") {
		t.Errorf("Expected 'failed to fetch' error, got: %v", err)
	}

	// Response should still have partial data
	if response.Data.Title != "Test Card" {
		t.Errorf("Expected partial data with title 'Test Card', got '%s'", response.Data.Title)
	}
}
