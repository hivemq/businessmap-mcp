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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestReadCard_WithLastEndTime(t *testing.T) {
	expectedTime := "2025-07-20T10:00:00Z"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/cards/1001" {
			response := CardDataResponse{
				Data: CardData{
					CardID:      1001,
					Title:       "Test Card",
					Description: "Test Description",
					LastEndTime: &expectedTime,
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

	if response.LastEndTime == nil {
		t.Fatal("Expected LastEndTime to be set, got nil")
	}

	expectedParsedTime, _ := time.Parse(time.RFC3339, expectedTime)
	if !response.LastEndTime.Equal(expectedParsedTime) {
		t.Errorf("Expected LastEndTime to be %v, got %v", expectedParsedTime, *response.LastEndTime)
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
