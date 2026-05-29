package main

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

var startTime = time.Now()

// ============================================================
// RATE LIMITER — anti spam per user
// ============================================================

type rateLimiter struct {
	mu      sync.Mutex
	lastMsg map[string]int64
}

var rl = &rateLimiter{lastMsg: make(map[string]int64)}

// isRateLimited returns true kalau user spam (< 2 detik dari pesan terakhir)
func isRateLimited(userId string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := time.Now().UnixMilli()
	last, ok := rl.lastMsg[userId]
	if ok && now-last < 2000 {
		return true
	}
	rl.lastMsg[userId] = now
	return false
}

// ============================================================
// UTILS
// ============================================================

func getUptime() string {
	d := time.Since(startTime)
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60
	return fmt.Sprintf("%dd %dh %dm %ds", days, hours, minutes, seconds)
}

func splitMessage(text string, maxLength int) []string {
	if len(text) <= maxLength {
		return []string{text}
	}
	var chunks []string
	current := ""
	for _, line := range strings.Split(text, "\n") {
		if len(current)+len(line) > maxLength {
			if current != "" {
				chunks = append(chunks, strings.TrimSpace(current))
			}
			current = line + "\n"
		} else {
			current += line + "\n"
		}
	}
	if strings.TrimSpace(current) != "" {
		chunks = append(chunks, strings.TrimSpace(current))
	}
	return chunks
}

func xpToNextLevel(level int) int {
	return 100 * (level + 1) * (level + 1)
}

func formatDuration(ms int64) string {
	s := ms / 1000
	m := s / 60
	h := m / 60
	if h > 0 {
		return fmt.Sprintf("%dj %dm", h, m%60)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s%60)
	}
	return fmt.Sprintf("%ds", s)
}

func nowISO() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func nowMs() int64 {
	return time.Now().UnixMilli()
}

func containsWord(text, word string) bool {
	return strings.Contains(strings.ToLower(text), strings.ToLower(word))
}

var availableModels = map[string]string{
	"llama70b": "llama-3.3-70b-versatile",
	"gpt120b":  "openai/gpt-oss-120b",
	"gpt20b":   "openai/gpt-oss-20b",
	"qwen32b":  "qwen/qwen3-32b",
}

func modelListText() string {
	return "`llama70b` → llama-3.3-70b-versatile\n`gpt120b` → openai/gpt-oss-120b\n`gpt20b` → openai/gpt-oss-20b\n`qwen32b` → qwen/qwen3-32b"
}
