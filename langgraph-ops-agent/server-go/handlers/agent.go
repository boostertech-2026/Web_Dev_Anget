package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"langgraph-ops-server/agent"
	"langgraph-ops-server/models"

	"github.com/gin-gonic/gin"
)

// AgentChatRequest is the request body for AI chat.
type AgentChatRequest struct {
	Message        string `json:"message"`
	ConversationID string `json:"conversation_id"`
	ThreadID       string `json:"thread_id"`
	Approved       bool   `json:"approved"` // set to true when user confirms a high-risk operation
	LlmConfig      *struct {
		ApiKey  string `json:"api_key"`
		BaseURL string `json:"base_url"`
		Model   string `json:"model"`
	} `json:"llm_config,omitempty"`
}

// AgentChat handles AI chat with SSE streaming — calls the Eino agent directly.
func AgentChat(c *gin.Context) {
	var req AgentChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	threadID := req.ThreadID
	if threadID == "" {
		threadID = fmt.Sprintf("thread-%d", time.Now().UnixNano())
	}

	// Build agent config
	llmCfg := make(map[string]string)
	if req.LlmConfig != nil {
		llmCfg["api_key"] = req.LlmConfig.ApiKey
		llmCfg["base_url"] = req.LlmConfig.BaseURL
		llmCfg["model"] = req.LlmConfig.Model
	}

	// Build initial state
	state := &agent.AgentState{
		UserInput: req.Message,
		LlmConfig: llmCfg,
		ThreadID:  threadID,
		Approved:  req.Approved,
	}

	// Run agent and stream SSE
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.WriteHeader(http.StatusOK)

	// Use non-streaming Invoke for now (graph Stream requires proper Eino stream wiring)
	// TODO: switch to agent.Stream when Eino streaming is fully configured
	finalState, err := agent.Invoke(c.Request.Context(), state)
	if err != nil {
		sendSSE(c.Writer, "error", map[string]string{"message": err.Error()})
		return
	}

	// Emulate streaming events from the final state
	if finalState.Intent != nil && len(finalState.Intent) > 0 {
		b, _ := json.Marshal(finalState.Intent)
		sendSSE(c.Writer, "intent", map[string]string{"intent": string(b)})
	}
	if finalState.Plan != "" {
		sendSSE(c.Writer, "plan", map[string]string{"content": finalState.Plan})
	}
	if finalState.FinalAnswer != "" {
		sendSSE(c.Writer, "done", map[string]string{
			"report": finalState.FinalAnswer,
			"thread_id": threadID,
		})
	} else {
		sendSSE(c.Writer, "done", map[string]string{
			"report":    "Agent completed processing.",
			"thread_id": threadID,
		})
	}

	sendSSE(c.Writer, "stream_end", map[string]string{"thread_id": threadID})
}

// sendSSE writes a single SSE event.
func sendSSE(w io.Writer, eventType string, data interface{}) {
	b, _ := json.Marshal(data)
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventType, string(b))
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
}

// ---- Dashboard / IPMI handlers (unchanged) ----

// GetDashboardSummary aggregates data for the dashboard overview.
func GetDashboardSummary(c *gin.Context) {
	var hostCount, onlineCount, offlineCount, alertCount int64

	models.DB.Model(&models.Host{}).Count(&hostCount)
	models.DB.Model(&models.Host{}).Where("status = ?", "online").Count(&onlineCount)
	offlineCount = hostCount - onlineCount
	models.DB.Model(&models.Alert{}).Where("status = ?", "firing").Count(&alertCount)

	var recentAlerts []models.Alert
	models.DB.Order("id desc").Limit(5).Find(&recentAlerts)

	var recentTasks []models.Task
	models.DB.Order("id desc").Limit(5).Find(&recentTasks)

	type AvgMetrics struct {
		AvgCpu  float64 `json:"avg_cpu"`
		AvgMem  float64 `json:"avg_mem"`
		AvgDisk float64 `json:"avg_disk"`
	}
	var avg AvgMetrics
	models.DB.Raw(`
		SELECT
			COALESCE(AVG(cpu_percent), 0) as avg_cpu,
			COALESCE(AVG(mem_percent), 0) as avg_mem,
			COALESCE(AVG(disk_percent), 0) as avg_disk
		FROM metrics
		WHERE created_at > datetime('now', '-1 hour')
	`).Scan(&avg)

	c.JSON(http.StatusOK, gin.H{
		"hosts": gin.H{
			"total":   hostCount,
			"online":  onlineCount,
			"offline": offlineCount,
		},
		"alerts": gin.H{
			"count":  alertCount,
			"recent": recentAlerts,
		},
		"tasks": gin.H{
			"recent": recentTasks,
		},
		"metrics": gin.H{
			"avg_cpu":  avg.AvgCpu,
			"avg_mem":  avg.AvgMem,
			"avg_disk": avg.AvgDisk,
		},
	})
}

// UpdateHostIpmi updates IPMI configuration for a host.
func UpdateHostIpmi(c *gin.Context) {
	id := c.Param("id")
	var host models.Host
	if err := models.DB.First(&host, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "host not found"})
		return
	}

	var req struct {
		IpmiHost     string `json:"ipmi_host"`
		IpmiUser     string `json:"ipmi_user"`
		IpmiPassword string `json:"ipmi_password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	host.IpmiHost = req.IpmiHost
	host.IpmiUser = req.IpmiUser
	if req.IpmiPassword != "" {
		enc, err := models.EncryptCredential(req.IpmiPassword)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encrypt IPMI password"})
			return
		}
		host.IpmiPassword = enc
	}
	host.IpmiStatus = "configured"
	models.DB.Save(&host)

	c.JSON(http.StatusOK, gin.H{"data": host})
}

// CheckIpmiConnectivity tests IPMI connectivity for a host.
func CheckIpmiConnectivity(c *gin.Context) {
	id := c.Param("id")
	var host models.Host
	if err := models.DB.First(&host, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "host not found"})
		return
	}

	if host.IpmiHost == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "IPMI not configured for this host"})
		return
	}

	// Use agent to check IPMI
	state := &agent.AgentState{
		UserInput: fmt.Sprintf("Check IPMI connectivity for %s using ipmi_sensor tool", host.Name),
		ThreadID:  fmt.Sprintf("ipmi-check-%s", id),
	}
	_, err := agent.Invoke(c.Request.Context(), state)
	if err != nil {
		log.Printf("[ipmi-check] agent error: %v", err)
	}

	host.IpmiStatus = "checking"
	models.DB.Save(&host)
	c.JSON(http.StatusOK, gin.H{"message": "IPMI check initiated"})
}
