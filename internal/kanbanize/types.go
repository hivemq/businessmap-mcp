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
	"fmt"
	"time"
)

type APIError struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// RateLimitError represents an HTTP 429 rate limit error with retry information
type RateLimitError struct {
	StatusCode int
	RetryAfter time.Duration // parsed from Retry-After header (seconds or HTTP-date)
	RawBody    string        // original body for diagnostics
}

func (e *RateLimitError) Error() string {
	if e.RetryAfter > 0 {
		return fmt.Sprintf("rate limit exceeded (HTTP %d): retry after %v", e.StatusCode, e.RetryAfter)
	}
	return fmt.Sprintf("rate limit exceeded (HTTP %d)", e.StatusCode)
}

type ReadCardResponse struct {
	Title                  string        `json:"title"`
	Description            string        `json:"description"`
	Subtasks               []Subtask     `json:"subtasks"`
	Comments               []Comment     `json:"comments"`
	LinkedCards            []LinkedCard  `json:"linked_cards"`
	CustomFields           []CustomField `json:"custom_fields,omitempty"`
	CreatedAt              *time.Time    `json:"created_at,omitempty"`
	LastModified           *time.Time    `json:"last_modified,omitempty"`
	InCurrentPositionSince *time.Time    `json:"in_current_position_since,omitempty"`
	FirstRequestTime       *time.Time    `json:"first_request_time,omitempty"`
	FirstStartTime         *time.Time    `json:"first_start_time,omitempty"`
	FirstEndTime           *time.Time    `json:"first_end_time,omitempty"`
	LastRequestTime        *time.Time    `json:"last_request_time,omitempty"`
	LastStartTime          *time.Time    `json:"last_start_time,omitempty"`
	LastEndTime            *time.Time    `json:"last_end_time,omitempty"`
	PlannedStartDate       *string       `json:"planned_start_date,omitempty"`
	PlannedEndDate         *string       `json:"planned_end_date,omitempty"`
	ActualStartTime        *time.Time    `json:"actual_start_time,omitempty"`
	ActualEndTime          *time.Time    `json:"actual_end_time,omitempty"`
}

type Subtask struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Completed   bool   `json:"completed"`
}

type CustomField struct {
	FieldID int         `json:"field_id"`
	Name    string      `json:"name"`
	Value   interface{} `json:"value"`
}

type Comment struct {
	ID        string    `json:"id"`
	Text      string    `json:"text"`
	Author    string    `json:"author"`
	CreatedAt time.Time `json:"created_at"`
}

type CardDataResponse struct {
	Data CardData `json:"data"`
}

type InitiativeDetails struct {
	PlannedStartDate *string `json:"planned_start_date"`
	PlannedEndDate   *string `json:"planned_end_date"`
	ActualStartTime  *string `json:"actual_start_time"`
	ActualEndTime    *string `json:"actual_end_time"`
}

type CardData struct {
	CardID                 int                `json:"card_id"`
	Title                  string             `json:"title"`
	Description            string             `json:"description"`
	LinkedCards            []LinkedCard       `json:"linked_cards"`
	CustomFields           []CustomField      `json:"custom_fields"`
	CreatedAt              *string            `json:"created_at"`
	LastModified           *string            `json:"last_modified"`
	InCurrentPositionSince *string            `json:"in_current_position_since"`
	FirstRequestTime       *string            `json:"first_request_time"`
	FirstStartTime         *string            `json:"first_start_time"`
	FirstEndTime           *string            `json:"first_end_time"`
	LastRequestTime        *string            `json:"last_request_time"`
	LastStartTime          *string            `json:"last_start_time"`
	LastEndTime            *string            `json:"last_end_time"`
	InitiativeDetails      *InitiativeDetails `json:"initiative_details"`
}

type LinkedCard struct {
	CardID   int    `json:"card_id"`
	LinkType string `json:"link_type"`
	Title    string `json:"title,omitempty"`
}

type CommentsResponse struct {
	Data []CommentData `json:"data"`
}

type CommentData struct {
	CommentID  int    `json:"comment_id"`
	Text       string `json:"text"`
	AuthorName string `json:"author_name"`
	CreatedAt  string `json:"created_at"`
}

type SubtasksResponse struct {
	Data []SubtaskData `json:"data"`
}

type SubtaskData struct {
	SubtaskID   int    `json:"subtask_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Position    int    `json:"position"`
	Finished    int    `json:"finished"`
}

type AddCommentRequest struct {
	Text string `json:"text"`
}

type AddCommentResponse struct {
	Data AddCommentData `json:"data"`
}

type AddCommentData struct {
	CommentID   int    `json:"comment_id"`
	Text        string `json:"text"`
	AuthorName  string `json:"author_name"`
	CreatedDate string `json:"created_date"`
}

// ReadCardWithRetryResponse wraps the card data with retry metadata
type ReadCardWithRetryResponse struct {
	CardID         string                    `json:"card_id"`
	Attempts       map[string]int            `json:"attempts"`
	WaitSeconds    float64                   `json:"wait_seconds"`
	RateLimitHits  int                       `json:"rate_limit_hits"`
	Completed      map[string]bool           `json:"completed"`
	PartialError   map[string]string         `json:"partial_error,omitempty"`
	Data           *ReadCardResponse         `json:"data"`
}
