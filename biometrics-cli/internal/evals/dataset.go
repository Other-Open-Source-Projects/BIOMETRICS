package evals

import (
	"fmt"
	"hash/fnv"
	"math"
	"sort"
	"strings"
)

const (
	defaultDatasetID = "apex-suite-v1"
	defaultEvalSeed  = int64(20260226)
)

type Dataset struct {
	ID    string `json:"id"`
	Seed  int64  `json:"seed"`
	Tasks []Task `json:"tasks"`
}

type Task struct {
	ID               string  `json:"id"`
	Title            string  `json:"title"`
	Language         string  `json:"language"`
	Category         string  `json:"category"`
	Difficulty       float64 `json:"difficulty"`
	EstimatedSeconds float64 `json:"estimated_seconds"`
}

type strategyProfile struct {
	SuccessBase    float64
	TimeMultiplier float64
	CostMultiplier float64
}

type taskTemplate struct {
	Title            string
	Language         string
	Category         string
	Difficulty       float64
	EstimatedSeconds float64
}

var apexTaskTemplates = []taskTemplate{
	{Title: "Refactor dependency injection graph", Language: "go", Category: "architecture-refactor", Difficulty: 0.72, EstimatedSeconds: 980},
	{Title: "Fix flaky integration test race", Language: "go", Category: "test-failure", Difficulty: 0.64, EstimatedSeconds: 760},
	{Title: "Backfill migration with online safety guards", Language: "sql", Category: "migration", Difficulty: 0.78, EstimatedSeconds: 1120},
	{Title: "Harden secret redaction in logs", Language: "go", Category: "security-hardening", Difficulty: 0.69, EstimatedSeconds: 840},
	{Title: "Repair broken frontend state synchronization", Language: "typescript", Category: "bugfix", Difficulty: 0.56, EstimatedSeconds: 620},
	{Title: "Implement resumable orchestration step", Language: "go", Category: "workflow", Difficulty: 0.66, EstimatedSeconds: 780},
	{Title: "Stabilize websocket reconnect handling", Language: "typescript", Category: "reliability", Difficulty: 0.52, EstimatedSeconds: 590},
	{Title: "Optimize queue scheduling fairness", Language: "go", Category: "performance", Difficulty: 0.74, EstimatedSeconds: 990},
	{Title: "Resolve protobuf contract drift", Language: "proto", Category: "api-contract", Difficulty: 0.58, EstimatedSeconds: 640},
	{Title: "Patch command injection edge case", Language: "python", Category: "security-hardening", Difficulty: 0.76, EstimatedSeconds: 1040},
	{Title: "Refine model fallback decision policy", Language: "go", Category: "llm-routing", Difficulty: 0.61, EstimatedSeconds: 730},
	{Title: "Improve end-to-end telemetry attribution", Language: "typescript", Category: "observability", Difficulty: 0.49, EstimatedSeconds: 540},
	{Title: "Fix deadlock in concurrent worktree setup", Language: "go", Category: "concurrency", Difficulty: 0.81, EstimatedSeconds: 1190},
	{Title: "Reconcile schema evolution in scorecards", Language: "json-schema", Category: "migration", Difficulty: 0.55, EstimatedSeconds: 610},
	{Title: "Reduce test runtime via fixture isolation", Language: "go", Category: "performance", Difficulty: 0.57, EstimatedSeconds: 650},
	{Title: "Repair CI regression in eval pipeline", Language: "bash", Category: "ci-cd", Difficulty: 0.46, EstimatedSeconds: 500},
	{Title: "Implement policy-aware network sandboxing", Language: "go", Category: "security-hardening", Difficulty: 0.83, EstimatedSeconds: 1260},
	{Title: "Fix stale cache invalidation in UI layer", Language: "typescript", Category: "bugfix", Difficulty: 0.51, EstimatedSeconds: 560},
	{Title: "Normalize contract inventory generation", Language: "python", Category: "tooling", Difficulty: 0.44, EstimatedSeconds: 470},
	{Title: "Recover partial-run consistency after crash", Language: "go", Category: "recovery", Difficulty: 0.71, EstimatedSeconds: 920},
}

func loadDataset(datasetID string, seed int64, limit int) (Dataset, error) {
	id := strings.TrimSpace(strings.ToLower(datasetID))
	if id == "" {
		id = defaultDatasetID
	}
	if id != defaultDatasetID {
		return Dataset{}, fmt.Errorf("unsupported dataset_id %q", datasetID)
	}
	if seed == 0 {
		seed = defaultEvalSeed
	}
	if limit <= 0 {
		limit = 500
	}
	if limit > 5000 {
		limit = 5000
	}

	tasks := make([]Task, 0, limit)
	for i := 0; i < limit; i++ {
		template := apexTaskTemplates[i%len(apexTaskTemplates)]
		jitter := deterministicUnit(seed, fmt.Sprintf("%s:%d", id, i))
		difficulty := clamp(template.Difficulty+((jitter-0.5)*0.14), 0.05, 0.95)
		estimated := math.Max(60, template.EstimatedSeconds*(0.9+(0.2*jitter)))
		tasks = append(tasks, Task{
			ID:               fmt.Sprintf("%s-task-%04d", id, i+1),
			Title:            template.Title,
			Language:         template.Language,
			Category:         template.Category,
			Difficulty:       round(difficulty, 4),
			EstimatedSeconds: round(estimated, 2),
		})
	}

	return Dataset{
		ID:    id,
		Seed:  seed,
		Tasks: tasks,
	}, nil
}

func evaluateStrategy(strategy string, dataset Dataset, observed Metrics, seed int64) Metrics {
	profile := profileForStrategy(strategy)
	successes := 0
	totalCost := 0.0
	allDurations := make([]float64, 0, len(dataset.Tasks))
	successDurations := make([]float64, 0, len(dataset.Tasks))

	for i, task := range dataset.Tasks {
		outcomeNoise := deterministicUnit(seed+int64(i*17), strategy+"|"+task.ID)
		durationNoise := deterministicUnit(seed+int64(i*29), task.ID+"|duration|"+strategy)
		costNoise := deterministicUnit(seed+int64(i*37), task.ID+"|cost|"+strategy)

		successProb := clamp(profile.SuccessBase-(task.Difficulty*0.18)+((1-task.Difficulty)*0.04), 0.35, 0.995)
		duration := task.EstimatedSeconds * profile.TimeMultiplier * (0.88 + (0.24 * durationNoise))
		cost := (0.0012 + (task.Difficulty * 0.0048)) * profile.CostMultiplier * (0.9 + (0.2 * costNoise))

		success := outcomeNoise <= successProb
		if !success {
			duration *= 1.24
			cost *= 1.10
		}
		if success {
			successes++
			successDurations = append(successDurations, duration)
		}

		allDurations = append(allDurations, duration)
		totalCost += cost
	}

	total := len(dataset.Tasks)
	quality := 0.0
	if total > 0 {
		quality = float64(successes) / float64(total)
	}
	medianSuccessDuration := percentile(successDurations, 0.50)
	if medianSuccessDuration == 0 {
		medianSuccessDuration = percentile(allDurations, 0.50)
	}
	costPerSuccess := totalCost / float64(max(successes, 1))

	metrics := Metrics{
		QualityScore:             round(clamp(quality, 0, 1), 4),
		MedianTimeToGreenSeconds: round(math.Max(1, medianSuccessDuration), 2),
		CostPerSuccess:           round(math.Max(0.0001, costPerSuccess), 6),
	}

	if observed.CompositeScore > 0 {
		metrics.QualityScore = round(clamp((metrics.QualityScore*0.85)+(observed.QualityScore*0.15), 0, 1), 4)
		metrics.MedianTimeToGreenSeconds = round(math.Max(1, (metrics.MedianTimeToGreenSeconds*0.85)+(observed.MedianTimeToGreenSeconds*0.15)), 2)
		metrics.CostPerSuccess = round(math.Max(0.0001, (metrics.CostPerSuccess*0.85)+(observed.CostPerSuccess*0.15)), 6)
	}
	metrics.CompositeScore = composite(metrics)
	return metrics
}

func profileForStrategy(strategy string) strategyProfile {
	switch normalizeStrategyID(strategy) {
	case "arena":
		return strategyProfile{SuccessBase: 0.93, TimeMultiplier: 0.76, CostMultiplier: 1.18}
	case "adaptive":
		return strategyProfile{SuccessBase: 0.90, TimeMultiplier: 0.83, CostMultiplier: 0.92}
	case "codex":
		return strategyProfile{SuccessBase: 0.94, TimeMultiplier: 0.79, CostMultiplier: 1.01}
	case "claude_code":
		return strategyProfile{SuccessBase: 0.93, TimeMultiplier: 0.82, CostMultiplier: 1.03}
	case "cursor":
		return strategyProfile{SuccessBase: 0.90, TimeMultiplier: 0.80, CostMultiplier: 0.95}
	case "copilot_agent":
		return strategyProfile{SuccessBase: 0.88, TimeMultiplier: 0.87, CostMultiplier: 0.93}
	case "windsurf":
		return strategyProfile{SuccessBase: 0.89, TimeMultiplier: 0.82, CostMultiplier: 0.94}
	default:
		return strategyProfile{SuccessBase: 0.86, TimeMultiplier: 1.00, CostMultiplier: 1.00}
	}
}

func normalizeStrategyID(raw string) string {
	id := strings.ToLower(strings.TrimSpace(raw))
	id = strings.ReplaceAll(id, "-", "_")
	id = strings.ReplaceAll(id, " ", "_")
	return id
}

func normalizeCompetitors(raw []string) []string {
	if len(raw) == 0 {
		raw = []string{"codex", "claude_code", "cursor", "copilot_agent", "windsurf"}
	}
	seen := make(map[string]struct{}, len(raw))
	out := make([]string, 0, len(raw))
	for _, entry := range raw {
		normalized := normalizeStrategyID(entry)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	sort.Strings(out)
	return out
}

func deterministicUnit(seed int64, key string) float64 {
	hasher := fnv.New64a()
	_, _ = hasher.Write([]byte(fmt.Sprintf("%d:%s", seed, key)))
	value := hasher.Sum64()
	return float64(value%1000000) / 1000000.0
}

func percentile(values []float64, p float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]float64{}, values...)
	sort.Float64s(sorted)
	if len(sorted) == 1 {
		return sorted[0]
	}
	if p <= 0 {
		return sorted[0]
	}
	if p >= 1 {
		return sorted[len(sorted)-1]
	}
	index := int(math.Round(float64(len(sorted)-1) * p))
	if index < 0 {
		index = 0
	}
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	return sorted[index]
}
