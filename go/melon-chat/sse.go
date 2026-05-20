package main

import (
	"fmt"
	"sync"
)

type SSEBroker struct {
	clients    map[chan string]bool
	register   chan chan string
	unregister chan chan string
	broadcast  chan string
	mu         sync.RWMutex
}

func newSSEBroker() *SSEBroker {
	return &SSEBroker{
		clients:    make(map[chan string]bool),
		register:   make(chan chan string),
		unregister: make(chan chan string),
		broadcast:  make(chan string, 256),
	}
}

func (b *SSEBroker) run() {
	for {
		select {
		case ch := <-b.register:
			b.mu.Lock()
			b.clients[ch] = true
			b.mu.Unlock()
		case ch := <-b.unregister:
			b.mu.Lock()
			if _, ok := b.clients[ch]; ok {
				delete(b.clients, ch)
				close(ch)
			}
			b.mu.Unlock()
		case msg := <-b.broadcast:
			b.mu.RLock()
			for ch := range b.clients {
				select {
				case ch <- msg:
				default:
				}
			}
			b.mu.RUnlock()
		}
	}
}

func (b *SSEBroker) Subscribe() chan string {
	ch := make(chan string, 64)
	b.register <- ch
	return ch
}

func (b *SSEBroker) Unsubscribe(ch chan string) {
	b.unregister <- ch
}

func (b *SSEBroker) Publish(eventType string, payload any) {
	msg := fmt.Sprintf("event: message\ndata: %s\n\n", toJSON(map[string]any{
		"type":    eventType,
		"payload": payload,
	}))
	b.broadcast <- msg
}
