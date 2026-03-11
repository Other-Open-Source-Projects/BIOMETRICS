package planning

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"biometrics-cli/internal/contracts"
)

var whitespace = regexp.MustCompile(`\s+`)

func BuildPlan(goal string) contracts.PlannerPlan {
	normalizedGoal := normalizeGoal(goal)
	parts := splitGoal(normalizedGoal)
	if len(parts) == 0 {
		parts = []string{normalizedGoal}
	}
	if cap := maxWorkPackages(); cap > 0 && len(parts) > cap {
		parts = parts[:cap]
	}

	plan := contracts.PlannerPlan{
		Version:      1,
		WorkPackages: make([]contracts.WorkPackage, 0, len(parts)),
	}

	for i, part := range parts {
		wpID := fmt.Sprintf("wp-%02d", i+1)
		plan.WorkPackages = append(plan.WorkPackages, contracts.WorkPackage{
			ID:        wpID,
			Title:     part,
			DependsOn: []string{},
			Priority:  100 - i,
			Scope:     inferScope(part),
		})
	}

	return plan
}

func maxWorkPackages() int {
	raw := strings.TrimSpace(os.Getenv("BIOMETRICS_MAX_WORK_PACKAGES"))
	if raw == "" {
		return 0
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return 0
	}
	return n
}

func normalizeGoal(goal string) string {
	goal = strings.TrimSpace(goal)
	if goal == "" {
		return "deliver requested changes"
	}
	goal = strings.ReplaceAll(goal, "\n", " ")
	goal = whitespace.ReplaceAllString(goal, " ")
	return strings.TrimSpace(goal)
}

func splitGoal(goal string) []string {
	if goal == "" {
		return []string{}
	}

	raw := strings.FieldsFunc(goal, func(r rune) bool {
		switch r {
		case ',', ';', '|':
			return true
		default:
			return false
		}
	})

	if len(raw) <= 1 && strings.Contains(strings.ToLower(goal), " and ") {
		raw = strings.Split(goal, " and ")
	}

	seen := make(map[string]struct{})
	parts := make([]string, 0, len(raw))
	for _, item := range raw {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		key := strings.ToLower(item)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		parts = append(parts, item)
	}
	return parts
}

func inferScope(text string) []string {
	lower := strings.ToLower(text)
	scope := make([]string, 0, 3)

	if containsAny(lower, "api", "backend", "service", "database", "db") {
		scope = append(scope, "backend")
	}
	if containsAny(lower, "ui", "frontend", "web", "react", "view") {
		scope = append(scope, "frontend")
	}
	if containsAny(lower, "test", "quality", "lint", "review") {
		scope = append(scope, "testing")
	}
	if len(scope) == 0 {
		scope = append(scope, "general")
	}
	return scope
}

func containsAny(text string, words ...string) bool {
	for _, word := range words {
		if strings.Contains(text, word) {
			return true
		}
	}
	return false
}
