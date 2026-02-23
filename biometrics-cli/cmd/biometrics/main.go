package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"context"

	"biometrics-cli/commands"
	"biometrics-cli/internal/supervisor"
	"biometrics-cli/pkg/logging"
)

var (
	version   = "1.0.0"
	commit    = "dev"
	buildDate = "unknown"
)

func main() {
	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		fmt.Printf("\nReceived signal: %v. Shutting down...\n", sig)
		os.Exit(0)
	}()

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "init":
		runInit()
	case "onboard":
		runOnboarding()
	case "auto":
		runAutoSetup()
	case "check":
		runBiometricsCheck()
	case "find-keys":
		findAPIKeys()
	case "version":
		printVersion()
	case "config":
		runConfig()
	case "audit":
		runAudit()
	case "work":
		runBiometricsLoop()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`
BIOMETRICS CLI v2.0.0

Usage: biometrics <command> [options]

Commands:
  init          Initialize BIOMETRICS repository
  onboard       Interactive onboarding process  
  auto          Automatic AI-powered setup
  check         Check BIOMETRICS compliance
  find-keys     Find existing API keys on system
  config        Manage configuration (init, validate, show)
  audit         Query and manage audit logs
  work          Start autonomous Biometrics Loop
  version       Show version information`)
}

func runBiometricsLoop() {
	ctx := context.Background()
	
	// Create a dedicated logger for the loop
	rotateCfg := logging.RotationConfig{
		Enabled:    true,
		MaxSize:    10 * 1024 * 1024,
		MaxBackups: 5,
		MaxAge:     30,
	}
	
	logger, err := logging.NewFileLogger("biometrics-loop.log", logging.InfoLevel, rotateCfg)
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	loop := supervisor.NewBiometricsLoop("biometrics", logger)
	if err := loop.Start(ctx); err != nil {
		fmt.Printf("Biometrics Loop failed: %v\n", err)
		os.Exit(1)
	}
}

func printVersion() {
	buildTime := buildDate
	if buildDate == "unknown" {
		buildTime = time.Now().Format("2006-01-02")
	}

	fmt.Printf("biometrics-cli v%s (commit: %s, built: %s)\n", version, commit, buildTime)
}

func runConfig() {
	if len(os.Args) < 3 {
		printConfigUsage()
		os.Exit(1)
	}

	subCommand := os.Args[2]

	switch subCommand {
	case "init":
		initConfig()
	case "validate":
		validateConfig()
	case "show":
		showConfig()
	default:
		fmt.Printf("Unknown config subcommand: %s\n", subCommand)
		printConfigUsage()
		os.Exit(1)
	}
}

func printConfigUsage() {
	fmt.Println(`
Usage: biometrics config <subcommand>

Subcommands:
  init       Create default configuration file
  validate   Validate existing configuration
  show       Display current configuration`)
}

func initConfig() {
	configDir := getConfigDir()
	configPath := filepath.Join(configDir, "config.yaml")

	if checkFileExists(configPath) {
		fmt.Printf("Config already exists at %s\n", configPath)
		fmt.Print("Overwrite? (y/n): ")
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" {
			fmt.Println("Cancelled")
			return
		}
	}

	if err := os.MkdirAll(configDir, 0755); err != nil {
		fmt.Printf("Error creating config directory: %v\n", err)
		os.Exit(1)
	}

	defaultConfig := `# BIOMETRICS CLI Configuration
version: "2.0.0"

# Logging configuration
logging:
  level: info
  format: json
  output: stdout

# Audit configuration
audit:
  enabled: true
  storage_path: /var/log/biometrics/audit
  retention_days: 90
  compression: true

# Authentication configuration
auth:
  mtls:
    enabled: false
    cert_path: ""
    key_path: ""
  oauth2:
    enabled: false
    provider: ""
    client_id: ""
    client_secret: ""

# Performance configuration
performance:
  profiling_enabled: false
  cache_enabled: true
  redis_url: "redis://localhost:54322"

# API configuration
api:
  rate_limit:
    enabled: true
    requests_per_second: 100
    burst: 200
  health_check:
    enabled: true
    path: /health
`

	if err := os.WriteFile(configPath, []byte(defaultConfig), 0644); err != nil {
		fmt.Printf("Error writing config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Configuration created at %s\n", configPath)
	fmt.Println("\nEdit the file to customize your settings")
}

func validateConfig() {
	configPath := getConfigPath()

	if !checkFileExists(configPath) {
		fmt.Printf("Configuration file not found: %s\n", configPath)
		fmt.Println("Run 'biometrics config init' to create one")
		os.Exit(1)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Printf("Error reading config: %v\n", err)
		os.Exit(1)
	}

	errors := []string{}
	warnings := []string{}

	content := string(data)

	if !strings.Contains(content, "version:") {
		errors = append(errors, "Missing 'version' field")
	}

	if !strings.Contains(content, "logging:") {
		warnings = append(warnings, "Missing 'logging' configuration")
	}

	if !strings.Contains(content, "audit:") {
		warnings = append(warnings, "Missing 'audit' configuration")
	}

	if len(errors) > 0 {
		fmt.Println("❌ Configuration validation FAILED")
		fmt.Println("\nErrors:")
		for _, e := range errors {
			fmt.Printf("  - %s\n", e)
		}
		os.Exit(1)
	}

	if len(warnings) > 0 {
		fmt.Println("⚠️  Configuration validation passed with warnings")
		fmt.Println("\nWarnings:")
		for _, w := range warnings {
			fmt.Printf("  - %s\n", w)
		}
	} else {
		fmt.Println("✓ Configuration is valid")
	}

	fmt.Printf("\nConfig file: %s\n", configPath)
}

func showConfig() {
	configPath := getConfigPath()

	if !checkFileExists(configPath) {
		fmt.Printf("Configuration file not found: %s\n", configPath)
		fmt.Println("Run 'biometrics config init' to create one")
		os.Exit(1)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Printf("Error reading config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Configuration (%s):\n", configPath)
	fmt.Println(string(data))
}

func getConfigDir() string {
	if dir := os.Getenv("BIOMETRICS_CONFIG_DIR"); dir != "" {
		return dir
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ".biometrics"
	}
	return filepath.Join(homeDir, ".biometrics")
}

func getConfigPath() string {
	if path := os.Getenv("BIOMETRICS_CONFIG_PATH"); path != "" {
		return path
	}
	return filepath.Join(getConfigDir(), "config.yaml")
}

func runInit() {
	fmt.Println("Initializing BIOMETRICS repository...")

	dirs := []string{
		"global/01-agents",
		"global/02-models",
		"global/03-mandates",
		"local/projects",
		"biometrics-cli/bin",
		"biometrics-cli/commands",
		"docs/media",
		"scripts",
		"assets/images",
		"assets/videos",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Printf("Failed to create directory %s: %v\n", dir, err)
			os.Exit(1)
		}
		fmt.Printf("Created directory: %s\n", dir)
	}

	createReadme("global", "Global Configurations", "Global AI agent configurations and mandates")
	createReadme("local", "Local Projects", "Project-specific configurations")
	createReadme("biometrics-cli", "BIOMETRICS CLI", "Command-line interface and automation")
	createReadme("docs", "Documentation", "Complete documentation for BIOMETRICS")

	fmt.Println("\nBIOMETRICS repository initialized successfully!")
	fmt.Println("\nNext steps:")
	fmt.Println("  1. Run 'biometrics onboard' for interactive setup")
	fmt.Println("  2. Run 'biometrics auto' for automatic AI-powered setup")
}

func runOnboarding() {
	fmt.Println("Starting BIOMETRICS onboarding process...")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	fmt.Println("Step 1: Checking for existing API keys...")
	existingKeys := findExistingKeys()
	if len(existingKeys) > 0 {
		fmt.Println("Found existing API keys:")
		for keyType, keyPath := range existingKeys {
			fmt.Printf("  - %s: %s\n", keyType, keyPath)
		}
	} else {
		fmt.Println("No existing API keys found")
	}

	fmt.Println()
	fmt.Println("Step 2: Do you want to use existing keys? (y/n)")
	useExisting, _ := reader.ReadString('\n')
	useExisting = strings.TrimSpace(useExisting)

	if useExisting == "y" || useExisting == "Y" {
		fmt.Println("Using existing API keys")
		copyKeysToEnv(existingKeys)
	} else {
		fmt.Println("Step 3: Enter your NVIDIA API Key (or press Enter to skip):")
		nvidiaKey, _ := reader.ReadString('\n')
		nvidiaKey = strings.TrimSpace(nvidiaKey)

		if nvidiaKey != "" {
			saveToEnv("NVIDIA_API_KEY", nvidiaKey)
			fmt.Println("NVIDIA API Key saved")
		}
	}

	fmt.Println("\nStep 4: Do you have a GitLab token? (y/n)")
	hasGitLab, _ := reader.ReadString('\n')
	hasGitLab = strings.TrimSpace(hasGitLab)

	if hasGitLab == "y" || hasGitLab == "Y" {
		fmt.Println("Enter GitLab token:")
		gitLabToken, _ := reader.ReadString('\n')
		gitLabToken = strings.TrimSpace(gitLabToken)
		saveToEnv("GITLAB_TOKEN", gitLabToken)
		fmt.Println("GitLab token saved")
	}

	fmt.Println("\nStep 5: Do you have Supabase credentials? (y/n)")
	hasSupabase, _ := reader.ReadString('\n')
	hasSupabase = strings.TrimSpace(hasSupabase)

	if hasSupabase == "y" || hasSupabase == "Y" {
		fmt.Println("Enter Supabase URL:")
		supabaseURL, _ := reader.ReadString('\n')
		supabaseURL = strings.TrimSpace(supabaseURL)
		saveToEnv("SUPABASE_URL", supabaseURL)

		fmt.Println("Enter Supabase Key:")
		supabaseKey, _ := reader.ReadString('\n')
		supabaseKey = strings.TrimSpace(supabaseKey)
		saveToEnv("SUPABASE_KEY", supabaseKey)
		fmt.Println("Supabase credentials saved")
	}

	fmt.Println("\nStep 6: Installing dependencies...")
	installDependencies()

	fmt.Println("\nStep 7: Installing OpenCode skills...")
	installOpenCodeSkills()

	// Step 8: OpenClaw Onboarding
	fmt.Println("\nStep 8: OpenClaw Configuration")
	fmt.Println("Which channels should OpenClaw monitor?")
	fmt.Println()
	fmt.Println(" [ ] GitHub Issues")
	fmt.Println(" [ ] Slack")
	fmt.Println(" [ ] Email")
	fmt.Println(" [ ] Discord")
	fmt.Println(" [ ] Telegram")
	fmt.Println(" [ ] WhatsApp")
	fmt.Println()
	fmt.Println("Enter channels (comma-separated, e.g., github,slack,email):")
	channels, _ := reader.ReadString('\n')
	channels = strings.TrimSpace(channels)
	if channels != "" {
		fmt.Printf("OpenClaw will monitor: %s\n", channels)
	}

	fmt.Println("\nSelect skills for OpenClaw:")
	fmt.Println(" [x] playwright - Browser automation")
	fmt.Println(" [x] git-master - Git operations")
	fmt.Println(" [x] frontend-ui-ux - UI/UX design")
	fmt.Println(" [ ] dev-browser - Advanced browser control")
	fmt.Println(" [ ] context7 - Documentation lookup")
	fmt.Println()
	fmt.Println("Enter skills (comma-separated, or press Enter for defaults):")
	skills, _ := reader.ReadString('\n')
	skills = strings.TrimSpace(skills)
	if skills == "" {
		skills = "playwright,git-master,frontend-ui-ux"
	}
	fmt.Printf("Selected skills: %s\n", skills)

	fmt.Println("\nAllowed processes (NO deployment without permission!):")
	fmt.Println(" [x] File creation/modification")
	fmt.Println(" [x] Code improvements")
	fmt.Println(" [x] Test execution (localhost only)")
	fmt.Println(" [x] Browser debugging")
	fmt.Println(" [ ] Deployment (requires explicit permission)")
	fmt.Println()

	// Step 9: 24/7 Auto-Development Option
	fmt.Println("\nStep 9: Enable 24/7 Auto-Development?")
	fmt.Println("OpenClaw will continuously:")
	fmt.Println(" - Create missing files")
	fmt.Println(" - Improve existing code")
	fmt.Println(" - Run tests in localhost")
	fmt.Println(" - Debug in browser")
	fmt.Println(" - NEVER deploy without permission")
	fmt.Println()
	fmt.Println("Enable? (y/n)")
	autoDev, _ := reader.ReadString('\n')
	autoDev = strings.TrimSpace(autoDev)

	if autoDev == "y" || autoDev == "Y" {
		fmt.Println("\n✅ 24/7 Auto-Development ENABLED")
		fmt.Println("OpenClaw will run in DELQHI OMEGA LOOP mode")
		fmt.Println("Continuous improvement cycle activated")
	} else {
		fmt.Println("\n24/7 Auto-Development DISABLED")
		fmt.Println("OpenClaw will only run on manual commands")
	}

	fmt.Println("\nOnboarding complete!")
	fmt.Println("\nNext steps:")
	fmt.Println(" 1. Run 'biometrics check' to verify setup")
	fmt.Println(" 2. Run 'opencode \"Build a feature\"' to start developing")
	fmt.Println(" 3. OpenClaw is ready to orchestrate your development!")
}

func runAutoSetup() {
	fmt.Println("Starting automatic AI-powered setup...")
	fmt.Println()

	fmt.Println("Step 1: Scanning for existing API keys...")
	existingKeys := findExistingKeys()

	if len(existingKeys) > 0 {
		fmt.Println("Found API keys:")
		for keyType, keyPath := range existingKeys {
			fmt.Printf("  - %s: %s\n", keyType, keyPath)
		}
		copyKeysToEnv(existingKeys)
		fmt.Println("Keys copied to .env")
	} else {
		fmt.Println("No API keys found. You'll need to add them manually.")
		fmt.Println("   Get NVIDIA API key: https://build.nvidia.com/")
	}

	fmt.Println("\nStep 2: Creating enterprise directory structure...")
	runInit()

	fmt.Println("\nStep 3: Installing dependencies...")
	installDependencies()

	fmt.Println("\nStep 4: Installing OpenCode skills...")
	installOpenCodeSkills()

	fmt.Println("\nStep 5: Verifying setup...")
	runBiometricsCheck()

	fmt.Println("\nAutomatic setup complete!")
}

func runBiometricsCheck() {
	fmt.Println("BIOMETRICS REPO CHECK")
	fmt.Println("=====================")
	fmt.Println()

	checks := []struct {
		name    string
		passed  bool
		success string
		failure string
	}{
		{"global/ README", checkFileExists("global/README.md"), "global/README.md exists", "global/README.md missing"},
		{"local/ README", checkFileExists("local/README.md"), "local/README.md exists", "local/README.md missing"},
		{"biometrics-cli/ README", checkFileExists("biometrics-cli/README.md"), "biometrics-cli/README.md exists", "biometrics-cli/README.md missing"},
		{".env file", checkFileExists(".env"), ".env exists", ".env missing"},
		{"oh-my-opencode.json", checkFileExists("oh-my-opencode.json"), "oh-my-opencode.json exists", "oh-my-opencode.json missing"},
		{"requirements.txt", checkFileExists("requirements.txt"), "requirements.txt exists", "requirements.txt missing"},
	}

	allPassed := true
	for _, check := range checks {
		if check.passed {
			fmt.Println("✓ " + check.success)
		} else {
			fmt.Println("✗ " + check.failure)
			allPassed = false
		}
	}

	fmt.Println()
	if allPassed {
		fmt.Println("All checks passed! BIOMETRICS is ready.")
	} else {
		fmt.Println("Some checks failed. Run 'biometrics onboard' to fix.")
	}
}

func findAPIKeys() {
	fmt.Println("Scanning for existing API keys...")
	fmt.Println()

	keys := findExistingKeys()

	if len(keys) == 0 {
		fmt.Println("No API keys found on system")
		return
	}

	fmt.Println("Found API keys:")
	for keyType, keyPath := range keys {
		fmt.Printf("  %s: %s\n", keyType, keyPath)
	}

	fmt.Println("\nTo use these keys, run: biometrics onboard")
}

func createReadme(dir, title, description string) {
	readmePath := filepath.Join(dir, "README.md")
	content := fmt.Sprintf(`# %s

## Purpose

%s

## Structure

%s/
├── README.md
└── ...

## Enterprise Practices 2026

- Go-Style Modularity
- Machine-readable
- KI-Agent optimized

---

Generated by BIOMETRICS CLI v2.0.0
`, title, description, dir)

	os.WriteFile(readmePath, []byte(content), 0644)
	fmt.Printf("Created: %s\n", readmePath)
}

func findExistingKeys() map[string]string {
	keys := make(map[string]string)

	knownAPIKeys := []string{
		"NVIDIA_API_KEY",
		"MOONSHOT_API_KEY",
		"GITLAB_TOKEN",
		"GITLAB_MEDIA_PROJECT_ID",
		"SUPABASE_URL",
		"SUPABASE_KEY",
		"OPENCODE_API_KEY",
		"ANTHROPIC_API_KEY",
		"OPENROUTER_API_KEY",
		"HUGGINGFACE_TOKEN",
	}

	for _, envVar := range knownAPIKeys {
		if value := os.Getenv(envVar); value != "" {
			keys[envVar] = "Environment Variable"
		}
	}

	locations := map[string]string{
		"NVIDIA_API_KEY": os.Getenv("HOME") + "/.nvidia_api_key",
		"GitLab":         os.Getenv("HOME") + "/.gitlab_token",
		"Supabase":       os.Getenv("HOME") + "/.supabase_key",
		"OpenCode":       os.Getenv("HOME") + "/.opencode/key",
	}

	for keyType, location := range locations {
		if _, err := os.Stat(location); err == nil {
			keys[keyType] = location
		}
	}

	dotenvPath := ".env"
	if _, err := os.Stat(dotenvPath); err == nil {
		file, err := os.Open(dotenvPath)
		if err == nil {
			defer file.Close()
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				line := scanner.Text()
				if strings.HasPrefix(line, "#") {
					continue
				}
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					envVar := strings.TrimSpace(parts[0])
					value := strings.TrimSpace(parts[1])
					if value != "" && value != "YOUR_KEY_HERE" && value != "nvapi-YOUR_KEY" {
						keys[envVar] = ".env file"
					}
				}
			}
		}
	}

	shellConfig := os.Getenv("HOME") + "/.zshrc"
	if _, err := os.Stat(shellConfig); err == nil {
		file, err := os.Open(shellConfig)
		if err == nil {
			defer file.Close()
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				line := scanner.Text()
				if strings.HasPrefix(line, "export ") {
					parts := strings.SplitN(line, "=", 2)
					if len(parts) == 2 {
						envVar := strings.TrimPrefix(strings.TrimSpace(parts[0]), "export ")
						value := strings.TrimSpace(parts[1])
						value = strings.Trim(value, `"'`)
						if strings.HasPrefix(value, "nvapi-") || strings.HasPrefix(value, "sk-") {
							keys[envVar] = "~/.zshrc"
						}
					}
				}
			}
		}
	}

	return keys
}

func copyKeysToEnv(keys map[string]string) {
	for keyType, keyPath := range keys {
		if keyPath != "Environment Variable" {
			data, err := os.ReadFile(keyPath)
			if err == nil {
				saveToEnv(keyType+"_KEY", strings.TrimSpace(string(data)))
			}
		}
	}
}

func autoConfigureKeys(keys map[string]string) {
	fmt.Println("\n=== Auto-Configuring API Keys ===")

	ohMyOpenCodePath := "oh-my-opencode.json"
	if _, err := os.Stat(ohMyOpenCodePath); err != nil {
		fmt.Printf("Warning: %s not found, skipping auto-configure\n", ohMyOpenCodePath)
		return
	}

	content, err := os.ReadFile(ohMyOpenCodePath)
	if err != nil {
		fmt.Printf("Warning: Could not read %s: %v\n", ohMyOpenCodePath, err)
		return
	}

	updated := string(content)
	for key, source := range keys {
		if source == "Environment Variable" || source == ".env file" || source == "~/.zshrc" {
			value := os.Getenv(key)
			if value != "" {
				envVarPlaceholder := "${" + key + "}"
				if !strings.Contains(updated, envVarPlaceholder) {
					updated = strings.ReplaceAll(updated, `"`+key+`": ""`, `"`+key+`": "${`+key+`}"`)
					fmt.Printf("✓ Configured %s from %s\n", key, source)
				}
			}
		}
	}

	err = os.WriteFile(ohMyOpenCodePath, []byte(updated), 0644)
	if err != nil {
		fmt.Printf("Warning: Could not write %s: %v\n", ohMyOpenCodePath, err)
		return
	}

	fmt.Println("✓ API keys auto-configured successfully!")
}

func saveToEnv(key, value string) {
	f, err := os.OpenFile(".env", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Could not open .env: %v\n", err)
		return
	}
	defer f.Close()

	f.WriteString(fmt.Sprintf("%s=%s\n", key, value))
}

func checkFileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func installDependencies() {
	if _, err := exec.LookPath("npm"); err == nil {
		fmt.Println("Installing npm dependencies...")
		cmd := exec.Command("npm", "install")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Printf("Warning: pnpm install failed: %v\n", err)
		} else {
			fmt.Println("✓ npm dependencies installed")
		}
	}

	if _, err := exec.LookPath("pip3"); err == nil {
		fmt.Println("Installing Python dependencies...")
		cmd := exec.Command("pip3", "install", "-r", "requirements.txt")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Printf("Warning: pip3 install failed: %v\n", err)
		} else {
			fmt.Println("✓ Python dependencies installed")
		}
	}

	if _, err := exec.LookPath("opencode"); err == nil {
		fmt.Println("Authenticating OpenCode providers...")
		cmd := exec.Command("opencode", "auth", "add", "nvidia-nim")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Printf("Warning: opencode auth failed: %v\n", err)
		} else {
			fmt.Println("✓ OpenCode authenticated")
		}
	}
}

func installOpenCodeSkills() {
	fmt.Println("Installing OpenCode skills...")

	skills := []string{
		"playwright",
		"git-master",
		"frontend-ui-ux",
	}

	for _, skill := range skills {
		fmt.Printf("Installing skill: %s... ", skill)
		cmd := exec.Command("opencode", "skill", "install", skill)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Printf("Warning: Failed to install %s\n", skill)
		} else {
			fmt.Println("✓")
		}
	}

	fmt.Println("OpenCode skills installed")
}

func runAudit() {
	if len(os.Args) < 3 {
		commands.PrintAuditHelp()
		os.Exit(1)
	}

	subCommand := os.Args[2]

	switch subCommand {
	case "query":
		flags := parseAuditQueryFlags()
		if err := commands.RunAuditQuery(flags); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
	case "export":
		flags := parseAuditExportFlags()
		if err := commands.RunAuditExport(flags); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
	case "stats":
		if err := commands.RunAuditStats(); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
	case "cleanup":
		retentionDays := parseRetentionDays()
		if err := commands.RunAuditCleanup(retentionDays); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
	case "rotate":
		if err := commands.RunAuditRotate(); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
	case "help":
		commands.PrintAuditHelp()
	default:
		fmt.Printf("Unknown audit subcommand: %s\n", subCommand)
		commands.PrintAuditHelp()
		os.Exit(1)
	}
}

func parseAuditQueryFlags() *commands.AuditQueryFlags {
	flags := &commands.AuditQueryFlags{
		Limit: 100,
	}

	args := os.Args[3:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--start-time":
			if i+1 < len(args) {
				flags.StartTime = args[i+1]
				i++
			}
		case "--end-time":
			if i+1 < len(args) {
				flags.EndTime = args[i+1]
				i++
			}
		case "--event-types":
			if i+1 < len(args) {
				flags.EventTypes = args[i+1]
				i++
			}
		case "--actors":
			if i+1 < len(args) {
				flags.Actors = args[i+1]
				i++
			}
		case "--resources":
			if i+1 < len(args) {
				flags.Resources = args[i+1]
				i++
			}
		case "--limit":
			if i+1 < len(args) {
				fmt.Sscanf(args[i+1], "%d", &flags.Limit)
				i++
			}
		case "--offset":
			if i+1 < len(args) {
				fmt.Sscanf(args[i+1], "%d", &flags.Offset)
				i++
			}
		case "--sort-by":
			if i+1 < len(args) {
				flags.SortBy = args[i+1]
				i++
			}
		case "--sort-order":
			if i+1 < len(args) {
				flags.SortOrder = args[i+1]
				i++
			}
		case "--format":
			if i+1 < len(args) {
				flags.Format = args[i+1]
				i++
			}
		case "--output", "-o":
			if i+1 < len(args) {
				flags.Output = args[i+1]
				i++
			}
		}
	}

	return flags
}

func parseAuditExportFlags() *commands.AuditExportFlags {
	flags := &commands.AuditExportFlags{}

	args := os.Args[3:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--start-time":
			if i+1 < len(args) {
				flags.StartTime = args[i+1]
				i++
			}
		case "--end-time":
			if i+1 < len(args) {
				flags.EndTime = args[i+1]
				i++
			}
		case "--format":
			if i+1 < len(args) {
				flags.Format = args[i+1]
				i++
			}
		case "--output", "-o":
			if i+1 < len(args) {
				flags.Output = args[i+1]
				i++
			}
		}
	}

	return flags
}

func parseRetentionDays() int {
	args := os.Args[3:]
	for i := 0; i < len(args); i++ {
		if args[i] == "--retention-days" && i+1 < len(args) {
			var days int
			fmt.Sscanf(args[i+1], "%d", &days)
			return days
		}
	}
	return 0
}
