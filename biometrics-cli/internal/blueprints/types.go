package blueprints

import "time"

type Catalog struct {
	Version string         `json:"version"`
	Source  SourceRef      `json:"source"`
	Profiles []ProfileSpec `json:"profiles"`
}

type SourceRef struct {
	Repo   string `json:"repo"`
	Commit string `json:"commit"`
}

type ProfileSpec struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Version     string       `json:"version"`
	Description string       `json:"description"`
	Core        CoreSpec     `json:"core"`
	Modules     []ModuleSpec `json:"modules"`
}

type CoreSpec struct {
	BlueprintTemplate string `json:"blueprint_template"`
	AgentsTemplate    string `json:"agents_template"`
}

type ModuleSpec struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Template    string `json:"template"`
}

type ProfileSummary struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Version     string          `json:"version"`
	Description string          `json:"description"`
	Modules     []ModuleSummary `json:"modules"`
}

type ModuleSummary struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type ApplyOptions struct {
	ProjectID string
	ProfileID string
	ModuleIDs []string
}

type ApplyResult struct {
	ProjectPath     string    `json:"project_path"`
	ProfileID       string    `json:"profile_id"`
	BlueprintPath   string    `json:"blueprint_path"`
	AgentsPath      string    `json:"agents_path"`
	AppliedModules  []string  `json:"applied_modules"`
	SkippedModules  []string  `json:"skipped_modules"`
	ChangedFiles    []string  `json:"changed_files"`
	GeneratedAt     time.Time `json:"generated_at"`
}
