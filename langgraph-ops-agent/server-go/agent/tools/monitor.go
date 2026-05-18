package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// MetricRow holds a snapshot of host metrics (matches GORM Metric model).
type MetricRow struct {
	CpuPercent  float64 `json:"cpu_percent"`
	MemPercent  float64 `json:"mem_percent"`
	MemUsed     int64   `json:"mem_used"`
	MemTotal    int64   `json:"mem_total"`
	DiskPercent float64 `json:"disk_percent"`
	Load1m      float64 `json:"load_1m"`
	Load5m      float64 `json:"load_5m"`
	NetRxRate   float64 `json:"net_rx_rate"`
	NetTxRate   float64 `json:"net_tx_rate"`
	ProcessTop  string  `json:"process_top"`
}

// TaskSummary is a minimal task record for deploy history.
type TaskSummary struct {
	Name      string `json:"name"`
	ExecType  string `json:"exec_type"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
	Result    string `json:"result"`
}

// MonitorStore abstracts metrics/log queries.
type MonitorStore interface {
	LatestMetric(hostIDOrName string) (*MetricRow, error)
	RecentTasks(hostIDOrName string, hours int, limit int) ([]TaskSummary, error)
}

// CheckMonitor queries real-time monitoring data for a host.
func CheckMonitor(ctx context.Context, store MonitorStore, host string) (string, error) {
	m, err := store.LatestMetric(host)
	if err != nil {
		return fmt.Sprintf(`{"host": "%s", "error": "%s"}`, host, err.Error()), nil
	}
	if m == nil {
		return fmt.Sprintf(`{"host": "%s", "error": "no metrics found"}`, host), nil
	}

	memUsedGB := roundTo1(float64(m.MemUsed) / (1024 * 1024 * 1024))
	memTotalGB := roundTo1(float64(m.MemTotal) / (1024 * 1024 * 1024))

	b, _ := json.Marshal(map[string]interface{}{
		"host":         host,
		"cpu_percent":  m.CpuPercent,
		"mem_percent":  m.MemPercent,
		"mem_used_gb":  memUsedGB,
		"mem_total_gb": memTotalGB,
		"disk_percent": m.DiskPercent,
		"load_1m":      m.Load1m,
		"load_5m":      m.Load5m,
		"net_rx_mbps":  m.NetRxRate,
		"net_tx_mbps":  m.NetTxRate,
		"process_top":  m.ProcessTop,
	})
	return string(b), nil
}

// QueryDeployHistory queries recent task/deployment history for a host.
func QueryDeployHistory(ctx context.Context, store MonitorStore, host string, hours int) (string, error) {
	tasks, err := store.RecentTasks(host, hours, 10)
	if err != nil {
		return fmt.Sprintf(`{"host": "%s", "hours": %d, "tasks": [], "note": "%s"}`, host, hours, err.Error()), nil
	}
	b, _ := json.Marshal(map[string]interface{}{
		"host":  host,
		"hours": hours,
		"tasks": tasks,
	})
	return string(b), nil
}

// QueryLogs builds an SSH command to query logs for a host.
// Returns the SSH command string — the caller should execute it via SSH.
func QueryLogs(ctx context.Context, host, service string, lines int, filterKeyword string) string {
	var cmd string
	if service != "" {
		cmd = fmt.Sprintf("journalctl -u %s -n %d --no-pager 2>/dev/null || tail -n %d /var/log/%s*.log 2>/dev/null",
			service, lines, lines, service)
	} else {
		cmd = fmt.Sprintf("journalctl -n %d --no-pager", lines)
	}
	if filterKeyword != "" {
		cmd += fmt.Sprintf(" | grep -i '%s'", filterKeyword)
	}
	cmd += fmt.Sprintf(" || echo 'No logs found for %s'", service)
	return cmd
}

func roundTo1(v float64) float64 {
	return float64(int(v*10)) / 10
}

// timeNow is a shim for time.Now (overridable in tests).
var timeNow = time.Now
