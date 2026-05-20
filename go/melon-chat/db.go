package main

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

var db *sql.DB

func initDB(dbURL string) error {
	var err error
	db, err = sql.Open("postgres", dbURL)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	if err = db.Ping(); err != nil {
		return fmt.Errorf("ping: %w", err)
	}
	return migrate()
}

func closeDB() {
	if db != nil {
		db.Close()
	}
}

func migrate() error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS conversations (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("create conversations: %w", err)
	}
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS messages (
			id BIGSERIAL PRIMARY KEY,
			conversation_id TEXT NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
			role TEXT NOT NULL,
			content_type TEXT NOT NULL DEFAULT 'text',
			content TEXT NOT NULL DEFAULT '',
			image_url TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("create messages: %w", err)
	}
	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_messages_conv ON messages(conversation_id)`)
	return err
}

func createConversation(title string) (*Conversation, error) {
	id := uuid.New().String()[:8]
	if title == "" {
		title = "New Chat"
	}
	now := time.Now()
	_, err := db.Exec(
		`INSERT INTO conversations (id, title, created_at, updated_at) VALUES ($1, $2, $3, $4)`,
		id, title, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("insert conv: %w", err)
	}
	return &Conversation{ID: id, Title: title, CreatedAt: now, UpdatedAt: now}, nil
}

func getConversations() ([]Conversation, error) {
	rows, err := db.Query(
		`SELECT id, title, created_at, updated_at FROM conversations ORDER BY updated_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("query convs: %w", err)
	}
	defer rows.Close()

	var convs []Conversation
	for rows.Next() {
		var c Conversation
		if err := rows.Scan(&c.ID, &c.Title, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan conv: %w", err)
		}
		convs = append(convs, c)
	}
	return convs, rows.Err()
}

func getConversation(id string) (*Conversation, error) {
	var c Conversation
	err := db.QueryRow(
		`SELECT id, title, created_at, updated_at FROM conversations WHERE id = $1`, id,
	).Scan(&c.ID, &c.Title, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("query conv: %w", err)
	}
	return &c, nil
}

func deleteConversation(id string) error {
	_, err := db.Exec(`DELETE FROM conversations WHERE id = $1`, id)
	return err
}

func addMessage(convID, role, contentType, content, imageURL string) (*Message, error) {
	now := time.Now()
	var msgID int64
	err := db.QueryRow(
		`INSERT INTO messages (conversation_id, role, content_type, content, image_url, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id`,
		convID, role, contentType, content, imageURL, now,
	).Scan(&msgID)
	if err != nil {
		return nil, fmt.Errorf("insert msg: %w", err)
	}

	_, _ = db.Exec(
		`UPDATE conversations SET updated_at = $1 WHERE id = $2`, now, convID,
	)

	return &Message{
		ID:             msgID,
		ConversationID: convID,
		Role:           role,
		ContentType:    contentType,
		Content:        content,
		ImageURL:       imageURL,
		CreatedAt:      now,
	}, nil
}

func getMessages(convID string) ([]Message, error) {
	rows, err := db.Query(
		`SELECT id, conversation_id, role, content_type, content, image_url, created_at
		 FROM messages WHERE conversation_id = $1 ORDER BY created_at ASC`,
		convID,
	)
	if err != nil {
		return nil, fmt.Errorf("query msgs: %w", err)
	}
	defer rows.Close()

	var msgs []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.Role, &m.ContentType, &m.Content, &m.ImageURL, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan msg: %w", err)
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}
