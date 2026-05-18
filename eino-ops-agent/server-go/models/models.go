package models

import (
	"log"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

// ---------------------------------------------------------------------------
// Core models
// ---------------------------------------------------------------------------
type User struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	Username string `gorm:"uniqueIndex" json:"username"`
	Password string `json:"-"`
}

type Host struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Name         string    `json:"name"`
	Host         string    `json:"host"`
	Port         int       `json:"port"`
	Username     string    `json:"username"`
	AuthType     string    `json:"auth_type"`
	Credential   string    `json:"-"`
	Status       string    `json:"status"`
	IpmiHost     string    `json:"ipmi_host"`
	IpmiUser     string    `json:"ipmi_user"`
	IpmiPassword string    `json:"-"`
	IpmiStatus   string    `json:"ipmi_status"`
	BmcVersion   string    `json:"bmc_version"`
	GroupTag     string    `json:"group_tag"`
	RegionTag    string    `json:"region_tag"`
	LastOperated time.Time `json:"last_operated"`
}

type Task struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `json:"name"`
	ExecType  string    `json:"exec_type"`
	Command   string    `json:"command"`
	HostIDs   string    `json:"host_ids"`
	ClientIDs string    `json:"client_ids"`
	Status    string    `json:"status"`
	Result    string    `json:"result"`
	Logs      string    `gorm:"type:text" json:"logs"`
	CreatedAt time.Time `json:"created_at"`
}

type Client struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `json:"name"`
	Host      string    `json:"host"`
	Status    string    `json:"status"`
	LastHeart time.Time `json:"last_heart"`
}

// ---------------------------------------------------------------------------
// Monitoring models (new)
// ---------------------------------------------------------------------------
type Metric struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	HostID       uint      `json:"host_id"`
	HostName     string    `json:"host_name"`
	CpuPercent   float64   `json:"cpu_percent"`
	MemTotal     int64     `json:"mem_total"`
	MemUsed      int64     `json:"mem_used"`
	MemPercent   float64   `json:"mem_percent"`
	DiskTotal    int64     `json:"disk_total"`
	DiskUsed     int64     `json:"disk_used"`
	DiskPercent  float64   `json:"disk_percent"`
	Load1m       float64   `json:"load_1m"`
	Load5m       float64   `json:"load_5m"`
	Load15m      float64   `json:"load_15m"`
	NetRxBytes   int64     `json:"net_rx_bytes"`
	NetTxBytes   int64     `json:"net_tx_bytes"`
	NetRxRate    float64   `json:"net_rx_rate"`
	NetTxRate    float64   `json:"net_tx_rate"`
	ProcessTop   string    `json:"process_top" gorm:"type:text"`
	DiskIORead   float64   `json:"disk_io_read"`
	DiskIOWrite  float64   `json:"disk_io_write"`
	CreatedAt    time.Time `json:"created_at"`
}

type Alert struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	HostID      uint      `json:"host_id"`
	HostName    string    `json:"host_name"`
	RuleName    string    `json:"rule_name"`
	Level       string    `json:"level"`
	Message     string    `json:"message"`
	Status      string    `json:"status"` // firing/acked/resolved
	AckedBy     string    `json:"acked_by"`
	ResolvedBy  string    `json:"resolved_by"`
	ResolveNote string    `json:"resolve_note"`
	CreatedAt   time.Time `json:"created_at"`
	ResolvedAt  time.Time `json:"resolved_at"`
}

type AlertRule struct {
	ID        uint    `gorm:"primaryKey" json:"id"`
	Name      string  `json:"name"`
	Metric    string  `json:"metric"`
	Condition string  `json:"condition"`
	Threshold float64 `json:"threshold"`
	Duration  int     `json:"duration"` // seconds
	Level     string  `json:"level"`
	Enabled   bool    `json:"enabled"`
}

type AuditLog struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	ThreadID       string    `json:"thread_id"`
	User           string    `json:"user"`
	Intent         string    `json:"intent"`
	Plan           string    `gorm:"type:text" json:"plan"`
	ToolsCalled    string    `gorm:"type:text" json:"tools_called"`
	HighRiskOps    string    `gorm:"type:text" json:"high_risk_ops"`
	Approved       bool      `json:"approved"`
	BackupPath     string    `json:"backup_path"`
	HostsAffected  string    `json:"hosts_affected"`
	FinalResult    string    `gorm:"type:text" json:"final_result"`
	Observations   string    `gorm:"type:text" json:"observations"`
	KnowledgeSaved bool      `json:"knowledge_saved"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
}

type ErrorHistory struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	HostID          uint      `json:"host_id"`
	HostName        string    `json:"host_name"`
	TaskID          uint      `json:"task_id"`
	Level           string    `json:"level"`
	Message         string    `gorm:"type:text" json:"message"`
	Source          string    `json:"source"`
	Status          string    `json:"status"` // pending/processing/resolved
	HandledBy       string    `json:"handled_by"`
	HandleNote      string    `json:"handle_note"`
	CreatedAt       time.Time `json:"created_at"`
	ResolvedAt      time.Time `json:"resolved_at"`
	ResolveDuration int64     `json:"resolve_duration"` // seconds
}

// ---------------------------------------------------------------------------
// Init
// ---------------------------------------------------------------------------
func InitDB() {
	db, err := gorm.Open(sqlite.Open("ops.db"), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	DB = db

	db.AutoMigrate(
		&User{}, &Host{}, &Task{}, &Client{},
		&Metric{}, &Alert{}, &AlertRule{}, &ErrorHistory{},
		&AuditLog{},
	)

	var count int64
	db.Model(&User{}).Count(&count)
	if count == 0 {
		hash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("[init] failed to hash default password: %v", err)
			hash = []byte("$2a$10$placeholder") // won't match, but prevents panic
		}
		db.Create(&User{Username: "admin", Password: string(hash)})
	}

	// Seed default alert rules
	db.Model(&AlertRule{}).Count(&count)
	if count == 0 {
		defaultRules := []AlertRule{
			{Name: "CPU过高", Metric: "cpu", Condition: ">", Threshold: 90, Duration: 300, Level: "critical", Enabled: true},
			{Name: "内存不足", Metric: "mem", Condition: ">", Threshold: 95, Duration: 300, Level: "critical", Enabled: true},
			{Name: "磁盘空间低", Metric: "disk", Condition: ">", Threshold: 85, Duration: 300, Level: "warning", Enabled: true},
			{Name: "主机离线", Metric: "host_down", Condition: "==", Threshold: 0, Duration: 120, Level: "critical", Enabled: true},
		}
		for _, r := range defaultRules {
			db.Create(&r)
		}
	}
}
