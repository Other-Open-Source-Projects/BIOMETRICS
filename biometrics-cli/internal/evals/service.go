package evals

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"biometrics-cli/internal/contracts"
	runtimeorchestrator "biometrics-cli/internal/runtime/orchestrator"
	"github.com/google/uuid"
)

type scorecardSource interface {
	ListScorecards() []runtimeorchestrator.Scorecard
}

type eventPublisher interface {
	Publish(ev contracts.Event) (contracts.Event, error)
}

type RunStatus string

const (
	RunQueued    RunStatus = "queued"
	RunRunning   RunStatus = "running"
	RunCompleted RunStatus = "completed"
	RunFailed    RunStatus = "failed"
)

type RunRequest struct {
	Name                string   `json:"name,omitempty"`
	CandidateStrategy   string   `json:"candidate_strategy_mode,omitempty"`
	BaselineStrategy    string   `json:"baseline_strategy_mode,omitempty"`
	SampleSize          int      `json:"sample_size,omitempty"`
	QualityTarget       float64  `json:"quality_target,omitempty"`
	SpeedImprovementMin float64  `json:"speed_improvement_min,omitempty"`
	CostReductionMin    float64  `json:"cost_reduction_min,omitempty"`
	DatasetID           string   `json:"dataset_id,omitempty"`
	Seed                int64    `json:"seed,omitempty"`
	TasksLimit          int      `json:"tasks_limit,omitempty"`
	CompetitorBaselines []string `json:"competitor_baselines,omitempty"`
}

type Comparison struct {
	QualityDelta                  float64 `json:"quality_delta"`
	TimeToGreenImprovementPercent float64 `json:"time_to_green_improvement_percent"`
	CostImprovementPercent        float64 `json:"cost_improvement_percent"`
	CompositeDelta                float64 `json:"composite_delta"`
}

type Run struct {
	ID                  string                `json:"id"`
	Name                string                `json:"name"`
	CandidateStrategy   string                `json:"candidate_strategy_mode"`
	BaselineStrategy    string                `json:"baseline_strategy_mode"`
	SampleSize          int                   `json:"sample_size"`
	DatasetID           string                `json:"dataset_id"`
	Seed                int64                 `json:"seed"`
	TasksLimit          int                   `json:"tasks_limit"`
	CompetitorBaselines []string              `json:"competitor_baselines,omitempty"`
	Status              RunStatus             `json:"status"`
	Metrics             Metrics               `json:"metrics"`
	Comparison          map[string]Comparison `json:"comparison,omitempty"`
	EvidencePaths       []string              `json:"evidence_paths,omitempty"`
	GatePackV1          GatePackV1            `json:"gate_pack_v1"`
	RegressionDetected  bool                  `json:"regression_detected"`
	Error               string                `json:"error,omitempty"`
	StartedAt           time.Time             `json:"started_at"`
	FinishedAt          time.Time             `json:"finished_at,omitempty"`
}

type GatePackV1 struct {
	Quality    bool `json:"quality"`
	Time       bool `json:"time"`
	Cost       bool `json:"cost"`
	Regression bool `json:"regression"`
	AllPass    bool `json:"all_pass"`
}

type Metrics struct {
	QualityScore             float64 `json:"quality_score"`
	MedianTimeToGreenSeconds float64 `json:"median_time_to_green_seconds"`
	CostPerSuccess           float64 `json:"cost_per_success"`
	CompositeScore           float64 `json:"composite_score"`
}

type LeaderboardEntry struct {
	Strategy                 string    `json:"strategy"`
	Runs                     int       `json:"runs"`
	QualityScore             float64   `json:"quality_score"`
	MedianTimeToGreenSeconds float64   `json:"median_time_to_green_seconds"`
	CostPerSuccess           float64   `json:"cost_per_success"`
	CompositeScore           float64   `json:"composite_score"`
	UpdatedAt                time.Time `json:"updated_at"`
}

type Service struct {
	source scorecardSource
	bus    eventPublisher

	workspaceRoot string
	evidenceDir   string

	mu          sync.RWMutex
	runs        map[string]Run
	leaderboard map[string]LeaderboardEntry
}

func NewService(source scorecardSource, bus eventPublisher) *Service {
	workspace := detectWorkspaceRoot()
	evidenceDir := filepath.Join(workspace, "logs", "evals")
	_ = os.MkdirAll(evidenceDir, 0o755)
	return &Service{
		source:        source,
		bus:           bus,
		workspaceRoot: workspace,
		evidenceDir:   evidenceDir,
		runs:          make(map[string]Run),
		leaderboard:   make(map[string]LeaderboardEntry),
	}
}

func (s *Service) StartRun(ctx context.Context, req RunRequest) (Run, error) {
	normalized, err := normalizeRunRequest(req)
	if err != nil {
		return Run{}, err
	}
	if ctx == nil {
		ctx = context.Background()
	}

	now := time.Now().UTC()
	run := Run{
		ID:                  "eval-run-" + uuid.NewString(),
		Name:                normalized.Name,
		CandidateStrategy:   normalized.CandidateStrategy,
		BaselineStrategy:    normalized.BaselineStrategy,
		SampleSize:          normalized.SampleSize,
		DatasetID:           normalized.DatasetID,
		Seed:                normalized.Seed,
		TasksLimit:          normalized.TasksLimit,
		CompetitorBaselines: append([]string{}, normalized.CompetitorBaselines...),
		Status:              RunRunning,
		StartedAt:           now,
	}

	s.mu.Lock()
	s.runs[run.ID] = run
	s.mu.Unlock()

	s.publish("", "eval.run.started", map[string]string{
		"eval_run_id":        run.ID,
		"candidate_strategy": run.CandidateStrategy,
		"baseline_strategy":  run.BaselineStrategy,
		"sample_size":        fmt.Sprintf("%d", run.SampleSize),
		"dataset_id":         run.DatasetID,
		"seed":               fmt.Sprintf("%d", run.Seed),
	})

	go s.executeRun(run.ID, normalized)
	return run, nil
}

func (s *Service) GetRun(runID string) (Run, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	run, ok := s.runs[strings.TrimSpace(runID)]
	if !ok {
		return Run{}, fmt.Errorf("eval run not found")
	}
	return run, nil
}

func (s *Service) Leaderboard() []LeaderboardEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entries := make([]LeaderboardEntry, 0, len(s.leaderboard))
	for _, entry := range s.leaderboard {
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].CompositeScore == entries[j].CompositeScore {
			return entries[i].UpdatedAt.After(entries[j].UpdatedAt)
		}
		return entries[i].CompositeScore > entries[j].CompositeScore
	})
	return entries
}

func (s *Service) executeRun(runID string, req RunRequest) {
	dataset, err := loadDataset(req.DatasetID, req.Seed, req.TasksLimit)
	if err != nil {
		s.finishRunFailure(runID, err)
		return
	}

	observed := aggregateFromScorecards(s.source)
	candidate := evaluateStrategy(req.CandidateStrategy, dataset, observed, req.Seed)
	baseline := evaluateStrategy(req.BaselineStrategy, dataset, observed, req.Seed+11)

	comparison := map[string]Comparison{
		req.BaselineStrategy: compareMetrics(candidate, baseline),
	}
	competitorMetrics := make(map[string]Metrics, len(req.CompetitorBaselines))
	for idx, competitor := range req.CompetitorBaselines {
		metrics := evaluateStrategy(competitor, dataset, observed, req.Seed+int64((idx+1)*37))
		competitorMetrics[competitor] = metrics
		comparison[competitor] = compareMetrics(candidate, metrics)
	}

	regression := comparison[req.BaselineStrategy].QualityDelta < -0.03
	baselineCmp := comparison[req.BaselineStrategy]
	gatePack := GatePackV1{
		Quality:    candidate.QualityScore >= 0.90,
		Time:       baselineCmp.TimeToGreenImprovementPercent >= 25.0,
		Cost:       baselineCmp.CostImprovementPercent >= 20.0,
		Regression: !regression,
	}
	gatePack.AllPass = gatePack.Quality && gatePack.Time && gatePack.Cost && gatePack.Regression
	if regression {
		s.publish("", "eval.metric.regression.detected", map[string]string{
			"eval_run_id":        runID,
			"metric":             "quality_score",
			"candidate_strategy": req.CandidateStrategy,
			"baseline_strategy":  req.BaselineStrategy,
			"candidate":          fmt.Sprintf("%.4f", candidate.QualityScore),
			"baseline":           fmt.Sprintf("%.4f", baseline.QualityScore),
		})
	}

	evidencePaths, err := s.writeEvidence(runID, req, dataset, candidate, baseline, competitorMetrics, comparison, gatePack)
	if err != nil {
		s.finishRunFailure(runID, err)
		return
	}

	now := time.Now().UTC()
	s.mu.Lock()
	run := s.runs[runID]
	run.Status = RunCompleted
	run.Metrics = candidate
	run.Comparison = comparison
	run.EvidencePaths = append([]string{}, evidencePaths...)
	run.GatePackV1 = gatePack
	run.RegressionDetected = regression
	run.FinishedAt = now
	s.runs[runID] = run
	s.updateLeaderboardLocked(req.CandidateStrategy, candidate, now)
	s.updateLeaderboardLocked(req.BaselineStrategy, baseline, now)
	for strategy, metrics := range competitorMetrics {
		s.updateLeaderboardLocked(strategy, metrics, now)
	}
	s.mu.Unlock()

	s.publish("", "eval.run.completed", map[string]string{
		"eval_run_id":         runID,
		"dataset_id":          req.DatasetID,
		"quality_score":       fmt.Sprintf("%.4f", candidate.QualityScore),
		"composite_score":     fmt.Sprintf("%.4f", candidate.CompositeScore),
		"regression_detected": fmt.Sprintf("%t", regression),
		"gate_pack_all_pass":  fmt.Sprintf("%t", gatePack.AllPass),
		"evidence_files":      strings.Join(evidencePaths, ","),
	})
}

func (s *Service) finishRunFailure(runID string, err error) {
	now := time.Now().UTC()
	message := strings.TrimSpace(err.Error())
	if message == "" {
		message = "eval run failed"
	}

	s.mu.Lock()
	run, ok := s.runs[runID]
	if ok {
		run.Status = RunFailed
		run.Error = message
		run.FinishedAt = now
		s.runs[runID] = run
	}
	s.mu.Unlock()

	s.publish("", "eval.run.failed", map[string]string{
		"eval_run_id": runID,
		"error":       message,
	})
}

func (s *Service) writeEvidence(
	runID string,
	req RunRequest,
	dataset Dataset,
	candidate Metrics,
	baseline Metrics,
	competitors map[string]Metrics,
	comparison map[string]Comparison,
	gatePack GatePackV1,
) ([]string, error) {
	if err := os.MkdirAll(s.evidenceDir, 0o755); err != nil {
		return nil, fmt.Errorf("create eval evidence dir: %w", err)
	}

	ts := time.Now().UTC().Format("20060102T150405Z")
	manifestPath := filepath.Join(s.evidenceDir, fmt.Sprintf("eval-manifest-%s-%s.json", runID, ts))
	resultsPath := filepath.Join(s.evidenceDir, fmt.Sprintf("eval-results-%s-%s.json", runID, ts))

	manifest := map[string]interface{}{
		"eval_run_id":           runID,
		"name":                  req.Name,
		"dataset_id":            req.DatasetID,
		"seed":                  req.Seed,
		"sample_size":           req.SampleSize,
		"tasks_limit":           req.TasksLimit,
		"tasks_total":           len(dataset.Tasks),
		"candidate_strategy":    req.CandidateStrategy,
		"baseline_strategy":     req.BaselineStrategy,
		"competitor_baselines":  req.CompetitorBaselines,
		"generated_at":          time.Now().UTC().Format(time.RFC3339Nano),
		"workspace":             s.workspaceRoot,
		"quality_target":        req.QualityTarget,
		"speed_improvement_min": req.SpeedImprovementMin,
		"cost_reduction_min":    req.CostReductionMin,
	}

	results := map[string]interface{}{
		"eval_run_id":           runID,
		"dataset_id":            req.DatasetID,
		"candidate_strategy":    req.CandidateStrategy,
		"baseline_strategy":     req.BaselineStrategy,
		"candidate_metrics":     candidate,
		"baseline_metrics":      baseline,
		"competitor_metrics":    competitors,
		"comparison":            comparison,
		"gate_pack_v1":          gatePack,
		"regression_detected":   comparison[req.BaselineStrategy].QualityDelta < -0.03,
		"evidence_generated_at": time.Now().UTC().Format(time.RFC3339Nano),
	}

	if err := writeJSONFile(manifestPath, manifest); err != nil {
		return nil, fmt.Errorf("write eval manifest: %w", err)
	}
	if err := writeJSONFile(resultsPath, results); err != nil {
		return nil, fmt.Errorf("write eval results: %w", err)
	}

	return []string{manifestPath, resultsPath}, nil
}

func writeJSONFile(path string, payload interface{}) error {
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

func (s *Service) updateLeaderboardLocked(strategy string, metrics Metrics, updatedAt time.Time) {
	entry := s.leaderboard[strategy]
	runs := entry.Runs + 1
	if runs <= 1 {
		s.leaderboard[strategy] = LeaderboardEntry{
			Strategy:                 strategy,
			Runs:                     1,
			QualityScore:             metrics.QualityScore,
			MedianTimeToGreenSeconds: metrics.MedianTimeToGreenSeconds,
			CostPerSuccess:           metrics.CostPerSuccess,
			CompositeScore:           metrics.CompositeScore,
			UpdatedAt:                updatedAt,
		}
		return
	}

	entry.Strategy = strategy
	entry.QualityScore = ((entry.QualityScore * float64(entry.Runs)) + metrics.QualityScore) / float64(runs)
	entry.MedianTimeToGreenSeconds = ((entry.MedianTimeToGreenSeconds * float64(entry.Runs)) + metrics.MedianTimeToGreenSeconds) / float64(runs)
	entry.CostPerSuccess = ((entry.CostPerSuccess * float64(entry.Runs)) + metrics.CostPerSuccess) / float64(runs)
	entry.CompositeScore = ((entry.CompositeScore * float64(entry.Runs)) + metrics.CompositeScore) / float64(runs)
	entry.Runs = runs
	entry.UpdatedAt = updatedAt
	s.leaderboard[strategy] = entry
}

func (s *Service) publish(runID, eventType string, payload map[string]string) {
	if s.bus == nil {
		return
	}
	_, _ = s.bus.Publish(contracts.Event{
		RunID:   runID,
		Type:    eventType,
		Source:  "evals",
		Payload: payload,
	})
}

func normalizeRunRequest(req RunRequest) (RunRequest, error) {
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		req.Name = "apex-suite"
	}
	req.CandidateStrategy = normalizeStrategyID(req.CandidateStrategy)
	if req.CandidateStrategy == "" {
		req.CandidateStrategy = "adaptive"
	}
	req.BaselineStrategy = normalizeStrategyID(req.BaselineStrategy)
	if req.BaselineStrategy == "" {
		req.BaselineStrategy = "deterministic"
	}

	req.SampleSize = max(1, req.SampleSize)
	if req.SampleSize < 500 {
		req.SampleSize = 500
	}
	if req.SampleSize > 5000 {
		req.SampleSize = 5000
	}

	req.TasksLimit = max(req.TasksLimit, req.SampleSize)
	if req.TasksLimit > 5000 {
		req.TasksLimit = 5000
	}
	if req.TasksLimit < req.SampleSize {
		req.TasksLimit = req.SampleSize
	}

	req.DatasetID = strings.TrimSpace(strings.ToLower(req.DatasetID))
	if req.DatasetID == "" {
		req.DatasetID = defaultDatasetID
	}
	if req.Seed == 0 {
		req.Seed = defaultEvalSeed
	}

	normalizedCompetitors := normalizeCompetitors(req.CompetitorBaselines)
	filteredCompetitors := make([]string, 0, len(normalizedCompetitors))
	for _, competitor := range normalizedCompetitors {
		if competitor == req.CandidateStrategy || competitor == req.BaselineStrategy {
			continue
		}
		filteredCompetitors = append(filteredCompetitors, competitor)
	}
	req.CompetitorBaselines = filteredCompetitors
	return req, nil
}

func aggregateFromScorecards(source scorecardSource) Metrics {
	if source == nil {
		return Metrics{}
	}
	cards := source.ListScorecards()
	if len(cards) == 0 {
		return Metrics{}
	}

	total := len(cards)
	sumQuality := 0.0
	sumTime := 0.0
	sumCost := 0.0
	for _, card := range cards {
		sumQuality += card.QualityScore
		sumTime += card.MedianTimeToGreenSeconds
		sumCost += card.CostPerSuccess
	}
	out := Metrics{
		QualityScore:             round(sumQuality/float64(total), 4),
		MedianTimeToGreenSeconds: round(sumTime/float64(total), 2),
		CostPerSuccess:           round(sumCost/float64(total), 6),
	}
	out.CompositeScore = composite(out)
	return out
}

func compareMetrics(candidate Metrics, baseline Metrics) Comparison {
	timeImprovement := 0.0
	if baseline.MedianTimeToGreenSeconds > 0 {
		timeImprovement = ((baseline.MedianTimeToGreenSeconds - candidate.MedianTimeToGreenSeconds) / baseline.MedianTimeToGreenSeconds) * 100.0
	}
	costImprovement := 0.0
	if baseline.CostPerSuccess > 0 {
		costImprovement = ((baseline.CostPerSuccess - candidate.CostPerSuccess) / baseline.CostPerSuccess) * 100.0
	}

	return Comparison{
		QualityDelta:                  round(candidate.QualityScore-baseline.QualityScore, 4),
		TimeToGreenImprovementPercent: round(timeImprovement, 4),
		CostImprovementPercent:        round(costImprovement, 4),
		CompositeDelta:                round(candidate.CompositeScore-baseline.CompositeScore, 4),
	}
}

func composite(m Metrics) float64 {
	speed := 1.0 / (1.0 + m.MedianTimeToGreenSeconds/1200.0)
	cost := 1.0 / (1.0 + m.CostPerSuccess)
	return round((0.6*m.QualityScore)+(0.25*speed)+(0.15*cost), 4)
}

func detectWorkspaceRoot() string {
	if env := strings.TrimSpace(os.Getenv("BIOMETRICS_WORKSPACE")); env != "" {
		return env
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "."
	}
	if filepath.Base(cwd) == "biometrics-cli" {
		return filepath.Dir(cwd)
	}
	return cwd
}

func clamp(value, minValue, maxValue float64) float64 {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func round(value float64, decimals int) float64 {
	factor := math.Pow10(decimals)
	return math.Round(value*factor) / factor
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
