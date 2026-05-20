package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

func handleChatPost(broker *SSEBroker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}

		if req.ContentType == "" {
			req.ContentType = "text"
		}

		convID := req.ConversationID
		if convID == "" {
			conv, err := createConversation("")
			if err != nil {
				log.Printf("create conv: %v", err)
				http.Error(w, "server error", http.StatusInternalServerError)
				return
			}
			convID = conv.ID
		} else {
			conv, err := getConversation(convID)
			if err != nil {
				log.Printf("get conv: %v", err)
				http.Error(w, "server error", http.StatusInternalServerError)
				return
			}
			if conv == nil {
				http.Error(w, "conversation not found", http.StatusNotFound)
				return
			}
		}

		imageURL := ""
		if req.Image != "" {
			imageURL = req.Image
		}

		msg, err := addMessage(convID, "user", req.ContentType, req.Content, imageURL)
		if err != nil {
			log.Printf("add msg: %v", err)
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}

		conv, _ := getConversation(convID)
		if conv != nil {
			if conv.Title == "New Chat" && req.ContentType == "text" && len(req.Content) > 0 {
				title := req.Content
				if len(title) > 50 {
					title = title[:50]
				}
				_, _ = db.Exec(`UPDATE conversations SET title = $1 WHERE id = $2`, title, convID)
			}
		}

		broker.Publish("message", msg)

		if req.Image != "" {
			go forwardToJacketEye(convID, req.Image, broker)
		}

		resp := map[string]any{
			"message":         msg,
			"conversation_id": convID,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

func handleChatReply(broker *SSEBroker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		convID := r.PathValue("id")
		var req ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}

		if req.ContentType == "" {
			req.ContentType = "text"
		}

		imageURL := ""
		if req.Image != "" {
			imageURL = req.Image
		}

		msg, err := addMessage(convID, "user", req.ContentType, req.Content, imageURL)
		if err != nil {
			log.Printf("add msg: %v", err)
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}

		broker.Publish("message", msg)

		if req.Image != "" {
			go forwardToJacketEye(convID, req.Image, broker)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(msg)
	}
}

func handleSSE(broker *SSEBroker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		ch := broker.Subscribe()
		defer broker.Unsubscribe(ch)

		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		fmt.Fprintf(w, "event: connected\ndata: {}\n\n")
		flusher.Flush()

		for {
			select {
			case <-r.Context().Done():
				return
			case <-ticker.C:
				fmt.Fprintf(w, ": keepalive\n\n")
				flusher.Flush()
			case msg, ok := <-ch:
				if !ok {
					return
				}
				fmt.Fprint(w, msg)
				flusher.Flush()
			}
		}
	}
}

func handleConversations(w http.ResponseWriter, r *http.Request) {
	convs, err := getConversations()
	if err != nil {
		log.Printf("get convs: %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	if convs == nil {
		convs = []Conversation{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(convs)
}

func handleConversation(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	conv, err := getConversation(id)
	if err != nil {
		log.Printf("get conv: %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	if conv == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	msgs, err := getMessages(id)
	if err != nil {
		log.Printf("get msgs: %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	if msgs == nil {
		msgs = []Message{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"conversation": conv,
		"messages":     msgs,
	})
}

func handleConversationDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := deleteConversation(id); err != nil {
		log.Printf("del conv: %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func handleMessagePost(broker *SSEBroker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req MessagePostRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}

		if req.ConversationID == "" {
			http.Error(w, "conversation_id required", http.StatusBadRequest)
			return
		}
		if req.Role == "" {
			req.Role = "assistant"
		}
		if req.ContentType == "" {
			req.ContentType = "text"
		}

		msg, err := addMessage(req.ConversationID, req.Role, req.ContentType, req.Content, req.ImageURL)
		if err != nil {
			log.Printf("add msg: %v", err)
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}

		broker.Publish("message", msg)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(msg)
	}
}

func forwardToJacketEye(convID, imageData string, broker *SSEBroker) {
	jacketURL := os.Getenv("JACKET_EYE_URL")
	if jacketURL == "" {
		jacketURL = "http://localhost:8085"
	}

	resp, err := http.Post(jacketURL+"/api/jacket/scan", "application/json",
		strings.NewReader(fmt.Sprintf(`{"image":"%s"}`, imageData)),
	)
	if err != nil {
		log.Printf("jacket-eye call: %v", err)
		_, _ = addMessage(convID, "assistant", "text",
			"⚠️ jacket-eye に接続できませんでした", "")
		broker.Publish("message", &Message{
			ConversationID: convID,
			Role:           "assistant",
			ContentType:    "text",
			Content:        "⚠️ jacket-eye に接続できませんでした",
		})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("read jacket-eye: %v", err)
		return
	}

	var result struct {
		Artist     string   `json:"artist"`
		Album      string   `json:"album"`
		Songs      []string `json:"songs"`
		Year       int      `json:"year"`
		Confidence float64  `json:"confidence"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		log.Printf("parse jacket-eye: %v", err)
		return
	}

	content := fmt.Sprintf("🎵 *%s* — %s", result.Artist, result.Album)
	if len(result.Songs) > 0 {
		content += "\n\n📝 収録曲:\n"
		for i, s := range result.Songs {
			if i >= 5 {
				content += fmt.Sprintf("  …他 %d 曲", len(result.Songs)-i)
				break
			}
			content += fmt.Sprintf("  %d. %s\n", i+1, s)
		}
	}
	if result.Year > 0 {
		content += fmt.Sprintf("\n📅 %d", result.Year)
	}
	content += fmt.Sprintf("\n\n信頼度: %.0f%%", result.Confidence*100)

	msg, err := addMessage(convID, "assistant", "recognition_result", content, "")
	if err != nil {
		log.Printf("add jacket result: %v", err)
		return
	}
	broker.Publish("message", msg)

	ledBoardURL := os.Getenv("LED_BOARD_URL")
	if ledBoardURL != "" {
		ledMsg := fmt.Sprintf("🎵 %s — %s", result.Artist, result.Album)
		http.Post(ledBoardURL+"/api/message", "application/json",
			strings.NewReader(fmt.Sprintf(`{"text":"%s","source":"jacket-eye"}`, ledMsg)),
		)
	}
}
