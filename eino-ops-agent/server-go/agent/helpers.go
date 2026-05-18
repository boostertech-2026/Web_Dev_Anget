package agent

import (
	"context"
	"fmt"
	"log"
	"time"

	"eino-ops-server/agent/tools"
	"eino-ops-server/models"
)

// ExecuteSSH performs a direct SSH command execution for a given host model.
// Used by the task dispatch system in handlers.
func ExecuteSSH(host models.Host, cmd string) (string, error) {
	if knowledgeStore == nil {
		return "", fmt.Errorf("agent store not initialized")
	}
	cfg, err := knowledgeStore.Resolve(fmt.Sprintf("%d", host.ID))
	if err != nil {
		return "", err
	}
	return tools.SshExec(context.Background(), knowledgeStore, cfg.Name, cmd)
}

// StartHealthCheck begins a background goroutine that periodically checks
// SSH connectivity for all hosts and updates their online/offline status.
func StartHealthCheck(interval time.Duration) {
	go func() {
		// Wait a few seconds for everything to initialize
		time.Sleep(5 * time.Second)
		log.Printf("[health] starting host health checker (interval=%s)", interval)

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			checkAllHosts()
		}
	}()
}

func checkAllHosts() {
	if knowledgeStore == nil {
		return
	}

	var hosts []models.Host
	if err := models.DB.Find(&hosts).Error; err != nil {
		log.Printf("[health] failed to query hosts: %v", err)
		return
	}

	for _, h := range hosts {
		prev := h.Status
		cfg, err := knowledgeStore.Resolve(fmt.Sprintf("%d", h.ID))
		if err != nil {
			if h.Status != "offline" {
				models.DB.Model(&h).Update("status", "offline")
				log.Printf("[health] %s (%s) → offline (resolve: %v)", h.Name, h.Host, err)
			}
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		result, err := tools.SshExec(ctx, knowledgeStore, cfg.Name, "echo OK")
		cancel()

		if err != nil || !isSSHSuccess(result) {
			if h.Status != "offline" {
				models.DB.Model(&h).Update("status", "offline")
				log.Printf("[health] %s (%s) → offline", h.Name, h.Host)
			}
		} else {
			if h.Status != "online" {
				models.DB.Model(&h).Update("status", "online")
				log.Printf("[health] %s (%s) → online", h.Name, h.Host)
			}
		}

		if prev != h.Status {
			h.LastOperated = time.Now()
			models.DB.Model(&h).Update("last_operated", time.Now())
		}
	}
}

func isSSHSuccess(jsonResult string) bool {
	// SshExec returns JSON like {"success":true,"stdout":"OK\n",...}
	for i := 0; i < len(jsonResult); i++ {
		if jsonResult[i] == '"' && i+9 < len(jsonResult) && jsonResult[i:i+9] == `"success"` {
			rest := jsonResult[i+9:]
			for j := 0; j < len(rest); j++ {
				if rest[j] == 't' && j+3 < len(rest) && rest[j:j+4] == "true" {
					return true
				}
			}
		}
	}
	return false
}
