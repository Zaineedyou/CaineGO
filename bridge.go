package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/bwmarrin/discordgo"
	"github.com/gorilla/websocket"
)

// ChatPayload adalah bentuk pesan JSON yang saling dikirim antara
// plugin Minecraft dan bot ini. Harus SAMA PERSIS dengan struktur
// yang dipakai di ChatListener.java / BridgeWebSocketClient.java sisi plugin.
type ChatPayload struct {
	Type     string `json:"type"`     // "chat" | "join" | "leave"
	Username string `json:"username"`
	Message  string `json:"message"`
}

var bridgeUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// bridgeConn menyimpan koneksi plugin Minecraft yang sedang terhubung
// beserta guild mana yang jadi tujuan pesan darinya.
// Diasumsikan 1 server Minecraft = 1 koneksi = 1 guild Discord.
// Kalau nanti mau multi-server, index by server_id yang dikirim plugin.
type bridgeConn struct {
	conn    *websocket.Conn
	guildID string
}

var (
	activeBridge   *bridgeConn
	activeBridgeMu sync.Mutex
)

// discordSession disimpan di sini supaya handler WS bisa kirim pesan ke Discord.
var (
	discordSession   *discordgo.Session
	discordSessionMu sync.RWMutex
)

// BRIDGE_TOKEN adalah secret yang harus sama dengan websocket.auth-token
// di config.yml plugin Minecraft. Ambil dari env var, JANGAN hardcode.
var BRIDGE_TOKEN string

// StartBridgeServer menjalankan HTTP server dengan endpoint /ws untuk
// menerima koneksi dari plugin Minecraft. Dipanggil sebagai goroutine
// terpisah dari main(), TIDAK memblokir runBot().
func StartBridgeServer(s *discordgo.Session) {
	discordSessionMu.Lock()
	discordSession = s
	discordSessionMu.Unlock()

	BRIDGE_TOKEN = os.Getenv("BRIDGE_TOKEN")
	if BRIDGE_TOKEN == "" {
		fmt.Println("⚠️  BRIDGE_TOKEN tidak diset — bridge WS akan menolak semua koneksi. Set env var BRIDGE_TOKEN untuk mengaktifkan.")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", handleBridgeWS)

	port := getEnvOrDefault("BRIDGE_PORT", "8080")

	fmt.Printf("🌉 Bridge WS server listening on :%s/ws\n", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		fmt.Println("❌ Bridge server error:", err)
	}
}

func handleBridgeWS(w http.ResponseWriter, r *http.Request) {
	if BRIDGE_TOKEN == "" {
		http.Error(w, "bridge not configured", http.StatusServiceUnavailable)
		return
	}

	auth := r.Header.Get("Authorization")
	if auth != "Bearer "+BRIDGE_TOKEN {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// guild_id dikirim sebagai query param oleh plugin, misal:
	// wss://host/ws?guild_id=123456789
	guildID := r.URL.Query().Get("guild_id")
	if guildID == "" {
		http.Error(w, "missing guild_id query param", http.StatusBadRequest)
		return
	}

	conn, err := bridgeUpgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("⚠️ bridge upgrade error:", err)
		return
	}

	bc := &bridgeConn{conn: conn, guildID: guildID}

	activeBridgeMu.Lock()
	activeBridge = bc
	activeBridgeMu.Unlock()

	fmt.Printf("🌉 Minecraft server connected to bridge (guild=%s)\n", guildID)

	defer func() {
		activeBridgeMu.Lock()
		if activeBridge == bc {
			activeBridge = nil
		}
		activeBridgeMu.Unlock()
		conn.Close()
		fmt.Printf("🌉 Minecraft server disconnected (guild=%s)\n", guildID)
	}()

	for {
		_, msgBytes, err := conn.ReadMessage()
		if err != nil {
			break
		}

		var payload ChatPayload
		if err := json.Unmarshal(msgBytes, &payload); err != nil {
			fmt.Println("⚠️ invalid bridge payload:", err)
			continue
		}

		handleMinecraftEvent(bc.guildID, payload)
	}
}

// handleMinecraftEvent meneruskan event dari Minecraft ke channel Discord
// yang sudah di-set lewat command "Caine setbridge #channel".
func handleMinecraftEvent(guildID string, payload ChatPayload) {
	discordSessionMu.RLock()
	s := discordSession
	discordSessionMu.RUnlock()
	if s == nil {
		return
	}

	channelID := getBridgeChannel(guildID)
	if channelID == "" {
		return // belum di-setup di guild ini
	}

	switch payload.Type {
	case "chat":
		s.ChannelMessageSend(channelID, fmt.Sprintf("**[MC] %s:** %s", payload.Username, payload.Message))
	case "join":
		s.ChannelMessageSend(channelID, fmt.Sprintf("➡️ **%s** joined the server", payload.Username))
	case "leave":
		s.ChannelMessageSend(channelID, fmt.Sprintf("⬅️ **%s** left the server", payload.Username))
	}
}

// SendToMinecraft dipanggil dari onMessageCreate (events.go) setiap kali
// ada pesan manusia di channel yang sudah di-set sebagai bridge channel
// untuk guild tersebut.
func SendToMinecraft(guildID, username, message string) {
	activeBridgeMu.Lock()
	bc := activeBridge
	activeBridgeMu.Unlock()

	if bc == nil || bc.guildID != guildID {
		return // server Minecraft untuk guild ini belum/tidak connect
	}

	payload := ChatPayload{
		Type:     "chat",
		Username: username,
		Message:  message,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}

	activeBridgeMu.Lock()
	defer activeBridgeMu.Unlock()
	if activeBridge != bc {
		return // sudah disconnect di antara pengecekan tadi
	}
	_ = bc.conn.WriteMessage(websocket.TextMessage, data)
}

// isBridgeConnected mengecek apakah ada server Minecraft yang terhubung untuk guild ini.
func isBridgeConnected(guildID string) bool {
	activeBridgeMu.Lock()
	defer activeBridgeMu.Unlock()
	return activeBridge != nil && activeBridge.guildID == guildID
}
