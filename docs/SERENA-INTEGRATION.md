# SERENA MCP INTEGRATION - BIOMETRICS

**Status:** ACTIVE - MANDATORY FOR ALL AGENTS
**Last Updated:** 2026-02-21
**Version:** 1.0.0

---

## OVERVIEW

Serena MCP is the central orchestration layer for BIOMETRICS. All agent activities MUST be coordinated through Serena.

**Purpose:**
- Project state management
- Cross-agent communication
- Task tracking and memory persistence
- Model collision prevention coordination
- "Sicher?" verification enforcement

---

## ARCHITECTURE

```
┌─────────────────────────────────────────────────────────────┐
│ BIOMETRICS ORCHESTRATION LAYER                              │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─────────────┐                                           │
│  │ Go Loop     │                                           │
│  │ Orchestrator│                                           │
│  │ (main.go)   │                                           │
│  └──────┬──────┘                                           │
│         │                                                   │
│         ▼                                                   │
│  ┌─────────────┐                                           │
│  │ Serena MCP  │ ◄─── Central coordination                 │
│  │ Server      │                                           │
│  └──────┬──────┘                                           │
│         │                                                   │
│    ┌────┴────┬────────────┬───────────┐                    │
│    │         │            │           │                    │
│    ▼         ▼            ▼           ▼                    │
│ ┌──────┐ ┌──────┐    ┌────────┐ ┌────────┐                │
│ │Sisy- │ │Atlas │    │Libr-   │ │Explore │                │
│ │phus  │ │      │    │arian   │ │        │                │
│ └──┬───┘ └──┬───┘    └───┬────┘ └───┬────┘                │
│    │        │            │          │                      │
│    └────────┴────────────┴──────────┘                      │
│              │                                              │
│              ▼                                              │
│    ┌─────────────────┐                                     │
│    │ Model Tracker   │                                     │
│    │ (Collision Prev)│                                     │
│    └─────────────────┘                                     │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

---

## INSTALLATION & SETUP

### 1. Install Serena

```bash
# Install via uv (Python package manager)
uv tool install serena

# Or run directly without installation
uvx --from git+https://github.com/oraios/serena serena start-mcp-server
```

### 2. Verify Installation

```bash
# Check if Serena is running
ps aux | grep serena | grep -v grep

# Expected output:
# simoneschulze 34192  0.0  0.1  ...  serena start-mcp-server
```

### 3. Configure OpenCode

Serena is already configured in `~/.config/opencode/opencode.json`:

```json
{
  "mcp": {
    "serena": {
      "type": "local",
      "command": [
        "uvx",
        "--from",
        "git+https://github.com/oraios/serena",
        "serena",
        "start-mcp-server"
      ],
      "enabled": true
    }
  }
}
```

---

## PROJECT STRUCTURE

BIOMETRICS uses the following Serena structure:

```
/Users/jeremy/dev/BIOMETRICS/
├── .serena/
│   ├── memories/
│   │   ├── agent-collaboration.md
│   │   ├── model-assignments.md
│   │   └── sicher-verification.md
│   ├── tasks/
│   │   ├── task-history.json
│   │   └── active-tasks.json
│   └── state.json
├── biometrics-cli/
│   └── cmd/
│       └── agent-loop/
│           └── main.go          # Go Orchestrator
└── docs/
    └── SERENA-INTEGRATION.md    # This file
```

---

## MANDATORY WORKFLOW

### BEFORE Starting Any Task

1. **Verify Serena is Running**
   ```bash
   ps aux | grep serena | grep -v grep
   ```

2. **Activate Project in Serena**
   ```bash
   # Automatic when using opencode CLI in BIOMETRICS directory
   cd /Users/jeremy/dev/BIOMETRICS
   opencode "task description"
   ```

3. **Check State Persistence**
   ```bash
   ls -la .serena/state.json
   ```

### AUTOMATIC CLEANUP (Go Orchestrator)

The Go orchestrator (`agent-loop`) automatically cleans up inactive sessions:

**Cleanup Criteria:**
- Status = `inactive`
- Last used > 7 days
- Project = empty or `default`

**Cleanup Process:**
1. Scan all Serena sessions
2. Identify sessions matching cleanup criteria
3. Archive inactive sessions (not delete - preserved in history)
4. Log cleanup actions for audit trail

**Manual Cleanup:**
```bash
# List all sessions
serena session list

# Archive specific session
serena session archive <session-id>

# View archived sessions
serena session list --archived
```

### DURING Task Execution

All agents MUST:
- Report status to Serena
- Check for model collisions via Serena
- Log task progress to Serena memories
- Request "Sicher?" verification when complete

### AFTER Task Completion

1. **Trigger "Sicher?" Verification**
   ```
   Sicher? Führe eine vollständige Selbstreflexion durch. Prüfe jede deiner Aussagen, verifiziere, ob ALLE Restriktionen des Initial-Prompts exakt eingehalten wurden. Stelle alles Fehlende fertig.
   ```

2. **Update Boulder.json**
   ```json
   {
     "active_plan": "plan-name",
     "started_at": "2026-02-21T...",
     "session_ids": ["ses_..."],
     "plan_name": "BIOMETRICS Omega Loop",
     "agent": "sisyphus"
   }
   ```

3. **Commit Changes**
   ```bash
   git add -A
   git commit -m "feat: description"
   git push origin main
   ```

---

## MODEL COLLISION PREVENTION

Serena coordinates model assignments to prevent collisions:

| Model | Category | Max Parallel | Assigned Agents |
|-------|----------|--------------|-----------------|
| `google/gemini-2.5-pro` | build, visual-engineering | 1 | sisyphus, oracle, build |
| `google/gemini-2.5-flash` | deep, think | 1 | atlas, deep, ultrabrain |
| `google/gemini-2.5-flash` | quick, explore | 1 | librarian, explore, quick |

**WICHTIG:** Gemini 3 Modelle (3.1-pro-preview, 3-flash-preview) sind NICHT verfuegbar via Standard API!

**ENFORCEMENT:**
- Serena tracks active model usage
- Go Orchestrator checks `ModelTracker` before assigning
- Collision = Task queued until model free

---

## TROUBLESHOOTING

### Serena Not Starting

```bash
# Clear uv cache
uv cache clean

# Retry
uvx --from git+https://github.com/oraios/serena serena start-mcp-server
```

### Project Not Activated

```bash
# Delete state and restart
rm .serena/state.json
opencode "test"
```

### MCP Connection Lost

```bash
# Restart OpenCode CLI
# Close terminal and reopen
opencode --version
```

### Multiple Serena Processes

```bash
# Kill all Serena processes
pkill -f serena

# Start fresh
uvx --from git+https://github.com/oraios/serena serena start-mcp-server
```

---

## API REFERENCE

### Serena MCP Tools

Serena provides the following tools to agents:

| Tool | Purpose | Example |
|------|---------|---------|
| `serena_activate_project` | Activate project context | `serena_activate_project("BIOMETRICS")` |
| `serena_log_task` | Log task execution | `serena_log_task({id, status})` |
| `serena_check_model` | Check model availability | `serena_check_model("gemini-3.1-pro")` |
| `serena_save_memory` | Persist context | `serena_save_memory("key", "value")` |
| `serena_get_state` | Retrieve project state | `serena_get_state()` |

---

## INTEGRATION WITH GO ORCHESTRATOR

The Go orchestrator (`biometrics-cli/cmd/agent-loop/main.go`) integrates with Serena:

```go
// Model collision prevention
tracker := NewModelTracker()
model := getModelForAgent(agent)
if err := tracker.Acquire(model); err != nil {
    // Collision detected - wait
    time.Sleep(5 * time.Second)
}

// Trigger Sicher verification
runSicherCheck(agent)
```

**Serena coordinates:**
- Model assignment tracking
- Task queue management
- State persistence
- Cross-agent communication

---

## BEST PRACTICES

1. **ALWAYS** verify Serena is running before starting work
2. **NEVER** bypass Serena for task coordination
3. **ALWAYS** trigger "Sicher?" verification after task completion
4. **NEVER** allow model collisions (max 1 agent per model)
5. **ALWAYS** commit changes after task completion

---

## REFERENCES

- **Global Mandate:** `/Users/jeremy/.config/opencode/AGENTS.md` (MANDATE 0.11)
- **Go Orchestrator:** `/Users/jeremy/dev/BIOMETRICS/biometrics-cli/cmd/agent-loop/main.go`
- **Boulder.json:** `/Users/jeremy/.sisyphus/boulder.json`
- **Serena GitHub:** https://github.com/oraios/serena

---

**END OF SERENA INTEGRATION DOCUMENTATION**
