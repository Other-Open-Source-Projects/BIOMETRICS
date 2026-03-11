package orchestrator

import (
	"biometrics-cli/internal/heartbeat"
	"biometrics-cli/internal/metrics"
	"biometrics-cli/internal/paths"
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"
)

type MultiProjectOrchestrator struct {
	mu          sync.RWMutex
	projects    map[string]*Project
	heartbeat   *heartbeat.Monitor
	schedulers  map[string]*ProjectScheduler
	workStealer *WorkStealer
}

type Project struct {
	Name          string
	Path          string
	PlanPath      string
	Active        bool
	LastProcessed time.Time
	Priority      int
}

type ProjectScheduler struct {
	ProjectName string
	Interval    time.Duration
	TaskQueue   []*ScheduledTask
	LastRun     time.Time
	Enabled     bool
}

type ScheduledTask struct {
	ID       string
	Project  string
	TaskType string
	Priority int
	Schedule string
	LastRun  time.Time
	NextRun  time.Time
	Callback func() error
}

func NewMultiProjectOrchestrator(heartbeat *heartbeat.Monitor) *MultiProjectOrchestrator {
	return &MultiProjectOrchestrator{
		projects:   make(map[string]*Project),
		heartbeat:  heartbeat,
		schedulers: make(map[string]*ProjectScheduler),
	}
}

func (mpo *MultiProjectOrchestrator) RegisterProject(name, path, planPath string, priority int) error {
	mpo.mu.Lock()
	defer mpo.mu.Unlock()

	if _, exists := mpo.projects[name]; exists {
		return fmt.Errorf("project %s already registered", name)
	}

	mpo.projects[name] = &Project{
		Name:          name,
		Path:          path,
		PlanPath:      planPath,
		Active:        true,
		LastProcessed: time.Now(),
		Priority:      priority,
	}

	metrics.ProjectsRegistered.WithLabelValues(name).Inc()

	return nil
}

func (mpo *MultiProjectOrchestrator) UnregisterProject(name string) error {
	mpo.mu.Lock()
	defer mpo.mu.Unlock()

	if _, exists := mpo.projects[name]; !exists {
		return fmt.Errorf("project %s not found", name)
	}

	delete(mpo.projects, name)
	metrics.ProjectsUnregistered.WithLabelValues(name).Inc()

	return nil
}

func (mpo *MultiProjectOrchestrator) GetProject(name string) (*Project, error) {
	mpo.mu.RLock()
	defer mpo.mu.RUnlock()

	if p, exists := mpo.projects[name]; exists {
		return p, nil
	}

	return nil, fmt.Errorf("project %s not found", name)
}

func (mpo *MultiProjectOrchestrator) GetAllProjects() []*Project {
	mpo.mu.RLock()
	defer mpo.mu.RUnlock()

	projects := make([]*Project, 0, len(mpo.projects))
	for _, p := range mpo.projects {
		projects = append(projects, p)
	}

	return projects
}

func (mpo *MultiProjectOrchestrator) GetActiveProjects() []*Project {
	mpo.mu.RLock()
	defer mpo.mu.RUnlock()

	var active []*Project
	for _, p := range mpo.projects {
		if p.Active {
			active = append(active, p)
		}
	}

	return active
}

func (mpo *MultiProjectOrchestrator) SetProjectActive(name string, active bool) error {
	mpo.mu.Lock()
	defer mpo.mu.Unlock()

	p, exists := mpo.projects[name]
	if !exists {
		return fmt.Errorf("project %s not found", name)
	}

	p.Active = active
	return nil
}

func (mpo *MultiProjectOrchestrator) GetNextProject() *Project {
	mpo.mu.RLock()
	defer mpo.mu.RUnlock()

	var next *Project
	var highestPriority int = -1

	for _, p := range mpo.projects {
		if !p.Active {
			continue
		}

		if p.Priority > highestPriority {
			highestPriority = p.Priority
			next = p
		}
	}

	if next != nil {
		next.LastProcessed = time.Now()
	}

	return next
}

func (mpo *MultiProjectOrchestrator) ScheduleTask(project, taskType string, interval time.Duration, callback func() error) error {
	mpo.mu.Lock()
	defer mpo.mu.Unlock()

	scheduler, exists := mpo.schedulers[project]
	if !exists {
		scheduler = &ProjectScheduler{
			ProjectName: project,
			Interval:    interval,
			TaskQueue:   make([]*ScheduledTask, 0),
			Enabled:     true,
		}
		mpo.schedulers[project] = scheduler
	}

	task := &ScheduledTask{
		ID:       fmt.Sprintf("%s_%s_%d", project, taskType, time.Now().Unix()),
		Project:  project,
		TaskType: taskType,
		Schedule: interval.String(),
		NextRun:  time.Now().Add(interval),
		Callback: callback,
	}

	scheduler.TaskQueue = append(scheduler.TaskQueue, task)
	metrics.ScheduledTasksTotal.WithLabelValues(project, taskType).Inc()

	return nil
}

func (mpo *MultiProjectOrchestrator) StartAllSchedulers(ctx context.Context) {
	mpo.mu.RLock()
	schedulers := make(map[string]*ProjectScheduler)
	for k, v := range mpo.schedulers {
		schedulers[k] = v
	}
	mpo.mu.RUnlock()

	for _, scheduler := range schedulers {
		go mpo.runScheduler(ctx, scheduler)
	}
}

func (mpo *MultiProjectOrchestrator) runScheduler(ctx context.Context, scheduler *ProjectScheduler) {
	ticker := time.NewTicker(scheduler.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !scheduler.Enabled {
				continue
			}

			for _, task := range scheduler.TaskQueue {
				if time.Now().After(task.NextRun) {
					if err := task.Callback(); err != nil {
						metrics.ScheduledTasksFailed.WithLabelValues(scheduler.ProjectName, task.TaskType).Inc()
						continue
					}

					task.LastRun = time.Now()
					task.NextRun = time.Now().Add(scheduler.Interval)
					metrics.ScheduledTasksCompleted.WithLabelValues(scheduler.ProjectName, task.TaskType).Inc()
				}
			}
		}
	}
}

func (mpo *MultiProjectOrchestrator) DiscoverProjects(basePath string) error {
	projectsDir := []string{
		filepath.Join(basePath, "BIOMETRICS"),
		filepath.Join(basePath, "SIN-Solver"),
		filepath.Join(basePath, "simone-webshop-01"),
	}

	for _, dir := range projectsDir {
		projectName := filepath.Base(dir)
		planPath := paths.SisyphusPlansDir(projectName)

		if err := mpo.RegisterProject(projectName, dir, planPath, 1); err != nil {
			continue
		}
	}

	return nil
}

func (mpo *MultiProjectOrchestrator) GetStats() map[string]interface{} {
	mpo.mu.RLock()
	defer mpo.mu.RUnlock()

	active := 0
	inactive := 0

	for _, p := range mpo.projects {
		if p.Active {
			active++
		} else {
			inactive++
		}
	}

	return map[string]interface{}{
		"total_projects":    len(mpo.projects),
		"active_projects":   active,
		"inactive_projects": inactive,
		"schedulers":        len(mpo.schedulers),
	}
}

func (mpo *MultiProjectOrchestrator) BalanceProjects() {
	projects := mpo.GetActiveProjects()

	if len(projects) == 0 {
		return
	}

	basePriority := 100 / len(projects)

	for i, p := range projects {
		p.Priority = basePriority * (len(projects) - i)
	}

	metrics.ProjectRebalanced.Inc()
}
