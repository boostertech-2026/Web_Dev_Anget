package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
)

var (
	serverAddr  = "localhost:8080"
	clientID    = uint(1)
	clientName  = "ops-client-1"
	clientHost  = "local-client"
	clientToken = ""
)

func init() {
	if addr := os.Getenv("SERVER_ADDR"); addr != "" {
		serverAddr = addr
	}
	if id := os.Getenv("CLIENT_ID"); id != "" {
		if v, err := strconv.Atoi(id); err == nil {
			clientID = uint(v)
		}
	}
	if name := os.Getenv("CLIENT_NAME"); name != "" {
		clientName = name
	}
	if host := os.Getenv("CLIENT_HOST"); host != "" {
		clientHost = host
	}
}

func main() {
	log.Printf("Starting LangGraph Ops Client %d...", clientID)
	log.Printf("Server: %s", serverAddr)

	go heartbeatLoop()

	for {
		if err := connectWS(); err != nil {
			log.Printf("WebSocket error: %v", err)
			time.Sleep(5 * time.Second)
		}
	}
}

func heartbeatLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		sendHeartbeat()
	}
}

func sendHeartbeat() {
	data := map[string]interface{}{
		"client_id": clientID,
		"name":      clientName,
		"host":      clientHost,
		"status":    "online",
	}
	jsonData, _ := json.Marshal(data)

	resp, err := http.Post("http://"+serverAddr+"/api/client/heartbeat", "application/json", bytes.NewReader(jsonData))
	if err != nil {
		log.Printf("Heartbeat failed: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Heartbeat request returned status %d", resp.StatusCode)
	}
}

func connectWS() error {
	url := "ws://" + serverAddr + "/ws/client"
	header := http.Header{}
	header.Set("X-Client-ID", strconv.Itoa(int(clientID)))

	ws, _, err := websocket.DefaultDialer.Dial(url, header)
	if err != nil {
		return err
	}
	defer ws.Close()

	log.Println("WebSocket connected")

	registerClient(ws)

	for {
		_, message, err := ws.ReadMessage()
		if err != nil {
			return err
		}
		handleServerMessage(ws, message)
	}
}

func registerClient(ws *websocket.Conn) {
	payload := map[string]interface{}{
		"type":      "register",
		"client_id": clientID,
		"name":      clientName,
		"host":      clientHost,
		"status":    "online",
	}
	ws.WriteJSON(payload)
}

func handleServerMessage(ws *websocket.Conn, message []byte) {
	var pkg struct {
		Type    string `json:"type"`
		Command string `json:"command"`
		TaskID  uint   `json:"task_id"`
	}
	if err := json.Unmarshal(message, &pkg); err != nil {
		log.Printf("Invalid server package: %v", err)
		return
	}

	switch pkg.Type {
	case "command":
		result, err := runShellCommand(pkg.Command)
		response := map[string]interface{}{
			"type":      "log",
			"client_id": clientID,
			"task_id":   pkg.TaskID,
			"result":    result,
		}
		if err != nil {
			response["error"] = err.Error()
		}
		if writeErr := ws.WriteJSON(response); writeErr != nil {
			log.Printf("Send log failed: %v", writeErr)
		}
	}
}

func runShellCommand(command string) (string, error) {
	if command == "" {
		return "", nil
	}

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", command)
	} else {
		cmd = exec.Command("sh", "-c", command)
	}

	output, err := cmd.CombinedOutput()
	return string(bytes.TrimSpace(output)), err
}
