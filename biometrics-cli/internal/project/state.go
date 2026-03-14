package project

import (
	"biometrics-cli/internal/paths"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Task struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

type Boulder struct {
	Tasks          []Task   `json:"tasks"`
	CompletedTasks []string `json:"completed_tasks"`
}

func GetNextTask(projectID string) (*Task, error) {
	boulderPath := filepath.Join(paths.SisyphusPlansDir(projectID), "boulder.json")

	data, err := os.ReadFile(boulderPath)
	if err != nil {
		return nil, err
	}

	var b Boulder
	if err := json.Unmarshal(data, &b); err != nil {
		return nil, err
	}

	for i, task := range b.Tasks {
		if task.Status == "pending" {
			b.Tasks[i].Status = "in_progress"

			newData, _ := json.MarshalIndent(b, "", "  ")
			os.WriteFile(boulderPath, newData, 0644)

			return &b.Tasks[i], nil
		}
	}
	return nil, fmt.Errorf("no pending tasks found")
}

func MarkTaskCompleted(projectID string, taskID string) error {
	boulderPath := filepath.Join(paths.SisyphusPlansDir(projectID), "boulder.json")

	data, err := os.ReadFile(boulderPath)
	if err != nil {
		return err
	}

	var b Boulder
	json.Unmarshal(data, &b)

	for i, task := range b.Tasks {
		if task.ID == taskID {
			b.Tasks[i].Status = "completed"
			b.CompletedTasks = append(b.CompletedTasks, taskID)
			break
		}
	}

	newData, _ := json.MarshalIndent(b, "", "  ")
	return os.WriteFile(boulderPath, newData, 0644)
}
