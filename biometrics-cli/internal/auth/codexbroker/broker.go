package codexbroker

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"biometrics-cli/internal/contracts"
)

const (
	defaultBinary = "codex"
)

type Interface interface {
	Status(ctx context.Context) (contracts.CodexAuthStatus, error)
	Login(ctx context.Context) (contracts.CodexAuthStatus, error)
	Logout(ctx context.Context) (contracts.CodexAuthStatus, error)
}

type Broker struct {
	binary string

	mu   sync.RWMutex
	last contracts.CodexAuthStatus
}

func New() *Broker {
	binary := strings.TrimSpace(os.Getenv("BIOMETRICS_CODEX_CLI_BINARY"))
	if binary == "" {
		binary = defaultBinary
	}
	return &Broker{
		binary: binary,
	}
}

func (b *Broker) Status(ctx context.Context) (contracts.CodexAuthStatus, error) {
	binaryPath, err := exec.LookPath(b.binary)
	if err != nil {
		status := contracts.CodexAuthStatus{
			Ready:       false,
			LoggedIn:    false,
			LastError:   fmt.Sprintf("codex cli not found in PATH: %s", b.binary),
			LastChecked: time.Now().UTC(),
		}
		b.setLast(status)
		return status, nil
	}

	status := contracts.CodexAuthStatus{
		Ready:       false,
		LoggedIn:    false,
		LastChecked: time.Now().UTC(),
	}

	output, _, execErr := runCommand(ctx, binaryPath, []string{"whoami"})
	if execErr == nil {
		status.Ready = true
		status.LoggedIn = true
		status.User = parseFirstLine(output)
		if status.User == "" {
			status.User = "codex-user"
		}
		status.LastError = ""
		b.setLast(status)
		return status, nil
	}

	output, _, execErr = runCommand(ctx, binaryPath, []string{"auth", "status"})
	if execErr == nil {
		status.Ready = true
		status.LoggedIn = detectLoggedIn(output)
		status.User = parseUserFromStatus(output)
		status.LastError = ""
		b.setLast(status)
		return status, nil
	}

	status.LastError = fmt.Sprintf("codex auth status check failed: %v", execErr)
	b.setLast(status)
	return status, nil
}

func (b *Broker) Login(ctx context.Context) (contracts.CodexAuthStatus, error) {
	binaryPath, err := exec.LookPath(b.binary)
	if err != nil {
		return contracts.CodexAuthStatus{
			Ready:       false,
			LoggedIn:    false,
			LastError:   fmt.Sprintf("codex cli not found in PATH: %s", b.binary),
			LastChecked: time.Now().UTC(),
		}, nil
	}

	candidates := [][]string{
		{"--login"},
		{"login"},
		{"auth", "login"},
	}
	var lastErr error
	for _, args := range candidates {
		if _, _, execErr := runCommand(ctx, binaryPath, args); execErr == nil {
			return b.Status(ctx)
		} else {
			lastErr = execErr
		}
	}

	status, _ := b.Status(ctx)
	if status.LastError == "" {
		status.LastError = fmt.Sprintf("codex login failed: %v", lastErr)
	}
	b.setLast(status)
	return status, fmt.Errorf("codex login failed")
}

func (b *Broker) Logout(ctx context.Context) (contracts.CodexAuthStatus, error) {
	binaryPath, err := exec.LookPath(b.binary)
	if err != nil {
		return contracts.CodexAuthStatus{
			Ready:       false,
			LoggedIn:    false,
			LastError:   fmt.Sprintf("codex cli not found in PATH: %s", b.binary),
			LastChecked: time.Now().UTC(),
		}, nil
	}

	candidates := [][]string{
		{"logout"},
		{"auth", "logout"},
		{"--logout"},
	}
	var lastErr error
	for _, args := range candidates {
		if _, _, execErr := runCommand(ctx, binaryPath, args); execErr == nil {
			status := contracts.CodexAuthStatus{
				Ready:       true,
				LoggedIn:    false,
				LastChecked: time.Now().UTC(),
			}
			b.setLast(status)
			return status, nil
		} else {
			lastErr = execErr
		}
	}

	status, _ := b.Status(ctx)
	if status.LastError == "" {
		status.LastError = fmt.Sprintf("codex logout failed: %v", lastErr)
	}
	b.setLast(status)
	return status, fmt.Errorf("codex logout failed")
}

func (b *Broker) Last() contracts.CodexAuthStatus {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.last
}

func (b *Broker) setLast(status contracts.CodexAuthStatus) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.last = status
}

func runCommand(ctx context.Context, binaryPath string, args []string) (string, string, error) {
	callCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	cmd := exec.CommandContext(callCtx, binaryPath, args...)
	cmd.Env = os.Environ()
	out, err := cmd.Output()
	if err == nil {
		return strings.TrimSpace(string(out)), "", nil
	}

	if exitErr, ok := err.(*exec.ExitError); ok {
		return strings.TrimSpace(string(out)), strings.TrimSpace(string(exitErr.Stderr)), err
	}
	return strings.TrimSpace(string(out)), "", err
}

func parseFirstLine(output string) string {
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func detectLoggedIn(output string) bool {
	text := strings.ToLower(output)
	if strings.Contains(text, "not logged in") || strings.Contains(text, "logged out") {
		return false
	}
	return strings.Contains(text, "logged in") || strings.Contains(text, "authenticated")
}

func parseUserFromStatus(output string) string {
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		lower := strings.ToLower(trimmed)
		if strings.HasPrefix(lower, "user:") || strings.HasPrefix(lower, "account:") {
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return ""
}
