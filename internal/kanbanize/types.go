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