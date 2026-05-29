package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const GROQ_API_URL = "https://api.groq.com/openai/v1/chat/completions"

type GroqMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

type GroqRequest struct {
	Model       string        `json:"model"`
	Messages    []GroqMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens"`
	Temperature float64       `json:"temperature"`
}

type GroqResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func askGroq(key, userMessage, displayName, guildId string) (string, error) {
	history := getHistory(key)
	systemPrompt := getGuildSystemPrompt(guildId)
	model := getGuildModel(guildId)

	messages := []GroqMessage{
		{
			Role: "system",
			Content: systemPrompt + fmt.Sprintf(
				"\n\nKamu lagi ngobrol di Discord sama user bernama %s. PENTING: jangan pernah tulis nama user dalam format [nama] atau kurung kotak apapun di responmu. Langsung bales natural aja kayak ngobrol biasa.",
				displayName,
			),
		},
	}
	for _, h := range history {
		messages = append(messages, GroqMessage{Role: h.Role, Content: h.Content})
	}
	messages = append(messages, GroqMessage{
		Role:    "user",
		Content: userMessage,
	})

	reqBody := GroqRequest{
		Model:       model,
		Messages:    messages,
		MaxTokens:   1024,
		Temperature: 0.8,
	}

	reply, err := doGroqRequest(reqBody)
	if err != nil {
		return "", err
	}

	addToHistory(key, "user", userMessage)
	addToHistory(key, "assistant", reply)
	return reply, nil
}

func askVision(key, userMessage, imageUrl, displayName, guildId string) (string, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	req, _ := http.NewRequest("GET", imageUrl, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("gagal fetch gambar: %v", err)
	}
	defer resp.Body.Close()

	imgBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	base64Img := base64.StdEncoding.EncodeToString(imgBytes)
	mimeType := resp.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "image/png"
	}

	systemPrompt := getGuildSystemPrompt(guildId)
	text := userMessage
	if text == "" {
		text = "Deskripsiin gambar ini."
	}

	reqBody := GroqRequest{
		Model: "meta-llama/llama-4-scout-17b-16e-instruct",
		Messages: []GroqMessage{
			{Role: "system", Content: systemPrompt},
			{
				Role: "user",
				Content: []interface{}{
					map[string]interface{}{
						"type": "image_url",
						"image_url": map[string]string{
							"url": fmt.Sprintf("data:%s;base64,%s", mimeType, base64Img),
						},
					},
					map[string]interface{}{
						"type": "text",
						"text": text,
					},
				},
			},
		},
		MaxTokens:   1024,
		Temperature: 0.8,
	}

	reply, err := doGroqRequest(reqBody)
	if err != nil {
		return "", err
	}

	addToHistory(key, "user", "[kirim gambar] "+userMessage)
	addToHistory(key, "assistant", reply)
	return reply, nil
}

// doGroqRequest mengirim request ke Groq API dengan retry otomatis.
// Retry dilakukan maksimal 3x dengan jeda 2 detik antar percobaan,
// khusus untuk error sementara (timeout, 429, 5xx).
func doGroqRequest(reqBody GroqRequest) (string, error) {
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal error: %w", err)
	}

	const maxRetries = 3
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		client := &http.Client{Timeout: 30 * time.Second}
		req, err := http.NewRequest("POST", GROQ_API_URL, bytes.NewReader(jsonData))
		if err != nil {
			return "", fmt.Errorf("buat request gagal: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+GROQ_API_KEY)

		resp, err := client.Do(req)
		if err != nil {
			// Network error / timeout — coba lagi
			lastErr = fmt.Errorf("request gagal (percobaan %d/%d): %w", attempt, maxRetries, err)
			time.Sleep(2 * time.Second)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("baca response gagal: %w", err)
			time.Sleep(2 * time.Second)
			continue
		}

		// Rate limit atau server error — coba lagi
		if resp.StatusCode == 429 || resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("groq HTTP %d (percobaan %d/%d)", resp.StatusCode, attempt, maxRetries)
			time.Sleep(2 * time.Second)
			continue
		}

		// Error lain (401, 400, dll) — langsung gagal, jangan retry
		if resp.StatusCode != 200 {
			return "", fmt.Errorf("groq HTTP %d: %s", resp.StatusCode, string(body))
		}

		var groqResp GroqResponse
		if err := json.Unmarshal(body, &groqResp); err != nil {
			return "", fmt.Errorf("parse response gagal: %w", err)
		}
		if groqResp.Error != nil {
			return "", fmt.Errorf("groq error: %s", groqResp.Error.Message)
		}
		if len(groqResp.Choices) == 0 || groqResp.Choices[0].Message.Content == "" {
			return "", fmt.Errorf("groq mengembalikan response kosong")
		}
		return groqResp.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("groq gagal setelah %d percobaan: %w", maxRetries, lastErr)
}
