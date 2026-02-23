package codegen

import (
	"biometrics-cli/internal/metrics"
	"biometrics-cli/internal/state"
	"fmt"
	"os/exec"
	"sync"
	"time"
)

type CodeGenerator struct {
	mu          sync.RWMutex
	Tasks       []*Task
	activeTasks map[string]*Task
	workers     int
	queue       chan *Task
}

type Task struct {
	ID          string
	Title       string
	Description string
	Agent       string
	Status      string
	Progress    int
	Output      string
	Error       string
	CreatedAt   time.Time
	StartedAt   time.Time
	CompletedAt time.Time
}

var (
	generator *CodeGenerator
	once      sync.Once
)

func NewCodeGenerator() *CodeGenerator {
	once.Do(func() {
		generator = &CodeGenerator{
			Tasks:       make([]*Task, 0),
			activeTasks: make(map[string]*Task),
			workers:     3,
			queue:       make(chan *Task, 100),
		}
		go generator.workerPool()
	})
	return generator
}

func (g *CodeGenerator) CreateTask(title, description, agent string) (*Task, error) {
	task := &Task{
		ID:          fmt.Sprintf("task-%d", time.Now().UnixNano()),
		Title:       title,
		Description: description,
		Agent:       agent,
		Status:      "pending",
		Progress:    0,
		CreatedAt:   time.Now(),
	}

	g.mu.Lock()
	g.Tasks = append(g.Tasks, task)
	g.mu.Unlock()

	state.GlobalState.Log("INFO", fmt.Sprintf("Created task: %s (%s)", task.Title, task.Agent))
	metrics.TasksCreatedTotal.Inc()

	return task, nil
}

func (g *CodeGenerator) RunCodeGeneration(taskID string, progressChan chan<- string) error {
	g.mu.Lock()
	var task *Task
	for _, t := range g.Tasks {
		if t.ID == taskID {
			task = t
			break
		}
	}

	if task == nil {
		g.mu.Unlock()
		return fmt.Errorf("task not found: %s", taskID)
	}

	task.Status = "running"
	task.StartedAt = time.Now()
	g.activeTasks[task.ID] = task
	g.mu.Unlock()

	progressChan <- "Starting code generation..."
	metrics.TasksStartedTotal.Inc()

	cmd := exec.Command("opencode", task.Description, "--agent", task.Agent)
	output, err := cmd.CombinedOutput()

	g.mu.Lock()
	defer g.mu.Unlock()

	if err != nil {
		task.Status = "failed"
		task.Error = err.Error()
		task.CompletedAt = time.Now()
		delete(g.activeTasks, task.ID)
		progressChan <- fmt.Sprintf("Error: %v", err)
		metrics.TasksFailedTotal.Inc()
		return err
	}

	task.Status = "completed"
	task.Output = string(output)
	task.CompletedAt = time.Now()
	task.Progress = 100
	delete(g.activeTasks, task.ID)

	progressChan <- "Task completed successfully"
	metrics.TasksCompletedTotal.Inc()

	state.GlobalState.Log("SUCCESS", fmt.Sprintf("Task %s completed", taskID))

	return nil
}

func (g *CodeGenerator) GetActiveTasks() []*Task {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if g.activeTasks == nil {
		return []*Task{}
	}

	tasks := make([]*Task, 0, len(g.activeTasks))
	for _, task := range g.activeTasks {
		tasks = append(tasks, task)
	}
	return tasks
}

func (g *CodeGenerator) ListTasks() []*Task {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if g.Tasks == nil {
		return []*Task{}
	}

	return g.Tasks
}

func (g *CodeGenerator) GetTask(id string) (*Task, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	for _, task := range g.Tasks {
		if task.ID == id {
			return task, nil
		}
	}
	return nil, fmt.Errorf("task not found")
}

func (g *CodeGenerator) DeleteTask(id string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	for i, task := range g.Tasks {
		if task.ID == id {
			g.Tasks = append(g.Tasks[:i], g.Tasks[i+1:]...)
			state.GlobalState.Log("INFO", fmt.Sprintf("Deleted task: %s", id))
			return nil
		}
	}
	return fmt.Errorf("task not found")
}

func (g *CodeGenerator) workerPool() {
	for i := 0; i < g.workers; i++ {
		go g.worker(i)
	}
}

func (g *CodeGenerator) worker(id int) {
	for task := range g.queue {
		g.runTask(task, id)
	}
}

func (g *CodeGenerator) runTask(task *Task, workerID int) {
	progressChan := make(chan string, 10)
	go g.RunCodeGeneration(task.ID, progressChan)

	for msg := range progressChan {
		state.GlobalState.Log("INFO", fmt.Sprintf("[Worker %d] %s: %s", workerID, task.ID, msg))
	}
}

func (g *CodeGenerator) QueueTask(task *Task) {
	select {
	case g.queue <- task:
	default:
		state.GlobalState.Log("WARN", "Task queue full, dropping task")
	}
}

func (g *CodeGenerator) GetStats() map[string]interface{} {
	g.mu.RLock()
	defer g.mu.RUnlock()

	pending := 0
	running := 0
	completed := 0
	failed := 0

	for _, task := range g.Tasks {
		switch task.Status {
		case "pending":
			pending++
		case "running":
			running++
		case "completed":
			completed++
		case "failed":
			failed++
		}
	}

	return map[string]interface{}{
		"total":     len(g.Tasks),
		"pending":   pending,
		"running":   running,
		"completed": completed,
		"failed":    failed,
		"queue_len": len(g.queue),
	}
}

func init() {
	NewCodeGenerator()
}

// Toggle features
func (g *CodeGenerator) SetSwarmEngine(enabled bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if enabled {
		g.workers = 10
	} else {
		g.workers = 3
	}
	// Note: changing workers dynamically requires restarting the worker pool in a real scenario,
	// but for now we update the config state.
}

func (g *CodeGenerator) SetDelqhiLoop(enabled bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	// Update internal state or trigger loop creation
}
