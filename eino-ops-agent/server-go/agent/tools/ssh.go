package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

// SSHConfig holds connection details for a host.
type SSHConfig struct {
	Name     string
	Host     string
	Port     int
	Username string
	AuthType string // "password" or "key"
	Password string // credential (password or private key)
}

// HostResolver looks up SSH config for a host identifier.
type HostResolver interface {
	Resolve(hostIdentifier string) (*SSHConfig, error)
}

// SshExec runs a command on a single host via SSH. Returns JSON.
func SshExec(ctx context.Context, resolver HostResolver, hostIdentifier, cmd string) (string, error) {
	cfg, err := resolver.Resolve(hostIdentifier)
	if err != nil {
		return jsonString(map[string]interface{}{"error": err.Error(), "host": hostIdentifier, "success": false}), nil
	}

	result := execSSH(cfg, cmd)
	result["host"] = cfg.Name
	return jsonString(result), nil
}

// SshBatchExec runs a command on multiple hosts in parallel. Returns JSON.
func SshBatchExec(ctx context.Context, resolver HostResolver, hostIDs []string, cmd string) (string, error) {
	type batched struct {
		Success int        `json:"success"`
		Failed  int        `json:"failed"`
		Total   int        `json:"total"`
		Results []string   `json:"results"`
	}

	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, 10) // max 10 concurrent

	out := batched{Total: len(hostIDs)}
	for _, hid := range hostIDs {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			r, _ := SshExec(ctx, resolver, id, cmd)
			mu.Lock()
			out.Results = append(out.Results, r)
			if isSuccess(r) {
				out.Success++
			} else {
				out.Failed++
			}
			mu.Unlock()
		}(hid)
	}
	wg.Wait()
	return jsonString(out), nil
}

func execSSH(cfg *SSHConfig, cmd string) map[string]interface{} {
	port := cfg.Port
	if port == 0 {
		port = 22
	}

	var auth ssh.AuthMethod
	if cfg.AuthType == "key" {
		signer, err := ssh.ParsePrivateKey([]byte(cfg.Password))
		if err != nil {
			return map[string]interface{}{"error": fmt.Sprintf("parse key: %v", err), "success": false}
		}
		auth = ssh.PublicKeys(signer)
	} else {
		auth = ssh.Password(cfg.Password)
	}

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", cfg.Host, port), &ssh.ClientConfig{
		User:            cfg.Username,
		Auth:            []ssh.AuthMethod{auth},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	})
	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("ssh connect: %v", err), "success": false}
	}
	defer client.Close()

	sess, err := client.NewSession()
	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("ssh session: %v", err), "success": false}
	}
	defer sess.Close()

	var stdout, stderr bytes.Buffer
	sess.Stdout = &stdout
	sess.Stderr = &stderr

	done := make(chan error, 1)
	go func() { done <- sess.Run(cmd) }()

	select {
	case err := <-done:
		if err != nil {
			return map[string]interface{}{
				"stdout":  truncate(stdout.String(), 8000),
				"stderr":  truncate(stderr.String(), 4000),
				"success": false,
				"error":   err.Error(),
			}
		}
	case <-time.After(60 * time.Second):
		return map[string]interface{}{"error": "ssh command timeout (30s)", "success": false}
	}

	return map[string]interface{}{
		"stdout":  truncate(stdout.String(), 8000),
		"stderr":  truncate(stderr.String(), 4000),
		"success": true,
	}
}

func truncate(s string, max int) string {
	if len(s) > max {
		return s[:max]
	}
	return s
}

func isSuccess(rawJSON string) bool {
	var m map[string]interface{}
	if json.Unmarshal([]byte(rawJSON), &m) != nil {
		return false
	}
	v, _ := m["success"].(bool)
	return v
}

func jsonString(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}
