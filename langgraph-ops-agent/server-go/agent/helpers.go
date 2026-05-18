package agent

import (
	"context"
	"fmt"

	"langgraph-ops-server/agent/tools"
	"langgraph-ops-server/models"
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
