package compiler

import (
	"fmt"
	"strings"

	"biometrics-cli/internal/context/indexer"
)

type Compiled struct {
	Prompt          string
	Budget          int
	UsedBytes       int
	SelectedSources []string
}

func Compile(basePrompt string, sources []indexer.Source, budget int) Compiled {
	if budget <= 0 {
		budget = 24000
	}

	builder := strings.Builder{}
	selected := make([]string, 0, len(sources))

	appendSegment := func(segment string) bool {
		segment = strings.TrimSpace(segment)
		if segment == "" {
			return true
		}
		if builder.Len()+len(segment)+1 > budget {
			return false
		}
		if builder.Len() > 0 {
			builder.WriteString("\n")
		}
		builder.WriteString(segment)
		return true
	}

	if !appendSegment(basePrompt) {
		return Compiled{
			Prompt:          "",
			Budget:          budget,
			UsedBytes:       0,
			SelectedSources: selected,
		}
	}

	for _, source := range sources {
		segment := fmt.Sprintf("[%s] %s", source.ID, source.Content)
		if !appendSegment(segment) {
			break
		}
		selected = append(selected, source.ID)
	}

	return Compiled{
		Prompt:          builder.String(),
		Budget:          budget,
		UsedBytes:       builder.Len(),
		SelectedSources: selected,
	}
}
