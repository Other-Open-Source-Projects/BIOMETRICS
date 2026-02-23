package workflows

import (
	"context"
	"fmt"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// Workflow represents a YAML-defined workflow
type Workflow struct {
	Name        string            `yaml:"name" json:"name"`
	Version     string            `yaml:"version" json:"version"`
	Description string            `yaml:"description" json:"description"`
	Trigger     TriggerConfig     `yaml:"trigger" json:"trigger"`
	Steps       []Step            `yaml:"steps" json:"steps"`
	Inputs      map[string]Field  `yaml:"inputs" json:"inputs"`
	Outputs     map[string]Field  `yaml:"outputs" json:"outputs"`
	Env         map[string]string `yaml:"env" json:"env"`
	Options     WorkflowOptions   `yaml:"options" json:"options"`
}

// TriggerConfig defines how a workflow is triggered
type TriggerConfig struct {
	Type     string   `yaml:"type" json:"type"`
	Cron     string   `yaml:"cron,omitempty" json:"cron,omitempty"`
	Events   []string `yaml:"events,omitempty" json:"events,omitempty"`
	Webhooks []string `yaml:"webhooks,omitempty" json:"webhooks,omitempty"`
}

// Step represents a single workflow step
type Step struct {
	ID        string           `yaml:"id" json:"id"`
	Name      string           `yaml:"name" json:"name"`
	Type      string           `yaml:"type" json:"type"`
	Agent     AgentConfig      `yaml:"agent,omitempty" json:"agent,omitempty"`
	Condition *ConditionConfig `yaml:"condition,omitempty" json:"condition,omitempty"`
	Parallel  *ParallelConfig  `yaml:"parallel,omitempty" json:"parallel,omitempty"`
	Loop      *LoopConfig      `yaml:"loop,omitempty" json:"loop,omitempty"`
	Transform *TransformConfig `yaml:"transform,omitempty" json:"transform,omitempty"`
	Timeout   time.Duration    `yaml:"timeout" json:"timeout"`
	Retry     RetryConfig      `yaml:"retry" json:"retry"`
	OnFailure string           `yaml:"on_failure" json:"on_failure"`
	DependsOn []string         `yaml:"depends_on" json:"depends_on"`
}

// AgentConfig defines agent execution
type AgentConfig struct {
	Provider  string            `yaml:"provider" json:"provider"`
	Model     string            `yaml:"model" json:"model"`
	Prompt    string            `yaml:"prompt" json:"prompt"`
	Tools     []string          `yaml:"tools" json:"tools"`
	Variables map[string]string `yaml:"variables" json:"variables"`
}

// ConditionConfig defines conditional execution
type ConditionConfig struct {
	Expression string   `yaml:"expression" json:"expression"`
	TrueSteps  []string `yaml:"true_steps" json:"true_steps"`
	FalseSteps []string `yaml:"false_steps" json:"false_steps"`
}

// ParallelConfig defines parallel execution
type ParallelConfig struct {
	Steps []string `yaml:"steps" json:"steps"`
}

// LoopConfig defines loop execution
type LoopConfig struct {
	Over          string `yaml:"over" json:"over"`
	MaxIterations int    `yaml:"max_iterations" json:"max_iterations"`
}

// TransformConfig defines data transformation
type TransformConfig struct {
	Input  string `yaml:"input" json:"input"`
	Output string `yaml:"output" json:"output"`
	Script string `yaml:"script" json:"script"`
}

// RetryConfig defines retry behavior
type RetryConfig struct {
	MaxAttempts int           `yaml:"max_attempts" json:"max_attempts"`
	Delay       time.Duration `yaml:"delay" json:"delay"`
	Backoff     string        `yaml:"backoff" json:"backoff"`
}

// Field represents an input/output field definition
type Field struct {
	Type        string `yaml:"type" json:"type"`
	Description string `yaml:"description" json:"description"`
	Required    bool   `yaml:"required" json:"required"`
	Default     any    `yaml:"default" json:"default"`
}

// WorkflowOptions defines workflow execution options
type WorkflowOptions struct {
	Concurrency int           `yaml:"concurrency" json:"concurrency"`
	Timeout     time.Duration `yaml:"timeout" json:"timeout"`
	Debug       bool          `yaml:"debug" json:"debug"`
}

// WorkflowEngine manages workflow execution
type WorkflowEngine struct {
	mu         sync.RWMutex
	workflows  map[string]*Workflow
	executors  map[string]StepExecutor
	registry   *TemplateRegistry
	maxWorkers int
	workerPool chan struct{}
}

// StepExecutor executes individual workflow steps
type StepExecutor interface {
	Execute(ctx context.Context, step *Step, inputs map[string]any) (map[string]any, error)
	Type() string
}

// NewWorkflowEngine creates a new workflow engine
func NewWorkflowEngine(registry *TemplateRegistry, maxWorkers int) *WorkflowEngine {
	engine := &WorkflowEngine{
		workflows:  make(map[string]*Workflow),
		executors:  make(map[string]StepExecutor),
		registry:   registry,
		maxWorkers: maxWorkers,
		workerPool: make(chan struct{}, maxWorkers),
	}

	// Register default executors
	engine.RegisterExecutor(&AgentExecutor{registry: registry})
	engine.RegisterExecutor(&ConditionExecutor{})
	engine.RegisterExecutor(&ParallelExecutor{engine: engine})
	engine.RegisterExecutor(&LoopExecutor{engine: engine})
	engine.RegisterExecutor(&TransformExecutor{})

	return engine
}

// RegisterExecutor registers a step executor
func (e *WorkflowEngine) RegisterExecutor(executor StepExecutor) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.executors[executor.Type()] = executor
}

// LoadWorkflow loads a workflow from YAML
func (e *WorkflowEngine) LoadWorkflow(data []byte) (*Workflow, error) {
	var workflow Workflow
	if err := yaml.Unmarshal(data, &workflow); err != nil {
		return nil, fmt.Errorf("failed to parse workflow: %w", err)
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	e.workflows[workflow.Name] = &workflow

	return &workflow, nil
}

// GetWorkflow retrieves a workflow by name
func (e *WorkflowEngine) GetWorkflow(name string) (*Workflow, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	wf, ok := e.workflows[name]
	return wf, ok
}

// ExecuteWorkflow runs a workflow with given inputs
func (e *WorkflowEngine) ExecuteWorkflow(ctx context.Context, name string, inputs map[string]any) (map[string]any, error) {
	wf, ok := e.GetWorkflow(name)
	if !ok {
		return nil, fmt.Errorf("workflow not found: %s", name)
	}

	mergedInputs := wf.mergeInputs(inputs)

	execCtx := &ExecutionContext{
		Workflow:  wf,
		Inputs:    mergedInputs,
		Outputs:   make(map[string]any),
		StepCache: make(map[string]map[string]any),
		Started:   time.Now(),
	}

	for _, step := range wf.Steps {
		if err := e.executeStep(ctx, execCtx, &step); err != nil {
			if step.OnFailure != "" {
				fmt.Printf("Step %s failed: %v, handling with: %s\n", step.ID, err, step.OnFailure)
				continue
			}
			return nil, fmt.Errorf("step %s failed: %w", step.ID, err)
		}
	}

	return execCtx.Outputs, nil
}

// mergeInputs merges user inputs with defaults
func (w *Workflow) mergeInputs(inputs map[string]any) map[string]any {
	merged := make(map[string]any)

	for name, field := range w.Inputs {
		if field.Default != nil {
			merged[name] = field.Default
		}
	}

	for k, v := range inputs {
		merged[k] = v
	}

	return merged
}

// executeStep executes a single workflow step
func (e *WorkflowEngine) executeStep(ctx context.Context, execCtx *ExecutionContext, step *Step) error {
	select {
	case e.workerPool <- struct{}{}:
		defer func() { <-e.workerPool }()
	case <-ctx.Done():
		return ctx.Err()
	}

	if err := e.checkDependencies(execCtx, step.DependsOn); err != nil {
		return err
	}

	e.mu.RLock()
	executor, ok := e.executors[step.Type]
	e.mu.RUnlock()

	if !ok {
		return fmt.Errorf("unknown step type: %s", step.Type)
	}

	inputs := e.prepareStepInputs(execCtx, step)

	stepCtx, cancel := context.WithTimeout(ctx, step.Timeout)
	defer cancel()

	outputs, err := executor.Execute(stepCtx, step, inputs)
	if err != nil {
		return err
	}

	execCtx.StepCache[step.ID] = outputs

	for k, v := range outputs {
		execCtx.Outputs[k] = v
	}

	return nil
}

// checkDependencies ensures all dependencies are met
func (e *WorkflowEngine) checkDependencies(execCtx *ExecutionContext, dependsOn []string) error {
	for _, dep := range dependsOn {
		if _, ok := execCtx.StepCache[dep]; !ok {
			return fmt.Errorf("dependency not met: %s", dep)
		}
	}
	return nil
}

// prepareStepInputs gathers inputs from dependencies
func (e *WorkflowEngine) prepareStepInputs(execCtx *ExecutionContext, step *Step) map[string]any {
	inputs := make(map[string]any)

	for k, v := range execCtx.Inputs {
		inputs[k] = v
	}

	for _, dep := range step.DependsOn {
		if outputs, ok := execCtx.StepCache[dep]; ok {
			for k, v := range outputs {
				inputs[fmt.Sprintf("%s.%s", dep, k)] = v
			}
		}
	}

	return inputs
}

// ExecutionContext holds runtime execution state
type ExecutionContext struct {
	Workflow  *Workflow
	Inputs    map[string]any
	Outputs   map[string]any
	StepCache map[string]map[string]any
	Started   time.Time
}

// ListWorkflows returns all registered workflow names
func (e *WorkflowEngine) ListWorkflows() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	names := make([]string, 0, len(e.workflows))
	for name := range e.workflows {
		names = append(names, name)
	}
	return names
}
