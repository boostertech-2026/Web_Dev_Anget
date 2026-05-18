package handlers

import (
	"net/http"
	"strconv"
	"time"

	"langgraph-ops-server/models"
	"strings"

	"github.com/gin-gonic/gin"
)

// GetLatestMetrics returns the most recent metrics for all or a specific host
func GetLatestMetrics(c *gin.Context) {
	hostParam := c.Query("host")
	var metrics []models.Metric

	query := models.DB.Order("id desc").Limit(50)
	if hostParam != "" {
		// Match by name or ID
		if id, err := strconv.Atoi(hostParam); err == nil {
			query = query.Where("host_id = ?", id)
		} else {
			escaped := strings.ReplaceAll(strings.ReplaceAll(hostParam, "%", "\\%"), "_", "\\_")
			query = query.Where("host_name LIKE ? ESCAPE '\\'", "%"+escaped+"%")
		}
	}
	query.Find(&metrics)

	// Return only the latest per host
	seen := map[uint]bool{}
	var result []models.Metric
	for _, m := range metrics {
		if !seen[m.HostID] {
			seen[m.HostID] = true
			result = append(result, m)
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

// GetMetricsHistory returns time-series metrics for a host
func GetMetricsHistory(c *gin.Context) {
	hostID := c.Query("host_id")
	duration := c.DefaultQuery("duration", "1h")
	since := time.Now()

	switch duration {
	case "5m":
		since = since.Add(-5 * time.Minute)
	case "1h":
		since = since.Add(-1 * time.Hour)
	case "24h":
		since = since.Add(-24 * time.Hour)
	case "7d":
		since = since.Add(-7 * 24 * time.Hour)
	default:
		since = since.Add(-1 * time.Hour)
	}

	var metrics []models.Metric
	models.DB.
		Where("host_id = ? AND created_at > ?", hostID, since).
		Order("created_at ASC").
		Limit(500).
		Find(&metrics)

	c.JSON(http.StatusOK, gin.H{"data": metrics})
}

// GetTrafficMetrics returns network traffic data for real-time monitoring
func GetTrafficMetrics(c *gin.Context) {
	hostID := c.Query("host_id")
	if hostID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "host_id required"})
		return
	}

	var metrics []models.Metric
	models.DB.
		Where("host_id = ? AND created_at > ?", hostID, time.Now().Add(-1*time.Hour)).
		Order("created_at ASC").
		Limit(200).
		Find(&metrics)

	c.JSON(http.StatusOK, gin.H{"data": metrics})
}

// GetAlerts returns alert list with optional filtering
func GetAlerts(c *gin.Context) {
	status := c.DefaultQuery("status", "")
	level := c.DefaultQuery("level", "")

	query := models.DB.Order("id desc").Limit(100)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if level != "" {
		query = query.Where("level = ?", level)
	}

	var alerts []models.Alert
	query.Find(&alerts)
	c.JSON(http.StatusOK, gin.H{"data": alerts})
}

// AckAlert marks an alert as acknowledged
func AckAlert(c *gin.Context) {
	id := c.Param("id")
	var alert models.Alert
	if err := models.DB.First(&alert, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "alert not found"})
		return
	}

	var req struct {
		AckedBy string `json:"acked_by"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.AckedBy == "" {
		req.AckedBy = "admin"
	}

	alert.Status = "acked"
	alert.AckedBy = req.AckedBy
	models.DB.Save(&alert)

	c.JSON(http.StatusOK, gin.H{"data": alert})
}

// ResolveAlert marks an alert as resolved
func ResolveAlert(c *gin.Context) {
	id := c.Param("id")
	var alert models.Alert
	if err := models.DB.First(&alert, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "alert not found"})
		return
	}

	var req struct {
		ResolvedBy  string `json:"resolved_by"`
		ResolveNote string `json:"resolve_note"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.ResolvedBy == "" {
		req.ResolvedBy = "admin"
	}

	now := time.Now()
	alert.Status = "resolved"
	alert.ResolvedBy = req.ResolvedBy
	alert.ResolveNote = req.ResolveNote
	alert.ResolvedAt = now
	models.DB.Save(&alert)

	c.JSON(http.StatusOK, gin.H{"data": alert})
}

// GetAlertRules returns configured alert rules
func GetAlertRules(c *gin.Context) {
	var rules []models.AlertRule
	models.DB.Find(&rules)
	c.JSON(http.StatusOK, gin.H{"data": rules})
}

// UpdateAlertRule enables/disables or modifies an alert rule
func UpdateAlertRule(c *gin.Context) {
	id := c.Param("id")
	var rule models.AlertRule
	if err := models.DB.First(&rule, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "rule not found"})
		return
	}

	var req struct {
		Enabled   *bool   `json:"enabled"`
		Threshold *float64 `json:"threshold"`
		Duration  *int    `json:"duration"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Enabled != nil {
		rule.Enabled = *req.Enabled
	}
	if req.Threshold != nil {
		rule.Threshold = *req.Threshold
	}
	if req.Duration != nil {
		rule.Duration = *req.Duration
	}
	models.DB.Save(&rule)

	c.JSON(http.StatusOK, gin.H{"data": rule})
}

// GetErrorHistory returns the error history list
func GetErrorHistory(c *gin.Context) {
	status := c.DefaultQuery("status", "")

	query := models.DB.Order("id desc").Limit(100)
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var errors []models.ErrorHistory
	query.Find(&errors)
	c.JSON(http.StatusOK, gin.H{"data": errors})
}

// GetErrorStats returns aggregated error statistics
func GetErrorStats(c *gin.Context) {
	// Total errors
	var total int64
	models.DB.Model(&models.ErrorHistory{}).Count(&total)

	// By level
	type LevelCount struct {
		Level string `json:"level"`
		Count int64  `json:"count"`
	}
	var byLevel []LevelCount
	models.DB.Raw(
		"SELECT level, COUNT(*) as count FROM error_histories GROUP BY level",
	).Scan(&byLevel)

	// By status
	type StatusCount struct {
		Status string `json:"status"`
		Count  int64  `json:"count"`
	}
	var byStatus []StatusCount
	models.DB.Raw(
		"SELECT status, COUNT(*) as count FROM error_histories GROUP BY status",
	).Scan(&byStatus)

	// Top error hosts
	type HostErrors struct {
		HostName string `json:"host_name"`
		Count    int64  `json:"count"`
	}
	var topHosts []HostErrors
	models.DB.Raw(
		"SELECT host_name, COUNT(*) as count FROM error_histories GROUP BY host_name ORDER BY count DESC LIMIT 10",
	).Scan(&topHosts)

	// Average resolve time
	var avgTime float64
	models.DB.Raw(
		"SELECT COALESCE(AVG(resolve_duration), 0) FROM error_histories WHERE status = 'resolved'",
	).Scan(&avgTime)

	c.JSON(http.StatusOK, gin.H{
		"total_errors":   total,
		"by_level":       byLevel,
		"by_status":      byStatus,
		"top_hosts":      topHosts,
		"avg_resolve_sec": avgTime,
	})
}

// ResolveError marks an error record as resolved
func ResolveError(c *gin.Context) {
	id := c.Param("id")
	var errRec models.ErrorHistory
	if err := models.DB.First(&errRec, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	var req struct {
		HandledBy  string `json:"handled_by"`
		HandleNote string `json:"handle_note"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.HandledBy == "" {
		req.HandledBy = "admin"
	}

	now := time.Now()
	errRec.Status = "resolved"
	errRec.HandledBy = req.HandledBy
	errRec.HandleNote = req.HandleNote
	errRec.ResolvedAt = now
	if !errRec.CreatedAt.IsZero() {
		errRec.ResolveDuration = int64(now.Sub(errRec.CreatedAt).Seconds())
	}
	models.DB.Save(&errRec)

	c.JSON(http.StatusOK, gin.H{"data": errRec})
}
