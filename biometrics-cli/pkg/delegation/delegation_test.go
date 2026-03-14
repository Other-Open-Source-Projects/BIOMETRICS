package delegation

import (
	"testing"
	"time"
)

func TestTaskCreation(t *testing.T) {
	task := NewTask("test-1", TaskTypeCode, PriorityHigh, nil)

	if task.ID != "test-1" {
		t.Errorf("Expected ID test-1, got %s", task.ID)
	}

	if task.Status != TaskStatusPending {
		t.Errorf("Expected status pending, got %s", task.Status)
	}

	if task.Priority != PriorityHigh {
		t.Errorf("Expected priority high, got %d", task.Priority)
	}
}

func TestTaskContext(t *testing.T) {
	task := NewTask("test-2", TaskTypeCode, PriorityNormal, nil)

	task.SetContext("key1", "value1")
	task.SetContext("key2", 123)

	if task.GetContext("key1") != "value1" {
		t.Errorf("Expected value1, got %v", task.GetContext("key1"))
	}

	if task.GetContext("key2") != 123 {
		t.Errorf("Expected 123, got %v", task.GetContext("key2"))
	}
}

func TestTaskRetry(t *testing.T) {
	task := NewTask("test-3", TaskTypeCode, PriorityNormal, nil)
	task.MaxRetries = 3

	if !task.IncrementRetry() {
		t.Error("Expected retry to succeed")
	}

	if task.RetryCount != 1 {
		t.Errorf("Expected retry count 1, got %d", task.RetryCount)
	}
}

func TestPriorityQueue(t *testing.T) {
	pq := NewPriorityQueue()

	task1 := NewTask("task-1", TaskTypeCode, PriorityNormal, nil)
	task2 := NewTask("task-2", TaskTypeCode, PriorityCritical, nil)
	task3 := NewTask("task-3", TaskTypeCode, PriorityHigh, nil)

	pq.Enqueue(task1)
	pq.Enqueue(task2)
	pq.Enqueue(task3)

	if pq.Len() != 3 {
		t.Errorf("Expected length 3, got %d", pq.Len())
	}

	first := pq.Dequeue()
	if first.ID != "task-2" {
		t.Errorf("Expected task-2 (critical) first, got %s", first.ID)
	}

	second := pq.Dequeue()
	if second.ID != "task-3" {
		t.Errorf("Expected task-3 (high) second, got %s", second.ID)
	}
}

func TestCircuitBreaker(t *testing.T) {
	cb := NewCircuitBreaker(3, 30*time.Second)

	if !cb.CanExecute() {
		t.Error("Expected circuit to be closed initially")
	}

	cb.RecordFailure()
	cb.RecordFailure()

	if !cb.CanExecute() {
		t.Error("Expected circuit to still allow execution after 2 failures")
	}

	cb.RecordFailure()

	if cb.CanExecute() {
		t.Error("Expected circuit to be open after 3 failures")
	}

	cb.RecordSuccess()

	if !cb.CanExecute() {
		t.Error("Expected circuit to be closed after success")
	}
}

func TestDelegationRouter(t *testing.T) {
	router := NewDelegationRouter()

	agent1 := &AgentCapability{
		Name:         "sisyphus",
		AgentID:      "agent-001",
		Capabilities: []string{"code", "testing"},
		Load:         5,
		Healthy:      true,
	}

	agent2 := &AgentCapability{
		Name:         "prometheus",
		AgentID:      "agent-002",
		Capabilities: []string{"planning", "architecture"},
		Load:         2,
		Healthy:      true,
	}

	router.RegisterAgent(agent1)
	router.RegisterAgent(agent2)

	task := NewTask("test-task", TaskTypeCode, PriorityHigh, nil)

	agentID, err := router.Route(task)
	if err != nil {
		t.Fatalf("Expected successful routing, got error: %v", err)
	}

	if agentID != "agent-001" {
		t.Errorf("Expected agent-001 (has code capability), got %s", agentID)
	}
}

func TestWorkerPool(t *testing.T) {
	router := NewDelegationRouter()
	router.RegisterAgent(&AgentCapability{
		Name:         "agent-1",
		AgentID:      "agent-001",
		Capabilities: []string{string(TaskTypeCode)},
		Load:         0,
		Healthy:      true,
	})
	engine := NewWorkerPool(5, router)
	defer engine.Shutdown()

	task := NewTask("test-task", TaskTypeCode, PriorityNormal, nil)
	engine.Submit(task)

	resultChan := engine.Results()
	select {
	case result := <-resultChan:
		if !result.Success {
			t.Errorf("Expected successful execution, got error: %v", result.Error)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for result")
	}
}

func TestResultAggregator(t *testing.T) {
	config := AggregatorConfig{
		Strategy: MergeStrategyConcat,
		Timeout:  10 * time.Second,
	}

	aggregator := NewResultAggregator(config)
	aggregator.SetTotalTasks("batch-1", 3)

	aggregator.Collect("batch-1", &TaskResult{
		TaskID:  "task-1",
		Success: true,
		Data:    "result1",
	})

	aggregator.Collect("batch-1", &TaskResult{
		TaskID:  "task-2",
		Success: true,
		Data:    "result2",
	})

	aggregator.Collect("batch-1", &TaskResult{
		TaskID:  "task-3",
		Success: true,
		Data:    "result3",
	})

	results, err := aggregator.GetResults("batch-1")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}
}
