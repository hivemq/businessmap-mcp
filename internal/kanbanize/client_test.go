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
	"testing"
	"time"
)

func TestReadCard_WithLastEndTime(t *testing.T) {
	expectedTime := "2025-07-20T10:00:00Z"
	createdAt := "2024-07-25T10:04:22Z"
	lastModified := "2025-09-29T14:01:56Z"
	plannedStart := "2024-11-01"
	plannedEnd := "2024-11-26"
	actualStart := "2024-12-20T09:59:10Z"
	actualEnd := "2025-09-16T09:18:25Z"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/cards/1001" {
			response := CardDataResponse{
				Data: CardData{
					CardID:                 1001,
					Title:                  "Test Card",
					Description:            "Test Description",
					CreatedAt:              &createdAt,
					LastModified:           &lastModified,
					InCurrentPositionSince: &lastModified,
					FirstRequestTime:       &createdAt,
					FirstStartTime:         &actualStart,
					FirstEndTime:           &actualEnd,
					LastRequestTime:        &createdAt,
					LastStartTime:          &actualStart,
					LastEndTime:            &expectedTime,
					InitiativeDetails: &InitiativeDetails{
						PlannedStartDate: &plannedStart,
						PlannedEndDate:   &plannedEnd,
						ActualStartTime:  &actualStart,
						ActualEndTime:    &actualEnd,
					},
					LinkedCards: []LinkedCard{
						{CardID: 2001, LinkType: "child"},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		} else if r.URL.Path == "/api/v2/cards/1001/comments" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(CommentsResponse{Data: []CommentData{}})
		} else if r.URL.Path == "/api/v2/cards/1001/subtasks" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(SubtasksResponse{Data: []SubtaskData{}})
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")
	response, err := client.ReadCard("1001")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if response.Title != "Test Card" {
		t.Errorf("Expected title 'Test Card', got '%s'", response.Title)
	}

	// Test LastEndTime
	if response.LastEndTime == nil {
		t.Fatal("Expected LastEndTime to be set, got nil")
	}
	expectedParsedTime, _ := time.Parse(time.RFC3339, expectedTime)
	if !response.LastEndTime.Equal(expectedParsedTime) {
		t.Errorf("Expected LastEndTime to be %v, got %v", expectedParsedTime, *response.LastEndTime)
	}

	// Test CreatedAt
	if response.CreatedAt == nil {
		t.Fatal("Expected CreatedAt to be set, got nil")
	}
	expectedCreatedAt, _ := time.Parse(time.RFC3339, createdAt)
	if !response.CreatedAt.Equal(expectedCreatedAt) {
		t.Errorf("Expected CreatedAt to be %v, got %v", expectedCreatedAt, *response.CreatedAt)
	}

	// Test LastModified
	if response.LastModified == nil {
		t.Fatal("Expected LastModified to be set, got nil")
	}
	expectedLastModified, _ := time.Parse(time.RFC3339, lastModified)
	if !response.LastModified.Equal(expectedLastModified) {
		t.Errorf("Expected LastModified to be %v, got %v", expectedLastModified, *response.LastModified)
	}

	// Test ActualStartTime
	if response.ActualStartTime == nil {
		t.Fatal("Expected ActualStartTime to be set, got nil")
	}
	expectedActualStart, _ := time.Parse(time.RFC3339, actualStart)
	if !response.ActualStartTime.Equal(expectedActualStart) {
		t.Errorf("Expected ActualStartTime to be %v, got %v", expectedActualStart, *response.ActualStartTime)
	}

	// Test ActualEndTime
	if response.ActualEndTime == nil {
		t.Fatal("Expected ActualEndTime to be set, got nil")
	}
	expectedActualEnd, _ := time.Parse(time.RFC3339, actualEnd)
	if !response.ActualEndTime.Equal(expectedActualEnd) {
		t.Errorf("Expected ActualEndTime to be %v, got %v", expectedActualEnd, *response.ActualEndTime)
	}

	// Test PlannedStartDate
	if response.PlannedStartDate == nil {
		t.Fatal("Expected PlannedStartDate to be set, got nil")
	}
	if *response.PlannedStartDate != plannedStart {
		t.Errorf("Expected PlannedStartDate to be %s, got %s", plannedStart, *response.PlannedStartDate)
	}

	// Test PlannedEndDate
	if response.PlannedEndDate == nil {
		t.Fatal("Expected PlannedEndDate to be set, got nil")
	}
	if *response.PlannedEndDate != plannedEnd {
		t.Errorf("Expected PlannedEndDate to be %s, got %s", plannedEnd, *response.PlannedEndDate)
	}
}

func TestReadCard_WithNullLastEndTime(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/cards/1002" {
			response := CardDataResponse{
				Data: CardData{
					CardID:      1002,
					Title:       "Test Card Without End Time",
					Description: "Test Description",
					LastEndTime: nil,
					LinkedCards: []LinkedCard{},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		} else if r.URL.Path == "/api/v2/cards/1002/comments" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(CommentsResponse{Data: []CommentData{}})
		} else if r.URL.Path == "/api/v2/cards/1002/subtasks" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(SubtasksResponse{Data: []SubtaskData{}})
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")
	response, err := client.ReadCard("1002")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if response.LastEndTime != nil {
		t.Errorf("Expected LastEndTime to be nil, got %v", *response.LastEndTime)
	}

	// Test that all timestamp fields are nil when not provided
	if response.CreatedAt != nil {
		t.Errorf("Expected CreatedAt to be nil, got %v", *response.CreatedAt)
	}
	if response.LastModified != nil {
		t.Errorf("Expected LastModified to be nil, got %v", *response.LastModified)
	}
	if response.ActualStartTime != nil {
		t.Errorf("Expected ActualStartTime to be nil, got %v", *response.ActualStartTime)
	}
	if response.ActualEndTime != nil {
		t.Errorf("Expected ActualEndTime to be nil, got %v", *response.ActualEndTime)
	}
}

func TestReadCard_WithLinkedCards(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/cards/1003" {
			response := CardDataResponse{
				Data: CardData{
					CardID:      1003,
					Title:       "Parent Card",
					Description: "Card with linked cards",
					LinkedCards: []LinkedCard{
						{CardID: 2001, LinkType: "child", Title: "Child Card 1"},
						{CardID: 2002, LinkType: "child", Title: "Child Card 2"},
						{CardID: 3001, LinkType: "parent", Title: "Parent Card"},
						{CardID: 4001, LinkType: "predecessor", Title: "Predecessor Card"},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		} else if r.URL.Path == "/api/v2/cards/1003/comments" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(CommentsResponse{Data: []CommentData{}})
		} else if r.URL.Path == "/api/v2/cards/1003/subtasks" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(SubtasksResponse{Data: []SubtaskData{}})
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")
	response, err := client.ReadCard("1003")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(response.LinkedCards) != 4 {
		t.Fatalf("Expected 4 linked cards, got %d", len(response.LinkedCards))
	}

	// Check child cards
	childCount := 0
	for _, card := range response.LinkedCards {
		if card.LinkType == "child" {
			childCount++
		}
	}
	if childCount != 2 {
		t.Errorf("Expected 2 child cards, got %d", childCount)
	}

	// Check first child card
	if response.LinkedCards[0].CardID != 2001 {
		t.Errorf("Expected first linked card ID to be 2001, got %d", response.LinkedCards[0].CardID)
	}
	if response.LinkedCards[0].LinkType != "child" {
		t.Errorf("Expected first linked card type to be 'child', got '%s'", response.LinkedCards[0].LinkType)
	}
}

func TestReadCard_WithEmptyLinkedCards(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/cards/1004" {
			response := CardDataResponse{
				Data: CardData{
					CardID:      1004,
					Title:       "Isolated Card",
					Description: "Card with no linked cards",
					LinkedCards: []LinkedCard{},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		} else if r.URL.Path == "/api/v2/cards/1004/comments" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(CommentsResponse{Data: []CommentData{}})
		} else if r.URL.Path == "/api/v2/cards/1004/subtasks" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(SubtasksResponse{Data: []SubtaskData{}})
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")
	response, err := client.ReadCard("1004")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(response.LinkedCards) != 0 {
		t.Errorf("Expected 0 linked cards, got %d", len(response.LinkedCards))
	}
}

func TestReadCard_WithInvalidLastEndTime(t *testing.T) {
	invalidTime := "invalid-time-format"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/cards/1005" {
			response := CardDataResponse{
				Data: CardData{
					CardID:      1005,
					Title:       "Test Card",
					Description: "Test Description",
					LastEndTime: &invalidTime,
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		} else if r.URL.Path == "/api/v2/cards/1005/comments" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(CommentsResponse{Data: []CommentData{}})
		} else if r.URL.Path == "/api/v2/cards/1005/subtasks" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(SubtasksResponse{Data: []SubtaskData{}})
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")
	response, err := client.ReadCard("1005")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Invalid time should result in nil LastEndTime
	if response.LastEndTime != nil {
		t.Errorf("Expected LastEndTime to be nil for invalid time format, got %v", *response.LastEndTime)
	}
}

func TestReadCard_WithCommentTimestamps(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/cards/1006" {
			response := CardDataResponse{
				Data: CardData{
					CardID:      1006,
					Title:       "Card with Comments",
					Description: "Test Description",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		} else if r.URL.Path == "/api/v2/cards/1006/comments" {
			response := CommentsResponse{
				Data: []CommentData{
					{
						CommentID:  1,
						Text:       "Comment 1 - RFC3339 format",
						AuthorName: "John Doe",
						CreatedAt:  "2024-01-15T10:30:00Z",
					},
					{
						CommentID:  2,
						Text:       "Comment 2 - Space-separated format",
						AuthorName: "Jane Doe",
						CreatedAt:  "2024-01-16 14:45:30",
					},
					{
						CommentID:  3,
						Text:       "Comment 3 - T-separated without timezone",
						AuthorName: "Bob Smith",
						CreatedAt:  "2024-01-17T09:15:00",
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		} else if r.URL.Path == "/api/v2/cards/1006/subtasks" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(SubtasksResponse{Data: []SubtaskData{}})
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")
	response, err := client.ReadCard("1006")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(response.Comments) != 3 {
		t.Fatalf("Expected 3 comments, got %d", len(response.Comments))
	}

	// Test first comment (RFC3339 format)
	comment1 := response.Comments[0]
	if comment1.Author != "John Doe" {
		t.Errorf("Expected author 'John Doe', got '%s'", comment1.Author)
	}
	if comment1.CreatedAt.IsZero() {
		t.Error("Expected Comment 1 CreatedAt to be parsed, got zero time")
	}
	expectedTime1, _ := time.Parse(time.RFC3339, "2024-01-15T10:30:00Z")
	if !comment1.CreatedAt.Equal(expectedTime1) {
		t.Errorf("Expected Comment 1 CreatedAt to be %v, got %v", expectedTime1, comment1.CreatedAt)
	}

	// Test second comment (space-separated format)
	comment2 := response.Comments[1]
	if comment2.CreatedAt.IsZero() {
		t.Error("Expected Comment 2 CreatedAt to be parsed, got zero time")
	}
	expectedTime2, _ := time.Parse("2006-01-02 15:04:05", "2024-01-16 14:45:30")
	if !comment2.CreatedAt.Equal(expectedTime2) {
		t.Errorf("Expected Comment 2 CreatedAt to be %v, got %v", expectedTime2, comment2.CreatedAt)
	}

	// Test third comment (T-separated without timezone)
	comment3 := response.Comments[2]
	if comment3.CreatedAt.IsZero() {
		t.Error("Expected Comment 3 CreatedAt to be parsed, got zero time")
	}
	expectedTime3, _ := time.Parse("2006-01-02T15:04:05", "2024-01-17T09:15:00")
	if !comment3.CreatedAt.Equal(expectedTime3) {
		t.Errorf("Expected Comment 3 CreatedAt to be %v, got %v", expectedTime3, comment3.CreatedAt)
	}
}

func TestReadCard_WithEmptyCommentTimestamp(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/cards/1007" {
			response := CardDataResponse{
				Data: CardData{
					CardID:      1007,
					Title:       "Card with Empty Comment Timestamp",
					Description: "Test Description",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		} else if r.URL.Path == "/api/v2/cards/1007/comments" {
			response := CommentsResponse{
				Data: []CommentData{
					{
						CommentID:  1,
						Text:       "Comment without timestamp",
						AuthorName: "Test User",
						CreatedAt:  "",
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		} else if r.URL.Path == "/api/v2/cards/1007/subtasks" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(SubtasksResponse{Data: []SubtaskData{}})
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")
	response, err := client.ReadCard("1007")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(response.Comments) != 1 {
		t.Fatalf("Expected 1 comment, got %d", len(response.Comments))
	}

	// Empty timestamp should result in zero time
	if !response.Comments[0].CreatedAt.IsZero() {
		t.Errorf("Expected CreatedAt to be zero time for empty timestamp, got %v", response.Comments[0].CreatedAt)
	}
}

func TestReadCard_WithCustomFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/cards/1008" {
			response := CardDataResponse{
				Data: CardData{
					CardID:      1008,
					Title:       "Card with Custom Fields",
					Description: "Test Description",
					CustomFields: []CustomField{
						{FieldID: 1, Name: "priority", Value: "High"},
						{FieldID: 2, Name: "team", Value: "Engineering"},
						{FieldID: 3, Name: "sprint", Value: float64(23)},
						{FieldID: 4, Name: "story_points", Value: float64(8)},
						{FieldID: 5, Name: "is_blocked", Value: false},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		} else if r.URL.Path == "/api/v2/cards/1008/comments" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(CommentsResponse{Data: []CommentData{}})
		} else if r.URL.Path == "/api/v2/cards/1008/subtasks" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(SubtasksResponse{Data: []SubtaskData{}})
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")
	response, err := client.ReadCard("1008")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if response.CustomFields == nil {
		t.Fatal("Expected CustomFields to be set, got nil")
	}

	if len(response.CustomFields) != 5 {
		t.Errorf("Expected 5 custom fields, got %d", len(response.CustomFields))
	}

	// Test first field (priority)
	if response.CustomFields[0].Name != "priority" {
		t.Errorf("Expected first field name to be 'priority', got '%s'", response.CustomFields[0].Name)
	}
	if priority, ok := response.CustomFields[0].Value.(string); !ok || priority != "High" {
		t.Errorf("Expected priority to be 'High', got %v", response.CustomFields[0].Value)
	}

	// Test numeric field (sprint)
	if response.CustomFields[2].Name != "sprint" {
		t.Errorf("Expected third field name to be 'sprint', got '%s'", response.CustomFields[2].Name)
	}
	if sprint, ok := response.CustomFields[2].Value.(float64); !ok || sprint != 23 {
		t.Errorf("Expected sprint to be 23, got %v", response.CustomFields[2].Value)
	}

	// Test boolean field (is_blocked)
	if response.CustomFields[4].Name != "is_blocked" {
		t.Errorf("Expected fifth field name to be 'is_blocked', got '%s'", response.CustomFields[4].Name)
	}
	if blocked, ok := response.CustomFields[4].Value.(bool); !ok || blocked != false {
		t.Errorf("Expected is_blocked to be false, got %v", response.CustomFields[4].Value)
	}
}

func TestReadCard_WithEmptyCustomFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/cards/1009" {
			response := CardDataResponse{
				Data: CardData{
					CardID:       1009,
					Title:        "Card without Custom Fields",
					Description:  "Test Description",
					CustomFields: []CustomField{},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		} else if r.URL.Path == "/api/v2/cards/1009/comments" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(CommentsResponse{Data: []CommentData{}})
		} else if r.URL.Path == "/api/v2/cards/1009/subtasks" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(SubtasksResponse{Data: []SubtaskData{}})
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")
	response, err := client.ReadCard("1009")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(response.CustomFields) != 0 {
		t.Errorf("Expected 0 custom fields, got %d", len(response.CustomFields))
	}
}

func TestReadCard_WithNullCustomFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/cards/1010" {
			response := CardDataResponse{
				Data: CardData{
					CardID:       1010,
					Title:        "Card with null Custom Fields",
					Description:  "Test Description",
					CustomFields: nil,
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		} else if r.URL.Path == "/api/v2/cards/1010/comments" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(CommentsResponse{Data: []CommentData{}})
		} else if r.URL.Path == "/api/v2/cards/1010/subtasks" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(SubtasksResponse{Data: []SubtaskData{}})
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")
	response, err := client.ReadCard("1010")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Null custom fields should result in nil or empty slice
	if len(response.CustomFields) != 0 {
		t.Errorf("Expected CustomFields to be empty, got %v", response.CustomFields)
	}
}

func TestGetCardsWithRetry_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/cards" {
			// Check query parameters
			query := r.URL.Query()
			if boardIDs := query.Get("board_ids"); boardIDs != "1,2,3" {
				t.Errorf("Expected board_ids=1,2,3, got %s", boardIDs)
			}

			// Return raw JSON response matching actual API format
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"data": {
					"pagination": {
						"all_pages": 1,
						"current_page": 1,
						"results_per_page": 200
					},
					"data": [
						{
							"card_id": 101,
							"title": "Card 1",
							"description": "Description 1",
							"board_id": 1,
							"lane_id": 10,
							"workflow_id": 100
						},
						{
							"card_id": 102,
							"title": "Card 2",
							"description": "Description 2",
							"board_id": 2,
							"lane_id": 20,
							"workflow_id": 200
						}
					]
				}
			}`))
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")
	filter := GetCardsRequest{
		BoardIDs: []int{1, 2, 3},
	}
	cfg := RetryConfig{
		MaxAttempts:       5,
		InitialDelay:      100 * time.Millisecond,
		MaxDelay:          1 * time.Second,
		Multiplier:        2.0,
		RespectRetryAfter: true,
		TotalWaitCap:      10 * time.Second,
	}

	ctx := context.Background()
	response, err := client.GetCardsWithRetry(ctx, filter, cfg, false)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if response.FilterUsed != "board_ids" {
		t.Errorf("Expected FilterUsed=board_ids, got %s", response.FilterUsed)
	}

	if len(response.FilterValues) != 3 {
		t.Errorf("Expected 3 filter values, got %d", len(response.FilterValues))
	}

	if len(response.Cards) != 2 {
		t.Fatalf("Expected 2 cards, got %d", len(response.Cards))
	}

	// Check that both cards are present (order may vary due to map iteration)
	foundCard101 := false
	foundCard102 := false
	for _, card := range response.Cards {
		if card.CardID == 101 && card.Title == "Card 1" {
			foundCard101 = true
		}
		if card.CardID == 102 && card.Title == "Card 2" {
			foundCard102 = true
		}
	}
	if !foundCard101 {
		t.Error("Expected to find card with ID 101 and title 'Card 1'")
	}
	if !foundCard102 {
		t.Error("Expected to find card with ID 102 and title 'Card 2'")
	}

	if response.Attempts["cards"] != 1 {
		t.Errorf("Expected 1 attempt, got %d", response.Attempts["cards"])
	}

	if !response.Completed["cards"] {
		t.Error("Expected cards to be completed")
	}

	if response.RateLimitHits != 0 {
		t.Errorf("Expected 0 rate limit hits, got %d", response.RateLimitHits)
	}
}

func TestGetCardsWithRetry_NoFilterProvided(t *testing.T) {
	client := NewClient("http://example.com", "test-api-key")
	filter := GetCardsRequest{}
	cfg := DefaultRetryConfig()

	ctx := context.Background()
	_, err := client.GetCardsWithRetry(ctx, filter, cfg, false)

	if err == nil {
		t.Fatal("Expected error when no filter is provided")
	}

	expectedMsg := "at least one filter parameter"
	if !contains(err.Error(), expectedMsg) {
		t.Errorf("Expected error to contain '%s', got %s", expectedMsg, err.Error())
	}
}

func TestGetCardsWithRetry_RateLimitThenSuccess(t *testing.T) {
	attemptCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/cards" {
			attemptCount++
			if attemptCount == 1 {
				// First attempt: return rate limit error
				w.Header().Set("Retry-After", "1")
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"error": "rate limit exceeded"}`))
				return
			}
			// Second attempt: success
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"data": {
					"pagination": {
						"all_pages": 1,
						"current_page": 1,
						"results_per_page": 200
					},
					"data": [
						{
							"card_id": 201,
							"title": "Card After Retry",
							"description": "Success",
							"board_id": 5,
							"lane_id": 50,
							"workflow_id": 500
						}
					]
				}
			}`))
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")
	filter := GetCardsRequest{
		LaneIDs: []int{50, 51},
	}
	cfg := RetryConfig{
		MaxAttempts:       5,
		InitialDelay:      100 * time.Millisecond,
		MaxDelay:          1 * time.Second,
		Multiplier:        2.0,
		RespectRetryAfter: true,
		TotalWaitCap:      10 * time.Second,
	}

	ctx := context.Background()
	response, err := client.GetCardsWithRetry(ctx, filter, cfg, false)

	if err != nil {
		t.Fatalf("Expected no error after retry, got %v", err)
	}

	if response.FilterUsed != "lane_ids" {
		t.Errorf("Expected FilterUsed=lane_ids, got %s", response.FilterUsed)
	}

	if response.Attempts["cards"] != 2 {
		t.Errorf("Expected 2 attempts, got %d", response.Attempts["cards"])
	}

	if response.RateLimitHits != 1 {
		t.Errorf("Expected 1 rate limit hit, got %d", response.RateLimitHits)
	}

	if len(response.Cards) != 1 {
		t.Fatalf("Expected 1 card, got %d", len(response.Cards))
	}

	if response.Cards[0].CardID != 201 {
		t.Errorf("Expected card ID to be 201, got %d", response.Cards[0].CardID)
	}
}

// Helper function for string contains
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && stringContains(s, substr)))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
