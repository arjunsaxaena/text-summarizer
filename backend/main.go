package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	genai "github.com/google/generative-ai-go/genai"
	"github.com/joho/godotenv"
	"google.golang.org/api/option"
)

type summarizeRequest struct {
	Text string `json:"text"`
}

type summarizeResponse struct {
	Summary string `json:"summary"`
}

func SummarizeWithGemini(ctx context.Context, apiKey string, text string) (string, error) {
	if strings.TrimSpace(apiKey) == "" {
		return "", errors.New("missing GEMINI_API_KEY")
	}
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return "", fmt.Errorf("genai client: %w", err)
	}
	defer client.Close()

	model := client.GenerativeModel("gemini-2.5-flash")
	prompt := fmt.Sprintf("Summarize the following text in exactly 3 concise lines. Keep it factual and clear.\n\nTEXT:\n%s", text)
	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", fmt.Errorf("generate content: %w", err)
	}

	var builder strings.Builder
	for _, cand := range resp.Candidates {
		if cand == nil || cand.Content == nil {
			continue
		}
		for _, part := range cand.Content.Parts {
			builder.WriteString(fmt.Sprint(part))
		}
	}

	output := strings.TrimSpace(builder.String())
	if output == "" {
		return "", errors.New("empty response from model")
	}
	lines := strings.Split(output, "\n")
	if len(lines) > 3 {
		output = strings.Join(lines[:3], "\n")
	}
	return output, nil
}

func main() {
	_ = godotenv.Load()
	_ = godotenv.Load("../.env")

	apiKey := os.Getenv("GEMINI_API_KEY")
	if strings.TrimSpace(apiKey) == "" {
		log.Println("Warning: GEMINI_API_KEY is empty. Set it in .env at project root.")
	}

	http.HandleFunc("/summarize", logRequests(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
		w.Header().Set("Vary", "Origin")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var req summarizeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON body"})
			return
		}
		input := strings.TrimSpace(req.Text)
		if input == "" {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "text is required"})
			return
		}

		result, err := SummarizeWithGemini(r.Context(), apiKey, input)
		if err != nil {
			log.Printf("summarize error: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "failed to summarize"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(summarizeResponse{Summary: result})
	}))

	log.Println("Backend running on http://localhost:4001")
	if err := http.ListenAndServe(":4001", nil); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

type loggingResponseWriter struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.status = code
	lrw.ResponseWriter.WriteHeader(code)
}

func (lrw *loggingResponseWriter) Write(b []byte) (int, error) {
	if lrw.status == 0 {
		lrw.status = http.StatusOK
	}
	n, err := lrw.ResponseWriter.Write(b)
	lrw.bytes += n
	return n, err
}

func logRequests(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		lrw := &loggingResponseWriter{ResponseWriter: w}
		next(lrw, r)
		dur := time.Since(start)
		log.Printf("%s %s %d %s %s bytes=%s", r.Method, r.URL.Path, lrw.status, dur, r.RemoteAddr, strconv.Itoa(lrw.bytes))
	}
}
