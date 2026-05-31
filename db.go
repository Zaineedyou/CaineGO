package main

import (
	"database/sql"
	"strconv"
	"os"
	"fmt"
	"sync"

	_ "modernc.org/sqlite"
)

func getDBPath() string {
	if p := os.Getenv("DB_PATH"); p != "" {
		return p
	}
	return "./caine.db"
}

var (
	db   *sql.DB
	dbMu sync.Mutex
)

// In-memory cache for per-guild config (kv store).
type kvCache struct {
	mu    sync.RWMutex
	store map[string]string 
}

var cache = &kvCache{store: make(map[string]string)}

const kvMiss = "\x00MISS" // sentinel: key has been looked up but is empty

func (c *kvCache) get(guildId, key string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.store[guildId+":"+key]
	if !ok {
		return "", false
	}
	if v == kvMiss {
		return "", true // cache hit: value is empty
	}
	return v, true
}

func (c *kvCache) set(guildId, key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.store[guildId+":"+key] = value
}

func (c *kvCache) del(guildId, key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.store, guildId+":"+key)
}

func initDB() {
	var err error
	db, err = sql.Open("sqlite", getDBPath()+"?_journal=WAL&_busy_timeout=5000&_timeout=5000")
	if err != nil {
		panic(fmt.Sprintf("❌ Failed to open SQLite: %v", err))
	}
	db.SetMaxOpenConns(1)
	createTables()
	fmt.Println("✅ SQLite database ready")
}

func flushDB() {
	if db != nil {
		db.Close()
	}
}

func createTables() {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS kv (
			guild_id TEXT NOT NULL,
			key TEXT NOT NULL,
			value TEXT NOT NULL,
			PRIMARY KEY (guild_id, key)
		)`,
		`CREATE TABLE IF NOT EXISTS warnings (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			guild_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			reason TEXT NOT NULL,
			time TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS banned_words (
			guild_id TEXT NOT NULL,
			word TEXT NOT NULL,
			PRIMARY KEY (guild_id, word)
		)`,
		`CREATE TABLE IF NOT EXISTS disabled_channels (
			guild_id TEXT NOT NULL,
			channel_id TEXT NOT NULL,
			PRIMARY KEY (guild_id, channel_id)
		)`,
		`CREATE TABLE IF NOT EXISTS history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			history_key TEXT NOT NULL,
			role TEXT NOT NULL,
			content TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS xp (
			guild_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			xp INTEGER DEFAULT 0,
			level INTEGER DEFAULT 0,
			last_message INTEGER DEFAULT 0,
			PRIMARY KEY (guild_id, user_id)
		)`,
		`CREATE TABLE IF NOT EXISTS afk (
			guild_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			reason TEXT NOT NULL,
			time INTEGER NOT NULL,
			PRIMARY KEY (guild_id, user_id)
		)`,
	}
	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			fmt.Printf("⚠️ createTables: %v\n", err)
		}
	}
}

func kvGet(guildId, key string) string {
	// Cek cache first before hit SQLite
	if v, ok := cache.get(guildId, key); ok {
		return v
	}
	var value string
	err := db.QueryRow(`SELECT value FROM kv WHERE guild_id=? AND key=?`, guildId, key).Scan(&value)
	if err != nil {
		// Cache "miss" So that the next query doesn't need to access SQLite again.
		cache.set(guildId, key, kvMiss)
		return ""
	}
	cache.set(guildId, key, value)
	return value
}

func kvSet(guildId, key, value string) {
	if _, err := db.Exec(`INSERT OR REPLACE INTO kv (guild_id, key, value) VALUES (?,?,?)`, guildId, key, value); err != nil {
		fmt.Printf("⚠️ kvSet [%s:%s]: %v\n", guildId, key, err)
		return
	}
	cache.set(guildId, key, value)
}

func kvDel(guildId, key string) {
	if _, err := db.Exec(`DELETE FROM kv WHERE guild_id=? AND key=?`, guildId, key); err != nil {
		fmt.Printf("⚠️ kvDel [%s:%s]: %v\n", guildId, key, err)
		return
	}
	cache.del(guildId, key)
}

func getGuildLogChannel(guildId string) string { return kvGet(guildId, "log_channel") }
func setGuildLogChannel(guildId, channelId string) {
	kvSet(guildId, "log_channel", channelId)
}

func getWarnings(userId, guildId string) []Warning {
	rows, err := db.Query(`SELECT reason, time FROM warnings WHERE guild_id=? AND user_id=? ORDER BY id`, guildId, userId)
	if err != nil {
		fmt.Printf("⚠️ getWarnings: %v\n", err)
		return nil
	}
	defer rows.Close()
	var warns []Warning
	for rows.Next() {
		var w Warning
		if err := rows.Scan(&w.Reason, &w.Time); err != nil {
			fmt.Printf("⚠️ getWarnings scan: %v\n", err)
			continue
		}
		warns = append(warns, w)
	}
	return warns
}

func addWarning(userId, guildId, reason string) int {
	if _, err := db.Exec(`INSERT INTO warnings (guild_id, user_id, reason, time) VALUES (?,?,?,?)`, guildId, userId, reason, nowISO()); err != nil {
		fmt.Printf("⚠️ addWarning: %v\n", err)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM warnings WHERE guild_id=? AND user_id=?`, guildId, userId).Scan(&count); err != nil {
		fmt.Printf("⚠️ addWarning count: %v\n", err)
	}
	return count
}

func clearWarnings(userId, guildId string) {
	if _, err := db.Exec(`DELETE FROM warnings WHERE guild_id=? AND user_id=?`, guildId, userId); err != nil {
		fmt.Printf("⚠️ clearWarnings: %v\n", err)
	}
}

func getBannedWords(guildId string) []string {
	rows, err := db.Query(`SELECT word FROM banned_words WHERE guild_id=?`, guildId)
	if err != nil {
		fmt.Printf("⚠️ getBannedWords: %v\n", err)
		return nil
	}
	defer rows.Close()
	var words []string
	for rows.Next() {
		var w string
		if err := rows.Scan(&w); err != nil {
			fmt.Printf("⚠️ getBannedWords scan: %v\n", err)
			continue
		}
		words = append(words, w)
	}
	return words
}

func addBannedWord(guildId, word string) {
	if _, err := db.Exec(`INSERT OR IGNORE INTO banned_words (guild_id, word) VALUES (?,?)`, guildId, word); err != nil {
		fmt.Printf("⚠️ addBannedWord: %v\n", err)
	}
}

func removeBannedWord(guildId, word string) {
	if _, err := db.Exec(`DELETE FROM banned_words WHERE guild_id=? AND word=?`, guildId, word); err != nil {
		fmt.Printf("⚠️ removeBannedWord: %v\n", err)
	}
}

func isChannelDisabled(guildId, channelId string) bool {
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM disabled_channels WHERE guild_id=? AND channel_id=?`, guildId, channelId).Scan(&count); err != nil {
		fmt.Printf("⚠️ isChannelDisabled: %v\n", err)
		return false
	}
	return count > 0
}

func disableChannel(guildId, channelId string) {
	if _, err := db.Exec(`INSERT OR IGNORE INTO disabled_channels (guild_id, channel_id) VALUES (?,?)`, guildId, channelId); err != nil {
		fmt.Printf("⚠️ disableChannel: %v\n", err)
	}
}

func enableChannel(guildId, channelId string) {
	if _, err := db.Exec(`DELETE FROM disabled_channels WHERE guild_id=? AND channel_id=?`, guildId, channelId); err != nil {
		fmt.Printf("⚠️ enableChannel: %v\n", err)
	}
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func getHistory(key string) []Message {
	rows, err := db.Query(`SELECT role, content FROM history WHERE history_key=? ORDER BY id`, key)
	if err != nil {
		fmt.Printf("⚠️ getHistory: %v\n", err)
		return nil
	}
	defer rows.Close()
	var msgs []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.Role, &m.Content); err != nil {
			fmt.Printf("⚠️ getHistory scan: %v\n", err)
			continue
		}
		msgs = append(msgs, m)
	}
	return msgs
}

func addToHistory(key, role, content string) {
	if _, err := db.Exec(`INSERT INTO history (history_key, role, content) VALUES (?,?,?)`, key, role, content); err != nil {
		fmt.Printf("⚠️ addToHistory: %v\n", err)
		return
	}
	// Trim kalau terlalu panjang — simpan MAX_HISTORY*2 entry terakhir
	if _, err := db.Exec(`DELETE FROM history WHERE history_key=? AND id NOT IN (
		SELECT id FROM history WHERE history_key=? ORDER BY id DESC LIMIT ?
	)`, key, key, MAX_HISTORY*2); err != nil {
		fmt.Printf("⚠️ addToHistory trim: %v\n", err)
	}
}

func clearHistory(key string) {
	if _, err := db.Exec(`DELETE FROM history WHERE history_key=?`, key); err != nil {
		fmt.Printf("⚠️ clearHistory: %v\n", err)
	}
}

func getWelcomeChannel(guildId string) string { return kvGet(guildId, "welcome_channel") }
func setWelcomeChannel(guildId, ch string)    { kvSet(guildId, "welcome_channel", ch) }
func getGoodbyeChannel(guildId string) string { return kvGet(guildId, "goodbye_channel") }
func setGoodbyeChannel(guildId, ch string)    { kvSet(guildId, "goodbye_channel", ch) }

func getWelcomeMessage(guildId string) string {
	v := kvGet(guildId, "welcome_msg")
	if v == "" {
		return "Selamat datang {user} di **{server}**! 🎉"
	}
	return v
}
func setWelcomeMessage(guildId, msg string) { kvSet(guildId, "welcome_msg", msg) }

func getGoodbyeMessage(guildId string) string {
	v := kvGet(guildId, "goodbye_msg")
	if v == "" {
		return "Selamat tinggal **{username}** dari **{server}**. 👋"
	}
	return v
}
func setGoodbyeMessage(guildId, msg string) { kvSet(guildId, "goodbye_msg", msg) }

func getAutoRole(guildId string) string   { return kvGet(guildId, "auto_role") }
func setAutoRole(guildId, roleId string)  { kvSet(guildId, "auto_role", roleId) }
func removeAutoRole(guildId string)       { kvDel(guildId, "auto_role") }

func getGuildSystemPrompt(guildId string) string {
	v := kvGet(guildId, "system_prompt")
	if v == "" {
		return DEFAULT_SYSTEM_PROMPT
	}
	return v
}
func setGuildSystemPrompt(guildId, prompt string) { kvSet(guildId, "system_prompt", prompt) }

func getGuildModel(guildId string) string {
	v := kvGet(guildId, "model")
	if v == "" {
		return DEFAULT_MODEL
	}
	return v
}
func setGuildModel(guildId, model string) { kvSet(guildId, "model", model) }

func getLevelChannel(guildId string) string { return kvGet(guildId, "level_channel") }
func setLevelChannel(guildId, ch string)    { kvSet(guildId, "level_channel", ch) }

type XPData struct {
	XP          int   `json:"xp"`
	Level       int   `json:"level"`
	LastMessage int64 `json:"lastMessage"`
}

func getUserXP(userId, guildId string) *XPData {
	data := &XPData{}
	err := db.QueryRow(`SELECT xp, level, last_message FROM xp WHERE guild_id=? AND user_id=?`, guildId, userId).
		Scan(&data.XP, &data.Level, &data.LastMessage)
	if err != nil {
		return &XPData{}
	}
	return data
}

func setUserXP(userId, guildId string, data *XPData) {
	if _, err := db.Exec(`INSERT OR REPLACE INTO xp (guild_id, user_id, xp, level, last_message) VALUES (?,?,?,?,?)`,
		guildId, userId, data.XP, data.Level, data.LastMessage); err != nil {
		fmt.Printf("⚠️ setUserXP: %v\n", err)
	}
}

func getAllXP(guildId string) []struct {
	UserID string
	Data   *XPData
} {
	rows, err := db.Query(`SELECT user_id, xp, level FROM xp WHERE guild_id=? ORDER BY level DESC, xp DESC LIMIT 10`, guildId)
	if err != nil {
		fmt.Printf("⚠️ getAllXP: %v\n", err)
		return nil
	}
	defer rows.Close()
	var result []struct {
		UserID string
		Data   *XPData
	}
	for rows.Next() {
		var uid string
		data := &XPData{}
		if err := rows.Scan(&uid, &data.XP, &data.Level); err != nil {
			fmt.Printf("⚠️ getAllXP scan: %v\n", err)
			continue
		}
		result = append(result, struct {
			UserID string
			Data   *XPData
		}{uid, data})
	}
	return result
}

type AFKData struct {
	Reason string `json:"reason"`
	Time   int64  `json:"time"`
}

func getAfkUser(userId, guildId string) *AFKData {
	data := &AFKData{}
	err := db.QueryRow(`SELECT reason, time FROM afk WHERE guild_id=? AND user_id=?`, guildId, userId).
		Scan(&data.Reason, &data.Time)
	if err != nil {
		return nil
	}
	return data
}

func setAfkUser(userId, guildId, reason string) {
	if _, err := db.Exec(`INSERT OR REPLACE INTO afk (guild_id, user_id, reason, time) VALUES (?,?,?,?)`,
		guildId, userId, reason, nowMs()); err != nil {
		fmt.Printf("⚠️ setAfkUser: %v\n", err)
	}
}

func removeAfkUser(userId, guildId string) {
	if _, err := db.Exec(`DELETE FROM afk WHERE guild_id=? AND user_id=?`, guildId, userId); err != nil {
		fmt.Printf("⚠️ removeAfkUser: %v\n", err)
	}
}

func getAllAfk(guildId string) map[string]*AFKData {
	rows, err := db.Query(`SELECT user_id, reason, time FROM afk WHERE guild_id=?`, guildId)
	if err != nil {
		fmt.Printf("⚠️ getAllAfk: %v\n", err)
		return nil
	}
	defer rows.Close()
	result := make(map[string]*AFKData)
	for rows.Next() {
		var uid string
		data := &AFKData{}
		if err := rows.Scan(&uid, &data.Reason, &data.Time); err != nil {
			fmt.Printf("⚠️ getAllAfk scan: %v\n", err)
			continue
		}
		result[uid] = data
	}
	return result
}

// Warning struct (use in moderation.go)
type Warning struct {
	Reason string
	Time   string
}

// getGuildMaxHistory returns the per-guild history limit, falling back to the global MAX_HISTORY.
func getGuildMaxHistory(guildId string) int {
	v := kvGet(guildId, "max_history")
	if v == "" {
		return MAX_HISTORY
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 5 {
		return MAX_HISTORY
	}
	return n
}

func setGuildMaxHistory(guildId string, limit int) {
	kvSet(guildId, "max_history", strconv.Itoa(limit))
}
