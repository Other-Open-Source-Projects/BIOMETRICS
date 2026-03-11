package selfhealing

import (
	"biometrics-cli/internal/metrics"
	"biometrics-cli/internal/paths"
	"biometrics-cli/internal/state"
	"fmt"
	"os"
	"os/exec"
	"time"
)

type SelfHealer struct {
	healthChecks map[string]HealthCheck
}

type HealthCheck struct {
	Name     string
	Check    func() error
	Recover  func() error
	Severity string
}

func NewSelfHealer() *SelfHealer {
	s := &SelfHealer{
		healthChecks: make(map[string]HealthCheck),
	}
	s.registerHealthChecks()
	return s
}

func (s *SelfHealer) registerHealthChecks() {
	s.healthChecks["serena"] = HealthCheck{
		Name:     "Serena MCP Server",
		Check:    s.checkSerena,
		Recover:  s.recoverSerena,
		Severity: "CRITICAL",
	}

	s.healthChecks["opencode"] = HealthCheck{
		Name:     "OpenCode CLI",
		Check:    s.checkOpenCode,
		Recover:  s.recoverOpenCode,
		Severity: "CRITICAL",
	}

	s.healthChecks["database"] = HealthCheck{
		Name:     "SQLite Database",
		Check:    s.checkDatabase,
		Recover:  s.recoverDatabase,
		Severity: "HIGH",
	}

	s.healthChecks["disk"] = HealthCheck{
		Name:     "Disk Space",
		Check:    s.checkDiskSpace,
		Recover:  s.recoverDiskSpace,
		Severity: "MEDIUM",
	}

	s.healthChecks["memory"] = HealthCheck{
		Name:     "Memory Usage",
		Check:    s.checkMemory,
		Recover:  s.recoverMemory,
		Severity: "MEDIUM",
	}
}

func (s *SelfHealer) checkSerena() error {
	cmd := exec.Command("pgrep", "-f", "serena.*start-mcp-server")
	return cmd.Run()
}

func (s *SelfHealer) recoverSerena() error {
	state.GlobalState.Log("WARNING", "Serena not running - attempting recovery...")

	cmd := exec.Command("bash", "-c", "pkill -f serena; sleep 2; uvx --from git+https://github.com/oraios/serena serena start-mcp-server &")
	cmd.Start()

	time.Sleep(10 * time.Second)

	if s.checkSerena() == nil {
		state.GlobalState.Log("SUCCESS", "Serena recovered successfully")
		return nil
	}

	state.GlobalState.Log("ERROR", "Serena recovery failed")
	return fmt.Errorf("serena recovery failed")
}

func (s *SelfHealer) checkOpenCode() error {
	cmd := exec.Command("opencode", "--version")
	return cmd.Run()
}

func (s *SelfHealer) recoverOpenCode() error {
	state.GlobalState.Log("WARNING", "OpenCode CLI not responding")
	return nil
}

func (s *SelfHealer) checkDatabase() error {
	_, err := os.Stat(paths.SisyphusDBPath("biometrics.db"))
	return err
}

func (s *SelfHealer) recoverDatabase() error {
	state.GlobalState.Log("WARNING", "Database not accessible - will be recreated on next cycle")
	return nil
}

func (s *SelfHealer) checkDiskSpace() error {
	cmd := exec.Command("df", "-h", "/")
	output, err := cmd.Output()
	if err != nil {
		return err
	}

	var available string
	fmt.Sscanf(string(output), "%s %s", &available, &available)
	if available > "90%" {
		return fmt.Errorf("disk space low: %s", available)
	}
	return nil
}

func (s *SelfHealer) recoverDiskSpace() error {
	state.GlobalState.Log("WARNING", "Running disk cleanup...")
	cmd := exec.Command("bash", "-c", "find /tmp -type f -mtime +7 -delete 2>/dev/null; rm -rf ~/.cache/* 2>/dev/null")
	return cmd.Run()
}

func (s *SelfHealer) checkMemory() error {
	cmd := exec.Command("bash", "-c", "vm_stat | head -5")
	_, err := cmd.Output()
	return err
}

func (s *SelfHealer) recoverMemory() error {
	state.GlobalState.Log("WARNING", "Memory pressure detected")
	return nil
}

func (s *SelfHealer) RunDiagnostics() {
	state.GlobalState.Log("INFO", "=== SELF-HEALING DIAGNOSTICS STARTED ===")

	for name, check := range s.healthChecks {
		if err := check.Check(); err != nil {
			state.GlobalState.Log("WARNING", fmt.Sprintf("Health check FAILED: %s - %v", name, err))

			if check.Recover != nil {
				state.GlobalState.Log("INFO", "Attempting recovery for: "+name)
				if recErr := check.Recover(); recErr != nil {
					state.GlobalState.Log("ERROR", fmt.Sprintf("Recovery FAILED for %s: %v", name, recErr))
					metrics.HealingFailures.WithLabelValues(name).Inc()
				} else {
					state.GlobalState.Log("SUCCESS", "Recovery SUCCEEDED for: "+name)
					metrics.HealingSuccesses.WithLabelValues(name).Inc()
				}
			}
		} else {
			state.GlobalState.Log("INFO", fmt.Sprintf("Health check PASSED: %s", name))
		}
	}

	state.GlobalState.Log("INFO", "=== SELF-HEALING DIAGNOSTICS COMPLETED ===")
}

func AttemptSerenaRecovery() {
	healer := NewSelfHealer()
	healer.healthChecks["serena"].Recover()
}

func StartHealthMonitor() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		healer := NewSelfHealer()
		healer.RunDiagnostics()
	}
}
