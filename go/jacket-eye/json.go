package main

import (
	"encoding/json"
	"log"
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
