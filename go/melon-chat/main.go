package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	loadEnv()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://localhost:5432/melon-chat?sslmode=disable"
	}
	if err := initDB(dbURL); err != nil {
		log.Fatalf("DB init: %v", err)
	}
	defer closeDB()

	broker := newSSEBroker()
	go broker.run()

	mux := http.NewServeMux()

	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, "static/index.html")
	})

	mux.HandleFunc("POST /api/chat", handleChatPost(broker))
	mux.HandleFunc("GET /api/events", handleSSE(broker))
	mux.HandleFunc("GET /api/conversations", handleConversations)
	mux.HandleFunc("GET /api/conversations/{id}", handleConversation)
	mux.HandleFunc("DELETE /api/conversations/{id}", handleConversationDelete)
	mux.HandleFunc("POST /api/message", handleMessagePost(broker))
	mux.HandleFunc("POST /api/chat/{id}", handleChatReply(broker))

	port := getEnv("PORT", getEnv("MELON_CHAT_PORT", "8086"))
	addr := ":" + port
	log.Printf("melon-chat starting on %s (Neon/PostgreSQL)", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
