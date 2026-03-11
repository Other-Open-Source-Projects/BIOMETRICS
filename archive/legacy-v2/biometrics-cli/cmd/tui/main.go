package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1).
			MarginBottom(1)

	statStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#04B575"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF5555")).
			Bold(true)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#50FA7B")).
			Bold(true)
)

type Task struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Agent       string    `json:"agent"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

type model struct {
	spinner     spinner.Model
	tasks       []Task
	status      string
	activeTasks int
	totalTasks  int
	loading     bool
	err         string
	inputMode   bool
	inputBuffer string
}

type tickMsg time.Time
type tasksMsg []Task
type statusMsg struct {
	status      string
	activeTasks int
	totalTasks  int
}

func main() {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4"))

	p := tea.NewProgram(model{
		spinner: s,
		loading: true,
	})

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		spinner.Tick,
		tickCmd(),
		fetchStatusCmd(),
		fetchTasksCmd(),
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "r":
			return m, tea.Batch(fetchStatusCmd(), fetchTasksCmd())
		case "n":
			newModel := m
			newModel.inputMode = true
			newModel.inputBuffer = ""
			return newModel, nil
		case "enter":
			if m.inputMode {
				// Create task
				go createTask(m.inputBuffer)
				newModel := m
				newModel.inputMode = false
				newModel.inputBuffer = ""
				return newModel, nil
			}
		default:
			if m.inputMode {
				newModel := m
				newModel.inputBuffer += msg.String()
				return newModel, nil
			}
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tickMsg:
		return m, tea.Batch(tickCmd(), fetchStatusCmd(), fetchTasksCmd())

	case tasksMsg:
		m.tasks = msg
		return m, nil

	case statusMsg:
		m.status = msg.status
		m.activeTasks = msg.activeTasks
		m.totalTasks = msg.totalTasks
		m.loading = false
		return m, nil
	}

	return m, nil
}

func (m model) View() string {
	var b string

	b += titleStyle.Render("🚀 BIOMETRICS - Code Generation TUI")
	b += "\n\n"

	if m.err != "" {
		b += errorStyle.Render("ERROR: "+m.err) + "\n"
	} else if m.loading {
		b += m.spinner.View() + " Loading...\n"
	} else {
		// Status
		b += fmt.Sprintf("Status: %s | Active: %s | Total: %s\n\n",
			successStyle.Render("● "+m.status),
			statStyle.Render(fmt.Sprintf("%d", m.activeTasks)),
			statStyle.Render(fmt.Sprintf("%d", m.totalTasks)))

		// Tasks
		b += "📋 TASKS:\n"
		b += strings.Repeat("─", 60) + "\n"

		if len(m.tasks) == 0 {
			b += "No tasks yet. Press 'n' to create one.\n"
		} else {
			for _, task := range m.tasks {
				statusIcon := "⏳"
				if task.Status == "completed" {
					statusIcon = "✅"
				} else if task.Status == "failed" {
					statusIcon = "❌"
				} else if task.Status == "running" {
					statusIcon = "🔨"
				}

				b += fmt.Sprintf("%s [%s] %s\n", statusIcon, task.Agent, task.Title)
				b += fmt.Sprintf("   ID: %s | Created: %s\n",
					task.ID[:20],
					task.CreatedAt.Format("15:04:05"))
				b += "\n"
			}
		}

		b += strings.Repeat("─", 60) + "\n\n"

		// Controls
		b += "Controls: [n] New Task  [r] Refresh  [q] Quit\n"

		if m.inputMode {
			b += "\n" + lipgloss.NewStyle().
				Foreground(lipgloss.Color("#F6AD55")).
				Render("Enter task description: "+m.inputBuffer+"█")
		}
	}

	b += "\n"
	b += lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).Render(strings.Repeat("─", 60))

	return b
}

func tickCmd() tea.Cmd {
	return tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func fetchStatusCmd() tea.Cmd {
	return func() tea.Msg {
		resp, err := http.Get("http://localhost:59003/api/status")
		if err != nil {
			return statusMsg{status: "disconnected"}
		}
		defer resp.Body.Close()

		var status struct {
			Status      string `json:"status"`
			ActiveTasks int    `json:"active_tasks"`
			TotalTasks  int    `json:"total_tasks"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
			return statusMsg{status: "error"}
		}

		return statusMsg{
			status:      status.Status,
			activeTasks: status.ActiveTasks,
			totalTasks:  status.TotalTasks,
		}
	}
}

func fetchTasksCmd() tea.Cmd {
	return func() tea.Msg {
		resp, err := http.Get("http://localhost:59003/api/tasks/list")
		if err != nil {
			return tasksMsg([]Task{})
		}
		defer resp.Body.Close()

		var tasks []Task
		if err := json.NewDecoder(resp.Body).Decode(&tasks); err != nil {
			return tasksMsg([]Task{})
		}

		return tasksMsg(tasks)
	}
}

func createTask(description string) {
	req := map[string]interface{}{
		"title":       description[:50],
		"description": description,
		"agent":       "sisyphus",
	}

	jsonData, _ := json.Marshal(req)
	resp, err := http.Post("http://localhost:59003/api/tasks/create", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return
	}
	defer resp.Body.Close()

	// Execute task
	var task Task
	json.NewDecoder(resp.Body).Decode(&task)

	executeReq, _ := json.Marshal(map[string]string{"task_id": task.ID})
	http.Post("http://localhost:59003/api/tasks/execute", "application/json", bytes.NewBuffer(executeReq))
}
