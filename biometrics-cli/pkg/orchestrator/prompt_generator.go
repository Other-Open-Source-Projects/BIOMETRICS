package orchestrator

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// MassivePromptGenerator creates context-rich prompts for sub-agents
type MassivePromptGenerator struct {
	projectRoot   string
	orchestrator  *Orchestrator
	requiredFiles []string
	optionalFiles []string
	contextCache  map[string]string
}

// NewMassivePromptGenerator creates a new prompt generator
func NewMassivePromptGenerator(orchestrator *Orchestrator) *MassivePromptGenerator {
	return &MassivePromptGenerator{
		projectRoot:  orchestrator.config.ProjectRoot,
		orchestrator: orchestrator,
		contextCache: make(map[string]string),
		requiredFiles: []string{
			"AGENTS-PLAN.md",
			"ARCHITECTURE.md",
			"README.md",
			"CHANGELOG.md",
			"docs/ORCHESTRATOR-MANDATE.md",
			"docs/agents/AGENT-MODEL-MAPPING.md",
		},
		optionalFiles: []string{
			"package.json",
			"go.mod",
			"requirements.txt",
			".env.example",
			"oh-my-opencode.json",
			"MEETING.md",
			"ONBOARDING.md",
		},
	}
}

// GenerateMassivePrompt creates a comprehensive prompt with ALL context
func (g *MassivePromptGenerator) GenerateMassivePrompt(agentName, category, task string) (string, error) {
	var sb strings.Builder

	// Header with agent identity
	sb.WriteString(g.generateHeader(agentName, category))

	// Critical rules (NEVER break)
	sb.WriteString(g.generateCriticalRules())

	// Task description
	sb.WriteString(g.generateTaskDescription(task))

	// Required files to read FIRST
	sb.WriteString(g.generateRequiredFilesSection())

	// Project context
	sb.WriteString(g.generateProjectContext())

	// Other parallel agents
	sb.WriteString(g.generateParallelAgentsInfo())

	// Acceptance criteria
	sb.WriteString(g.generateAcceptanceCriteria(task))

	// Output format requirements
	sb.WriteString(g.generateOutputFormat())

	// Warnings and common mistakes
	sb.WriteString(g.generateWarnings())

	prompt := sb.String()

	// Save prompt to file for audit
	g.savePrompt(agentName, prompt)

	return prompt, nil
}

func (g *MassivePromptGenerator) generateHeader(agentName, category string) string {
	model := g.orchestrator.getModelForCategory(category)

	return fmt.Sprintf(`# 🎯 ORCHESTRATOR → AGENT %s (%s)

**Generated:** %s
**Session:** Will be assigned on execution
**Model:** %s
**Category:** %s
**Orchestrator:** Active monitoring enabled

---

`, agentName, category, time.Now().Format("2006-01-02 15:04:05"), model, category)
}

func (g *MassivePromptGenerator) generateCriticalRules() string {
	return `## 🚨 KRITISCHE REGELN (NIEMALS BRECHEN!)

### ❌ ABSOLUT VERBOTEN:
1. **NIEMALS 2 Agents mit gleichem Modell parallel!**
   - Qwen 3.5: MAX 1 Agent
   - Kimi K2.5: MAX 1 Agent  
   - MiniMax M2.5: MAX 1 Agent
   - **MAXIMAL 3 Agents parallel (alle verschiedene Modelle!)**

2. **NIEMALS Dateien erstellen ohne zu lesen!**
   - IMMER zuerst glob() oder ls verwenden
   - IMMER existierende Dateien KOMPLETT lesen (bis letzte Zeile!)
   - NIEMALS Duplikate erstellen!

3. **NIEMALS "fertig" sagen ohne Evidenz!**
   - IMMER Dateiinhalt zeigen
   - IMMER Tests durchführen
   - IMMER "Sicher?"-Check bestehen
   - IMMER Git Commit machen

4. **NIEMALS User-Onboarding überspringen!**
   - IMMER mit User zusammen Config erstellen
   - IMMER API Keys erklären
   - IMMER Tests gemeinsam durchführen

### ✅ ABSOLUT PFLICHT:
1. **IMMER Serena MCP nutzen** für Projekt-Kontext
2. **IMMER ALLE Dateien lesen** bevor du arbeitest
3. **IMMER bestehende erweitern** statt neu erstellen
4. **IMMER "Sicher?"-Check** nach jeder Completion
5. **IMMER Git Commit** nach jeder Änderung

---

`
}

func (g *MassivePromptGenerator) generateTaskDescription(task string) string {
	return fmt.Sprintf(`## 🎯 DEINE AUFGABE

%s

**WICHTIG:** Diese Aufgabe ist Teil des 20-Task Infinity Loop.
Nach deiner Completion werden sofort 5 neue Tasks generiert.

---

`, task)
}

func (g *MassivePromptGenerator) generateRequiredFilesSection() string {
	var sb strings.Builder
	sb.WriteString("## 📖 DATEIEN DIE DU ZUERST LESEN MUSST (PFLICHT!)\n\n")
	sb.WriteString("### Zwingend erforderliche Dateien:\n\n")

	for i, file := range g.requiredFiles {
		fullPath := filepath.Join(g.projectRoot, file)
		content := g.readFileContent(fullPath)
		if content != "" {
			sb.WriteString(fmt.Sprintf("#### %d. %s\n", i+1, file))
			sb.WriteString("```markdown\n")
			sb.WriteString(content)
			sb.WriteString("\n```\n\n")
		}
	}

	sb.WriteString("### Optionale Dateien (falls relevant):\n\n")
	for i, file := range g.optionalFiles {
		fullPath := filepath.Join(g.projectRoot, file)
		content := g.readFileContent(fullPath)
		if content != "" {
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, file))
		}
	}

	sb.WriteString("\n**WICHTIG:** Du MUSST ALLE oben genannten Dateien gelesen haben bevor du beginnst!\n\n---\n\n")

	return sb.String()
}

func (g *MassivePromptGenerator) generateProjectContext() string {
	var sb strings.Builder
	sb.WriteString("## 🏗️ PROJEKT-KONTEXT\n\n")

	// Architecture overview
	archPath := filepath.Join(g.projectRoot, "ARCHITECTURE.md")
	if content := g.readFileContent(archPath); content != "" {
		sb.WriteString("### Architektur-Übersicht\n\n")
		sb.WriteString(content)
		sb.WriteString("\n\n")
	}

	// Recent changes
	changelogPath := filepath.Join(g.projectRoot, "CHANGELOG.md")
	if content := g.readFileContent(changelogPath); content != "" {
		sb.WriteString("### Letzte Änderungen\n\n")
		sb.WriteString(content)
		sb.WriteString("\n\n")
	}

	sb.WriteString("---\n\n")
	return sb.String()
}

func (g *MassivePromptGenerator) generateParallelAgentsInfo() string {
	status := g.orchestrator.GetStatus()

	var sb strings.Builder
	sb.WriteString("## 📊 ANDERE AGENTS (PARALLEL AKTIV)\n\n")
	sb.WriteString(fmt.Sprintf("**Aktive Sessions:** %v\n", status["active_sessions"]))
	sb.WriteString(fmt.Sprintf("**Abgeschlossene Sessions:** %v\n", status["completed_sessions"]))
	sb.WriteString("\n**Modell-Auslastung:**\n")

	for model, usage := range status["model_usage"].(map[string]int) {
		limit := g.orchestrator.config.ModelLimits[model]
		sb.WriteString(fmt.Sprintf("- %s: %d/%d verwendet\n", model, usage, limit))
	}

	sb.WriteString("\n**WICHTIG:** Dein Code muss konsistent mit der Arbeit anderer Agents sein!\n\n---\n\n")

	return sb.String()
}

func (g *MassivePromptGenerator) generateAcceptanceCriteria(task string) string {
	return `## ✅ ACCEPTANCE CRITERIA

Deine Aufgabe ist NUR dann abgeschlossen wenn ALLE folgenden Kriterien erfüllt sind:

- [ ] Alle erforderlichen Dateien gelesen (siehe oben)
- [ ] Aufgabe vollständig implementiert
- [ ] Tests geschrieben und bestanden
- [ ] Git Commit mit konventioneller Commit-Message
- [ ] "Sicher?"-Check bestanden (alle 6 Checks)
- [ ] Dokumentation aktualisiert (AGENTS-PLAN.md, MEETING.md)
- [ ] Keine Duplikate erstellt
- [ ] Keine LSP/vet Fehler
- [ ] Konsistent mit existierendem Code

**OHNE ALLE HAKEN = NICHT FERTIG!**

---

`
}

func (g *MassivePromptGenerator) generateOutputFormat() string {
	return `## 🚀 OUTPUT FORMAT

Deine Antwort MUSS folgende Struktur haben:

### 1. Gelesene Dateien
Liste ALLE Dateien auf die du gelesen hast mit Zeilenzahlen:
- AGENTS-PLAN.md (303 Zeilen) - KOMPLETT GELESEN ✅
- ARCHITECTURE.md (XXX Zeilen) - KOMPLETT GELESEN ✅
- ...

### 2. Status der Aufgabe
Beschreibe detailliert was du gemacht hast:
- Welche Dateien erstellt/bearbeitet
- Welche Tests geschrieben
- Welche Commits gemacht

### 3. "Sicher?"-Check
Bestätige dass der Sicher?-Check bestanden wurde:
- Files Created/Modified: ✅
- Tests Pass: ✅
- Git Commit: ✅
- No Duplicates: ✅
- LSP Diagnostics Clean: ✅
- Agent Honesty: ✅

### 4. Nächste 3 Schritte
Was sind die nächsten Tasks im Infinity Loop?
1. [Nächster Task]
2. [Übernächster Task]
3. [Dritter Task]

---

`
}

func (g *MassivePromptGenerator) generateWarnings() string {
	return `## ⚠️ HÄUFIGE FEHLER + LÖSUNGEN

### FEHLER 1: Duplikate erstellt
**Lösung:** STOPP! Datei existiert bereits - erst lesen, dann erweitern!

### FEHLER 2: Falsches Modell genutzt
**Lösung:** Immer explizites Modell angeben in task()!

### FEHLER 3: "Fertig" ohne Evidenz
**Lösung:** "Sicher?"-Check - zeige alle Dateien, Tests, Commits!

### FEHLER 4: Dateien nicht gelesen
**Lösung:** IMMER zuerst lesen - ohne Ausnahme!

---

## 🔥 INFINITY LOOP REMINDER

**Nach deiner Completion:**
1. Orchestrator führt Sicher?-Check durch
2. Bei Bestehen: +5 neue Tasks werden generiert
3. Git Commit wird gemacht
4. Nächster Agent startet sofort

**"Ein Task endet, fünf neue beginnen"**
**"Kein Warten, nur Arbeiten"**
**"Kein Fertig, nur Weiter"**

---

**ENDE DES PROMPTS - JETZT ARBEITEN!**
`
}

func (g *MassivePromptGenerator) readFileContent(path string) string {
	// Check cache first
	if content, ok := g.contextCache[path]; ok {
		return content
	}

	// Read file
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return "" // File doesn't exist or can't be read
	}

	content := string(data)

	// Cache for future use
	g.contextCache[path] = content

	return content
}

func (g *MassivePromptGenerator) savePrompt(agentName, prompt string) error {
	dir := filepath.Join(g.projectRoot, ".sisyphus", "prompts")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	filename := filepath.Join(dir, fmt.Sprintf("%s_%s.md", agentName, time.Now().Format("20060102_150405")))
	return ioutil.WriteFile(filename, []byte(prompt), 0644)
}

// RefreshContext reloads all context files
func (g *MassivePromptGenerator) RefreshContext() error {
	g.contextCache = make(map[string]string)

	// Pre-load required files
	for _, file := range g.requiredFiles {
		fullPath := filepath.Join(g.projectRoot, file)
		g.readFileContent(fullPath)
	}

	return nil
}

// GetContextStats returns statistics about loaded context
func (g *MassivePromptGenerator) GetContextStats() map[string]interface{} {
	totalLines := 0
	totalFiles := len(g.contextCache)

	for _, content := range g.contextCache {
		lines := strings.Count(content, "\n") + 1
		totalLines += lines
	}

	return map[string]interface{}{
		"files_loaded": totalFiles,
		"total_lines":  totalLines,
		"cache_size":   len(g.contextCache),
	}
}
