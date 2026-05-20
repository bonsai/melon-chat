package main

import "time"

type Conversation struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Message struct {
	ID             int64     `json:"id"`
	ConversationID string    `json:"conversation_id"`
	Role           string    `json:"role"`
	ContentType    string    `json:"content_type"`
	Content        string    `json:"content"`
	ImageURL       string    `json:"image_url,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

type ChatRequest struct {
	ConversationID string `json:"conversation_id"`
	ContentType    string `json:"content_type"`
	Content        string `json:"content"`
	Image          string `json:"image,omitempty"`
}

type MessagePostRequest struct {
	ConversationID string `json:"conversation_id"`
	Role           string `json:"role"`
	ContentType    string `json:"content_type"`
	Content        string `json:"content"`
	ImageURL       string `json:"image_url,omitempty"`
}

type SSEEvent struct {
	Type    string `json:"type"`
	Payload any    `json:"payload"`
}
