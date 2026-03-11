package skillkit

import "time"

type Scope string

const (
	ScopeRepo   Scope = "repo"
	ScopeUser   Scope = "user"
	ScopeSystem Scope = "system"
	ScopeAdmin  Scope = "admin"
)

type SelectionMode string

const (
	SelectionModeAuto     SelectionMode = "auto"
	SelectionModeExplicit SelectionMode = "explicit"
	SelectionModeOff      SelectionMode = "off"
)

func NormalizeSelectionMode(raw string) SelectionMode {
	switch SelectionMode(raw) {
	case SelectionModeExplicit:
		return SelectionModeExplicit
	case SelectionModeOff:
		return SelectionModeOff
	default:
		return SelectionModeAuto
	}
}

type SkillInterface struct {
	DisplayName      string `json:"display_name,omitempty"`
	ShortDescription string `json:"short_description,omitempty"`
	IconSmall        string `json:"icon_small,omitempty"`
	IconLarge        string `json:"icon_large,omitempty"`
	BrandColor       string `json:"brand_color,omitempty"`
	DefaultPrompt    string `json:"default_prompt,omitempty"`
}

type SkillToolDependency struct {
	Type        string `json:"type"`
	Value       string `json:"value"`
	Description string `json:"description,omitempty"`
	Transport   string `json:"transport,omitempty"`
	Command     string `json:"command,omitempty"`
	URL         string `json:"url,omitempty"`
}

type SkillDependencies struct {
	Tools []SkillToolDependency `json:"tools,omitempty"`
}

type SkillPolicy struct {
	AllowImplicitInvocation *bool `json:"allow_implicit_invocation,omitempty"`
}

type SkillMetadata struct {
	Name             string             `json:"name"`
	Description      string             `json:"description"`
	ShortDescription string             `json:"short_description,omitempty"`
	PathToSkillMD    string             `json:"path_to_skill_md"`
	Scope            Scope              `json:"scope"`
	Enabled          bool               `json:"enabled"`
	Interface        *SkillInterface    `json:"interface,omitempty"`
	Dependencies     *SkillDependencies `json:"dependencies,omitempty"`
	Policy           SkillPolicy        `json:"policy,omitempty"`
}

type SkillError struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

type LoadOutcome struct {
	Skills         []SkillMetadata     `json:"skills"`
	Errors         []SkillError        `json:"errors,omitempty"`
	DisabledPaths  map[string]struct{} `json:"-"`
	LoadedAt       time.Time           `json:"loaded_at"`
	ConfigFilePath string              `json:"config_file_path,omitempty"`
}

func (o LoadOutcome) EnabledSkills() []SkillMetadata {
	out := make([]SkillMetadata, 0, len(o.Skills))
	for _, skill := range o.Skills {
		if skill.Enabled {
			out = append(out, skill)
		}
	}
	return out
}

type BlockedSkill struct {
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

type SelectionResult struct {
	Mode     SelectionMode   `json:"mode"`
	Selected []SkillMetadata `json:"selected"`
	Blocked  []BlockedSkill  `json:"blocked,omitempty"`
}

type InstallRequest struct {
	Name         string   `json:"name,omitempty"`
	Repo         string   `json:"repo,omitempty"`
	URL          string   `json:"url,omitempty"`
	Paths        []string `json:"paths,omitempty"`
	Ref          string   `json:"ref,omitempty"`
	Dest         string   `json:"dest,omitempty"`
	Method       string   `json:"method,omitempty"`
	Experimental bool     `json:"experimental,omitempty"`
}

type CreateRequest struct {
	Name      string            `json:"name"`
	Path      string            `json:"path,omitempty"`
	Resources []string          `json:"resources,omitempty"`
	Examples  bool              `json:"examples,omitempty"`
	Interface map[string]string `json:"interface,omitempty"`
	Validate  bool              `json:"validate,omitempty"`
}

type OperationResult struct {
	Status  string   `json:"status"`
	Message string   `json:"message,omitempty"`
	Output  []string `json:"output,omitempty"`
}
