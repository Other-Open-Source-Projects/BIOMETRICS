package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"biometrics-cli/internal/codegen"
	"github.com/gorilla/websocket"
)

var (
	generator *codegen.CodeGenerator
	wsClients = make(map[*websocket.Conn]bool)
	wsChan    = make(chan string, 100)
	// Global state for Mandates
	delqhiLoopEnabled  bool = false
	swarmEngineEnabled bool = true

)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all for dev
	},
}

func main() {
	generator = codegen.NewCodeGenerator()

	// Start WebSocket broadcaster
	go broadcastWebSocket()

	// Setup routes
	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/api/health", handleHealth)
	http.HandleFunc("/api/tasks", handleTasks)
	http.HandleFunc("/api/tasks/create", handleCreateTask)
	http.HandleFunc("/api/tasks/list", handleListTasks)
	http.HandleFunc("/api/tasks/execute", handleExecuteTask)
	http.HandleFunc("/api/tasks/", handleTaskByID)
	http.HandleFunc("/api/status", handleStatus)
	http.HandleFunc("/api/metrics", handleMetrics)
	http.HandleFunc("/api/docker/containers", handleDockerContainers)
	http.HandleFunc("/api/docker/stats", handleDockerStats)
	http.HandleFunc("/api/scheduler/jobs", handleSchedulerJobs)
	http.HandleFunc("/api/config", handleConfig)
	http.HandleFunc("/api/config/toggle-delqhi-loop", handleToggleDelqhiLoop)
	http.HandleFunc("/api/config/toggle-swarm-engine", handleToggleSwarmEngine)
	http.HandleFunc("/api/projects", handleProjectsList)

	http.HandleFunc("/api/ratelimit/stats", handleRateLimitStats)
	http.HandleFunc("/ws", handleWebSocket)

	// Static files for web UI
	fs := http.FileServer(http.Dir("./web-ui"))
	http.Handle("/ui/", http.StripPrefix("/ui/", fs))

	port := os.Getenv("PORT")
	if port == "" {
		port = "59003"
	}

	fmt.Printf("🚀 BIOMETRICS API Server starting on http://localhost:%s\n", port)
	fmt.Printf("📊 Web UI: http://localhost:%s/ui/\n", port)
	fmt.Printf("🔌 WebSocket: ws://localhost:%s/ws\n", port)
	fmt.Printf("📚 API Docs: http://localhost:%s/ui/api-docs\n", port)

	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, "/ui/", http.StatusFound)
}

func handleTasks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handleListTasks(w, r)
	case http.MethodPost:
		handleCreateTask(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleCreateTask(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Agent       string `json:"agent"` // sisyphus, prometheus, oracle
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	task, err := generator.CreateTask(req.Title, req.Description, req.Agent)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	broadcastUpdate(fmt.Sprintf("Task created: %s (%s)", task.Title, task.ID))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

func handleListTasks(w http.ResponseWriter, r *http.Request) {
	tasks := generator.ListTasks()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}

func handleExecuteTask(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TaskID string `json:"task_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Execute in background
	go func() {
		progressChan := make(chan string, 10)
		err := generator.RunCodeGeneration(req.TaskID, progressChan)

		for msg := range progressChan {
			broadcastUpdate(msg)
		}

		if err != nil {
			broadcastUpdate(fmt.Sprintf("ERROR: %v", err))
		} else {
			broadcastUpdate(fmt.Sprintf("Task completed: %s", req.TaskID))
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "started",
		"task_id": req.TaskID,
	})
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"status":       "running",
		"active_tasks": len(generator.GetActiveTasks()),
		"total_tasks":  len(generator.Tasks),
		"timestamp":    time.Now().Format(time.RFC3339),
		"version":      "1.0.0",
		"orchestrator": "biometrics-cli",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	wsClients[conn] = true
	defer func() {
		delete(wsClients, conn)
		conn.Close()
	}()

	// Send initial status
	initial := map[string]interface{}{
		"type":   "connected",
		"status": "ready",
	}
	conn.WriteJSON(initial)

	// Keep connection alive
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func broadcastWebSocket() {
	for msg := range wsChan {
		data := map[string]interface{}{
			"type":    "update",
			"message": msg,
			"time":    time.Now().Format(time.RFC3339),
		}

		jsonData, err := json.Marshal(data)
		if err != nil {
			log.Printf("JSON marshal error: %v", err)
			continue
		}

		for client := range wsClients {
			if err := client.WriteMessage(websocket.TextMessage, jsonData); err != nil {
				log.Printf("WebSocket send error: %v", err)
				delete(wsClients, client)
			}
		}
	}
}

func broadcastUpdate(message string) {
	select {
	case wsChan <- message:
	default:
		// Channel full, drop message
	}
	log.Printf("Broadcast: %s", message)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":    "healthy",
		"service":   "biometrics-api",
		"version":   "1.0.0",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

func handleTaskByID(w http.ResponseWriter, r *http.Request) {
	taskID := r.URL.Path[len("/api/tasks/"):]

	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		json.NewEncoder(w).Encode(map[string]string{"task_id": taskID, "status": "found"})
	case http.MethodDelete:
		json.NewEncoder(w).Encode(map[string]string{"task_id": taskID, "status": "deleted"})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"orchestrator": map[string]interface{}{
			"cycles_total":       0,
			"model_acquisitions": map[string]int{},
			"tasks_created":      0,
			"tasks_completed":    0,
			"tasks_failed":       0,
		},
		"notifications": map[string]int{
			"sent":    0,
			"failed":  0,
			"dropped": 0,
		},
		"scheduler": map[string]int{
			"jobs_run":    0,
			"jobs_failed": 0,
		},
		"git": map[string]int{
			"commits": 0,
			"pushes":  0,
			"pulls":   0,
		},
	})
}

func handleDockerContainers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode([]map[string]string{
		{"name": "serena", "status": "running"},
		{"name": "opencode", "status": "running"},
	})
}

func handleDockerStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"containers": 5,
		"networks":   3,
		"images":     20,
	})
}

func handleSchedulerJobs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode([]map[string]string{
		{"id": "health-check", "status": "enabled"},
		{"id": "cleanup-logs", "status": "enabled"},
	})
}

func handleConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"provider":     "nvidia-nim",
		"model":        "google/antigravity-gemini-3.1-pro",
		"auto_healing": true,
		"scheduler":    true,
		"delqhi_loop":  delqhiLoopEnabled,
		"swarm_engine": swarmEngineEnabled,
	})
}

func handleRateLimitStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"limits": map[string]interface{}{
			"default": map[string]int{"rate": 100, "burst": 10},
		},
		"current": map[string]int{},
	})
}

func handleToggleDelqhiLoop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	delqhiLoopEnabled = req.Enabled
	generator.SetDelqhiLoop(req.Enabled)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "success", "delqhi_loop": delqhiLoopEnabled})
	broadcastUpdate(fmt.Sprintf("Delqhi-Loop set to %v", delqhiLoopEnabled))
}

func handleToggleSwarmEngine(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	swarmEngineEnabled = req.Enabled
	generator.SetSwarmEngine(req.Enabled)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "success", "swarm_engine": swarmEngineEnabled})
	broadcastUpdate(fmt.Sprintf("Swarm Engine set to %v", swarmEngineEnabled))
}

func handleProjectsList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode([]map[string]string{
		{"id": "biometrics", "name": "BIOMETRICS (Core)"},
		{"id": "sin-solver", "name": "SIN-Solver"},
		{"id": "simone-webshop", "name": "Simone-Webshop"},
	})
}
