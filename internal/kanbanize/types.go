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
	"time"
)

type APIError struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

type ReadCardResponse struct {
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Subtasks    []Subtask `json:"subtasks"`
	Comments    []Comment `json:"comments"`
}

type Subtask struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Completed   bool   `json:"completed"`
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

type CardData struct {
	CardID      int    `json:"card_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

type CommentsResponse struct {
	Data []CommentData `json:"data"`
}

type CommentData struct {
	CommentID   int    `json:"comment_id"`
	Text        string `json:"text"`
	AuthorName  string `json:"author_name"`
	CreatedDate string `json:"created_date"`
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