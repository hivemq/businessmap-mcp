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
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

// RetryConfig holds configuration for retry behavior
type RetryConfig struct {
	MaxAttempts       int
	InitialDelay      time.Duration
	MaxDelay          time.Duration
	Multiplier        float64
	RespectRetryAfter bool
	TotalWaitCap      time.Duration
}

// DefaultRetryConfig returns sensible default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:       10,
		InitialDelay:      5 * time.Second,
		MaxDelay:          5 * time.Minute,
		Multiplier:        2.0,
		RespectRetryAfter: true,
		TotalWaitCap:      20 * time.Minute,
	}
}

// Validate checks if the retry configuration is valid
func (rc *RetryConfig) Validate() error {
	if rc.MaxAttempts < 1 {
		return fmt.Errorf("MaxAttempts must be >= 1, got %d", rc.MaxAttempts)
	}
	if rc.Multiplier < 1.0 {
		return fmt.Errorf("Multiplier must be >= 1.0, got %f", rc.Multiplier)
	}
	if rc.InitialDelay <= 0 {
		return fmt.Errorf("InitialDelay must be > 0, got %v", rc.InitialDelay)
	}
	if rc.MaxDelay < rc.InitialDelay {
		return fmt.Errorf("MaxDelay (%v) must be >= InitialDelay (%v)", rc.MaxDelay, rc.InitialDelay)
	}
	if rc.TotalWaitCap < rc.InitialDelay {
		return fmt.Errorf("TotalWaitCap (%v) must be >= InitialDelay (%v)", rc.TotalWaitCap, rc.InitialDelay)
	}
	return nil
}

// parseRetryAfter parses the Retry-After header value
// It supports both delay-seconds (integer) and HTTP-date formats
func parseRetryAfter(retryAfterHeader string) time.Duration {
	if retryAfterHeader == "" {
		return 0
	}

	// Try parsing as integer (seconds)
	if seconds, err := strconv.Atoi(retryAfterHeader); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}

	// Try parsing as HTTP-date (RFC1123, RFC850, ANSI C formats)
	formats := []string{
		time.RFC1123,
		time.RFC850,
		time.ANSIC,
		time.RFC3339,
	}

	for _, format := range formats {
		if t, err := time.Parse(format, retryAfterHeader); err == nil {
			duration := time.Until(t)
			if duration > 0 {
				return duration
			}
			return 0
		}
	}

	return 0
}

// exponentialBackoffWithJitter calculates the backoff delay using full jitter
// Returns the delay to wait before the next retry attempt
func exponentialBackoffWithJitter(cfg RetryConfig, attempt int, retryAfter time.Duration) time.Duration {
	// If Retry-After header is present and we respect it, use it
	if retryAfter > 0 && cfg.RespectRetryAfter {
		return retryAfter
	}

	// Calculate base delay with exponential backoff
	base := cfg.InitialDelay
	if attempt > 0 {
		base = time.Duration(float64(cfg.InitialDelay) * math.Pow(cfg.Multiplier, float64(attempt)))
	}

	// Cap at max delay
	if base > cfg.MaxDelay {
		base = cfg.MaxDelay
	}

	// Apply full jitter: random value between 0 and base
	maxNanos := base.Nanoseconds()
	if maxNanos <= 0 {
		return 0
	}

	jitteredNanos := rand.Int63n(maxNanos + 1)
	return time.Duration(jitteredNanos)
}

// isRateLimitError checks if an error is a rate limit error
func isRateLimitError(err error) (*RateLimitError, bool) {
	if rateLimitErr, ok := err.(*RateLimitError); ok {
		return rateLimitErr, true
	}
	return nil, false
}

// makeRequestWithRetry executes an HTTP request with retry logic for rate limits
func (c *Client) makeRequestWithRetry(ctx context.Context, cfg RetryConfig, method, url string, body interface{}) ([]byte, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid retry config: %w", err)
	}

	var lastErr error
	totalWaitTime := time.Duration(0)
	startTime := time.Now()

	for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("request canceled: %w", ctx.Err())
		default:
		}

		// Attempt the request
		result, err := c.makeAPIRequestWithBody(method, url, body)
		if err == nil {
			if attempt > 0 {
				log.Printf("[RETRY] Success after %d attempts, total wait: %v", attempt+1, totalWaitTime)
			}
			return result, nil
		}

		// Check if it's a rate limit error
		rateLimitErr, isRateLimit := isRateLimitError(err)
		if !isRateLimit {
			// Non-rate-limit error, fail fast
			return nil, err
		}

		lastErr = err

		// Check if we've exhausted attempts
		if attempt >= cfg.MaxAttempts-1 {
			log.Printf("[RETRY] Max attempts (%d) exceeded for %s", cfg.MaxAttempts, url)
			return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
		}

		// Calculate backoff delay
		backoffDelay := exponentialBackoffWithJitter(cfg, attempt, rateLimitErr.RetryAfter)

		// Check if waiting would exceed total wait cap
		if totalWaitTime+backoffDelay > cfg.TotalWaitCap {
			log.Printf("[RETRY] Would exceed total wait cap (%v), aborting", cfg.TotalWaitCap)
			return nil, fmt.Errorf("total wait time would exceed cap (%v): %w", cfg.TotalWaitCap, lastErr)
		}

		// Log retry attempt
		if rateLimitErr.RetryAfter > 0 {
			log.Printf("[RETRY] Attempt %d/%d failed: rate limit hit (Retry-After: %v), waiting %v",
				attempt+1, cfg.MaxAttempts, rateLimitErr.RetryAfter, backoffDelay)
		} else {
			log.Printf("[RETRY] Attempt %d/%d failed: rate limit hit, waiting %v",
				attempt+1, cfg.MaxAttempts, backoffDelay)
		}

		// Wait with context awareness
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("request canceled during backoff: %w", ctx.Err())
		case <-time.After(backoffDelay):
			totalWaitTime = time.Since(startTime)
		}
	}

	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

// enhanceErrorWithRateLimit wraps HTTP errors to detect rate limiting
func enhanceErrorWithRateLimit(resp *http.Response, originalErr error, body []byte) error {
	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
		return &RateLimitError{
			StatusCode: resp.StatusCode,
			RetryAfter: retryAfter,
			RawBody:    string(body),
		}
	}
	return originalErr
}

// endpointResult tracks the result of fetching a single endpoint
type endpointResult struct {
	name          string
	data          []byte
	attempts      int
	rateLimitHits int
	success       bool
	err           error
}

// ReadCardWithRetry fetches card data with retry logic for rate limiting
// It returns a structured response with metadata about retry attempts
func (c *Client) ReadCardWithRetry(ctx context.Context, cardIDOrURL string, cfg RetryConfig, failOnPartial bool) (*ReadCardWithRetryResponse, error) {
	cardID, err := c.extractCardID(cardIDOrURL)
	if err != nil {
		return nil, err
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid retry config: %w", err)
	}

	startTime := time.Now()
	response := &ReadCardWithRetryResponse{
		CardID:       cardID,
		Attempts:     make(map[string]int),
		Completed:    make(map[string]bool),
		PartialError: make(map[string]string),
		Data:         &ReadCardResponse{},
	}

	// Fetch primary card data (required)
	cardResult := c.fetchWithRetry(ctx, cfg, "card", fmt.Sprintf("%s/api/v2/cards/%s", c.baseURL, cardID))
	response.Attempts["card"] = cardResult.attempts
	response.RateLimitHits += cardResult.rateLimitHits
	response.Completed["card"] = cardResult.success

	if !cardResult.success {
		response.PartialError["card"] = cardResult.err.Error()
		response.WaitSeconds = time.Since(startTime).Seconds()
		return response, fmt.Errorf("failed to fetch card: %w", cardResult.err)
	}

	// Parse card data
	var cardDataResp CardDataResponse
	if err := json.Unmarshal(cardResult.data, &cardDataResp); err != nil {
		return response, fmt.Errorf("failed to parse card data: %w", err)
	}

	// Populate basic card fields
	response.Data.Title = cardDataResp.Data.Title
	response.Data.Description = cardDataResp.Data.Description
	response.Data.LinkedCards = cardDataResp.Data.LinkedCards
	response.Data.CustomFields = cardDataResp.Data.CustomFields
	response.Data.CreatedAt = parseTimestamp(cardDataResp.Data.CreatedAt)
	response.Data.LastModified = parseTimestamp(cardDataResp.Data.LastModified)
	response.Data.InCurrentPositionSince = parseTimestamp(cardDataResp.Data.InCurrentPositionSince)
	response.Data.FirstRequestTime = parseTimestamp(cardDataResp.Data.FirstRequestTime)
	response.Data.FirstStartTime = parseTimestamp(cardDataResp.Data.FirstStartTime)
	response.Data.FirstEndTime = parseTimestamp(cardDataResp.Data.FirstEndTime)
	response.Data.LastRequestTime = parseTimestamp(cardDataResp.Data.LastRequestTime)
	response.Data.LastStartTime = parseTimestamp(cardDataResp.Data.LastStartTime)
	response.Data.LastEndTime = parseTimestamp(cardDataResp.Data.LastEndTime)

	// Parse initiative details if present
	if cardDataResp.Data.InitiativeDetails != nil {
		response.Data.PlannedStartDate = cardDataResp.Data.InitiativeDetails.PlannedStartDate
		response.Data.PlannedEndDate = cardDataResp.Data.InitiativeDetails.PlannedEndDate
		response.Data.ActualStartTime = parseTimestamp(cardDataResp.Data.InitiativeDetails.ActualStartTime)
		response.Data.ActualEndTime = parseTimestamp(cardDataResp.Data.InitiativeDetails.ActualEndTime)
	}

	// Fetch comments and subtasks in parallel (secondary endpoints)
	type fetchResult struct {
		name   string
		result *endpointResult
		data   []byte
	}

	resultsChan := make(chan fetchResult, 2)

	// Fetch comments
	go func() {
		result := c.fetchWithRetry(ctx, cfg, "comments", fmt.Sprintf("%s/api/v2/cards/%s/comments", c.baseURL, cardID))
		resultsChan <- fetchResult{name: "comments", result: result, data: result.data}
	}()

	// Fetch subtasks
	go func() {
		result := c.fetchWithRetry(ctx, cfg, "subtasks", fmt.Sprintf("%s/api/v2/cards/%s/subtasks", c.baseURL, cardID))
		resultsChan <- fetchResult{name: "subtasks", result: result, data: result.data}
	}()

	// Collect results
	for i := 0; i < 2; i++ {
		select {
		case <-ctx.Done():
			return response, fmt.Errorf("context canceled: %w", ctx.Err())
		case result := <-resultsChan:
			response.Attempts[result.name] = result.result.attempts
			response.RateLimitHits += result.result.rateLimitHits
			response.Completed[result.name] = result.result.success

			if !result.result.success {
				response.PartialError[result.name] = result.result.err.Error()
				if failOnPartial {
					response.WaitSeconds = time.Since(startTime).Seconds()
					return response, fmt.Errorf("failed to fetch %s: %w", result.name, result.result.err)
				}
			} else {
				// Parse successful results
				if result.name == "comments" {
					var commentsResp CommentsResponse
					if err := json.Unmarshal(result.data, &commentsResp); err == nil {
						comments := make([]Comment, len(commentsResp.Data))
						for i, commentData := range commentsResp.Data {
							comments[i] = Comment{
								ID:        strconv.Itoa(commentData.CommentID),
								Text:      commentData.Text,
								Author:    commentData.AuthorName,
								CreatedAt: parseCommentTimestamp(commentData.CreatedAt),
							}
						}
						response.Data.Comments = comments
					}
				} else if result.name == "subtasks" {
					var subtasksResp SubtasksResponse
					if err := json.Unmarshal(result.data, &subtasksResp); err == nil {
						subtasks := make([]Subtask, len(subtasksResp.Data))
						for i, subtaskData := range subtasksResp.Data {
							subtasks[i] = Subtask{
								ID:          strconv.Itoa(subtaskData.SubtaskID),
								Title:       subtaskData.Title,
								Description: subtaskData.Description,
								Completed:   subtaskData.Finished == 1,
							}
						}
						response.Data.Subtasks = subtasks
					}
				}
			}
		}
	}

	response.WaitSeconds = time.Since(startTime).Seconds()
	return response, nil
}

// fetchWithRetry is a helper that wraps makeRequestWithRetry with result tracking
func (c *Client) fetchWithRetry(ctx context.Context, cfg RetryConfig, name, url string) *endpointResult {
	result := &endpointResult{
		name:     name,
		attempts: 0,
	}

	var lastErr error
	totalWaitTime := time.Duration(0)
	startTime := time.Now()

	for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
		result.attempts = attempt + 1

		// Check context cancellation
		select {
		case <-ctx.Done():
			result.err = fmt.Errorf("request canceled: %w", ctx.Err())
			return result
		default:
		}

		// Attempt the request
		data, err := c.makeAPIRequest(url)
		if err == nil {
			result.success = true
			result.data = data
			if attempt > 0 {
				log.Printf("[RETRY] Success for %s after %d attempts, total wait: %v", name, attempt+1, totalWaitTime)
			}
			return result
		}

		// Check if it's a rate limit error
		rateLimitErr, isRateLimit := isRateLimitError(err)
		if isRateLimit {
			result.rateLimitHits++
		}

		if !isRateLimit {
			// Non-rate-limit error, fail fast
			result.err = err
			return result
		}

		lastErr = err

		// Check if we've exhausted attempts
		if attempt >= cfg.MaxAttempts-1 {
			log.Printf("[RETRY] Max attempts (%d) exceeded for %s", cfg.MaxAttempts, name)
			result.err = fmt.Errorf("max retries exceeded: %w", lastErr)
			return result
		}

		// Calculate backoff delay
		backoffDelay := exponentialBackoffWithJitter(cfg, attempt, rateLimitErr.RetryAfter)

		// Check if waiting would exceed total wait cap
		if totalWaitTime+backoffDelay > cfg.TotalWaitCap {
			log.Printf("[RETRY] Would exceed total wait cap (%v) for %s, aborting", cfg.TotalWaitCap, name)
			result.err = fmt.Errorf("total wait time would exceed cap (%v): %w", cfg.TotalWaitCap, lastErr)
			return result
		}

		// Log retry attempt
		if rateLimitErr.RetryAfter > 0 {
			log.Printf("[RETRY] %s attempt %d/%d failed: rate limit hit (Retry-After: %v), waiting %v",
				name, attempt+1, cfg.MaxAttempts, rateLimitErr.RetryAfter, backoffDelay)
		} else {
			log.Printf("[RETRY] %s attempt %d/%d failed: rate limit hit, waiting %v",
				name, attempt+1, cfg.MaxAttempts, backoffDelay)
		}

		// Wait with context awareness
		select {
		case <-ctx.Done():
			result.err = fmt.Errorf("request canceled during backoff: %w", ctx.Err())
			return result
		case <-time.After(backoffDelay):
			totalWaitTime = time.Since(startTime)
		}
	}

	result.err = fmt.Errorf("max retries exceeded: %w", lastErr)
	return result
}
