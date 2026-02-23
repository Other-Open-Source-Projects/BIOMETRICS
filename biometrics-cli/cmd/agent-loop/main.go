package main

import (
	"biometrics-cli/internal/cache"
	"biometrics-cli/internal/chaos"
	"biometrics-cli/internal/metrics"
	"biometrics-cli/internal/models"
	"biometrics-cli/internal/orchestrator"
	"biometrics-cli/internal/selfhealing"
	"biometrics-cli/internal/state"
	"biometrics-cli/internal/tracker"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func runDoctor() {
	fmt.Println("=== BIOMETRICS DOCTOR ===")
	paths := []string{"/Users/jeremy/.sisyphus", "/Users/jeremy/.config/opencode/opencode.json"}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			fmt.Printf("OK: %s\n", p)
		} else {
			fmt.Printf("ERROR: %s\n", p)
		}
	}
}

func runSicherCheck(agent string) {
	prompt := "Sicher? Führe eine vollständige Selbstreflexion durch."
	_ = exec.Command("opencode", "prompt", prompt, "--agent", agent).Run()
}

func getModelForAgent(agent string) string {
	switch agent {
	case "sisyphus", "build", "atlas", "deep", "oracle", "ultrabrain", "visual-engineering":
		return "google/antigravity-gemini-3.1-pro"
	case "librarian", "explore", "quick", "metis", "momus":
		return "minimax"
	default:
		return "google/antigravity-gemini-3.1-pro"
	}
}

func readBoulder(path string) (*models.Boulder, error) {
	if state.GlobalState.GetChaos() && rand.Intn(10) < 3 {
		return nil, fmt.Errorf("corrupted state")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var b models.Boulder
	err = json.Unmarshal(data, &b)
	return &b, err
}

func verifySerenaProcess() error {
	if state.GlobalState.GetChaos() && rand.Intn(10) < 5 {
		return fmt.Errorf("chaos monkey killed connection")
	}
	return exec.Command("pgrep", "-f", "serena.*start-mcp-server").Run()
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "doctor" {
		runDoctor()
		return
	}

	cache.New(&cache.CacheConfig{
		DiskPath:        "./cache",
		TTL:             5 * time.Minute,
		CleanupInterval: 1 * time.Minute,
	})
	modelCache := cache.NewModelCache(2 * time.Minute)

	state.GlobalState.InitDB()
	go orchestrator.DisplayDashboard()
	go chaos.RunChaosMonkey()
	go selfhealing.StartHealthMonitor()

	go func() {
		http.Handle("/metrics", promhttp.Handler())
		_ = http.ListenAndServe(":59002", nil)
	}()

	modelTracker := tracker.NewModelTracker()
	state.GlobalState.Log("INFO", "Started modular orchestrator with cache")

	for {
		start := time.Now()
		metrics.CyclesTotal.Inc()

		healer := selfhealing.NewSelfHealer()
		healer.RunDiagnostics()

		if err := verifySerenaProcess(); err != nil {
			state.GlobalState.Log("ERROR", "Serena MCP check failed: "+err.Error())
			time.Sleep(10 * time.Second)
			continue
		}

		b, err := readBoulder("/Users/jeremy/.sisyphus/boulder.json")
		if err != nil {
			state.GlobalState.Log("ERROR", "Failed to read boulder: "+err.Error())
			time.Sleep(10 * time.Second)
			continue
		}

		state.GlobalState.PlanName = b.PlanName
		state.GlobalState.CurrentAgent = b.Agent

		if b.ActivePlan == "" {
			time.Sleep(10 * time.Second)
			continue
		}

		model := getModelForAgent(b.Agent)

		cacheKey := fmt.Sprintf("cycle:%d:agent:%s", time.Now().Unix(), b.Agent)
		if _, found := cache.Get().Get(cacheKey); found {
			state.GlobalState.Log("INFO", "Skipping cached cycle for "+b.Agent)
			time.Sleep(30 * time.Second)
			continue
		}

		if err := modelTracker.Acquire(model); err != nil {
			time.Sleep(5 * time.Second)
			continue
		}

		state.GlobalState.ActiveModel = model
		metrics.ModelAcquisitions.WithLabelValues(model).Inc()
		state.GlobalState.Log("SUCCESS", "Acquired "+model)

		runSicherCheck(b.Agent)

		modelCache.StoreModelResult(model, b.PlanName, "completed")
		modelTracker.Release(model)
		state.GlobalState.ActiveModel = "NONE"

		duration := time.Since(start).Seconds()
		metrics.CycleDuration.Observe(duration)

		time.Sleep(60 * time.Second)
	}
}
