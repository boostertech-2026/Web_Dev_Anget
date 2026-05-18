package agent

import (
	"fmt"
	"strings"

	"langgraph-ops-server/agent/tools"
	"langgraph-ops-server/models"

	"gorm.io/gorm"
)

// gormStore implements all four tool-store interfaces via GORM.
type gormStore struct {
	db *gorm.DB
}

// ---- KnowledgeStore ----

func (s *gormStore) EnsureTable() error {
	return s.db.Exec(`
		CREATE TABLE IF NOT EXISTS knowledge_cases (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			symptoms TEXT,
			diagnosis TEXT,
			root_cause TEXT,
			solution TEXT,
			hosts TEXT,
			tags TEXT,
			resolved_at TEXT,
			created_at TEXT DEFAULT (datetime('now'))
		)
	`).Error
}

func (s *gormStore) Query(symptomPattern, tagPattern, hostPattern string, limit int) ([]tools.KnowledgeCase, error) {
	var rows []struct {
		ID         uint   `gorm:"column:id"`
		Symptoms   string `gorm:"column:symptoms"`
		Diagnosis  string `gorm:"column:diagnosis"`
		RootCause  string `gorm:"column:root_cause"`
		Solution   string `gorm:"column:solution"`
		Hosts      string `gorm:"column:hosts"`
		Tags       string `gorm:"column:tags"`
		ResolvedAt string `gorm:"column:resolved_at"`
	}
	err := s.db.Raw(`
		SELECT id, symptoms, diagnosis, root_cause, solution, hosts, tags, resolved_at
		FROM knowledge_cases
		WHERE symptoms LIKE ? OR tags LIKE ? OR hosts LIKE ?
		ORDER BY created_at DESC LIMIT ?
	`, symptomPattern, tagPattern, hostPattern, limit).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]tools.KnowledgeCase, len(rows))
	for i, r := range rows {
		out[i] = tools.KnowledgeCase{
			ID: r.ID, Symptoms: r.Symptoms, Diagnosis: r.Diagnosis,
			RootCause: r.RootCause, Solution: r.Solution,
			Hosts: r.Hosts, Tags: r.Tags, ResolvedAt: r.ResolvedAt,
		}
	}
	return out, nil
}

func (s *gormStore) Insert(symptoms, diagnosis, rootCause, solution, hosts, tags string) (int64, error) {
	res := s.db.Exec(`
		INSERT INTO knowledge_cases (symptoms, diagnosis, root_cause, solution, hosts, tags, resolved_at)
		VALUES (?, ?, ?, ?, ?, ?, datetime('now'))
	`, symptoms, diagnosis, rootCause, solution, hosts, tags)
	if res.Error != nil {
		return 0, res.Error
	}
	return res.RowsAffected, nil
}

// ---- MonitorStore ----

func (s *gormStore) LatestMetric(hostIDOrName string) (*tools.MetricRow, error) {
	var m tools.MetricRow
	err := s.db.Raw(`
		SELECT cpu_percent, mem_percent, mem_used, mem_total,
		       disk_percent, load_1m, load_5m, net_rx_rate, net_tx_rate, process_top
		FROM metrics
		WHERE host_name = ? OR host_id = ?
		ORDER BY created_at DESC LIMIT 1
	`, hostIDOrName, hostIDOrName).Scan(&m).Error
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (s *gormStore) RecentTasks(hostIDOrName string, hours int, limit int) ([]tools.TaskSummary, error) {
	escaped := strings.ReplaceAll(strings.ReplaceAll(hostIDOrName, "%", "\\%"), "_", "\\_")
	var rows []tools.TaskSummary
	err := s.db.Raw(`
		SELECT name, exec_type, status, created_at, COALESCE(result,'') AS result
		FROM tasks
		WHERE (host_ids LIKE ? ESCAPE '\' OR host_ids LIKE ? ESCAPE '\')
		  AND status IN ('success','failed')
		ORDER BY created_at DESC LIMIT ?
	`, "%"+escaped+"%", "%"+escaped+"%", limit).Scan(&rows).Error
	return rows, err
}

// ---- HostResolver (SSH) ----

func (s *gormStore) Resolve(hostIdentifier string) (*tools.SSHConfig, error) {
	var h models.Host
	err := s.db.Where("id = ? OR name = ? OR host = ?", hostIdentifier, hostIdentifier, hostIdentifier).First(&h).Error
	if err != nil {
		return nil, fmt.Errorf("host not found: %s", hostIdentifier)
	}
	password, err := models.DecryptCredential(h.Credential)
	if err != nil {
		return nil, fmt.Errorf("decrypt credential for host %s: %w", h.Name, err)
	}
	return &tools.SSHConfig{
		Name:     h.Name,
		Host:     h.Host,
		Port:     h.Port,
		Username: h.Username,
		AuthType: h.AuthType,
		Password: password,
	}, nil
}

// ---- IPMIResolver ----

func (s *gormStore) ResolveIPMI(hostIdentifier string) (*tools.IPMIConfig, error) {
	var h models.Host
	err := s.db.Where("id = ? OR name = ? OR host = ?", hostIdentifier, hostIdentifier, hostIdentifier).First(&h).Error
	if err != nil {
		return nil, fmt.Errorf("host not found: %s", hostIdentifier)
	}
	if h.IpmiHost == "" {
		return nil, fmt.Errorf("host %s has no IPMI configured", h.Name)
	}
	ipmiPassword, err := models.DecryptCredential(h.IpmiPassword)
	if err != nil {
		return nil, fmt.Errorf("decrypt IPMI credential for host %s: %w", h.Name, err)
	}
	return &tools.IPMIConfig{
		Name:     h.Name,
		Host:     h.IpmiHost,
		User:     h.IpmiUser,
		Password: ipmiPassword,
	}, nil
}

// ---- AuditStore ----

func (s *gormStore) EnsureAuditTable() error {
	return s.db.Exec(`
		CREATE TABLE IF NOT EXISTS audit_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			thread_id TEXT,
			user TEXT,
			intent TEXT,
			plan TEXT,
			tools_called TEXT,
			high_risk_ops TEXT,
			approved INTEGER DEFAULT 0,
			backup_path TEXT,
			hosts_affected TEXT,
			final_result TEXT,
			observations TEXT,
			knowledge_saved INTEGER DEFAULT 0,
			status TEXT DEFAULT 'completed',
			created_at TEXT DEFAULT (datetime('now'))
		)
	`).Error
}

func (s *gormStore) InsertAuditLog(entry AuditEntry) error {
	return s.db.Exec(`
		INSERT INTO audit_logs (thread_id, user, intent, plan, tools_called, high_risk_ops,
			approved, backup_path, hosts_affected, final_result, observations,
			knowledge_saved, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, entry.ThreadID, entry.User, entry.Intent, entry.Plan, entry.ToolsCalled,
		entry.HighRiskOps, entry.Approved, entry.BackupPath, entry.HostsAffected,
		entry.FinalResult, entry.Observations, entry.KnowledgeSaved, entry.Status).Error
}

// AuditEntry is a flat struct for passing audit data to the store.
type AuditEntry struct {
	ThreadID       string
	User           string
	Intent         string
	Plan           string
	ToolsCalled    string
	HighRiskOps    string
	Approved       bool
	BackupPath     string
	HostsAffected  string
	FinalResult    string
	Observations   string
	KnowledgeSaved bool
	Status         string
}

// Compile-time interface checks
var (
	_ tools.KnowledgeStore = (*gormStore)(nil)
	_ tools.MonitorStore   = (*gormStore)(nil)
	_ tools.HostResolver   = (*gormStore)(nil)
	_ tools.IPMIResolver   = (*gormStore)(nil)
)

