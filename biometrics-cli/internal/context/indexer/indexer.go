package indexer

import (
	"fmt"
	"sort"
	"strings"

	"biometrics-cli/internal/contracts"
)

type Source struct {
	ID      string
	Kind    string
	Content string
}

func Build(run contracts.Run, task contracts.Task, input map[string]string) []Source {
	sources := make([]Source, 0, len(input)+4)
	sources = append(sources, Source{
		ID:      "run.goal",
		Kind:    "goal",
		Content: strings.TrimSpace(run.Goal),
	})
	sources = append(sources, Source{
		ID:      "task.name",
		Kind:    "task",
		Content: strings.TrimSpace(task.Name),
	})
	if run.BlueprintProfile != "" {
		sources = append(sources, Source{
			ID:      "blueprint.profile",
			Kind:    "blueprint",
			Content: run.BlueprintProfile,
		})
	}
	if len(run.BlueprintModules) > 0 {
		sources = append(sources, Source{
			ID:      "blueprint.modules",
			Kind:    "blueprint",
			Content: strings.Join(run.BlueprintModules, ","),
		})
	}

	keys := make([]string, 0, len(input))
	for key := range input {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := strings.TrimSpace(input[key])
		if value == "" {
			continue
		}
		sources = append(sources, Source{
			ID:      fmt.Sprintf("input.%s", key),
			Kind:    "input",
			Content: value,
		})
	}
	return sources
}
