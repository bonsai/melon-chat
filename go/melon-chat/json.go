package main

import (
	"encoding/json"
	"log"
	"strings"
)

func toJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		log.Printf("json marshal: %v", err)
		return "{}"
	}
	return string(b)
}

func toJSONPretty(v any) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		log.Printf("json marshal: %v", err)
		return "{}"
	}
	return string(b)
}

func detectContentType(msg string) string {
	msg = strings.TrimSpace(msg)
	if strings.HasPrefix(msg, "{") || strings.HasPrefix(msg, "[") {
		return "text"
	}
	return "text"
}
