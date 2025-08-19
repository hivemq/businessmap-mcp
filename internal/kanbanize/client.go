package kanbanize

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

func (c *Client) ReadCard(cardID string) (*ReadCardResponse, error) {
	if cardID == "" {
		return nil, fmt.Errorf("card ID cannot be empty")
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
	req, err := http.NewRequest("GET", url, nil)
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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var apiErr APIError
		if err := json.Unmarshal(body, &apiErr); err == nil {
			return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, apiErr.Message)
		}
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}