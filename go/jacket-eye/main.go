package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "scan" {
		runCLI(os.Args[2:])
		return
	}

	loadEnv()
	port := getEnv("PORT", getEnv("JACKET_EYE_PORT", "8085"))
	addr := ":" + port

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/jacket/scan", handleScan)
	mux.HandleFunc("GET /api/health", handleHealth)

	log.Printf("jacket-eye starting on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

func runCLI(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: jacket-eye scan <image-path>")
		os.Exit(1)
	}

	imagePath := args[0]
	data, err := os.ReadFile(imagePath)
	if err != nil {
		log.Fatalf("read image: %v", err)
	}

	result, err := scanImage(data, imagePath)
	if err != nil {
		log.Fatalf("scan: %v", err)
	}

	fmt.Println(toJSONPretty(result))
}

func handleScan(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Image string `json:"image"`
		URL   string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
		return
	}

	var imageData []byte
	var filename string

	if req.URL != "" {
		resp, err := http.Get(req.URL)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"fetch url: %s"}`, err), http.StatusBadRequest)
			return
		}
		defer resp.Body.Close()
		imageData, _ = io.ReadAll(resp.Body)
		filename = req.URL
	} else if strings.HasPrefix(req.Image, "data:image") {
		parts := strings.SplitN(req.Image, ",", 2)
		if len(parts) != 2 {
			http.Error(w, `{"error":"invalid data uri"}`, http.StatusBadRequest)
			return
		}
		imageData = []byte(parts[1])
		filename = "upload"
	} else {
		imageData = []byte(req.Image)
		filename = "upload"
	}

	result, err := scanImage(imageData, filename)
	if err != nil {
		log.Printf("scan error: %v", err)
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]any{
		"ok":    true,
		"service": "jacket-eye",
	})
}

func scanImage(imageData []byte, filename string) (*ScanResult, error) {
	endpoint := os.Getenv("SAKURA_ENDPOINT")
	if endpoint == "" {
		endpoint = "https://api.sakura.ai/v1"
	}
	apiKey := os.Getenv("SAKURA_API_KEY")
	secret := os.Getenv("SAKURA_SECRET")

	model := os.Getenv("SAKURA_VLM_MODEL")
	if model == "" {
		model = "qwen-vl"
	}

	payload := map[string]any{
		"model": model,
		"messages": []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					{
						"type": "text",
						"text": `Analyze this album cover image and respond in JSON format:
{
  "artist": "artist name",
  "album": "album name",
  "songs": ["song1", "song2"],
  "year": 2024
}
If uncertain, set fields to null. Respond with ONLY the JSON, no other text.`,
					},
					{
						"type": "image_base64",
						"image_base64": string(imageData),
					},
				},
			},
		},
		"temperature": 0.1,
		"max_tokens": 1024,
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", endpoint+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create req: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	if secret != "" {
		req.Header.Set("X-API-Secret", secret)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sakura api: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("sakura api %d: %s", resp.StatusCode, string(respBody))
	}

	var sakuraResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &sakuraResp); err != nil {
		return nil, fmt.Errorf("parse sakura: %w", err)
	}

	if len(sakuraResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	content := sakuraResp.Choices[0].Message.Content
	content = extractJSON(content)

	var result ScanResult
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		result.Artist = "Unknown"
		result.Album = "Unknown"
		result.RawResponse = content
		result.Confidence = 0.3
	}

	if result.Confidence == 0 {
		result.Confidence = 0.85
	}

	return &result, nil
}

func extractJSON(s string) string {
	start := strings.Index(s, "{")
	if start == -1 {
		return s
	}
	end := strings.LastIndex(s, "}")
	if end == -1 || end < start {
		return s
	}
	return s[start : end+1]
}

type ScanResult struct {
	Artist     string   `json:"artist"`
	Album      string   `json:"album"`
	Songs      []string `json:"songs"`
	Year       int      `json:"year"`
	Confidence float64  `json:"confidence"`
	RawResponse string  `json:"raw_response,omitempty"`
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
