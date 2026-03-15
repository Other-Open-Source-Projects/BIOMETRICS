package opencode

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

const (
	defaultInstallTimeout = 5 * time.Minute
)

// InstallEventHook emits opencode installer lifecycle events.
type InstallEventHook func(runID, eventType string, payload map[string]string)

// Option customizes adapter behavior.
type Option func(*Adapter)

type Adapter struct {
	binary         string
	autoInstall    bool
	installTimeout time.Duration
	installCommand string
	installArgs    []string
	installMu      sync.Mutex
	installHook    InstallEventHook
}

type routingMetadata struct {
	providerID string
	modelID    string
}

func NewAdapter(opts ...Option) *Adapter {
	adapter := &Adapter{
		binary:         "opencode",
		autoInstall:    envBoolDefaultTrue("BIOMETRICS_OPENCODE_AUTO_INSTALL"),
		installTimeout: defaultInstallTimeout,
		installCommand: "brew",
		installArgs:    []string{"install", "opencode"},
	}
	for _, opt := range opts {
		if opt != nil {
			opt(adapter)
		}
	}
	return adapter
}

func WithInstallEventHook(hook InstallEventHook) Option {
	return func(a *Adapter) {
		a.installHook = hook
	}
}

func WithAutoInstall(enabled bool) Option {
	return func(a *Adapter) {
		a.autoInstall = enabled
	}
}

func WithBinary(binary string) Option {
	return func(a *Adapter) {
		if strings.TrimSpace(binary) != "" {
			a.binary = strings.TrimSpace(binary)
		}
	}
}

func WithInstallTimeout(timeout time.Duration) Option {
	return func(a *Adapter) {
		if timeout > 0 {
			a.installTimeout = timeout
		}
	}
}

func WithInstaller(command string, args ...string) Option {
	return func(a *Adapter) {
		trimmed := strings.TrimSpace(command)
		if trimmed == "" {
			return
		}
		a.installCommand = trimmed
		if len(args) > 0 {
			cloned := make([]string, 0, len(args))
			for _, arg := range args {
				if strings.TrimSpace(arg) == "" {
					continue
				}
				cloned = append(cloned, arg)
			}
			if len(cloned) > 0 {
				a.installArgs = cloned
			}
		}
	}
}

func IsAvailable() bool {
	_, err := lookupBinary("opencode")
	return err == nil
}

func (a *Adapter) Execute(ctx context.Context, runID, agentName, prompt, projectID string) (string, error) {
	if strings.TrimSpace(prompt) == "" {
		return "", fmt.Errorf("empty prompt")
	}

	if err := a.ensureBinary(ctx, runID); err != nil {
		return "", err
	}

	binaryPath, err := lookupBinary(a.binary)
	if err != nil {
		return "", fmt.Errorf("%s unavailable after install check: %w", a.binary, err)
	}

	execPrompt, metadata := extractRoutingMetadata(prompt)
	result, err := a.executeOnce(ctx, binaryPath, runID, agentName, execPrompt, projectID, metadata)
	if err != nil && metadata.modelID != "" && isUnsupportedModelFlagError(err) {
		metadata.modelID = ""
		result, err = a.executeOnce(ctx, binaryPath, runID, agentName, execPrompt, projectID, metadata)
	}
	if err != nil {
		return "", err
	}
	return result, nil
}

func (a *Adapter) executeOnce(
	ctx context.Context,
	binaryPath, runID, agentName, prompt, projectID string,
	metadata routingMetadata,
) (string, error) {
	execCtx := ctx
	var cancel context.CancelFunc
	if _, ok := ctx.Deadline(); !ok {
		execCtx, cancel = context.WithTimeout(ctx, 10*time.Minute)
	}
	if cancel != nil {
		defer cancel()
	}

	args := []string{"run"}
	if strings.TrimSpace(agentName) != "" {
		args = append(args, "--agent", agentName)
	}
	if metadata.modelID != "" {
		args = append(args, "--model", metadata.modelID)
	}
	if dir := resolveOpenCodeRunDir(projectID); dir != "" {
		args = append(args, "--dir", dir)
	}
	args = append(args, prompt)

	cmd := exec.CommandContext(execCtx, binaryPath, args...)
	cmd.Env = append(os.Environ(),
		"PROJECT_ID="+projectID,
		"BIOMETRICS_RUN_ID="+runID,
		"BIOMETRICS_MODEL_PROVIDER="+metadata.providerID,
		"BIOMETRICS_MODEL_ID="+metadata.modelID,
	)

	var out bytes.Buffer
	var errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("opencode execute: %w (%s)", err, strings.TrimSpace(errOut.String()))
	}

	result := strings.TrimSpace(out.String())
	if result == "" {
		result = "opencode completed with empty output"
	}
	return result, nil
}

func resolveOpenCodeRunDir(projectID string) string {
	// Explicit override for operator/testing.
	candidates := []string{
		strings.TrimSpace(os.Getenv("BIOMETRICS_OPENCODE_DIR")),
		strings.TrimSpace(os.Getenv("BIOMETRICS_WORKSPACE")),
	}

	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if stat, err := os.Stat(candidate); err == nil && stat.IsDir() {
			return candidate
		}
	}

	if cwd, err := os.Getwd(); err == nil {
		if stat, statErr := os.Stat(cwd); statErr == nil && stat.IsDir() {
			return cwd
		}
	}
	return ""
}

func (a *Adapter) ensureBinary(ctx context.Context, runID string) error {
	if _, err := lookupBinary(a.binary); err == nil {
		return nil
	}

	if !a.autoInstall {
		return fmt.Errorf(
			"%s binary not found; install with `brew install opencode` or run `biometrics-onboard`",
			a.binary,
		)
	}

	a.installMu.Lock()
	defer a.installMu.Unlock()

	if _, err := lookupBinary(a.binary); err == nil {
		return nil
	}

	return a.installBinary(ctx, runID)
}

func (a *Adapter) installBinary(ctx context.Context, runID string) error {
	installCommand := strings.TrimSpace(a.installCommand)
	installArgs := append([]string{}, a.installArgs...)
	if installCommand == "" {
		installCommand = "brew"
	}
	if len(installArgs) == 0 {
		installArgs = []string{"install", "opencode"}
	}

	installCmdDisplay := installCommand + " " + strings.Join(installArgs, " ")
	payload := map[string]string{
		"binary":      a.binary,
		"install_cmd": installCmdDisplay,
	}

	if _, err := lookupBinary(installCommand); err != nil {
		a.emitInstallEvent(runID, "opencode.install.failed", withError(payload, fmt.Sprintf("%s not available in PATH", installCommand)))
		return fmt.Errorf(
			"%s binary not found and installer command %q is unavailable; install Homebrew, then run `brew install opencode` or `biometrics-onboard`",
			a.binary,
			installCommand,
		)
	}

	a.emitInstallEvent(runID, "opencode.install.started", payload)

	installCtx, cancel := context.WithTimeout(ctx, a.installTimeout)
	defer cancel()

	cmd := exec.CommandContext(installCtx, installCommand, installArgs...)
	cmd.Env = withHomebrewPathEnv(os.Environ())

	var out bytes.Buffer
	var errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut

	if err := cmd.Run(); err != nil {
		detail := strings.TrimSpace(errOut.String())
		if detail == "" {
			detail = strings.TrimSpace(out.String())
		}
		if detail == "" {
			detail = err.Error()
		}
		a.emitInstallEvent(runID, "opencode.install.failed", withError(payload, detail))
		return fmt.Errorf(
			"automatic opencode install failed: %s; remediate with `brew install opencode` or run `biometrics-onboard`",
			detail,
		)
	}

	if _, err := lookupBinary(a.binary); err != nil {
		msg := "opencode still unavailable after brew install"
		a.emitInstallEvent(runID, "opencode.install.failed", withError(payload, msg))
		return fmt.Errorf("%s; ensure /opt/homebrew/bin or /usr/local/bin is in PATH", msg)
	}

	a.emitInstallEvent(runID, "opencode.install.succeeded", payload)
	return nil
}

func (a *Adapter) emitInstallEvent(runID, eventType string, payload map[string]string) {
	if a.installHook == nil {
		return
	}
	cloned := make(map[string]string, len(payload))
	for key, value := range payload {
		cloned[key] = value
	}
	a.installHook(runID, eventType, cloned)
}

func lookupBinary(name string) (string, error) {
	if path, err := exec.LookPath(name); err == nil {
		return path, nil
	}

	updatedPath := ensureHomebrewPaths(strings.Split(os.Getenv("PATH"), string(os.PathListSeparator)))
	_ = os.Setenv("PATH", strings.Join(updatedPath, string(os.PathListSeparator)))
	return exec.LookPath(name)
}

func ensureHomebrewPaths(pathEntries []string) []string {
	seen := make(map[string]struct{}, len(pathEntries))
	for _, entry := range pathEntries {
		trimmed := strings.TrimSpace(entry)
		if trimmed == "" {
			continue
		}
		seen[trimmed] = struct{}{}
	}

	candidates := []string{"/opt/homebrew/bin", "/usr/local/bin"}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err != nil {
			continue
		}
		if _, ok := seen[candidate]; ok {
			continue
		}
		pathEntries = append([]string{candidate}, pathEntries...)
		seen[candidate] = struct{}{}
	}
	return pathEntries
}

func withHomebrewPathEnv(env []string) []string {
	if len(env) == 0 {
		pathEntries := ensureHomebrewPaths(strings.Split(os.Getenv("PATH"), string(os.PathListSeparator)))
		return []string{"PATH=" + strings.Join(pathEntries, string(os.PathListSeparator))}
	}

	updated := append([]string{}, env...)
	pathIndex := -1
	pathValue := os.Getenv("PATH")
	for i, entry := range updated {
		if strings.HasPrefix(entry, "PATH=") {
			pathIndex = i
			pathValue = strings.TrimPrefix(entry, "PATH=")
			break
		}
	}
	pathEntries := ensureHomebrewPaths(strings.Split(pathValue, string(os.PathListSeparator)))
	pathEnv := "PATH=" + strings.Join(pathEntries, string(os.PathListSeparator))
	if pathIndex >= 0 {
		updated[pathIndex] = pathEnv
	} else {
		updated = append(updated, pathEnv)
	}
	return updated
}

func envBoolDefaultTrue(key string) bool {
	raw := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	switch raw {
	case "", "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return true
	}
}

func extractRoutingMetadata(prompt string) (string, routingMetadata) {
	trimmed := strings.TrimLeft(prompt, " \t\r\n")
	if !strings.HasPrefix(trimmed, "[") {
		return prompt, routingMetadata{}
	}

	newline := strings.IndexByte(trimmed, '\n')
	if newline < 0 {
		return prompt, routingMetadata{}
	}

	header := strings.TrimSpace(trimmed[:newline])
	body := strings.TrimLeft(trimmed[newline+1:], " \t\r\n")
	if !strings.HasPrefix(header, "[") || !strings.HasSuffix(header, "]") {
		return prompt, routingMetadata{}
	}

	content := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(header, "["), "]"))
	if content == "" {
		return prompt, routingMetadata{}
	}

	meta := routingMetadata{}
	for _, token := range strings.Fields(content) {
		parts := strings.SplitN(strings.TrimSpace(token), "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(parts[0]))
		value := strings.TrimSpace(parts[1])
		switch key {
		case "provider":
			meta.providerID = value
		case "model":
			meta.modelID = value
		}
	}

	if meta.providerID == "" && meta.modelID == "" {
		return prompt, routingMetadata{}
	}
	if body == "" {
		return prompt, meta
	}
	return body, meta
}

func isUnsupportedModelFlagError(err error) bool {
	if err == nil {
		return false
	}
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "unknown flag: --model") ||
		strings.Contains(lower, "flag provided but not defined") ||
		strings.Contains(lower, "unknown option '--model'")
}

func withError(payload map[string]string, message string) map[string]string {
	out := make(map[string]string, len(payload)+1)
	for key, value := range payload {
		out[key] = value
	}
	out["error"] = strings.TrimSpace(message)
	return out
}
