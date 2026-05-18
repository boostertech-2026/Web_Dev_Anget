package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// IPMIConfig holds IPMI connection info.
type IPMIConfig struct {
	Name     string
	Host     string
	User     string
	Password string
}

// IPMIResolver looks up IPMI config for a host.
type IPMIResolver interface {
	ResolveIPMI(hostIdentifier string) (*IPMIConfig, error)
}

func runIPMI(cfg *IPMIConfig, args ...string) map[string]interface{} {
	base := []string{"-H", cfg.Host, "-U", cfg.User, "-P", cfg.Password, "-I", "lanplus"}
	cmdArgs := append(base, args...)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ipmitool", cmdArgs...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("%v: %s", err, strings.TrimSpace(string(out))),
		}
	}
	return map[string]interface{}{
		"success": true,
		"stdout":  truncateString(string(out), 4000),
	}
}

func ipmiResult(cfg *IPMIConfig, action string, result map[string]interface{}) string {
	result["host"] = cfg.Name
	result["action"] = action
	b, _ := json.Marshal(result)
	return string(b)
}

// IpmiPower controls host power. action: status/on/off/reset/cycle
func IpmiPower(ctx context.Context, resolver IPMIResolver, host, action string) (string, error) {
	valid := map[string]bool{"status": true, "on": true, "off": true, "reset": true, "cycle": true}
	if !valid[action] {
		return `{"error": "Invalid power action, choices: status/on/off/reset/cycle"}`, nil
	}
	cfg, err := resolver.ResolveIPMI(host)
	if err != nil {
		return fmt.Sprintf(`{"error": "%s"}`, err.Error()), nil
	}
	if cfg == nil {
		return fmt.Sprintf(`{"error": "Host %s not found or no IPMI configured"}`, host), nil
	}
	r := runIPMI(cfg, "chassis", "power", action)
	return ipmiResult(cfg, "power_"+action, r), nil
}

// IpmiBootdev sets boot device. device: pxe/cdrom/bios/disk
func IpmiBootdev(ctx context.Context, resolver IPMIResolver, host, device string) (string, error) {
	valid := map[string]bool{"pxe": true, "cdrom": true, "bios": true, "disk": true}
	if !valid[device] {
		return `{"error": "Invalid boot device, choices: pxe/cdrom/bios/disk"}`, nil
	}
	cfg, err := resolver.ResolveIPMI(host)
	if err != nil {
		return fmt.Sprintf(`{"error": "%s"}`, err.Error()), nil
	}
	if cfg == nil {
		return fmt.Sprintf(`{"error": "Host %s not found or no IPMI configured"}`, host), nil
	}
	r := runIPMI(cfg, "chassis", "bootdev", device)
	return ipmiResult(cfg, "bootdev_"+device, r), nil
}

// IpmiResetPassword resets BMC user password.
func IpmiResetPassword(ctx context.Context, resolver IPMIResolver, host, userID, newPassword string) (string, error) {
	cfg, err := resolver.ResolveIPMI(host)
	if err != nil {
		return fmt.Sprintf(`{"error": "%s"}`, err.Error()), nil
	}
	if cfg == nil {
		return fmt.Sprintf(`{"error": "Host %s not found or no IPMI configured"}`, host), nil
	}
	r := runIPMI(cfg, "user", "set", "password", userID, newPassword)
	if r["success"] == true {
		r["message"] = fmt.Sprintf("BMC password for user %s has been reset successfully.", userID)
	}
	return ipmiResult(cfg, "reset_password", r), nil
}

// IpmiSensor reads IPMI sensor data.
func IpmiSensor(ctx context.Context, resolver IPMIResolver, host string) (string, error) {
	cfg, err := resolver.ResolveIPMI(host)
	if err != nil {
		return fmt.Sprintf(`{"error": "%s"}`, err.Error()), nil
	}
	if cfg == nil {
		return fmt.Sprintf(`{"error": "Host %s not found or no IPMI configured"}`, host), nil
	}
	r := runIPMI(cfg, "sensor", "list")
	return ipmiResult(cfg, "sensor_list", r), nil
}

// IpmiSEL reads IPMI System Event Log.
func IpmiSEL(ctx context.Context, resolver IPMIResolver, host string) (string, error) {
	cfg, err := resolver.ResolveIPMI(host)
	if err != nil {
		return fmt.Sprintf(`{"error": "%s"}`, err.Error()), nil
	}
	if cfg == nil {
		return fmt.Sprintf(`{"error": "Host %s not found or no IPMI configured"}`, host), nil
	}
	r := runIPMI(cfg, "sel", "list")
	return ipmiResult(cfg, "sel_list", r), nil
}

func truncateString(s string, max int) string {
	if len(s) > max {
		return s[:max]
	}
	return s
}
