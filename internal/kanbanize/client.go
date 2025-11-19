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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// extractCardID extracts card ID from either a card ID or a full BusinessMap URL
func (c *Client) extractCardID(input string) (string, error) {
	if input == "" {
		return "", fmt.Errorf("card ID or URL cannot be empty")
	}

	// If input doesn't contain "http" or "/", assume it's already a card ID
	if !strings.Contains(input, "http") && !strings.Contains(input, "/") {
		return input, nil
	}

	// Pattern to match BusinessMap URLs with various endings:
	// .../ctrl_board/<board_id>/cards/<card_id>
	// .../ctrl_board/<board_id>/cards/<card_id>/
	// .../ctrl_board/<board_id>/cards/<card_id>/any_string
	// .../ctrl_board/<board_id>/cards/<card_id>/any_string/
	// Also matches variations like .../crl_board/
	pattern := regexp.MustCompile(`/c(?:tr|r)l_board/\d+/cards/(\d+)(?:/.*)?`)
	matches := pattern.FindStringSubmatch(input)

	if len(matches) < 2 {
		return "", fmt.Errorf("invalid BusinessMap URL format: %s", input)
	}

	return matches[1], nil
}

// parseTimestamp parses an RFC3339 timestamp string into a time.Time pointer
func parseTimestamp(ts *string) *time.Time {
	if ts == nil || *ts == "" {
		return nil
	}
	if parsed, err := time.Parse(time.RFC3339, *ts); err == nil {
		return &parsed
	}
	return nil
}

// parseCommentTimestamp tries multiple date formats to parse comment timestamps
func parseCommentTimestamp(dateStr string) time.Time {
	if dateStr == "" {
		return time.Time{}
	}

	// Try common formats
	formats := []string{
		time.RFC3339,           // "2006-01-02T15:04:05Z07:00"
		"2006-01-02T15:04:05Z", // RFC3339 without timezone offset
		"2006-01-02 15:04:05",  // Space-separated format
		"2006-01-02T15:04:05",  // T-separated without timezone
		time.RFC3339Nano,       // With nanoseconds
	}

	for _, format := range formats {
		if parsed, err := time.Parse(format, dateStr); err == nil {
			return parsed
		}
	}

	// If all parsing fails, return zero time
	return time.Time{}
}

func (c *Client) ReadCard(cardIDOrURL string) (*ReadCardResponse, error) {
	cardID, err := c.extractCardID(cardIDOrURL)
	if err != nil {
		return nil, err
	}

	cardData, err := c.getCard(cardID)
	if err != nil {
		return nil, fmt.Errorf("failed to get card data: %w", err)
	}

	comments, err := c.getCardComments(cardID)
	if err != nil {
		comments = []Comment{}
	}

	subtasks, err := c.getCardSubtasks(cardID)
	if err != nil {
		subtasks = []Subtask{}
	}

	response := &ReadCardResponse{
		Title:                  cardData.Title,
		Description:            cardData.Description,
		Comments:               comments,
		Subtasks:               subtasks,
		LinkedCards:            cardData.LinkedCards,
		CustomFields:           cardData.CustomFields,
		CreatedAt:              parseTimestamp(cardData.CreatedAt),
		LastModified:           parseTimestamp(cardData.LastModified),
		InCurrentPositionSince: parseTimestamp(cardData.InCurrentPositionSince),
		FirstRequestTime:       parseTimestamp(cardData.FirstRequestTime),
		FirstStartTime:         parseTimestamp(cardData.FirstStartTime),
		FirstEndTime:           parseTimestamp(cardData.FirstEndTime),
		LastRequestTime:        parseTimestamp(cardData.LastRequestTime),
		LastStartTime:          parseTimestamp(cardData.LastStartTime),
		LastEndTime:            parseTimestamp(cardData.LastEndTime),
	}

	// Parse initiative details if present
	if cardData.InitiativeDetails != nil {
		response.PlannedStartDate = cardData.InitiativeDetails.PlannedStartDate
		response.PlannedEndDate = cardData.InitiativeDetails.PlannedEndDate
		response.ActualStartTime = parseTimestamp(cardData.InitiativeDetails.ActualStartTime)
		response.ActualEndTime = parseTimestamp(cardData.InitiativeDetails.ActualEndTime)
	}

	return response, nil
}

func (c *Client) AddCardComment(cardIDOrURL, text string) error {
	cardID, err := c.extractCardID(cardIDOrURL)
	if err != nil {
		return err
	}
	if text == "" {
		return fmt.Errorf("comment text cannot be empty")
	}

	url := fmt.Sprintf("%s/api/v2/cards/%s/comments", c.baseURL, cardID)
	request := AddCommentRequest{Text: text}

	body, err := c.makeAPIRequestWithBody("POST", url, request)
	if err != nil {
		return fmt.Errorf("failed to add comment: %w", err)
	}

	var response AddCommentResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	return nil
}

func (c *Client) getCard(cardID string) (*CardData, error) {
	url := fmt.Sprintf("%s/api/v2/cards/%s", c.baseURL, cardID)

	body, err := c.makeAPIRequest(url)
	if err != nil {
		return nil, err
	}

	var response CardDataResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse card data: %w", err)
	}

	return &response.Data, nil
}

func (c *Client) getCardComments(cardID string) ([]Comment, error) {
	url := fmt.Sprintf("%s/api/v2/cards/%s/comments", c.baseURL, cardID)

	body, err := c.makeAPIRequest(url)
	if err != nil {
		return []Comment{}, nil
	}

	var response CommentsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return []Comment{}, nil
	}

	comments := make([]Comment, len(response.Data))
	for i, commentData := range response.Data {
		comments[i] = Comment{
			ID:        strconv.Itoa(commentData.CommentID),
			Text:      commentData.Text,
			Author:    commentData.AuthorName,
			CreatedAt: parseCommentTimestamp(commentData.CreatedAt),
		}
	}

	return comments, nil
}

func (c *Client) getCardSubtasks(cardID string) ([]Subtask, error) {
	url := fmt.Sprintf("%s/api/v2/cards/%s/subtasks", c.baseURL, cardID)

	body, err := c.makeAPIRequest(url)
	if err != nil {
		return []Subtask{}, nil
	}

	var response SubtasksResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return []Subtask{}, nil
	}

	subtasks := make([]Subtask, len(response.Data))
	for i, subtaskData := range response.Data {
		subtasks[i] = Subtask{
			ID:          strconv.Itoa(subtaskData.SubtaskID),
			Title:       subtaskData.Title,
			Description: subtaskData.Description,
			Completed:   subtaskData.Finished == 1,
		}
	}

	return subtasks, nil
}

func (c *Client) makeAPIRequest(url string) ([]byte, error) {
	return c.makeAPIRequestWithBody("GET", url, nil)
}

// GetCardsWithRetry queries multiple cards with retry logic for rate limiting
// It returns a structured response with metadata about retry attempts
func (c *Client) GetCardsWithRetry(ctx context.Context, filter GetCardsRequest, cfg RetryConfig, failOnPartial bool) (*GetCardsWithRetryResponse, error) {
	// Validate at least one filter is provided
	if len(filter.BoardIDs) == 0 && len(filter.LaneIDs) == 0 &&
		len(filter.WorkflowIDs) == 0 && len(filter.CardIDs) == 0 {
		return nil, fmt.Errorf("at least one filter parameter (board_ids, lane_ids, workflow_ids, or card_ids) must be provided")
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid retry config: %w", err)
	}

	startTime := time.Now()
	response := &GetCardsWithRetryResponse{
		Attempts:     make(map[string]int),
		Completed:    make(map[string]bool),
		PartialError: make(map[string]string),
		Cards:        []CardSummary{},
	}

	// Determine which filter is being used
	if len(filter.BoardIDs) > 0 {
		response.FilterUsed = "board_ids"
		response.FilterValues = filter.BoardIDs
	} else if len(filter.LaneIDs) > 0 {
		response.FilterUsed = "lane_ids"
		response.FilterValues = filter.LaneIDs
	} else if len(filter.WorkflowIDs) > 0 {
		response.FilterUsed = "workflow_ids"
		response.FilterValues = filter.WorkflowIDs
	} else if len(filter.CardIDs) > 0 {
		response.FilterUsed = "card_ids"
		response.FilterValues = filter.CardIDs
	}

	// Build the URL with query parameters
	url := fmt.Sprintf("%s/api/v2/cards", c.baseURL)
	queryParams := []string{}

	if len(filter.BoardIDs) > 0 {
		boardIDs := make([]string, len(filter.BoardIDs))
		for i, id := range filter.BoardIDs {
			boardIDs[i] = strconv.Itoa(id)
		}
		queryParams = append(queryParams, "board_ids="+strings.Join(boardIDs, ","))
	}

	if len(filter.LaneIDs) > 0 {
		laneIDs := make([]string, len(filter.LaneIDs))
		for i, id := range filter.LaneIDs {
			laneIDs[i] = strconv.Itoa(id)
		}
		queryParams = append(queryParams, "lane_ids="+strings.Join(laneIDs, ","))
	}

	if len(filter.WorkflowIDs) > 0 {
		workflowIDs := make([]string, len(filter.WorkflowIDs))
		for i, id := range filter.WorkflowIDs {
			workflowIDs[i] = strconv.Itoa(id)
		}
		queryParams = append(queryParams, "workflow_ids="+strings.Join(workflowIDs, ","))
	}

	if len(filter.CardIDs) > 0 {
		cardIDs := make([]string, len(filter.CardIDs))
		for i, id := range filter.CardIDs {
			cardIDs[i] = strconv.Itoa(id)
		}
		queryParams = append(queryParams, "card_ids="+strings.Join(cardIDs, ","))
	}

	if len(queryParams) > 0 {
		url += "?" + strings.Join(queryParams, "&")
	}

	// Fetch cards with retry
	cardsResult := c.fetchWithRetry(ctx, cfg, "cards", url)
	response.Attempts["cards"] = cardsResult.attempts
	response.RateLimitHits = cardsResult.rateLimitHits
	response.Completed["cards"] = cardsResult.success

	if !cardsResult.success {
		response.PartialError["cards"] = cardsResult.err.Error()
		response.WaitSeconds = time.Since(startTime).Seconds()
		return response, fmt.Errorf("failed to fetch cards: %w", cardsResult.err)
	}

	// Parse cards data - the API returns nested structure: data.pagination and data.data
	var cardsResp GetCardsResponse
	if err := json.Unmarshal(cardsResult.data, &cardsResp); err != nil {
		// Include raw data in error for debugging
		return response, fmt.Errorf("failed to parse cards data: %w (raw: %s)", err, string(cardsResult.data))
	}

	response.Cards = cardsResp.Data.Data
	response.WaitSeconds = time.Since(startTime).Seconds()
	return response, nil
}

func (c *Client) makeAPIRequestWithBody(method, url string, body interface{}) ([]byte, error) {
	var requestBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		requestBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, url, requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("apikey", c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		// Check for rate limiting first
		if resp.StatusCode == http.StatusTooManyRequests {
			retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
			return nil, &RateLimitError{
				StatusCode: resp.StatusCode,
				RetryAfter: retryAfter,
				RawBody:    string(responseBody),
			}
		}

		// Handle other API errors
		var apiErr APIError
		if err := json.Unmarshal(responseBody, &apiErr); err == nil {
			return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, apiErr.Message)
		}
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(responseBody))
	}

	return responseBody, nil
}
