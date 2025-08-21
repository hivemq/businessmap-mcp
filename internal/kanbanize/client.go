package kanbanize

import (
	"bytes"
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

	return &ReadCardResponse{
		Title:       cardData.Title,
		Description: cardData.Description,
		Comments:    comments,
		Subtasks:    subtasks,
	}, nil
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
		createdAt, _ := time.Parse("2006-01-02 15:04:05", commentData.CreatedDate)
		comments[i] = Comment{
			ID:        strconv.Itoa(commentData.CommentID),
			Text:      commentData.Text,
			Author:    commentData.AuthorName,
			CreatedAt: createdAt,
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
		var apiErr APIError
		if err := json.Unmarshal(responseBody, &apiErr); err == nil {
			return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, apiErr.Message)
		}
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(responseBody))
	}

	return responseBody, nil
}