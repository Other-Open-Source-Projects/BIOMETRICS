package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00FF00")).
			Background(lipgloss.Color("#004400")).
			Padding(0, 2).
			MarginBottom(1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Italic(true).
			MarginBottom(2)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FF00")).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000")).
			Bold(true)

	runningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FF00"))

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888"))

	commandStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFF00")).
			Bold(true)
)

// Step represents an onboarding step
type Step struct {
	Title   string
	Status  string
	Message string
}

// Model for bubbletea
type Model struct {
	steps      []Step
	current    int
	quitting   bool
	done       bool
	spinner    spinner.Model
	width      int
	height     int
	config     Config
	showMenu   bool
	menuCursor int
}

// Config holds user configuration
type Config struct {
	InstallOpenCode bool
	InstallOpenClaw bool
	WhatsApp        bool
	Telegram        bool
}

// Messages
type statusMsg struct {
	Index   int
	Status  string
	Message string
}

func initialModel() Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = runningStyle

	steps := []Step{
		{Title: "Checking system requirements", Status: "pending"},
		{Title: "Installing Git", Status: "pending"},
		{Title: "Installing Node.js", Status: "pending"},
		{Title: "Installing pnpm", Status: "pending"},
		{Title: "Installing Homebrew", Status: "pending"},
		{Title: "Installing Python 3", Status: "pending"},
		{Title: "Configuring PATH", Status: "pending"},
		{Title: "Installing NLM CLI", Status: "pending"},
		{Title: "Installing OpenCode", Status: "pending"},
		{Title: "Installing OpenClaw", Status: "pending"},
		{Title: "Installing Serena MCP (Orchestration)", Status: "pending"},
		{Title: "Configuring WhatsApp (QR Code)", Status: "pending"},
		{Title: "Configuring Telegram", Status: "pending"},
		{Title: "Installing skills (ordercli, github, gitlab)", Status: "pending"},
	}

	return Model{
		steps:   steps,
		spinner: s,
		config: Config{
			InstallOpenCode: true,
			InstallOpenClaw: true,
			WhatsApp:        true,
			Telegram:        true,
		},
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, checkSystemRequirements)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "up", "k":
			if m.showMenu && m.done {
				if m.menuCursor > 0 {
					m.menuCursor--
				}
			}
		case "down", "j":
			if m.showMenu && m.done {
				if m.menuCursor < 2 {
					m.menuCursor++
				}
			}
		case "enter":
			if m.done && m.showMenu {
				switch m.menuCursor {
				case 0:
					m.quitting = true
					go func() {
						cmd := exec.Command("opencode")
						cmd.Stdout = os.Stdout
						cmd.Stderr = os.Stderr
						cmd.Stdin = os.Stdin
						cmd.Run()
					}()
					return m, tea.Quit
				case 1:
					m.quitting = true
					go func() {
						cmd := exec.Command("openclaw")
						cmd.Stdout = os.Stdout
						cmd.Stderr = os.Stderr
						cmd.Stdin = os.Stdin
						cmd.Run()
					}()
					return m, tea.Quit
				case 2:
					m.quitting = true
					return m, tea.Quit
				}
			}
		case " ":
			if m.done && !m.showMenu {
				m.showMenu = true
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case statusMsg:
		if msg.Index < len(m.steps) {
			m.steps[msg.Index].Status = msg.Status
			m.steps[msg.Index].Message = msg.Message

			if msg.Status == "success" {
				if msg.Index < len(m.steps)-1 {
					m.current = msg.Index + 1
					return m, nextStep(msg.Index + 1)
				}
				m.done = true
				return m, nil
			}
		}
	}

	return m, nil
}

func (m Model) View() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(titleStyle.Render(" BIOMETRICS ONBOARD "))
	b.WriteString("\n")
	b.WriteString(subtitleStyle.Render(" Professional Setup - OpenCode + OpenClaw + Serena MCP "))
	b.WriteString("\n")

	for i, step := range m.steps {
		status := "  "
		style := dimStyle

		switch step.Status {
		case "running":
			if i == m.current {
				status = runningStyle.Render(m.spinner.View()+" ") + " "
				style = runningStyle
			}
		case "success":
			status = successStyle.Render("✓ ")
			style = successStyle
		case "error":
			status = errorStyle.Render("✗ ")
			style = errorStyle
		}

		b.WriteString(style.Render(status + step.Title))
		if step.Message != "" && step.Status == "running" {
			b.WriteString(dimStyle.Render(" - " + step.Message))
		}
		b.WriteString("\n")
	}

	if m.done {
		b.WriteString("\n")
		b.WriteString(successStyle.Render("✓ Setup complete!\n"))
		b.WriteString("\n")

		if m.showMenu {
			b.WriteString(commandStyle.Render("What would you like to do?\n"))
			b.WriteString("\n")
			options := []string{"Start OpenCode", "Start OpenClaw", "Exit"}
			for i, option := range options {
				cursor := "  "
				style := dimStyle
				if i == m.menuCursor {
					cursor = runningStyle.Render("› ")
					style = runningStyle
				}
				b.WriteString(style.Render(cursor + option))
				b.WriteString("\n")
			}
			b.WriteString("\n")
			b.WriteString(dimStyle.Render("Use ↑/↓ to navigate, Enter to select\n"))
		} else {
			b.WriteString(dimStyle.Render("Press SPACE to show menu\n"))
		}
	}

	if m.quitting {
		b.WriteString("\nAborted.\n")
	}

	return b.String()
}

func nextStep(index int) tea.Cmd {
	switch index {
	case 0:
		return checkSystemRequirements
	case 1:
		return installGit
	case 2:
		return installNode
	case 3:
		return installPnpm
	case 4:
		return installHomebrew
	case 5:
		return installPython
	case 6:
		return configurePATH
	case 7:
		return installNLMCLI
	case 8:
		return installOpenCode
	case 9:
		return installOpenClaw
	case 10:
		return installSerenaMCP
	case 11:
		return configureWhatsApp
	case 12:
		return configureTelegram
	case 13:
		return installSkills
	default:
		return nil
	}
}

func checkSystemRequirements() tea.Msg {
	installed := make(map[string]bool)
	tools := []string{"git", "node", "pnpm", "brew", "python3"}

	for _, tool := range tools {
		_, err := exec.LookPath(tool)
		installed[tool] = err == nil
	}

	for i, tool := range []string{"git", "node", "pnpm", "brew", "python3"} {
		if !installed[tool] {
			return statusMsg{Index: i + 1, Status: "pending", Message: "not installed"}
		}
	}

	return statusMsg{Index: 0, Status: "success", Message: "all requirements met"}
}

func installGit() tea.Msg {
	_, err := runCommand("brew", "install", "git")
	if err != nil {
		return statusMsg{Index: 1, Status: "error", Message: err.Error()}
	}
	return statusMsg{Index: 1, Status: "success", Message: "installed"}
}

func installNode() tea.Msg {
	_, err := runCommand("brew", "install", "node")
	if err != nil {
		return statusMsg{Index: 2, Status: "error", Message: err.Error()}
	}
	return statusMsg{Index: 2, Status: "success", Message: "installed"}
}

func installPnpm() tea.Msg {
	_, err := runCommand("brew", "install", "pnpm")
	if err != nil {
		return statusMsg{Index: 3, Status: "error", Message: err.Error()}
	}
	return statusMsg{Index: 3, Status: "success", Message: "installed"}
}

func installHomebrew() tea.Msg {
	if _, err := exec.LookPath("brew"); err == nil {
		return statusMsg{Index: 4, Status: "success", Message: "already installed"}
	}
	return statusMsg{Index: 4, Status: "pending", Message: "manual installation required"}
}

func installPython() tea.Msg {
	_, err := runCommand("brew", "install", "python@3.11")
	if err != nil {
		return statusMsg{Index: 5, Status: "error", Message: err.Error()}
	}
	return statusMsg{Index: 5, Status: "success", Message: "installed"}
}

func configurePATH() tea.Msg {
	homeDir, _ := os.UserHomeDir()
	zshrc := filepath.Join(homeDir, ".zshrc")

	pathLine := "\nexport PATH=\"$HOME/Library/pnpm:$PATH\"\n"

	data, _ := os.ReadFile(zshrc)
	content := string(data)

	if !strings.Contains(content, "pnpm") {
		f, _ := os.OpenFile(zshrc, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		defer f.Close()
		f.WriteString(pathLine)
	}

	os.Setenv("PATH", os.Getenv("HOME")+"/Library/pnpm:"+os.Getenv("PATH"))

	return statusMsg{Index: 6, Status: "success", Message: "PATH configured"}
}

func installNLMCLI() tea.Msg {
	_, err := runCommand("pnpm", "add", "-g", "nlm-cli")
	if err != nil {
		return statusMsg{Index: 7, Status: "error", Message: err.Error()}
	}
	return statusMsg{Index: 7, Status: "success", Message: "installed"}
}

func installOpenCode() tea.Msg {
	_, err := runCommand("brew", "install", "opencode")
	if err != nil {
		return statusMsg{Index: 8, Status: "error", Message: err.Error()}
	}
	return statusMsg{Index: 8, Status: "success", Message: "installed"}
}

func installOpenClaw() tea.Msg {
	_, err := runCommand("pnpm", "add", "-g", "@delqhi/openclaw")
	if err != nil {
		return statusMsg{Index: 9, Status: "error", Message: err.Error()}
	}
	return statusMsg{Index: 9, Status: "success", Message: "installed"}
}

func installSerenaMCP() tea.Msg {
	// Install Serena MCP for OpenCode
	fmt.Println("\n🔧 Installing Serena MCP for OpenCode...")
	runCommand("opencode", "mcp", "add", "serena")

	// Install Serena skill for OpenClaw
	fmt.Println("🔧 Installing Serena skill for OpenClaw...")
	runCommand("openclaw", "skills", "install", "openclaw/skills--serena")

	return statusMsg{Index: 10, Status: "success", Message: "Serena MCP installed"}
}

func configureWhatsApp() tea.Msg {
	fmt.Println("\n📱 WhatsApp QR Code Pairing")
	fmt.Println("OpenClaw will now show a QR code...")
	fmt.Println("Scan it with your WhatsApp app to connect.")

	cmd := exec.Command("openclaw", "channels", "login", "--channel", "whatsapp")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return statusMsg{Index: 11, Status: "error", Message: err.Error()}
	}

	return statusMsg{Index: 11, Status: "success", Message: "WhatsApp connected"}
}

func configureTelegram() tea.Msg {
	fmt.Println("\n📱 Telegram Bot Setup")
	fmt.Println("Follow the instructions to create your bot...")

	cmd := exec.Command("openclaw", "channels", "login", "--channel", "telegram")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return statusMsg{Index: 12, Status: "error", Message: err.Error()}
	}

	return statusMsg{Index: 12, Status: "success", Message: "Telegram connected"}
}

func installSkills() tea.Msg {
	skills := []string{
		"openclaw/skills--ordercli",
		"openclaw/skills--github",
		"openclaw/skills--gitlab",
	}

	for _, skill := range skills {
		runCommand("openclaw", "skills", "install", skill)
	}

	return statusMsg{Index: 13, Status: "success", Message: "skills installed"}
}

func runCommand(cmd string, args ...string) (string, error) {
	output, err := exec.Command(cmd, args...).CombinedOutput()
	return strings.TrimSpace(string(output)), err
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
