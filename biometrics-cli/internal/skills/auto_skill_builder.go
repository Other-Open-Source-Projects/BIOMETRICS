package skills

import (
	"biometrics-cli/internal/paths"
	"biometrics-cli/internal/state"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type AutoSkillBuilder struct {
	patterns  []SkillPattern
	newSkills []Skill
}

type SkillPattern struct {
	Prompt        string
	DetectedSkill string
	SuccessRate   float64
	Frequency     int
}

type GeneratedSkill struct {
	Name        string
	Keywords    []string
	Description string
	Confidence  float64
}

func NewAutoSkillBuilder() *AutoSkillBuilder {
	return &AutoSkillBuilder{
		patterns:  make([]SkillPattern, 0),
		newSkills: make([]Skill, 0),
	}
}

func (a *AutoSkillBuilder) AnalyzePatterns() {
	state.GlobalState.Log("INFO", "=== AUTO-SKILL-BUILDER: Analyzing patterns ===")

	promptHistory := a.loadPromptHistory()

	for _, prompt := range promptHistory {
		matchedSkill := a.detectSkillFromPrompt(prompt)
		if matchedSkill != "" {
			a.recordPattern(prompt, matchedSkill)
		}
	}

	a.calculateSkillSuccessRates()

	state.GlobalState.Log("INFO", fmt.Sprintf("Analyzed %d prompts, found %d skill patterns", len(promptHistory), len(a.patterns)))
}

func (a *AutoSkillBuilder) loadPromptHistory() []string {
	var prompts []string

	data, err := os.ReadFile(paths.SisyphusPromptHistoryPath())
	if err != nil {
		return prompts
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if len(line) > 10 {
			prompts = append(prompts, line)
		}
	}

	return prompts
}

func (a *AutoSkillBuilder) detectSkillFromPrompt(prompt string) string {
	lowerPrompt := strings.ToLower(prompt)

	skillIndicators := map[string][]string{
		"browser-automation": {"navigate", "click", "screenshot", "fill form", "scroll"},
		"database":           {"query", "insert", "update", "delete", "select", "sql"},
		"api-integration":    {"endpoint", "request", "http", "rest", "json"},
		"file-operations":    {"read", "write", "file", "directory", "path"},
		"testing":            {"test", "assert", "expect", "mock", "coverage"},
		"security":           {"auth", "token", "password", "encrypt", "validate"},
		"devops":             {"docker", "deploy", "ci/cd", "pipeline", "kubernetes"},
		"ai-ml":              {"model", "train", "predict", "embedding", "vector"},
	}

	for skill, indicators := range skillIndicators {
		for _, indicator := range indicators {
			if strings.Contains(lowerPrompt, indicator) {
				return skill
			}
		}
	}

	return ""
}

func (a *AutoSkillBuilder) recordPattern(prompt, skill string) {
	for i := range a.patterns {
		if a.patterns[i].DetectedSkill == skill {
			a.patterns[i].Frequency++
			return
		}
	}

	a.patterns = append(a.patterns, SkillPattern{
		Prompt:        prompt,
		DetectedSkill: skill,
		SuccessRate:   0.8,
		Frequency:     1,
	})
}

func (a *AutoSkillBuilder) calculateSkillSuccessRates() {
	for i := range a.patterns {
		if a.patterns[i].Frequency > 5 {
			a.patterns[i].SuccessRate = 0.95
		} else if a.patterns[i].Frequency > 2 {
			a.patterns[i].SuccessRate = 0.85
		}
	}
}

func (a *AutoSkillBuilder) GenerateNewSkills() {
	state.GlobalState.Log("INFO", "=== AUTO-SKILL-BUILDER: Generating new skills ===")

	a.AnalyzePatterns()

	generatedSkills := a.generateSkillsFromPatterns()

	for _, gs := range generatedSkills {
		if gs.Confidence > 0.7 {
			newSkill := Skill{
				Name:        gs.Name,
				Keywords:    gs.Keywords,
				Description: gs.Description,
			}
			Registry = append(Registry, newSkill)
			a.newSkills = append(a.newSkills, newSkill)
			state.GlobalState.Log("SUCCESS", fmt.Sprintf("Auto-generated skill: %s (confidence: %.2f)", gs.Name, gs.Confidence))
		}
	}

	if len(a.newSkills) > 0 {
		a.persistGeneratedSkills()
	}

	state.GlobalState.Log("INFO", fmt.Sprintf("Generated %d new skills", len(a.newSkills)))
}

func (a *AutoSkillBuilder) generateSkillsFromPatterns() []GeneratedSkill {
	var generated []GeneratedSkill

	skillGenerators := map[string]GeneratedSkill{
		"browser-automation": {
			Name:        "browser-automation",
			Keywords:    []string{"navigate", "click", "screenshot", "fill", "form", "automation"},
			Description: "Browser automation for web scraping and testing",
			Confidence:  0.85,
		},
		"database": {
			Name:        "database-ops",
			Keywords:    []string{"query", "sql", "database", "postgres", "mysql"},
			Description: "Database operations and query optimization",
			Confidence:  0.90,
		},
		"api-integration": {
			Name:        "api-integration",
			Keywords:    []string{"api", "rest", "http", "endpoint", "webhook"},
			Description: "REST API integration and webhook handling",
			Confidence:  0.88,
		},
	}

	for _, pattern := range a.patterns {
		if pattern.Frequency >= 3 && pattern.SuccessRate > 0.7 {
			if gen, ok := skillGenerators[pattern.DetectedSkill]; ok {
				generated = append(generated, gen)
			}
		}
	}

	return generated
}

func (a *AutoSkillBuilder) persistGeneratedSkills() {
	content := fmt.Sprintf("// Auto-generated skills - %s\npackage skills\n\nvar Registry = []Skill{\n", time.Now().Format("2006-01-02"))

	for _, skill := range Registry {
		content += fmt.Sprintf("\t{Name: \"%s\", Keywords: []string{", skill.Name)
		for i, kw := range skill.Keywords {
			if i > 0 {
				content += ", "
			}
			content += fmt.Sprintf("\"%s\"", kw)
		}
		content += fmt.Sprintf("}, Description: \"%s\"},\n", skill.Description)
	}

	content += "}\n"

	repoRoot, err := paths.FindRepoRoot()
	if err != nil {
		state.GlobalState.Log("WARNING", fmt.Sprintf("AUTO-SKILL-BUILDER: cannot locate repo root: %v", err))
		return
	}
	outPath := filepath.Join(repoRoot, "biometrics-cli", "internal", "skills", "registry.go")
	_ = os.MkdirAll(filepath.Dir(outPath), 0755)
	_ = os.WriteFile(outPath, []byte(content), 0644)
}

func SelfTrain() {
	builder := NewAutoSkillBuilder()
	builder.AnalyzePatterns()
	builder.GenerateNewSkills()
}
