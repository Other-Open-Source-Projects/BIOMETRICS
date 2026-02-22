# OPENCODE Configuration Guide - Google Gemini Setup

**Project:** BIOMETRICS  
**Last Updated:** 2026-02-22  
**Status:** Active - All Gemini Models Working

---

## Das Problem (February 2026)

Google Gemini funktionierte NICHT in OpenCode. Alle Versuche schlugen mit `API key not valid` fehl.

### Root Causes:

1. **Leaked API Key**: Der alte Key `AIzaSyAVWKxhWCT64Z0VxxmskWzPNTwfWVecC_U` wurde von Google wegen "leaked" permanent gesperrt
2. **Corrupted auth.json**: Der "google" Eintrag in `~/.local/share/opencode/auth.json` enthielt stattdessen deutschen Text: "ie custom_agents.json mit den neuen Modellen aktualisieren?"
3. **Duplicate JSON Keys**: In `opencode.json` gab es doppelte `gemini-3.1-pro-preview` und `gemini-3-flash-preview` Einträge
4. **Falscher Speicherort**: Der API Key wurde NUR in ~/.zshrc exportiert, aber NICHT in auth.json

---

## Die Lösung - Schritt für Schritt

### Schritt 1: Neuen API Key generieren

1. Gehe zu: https://aistudio.google.com/app/apikey
2. Login mit Google Account
3. "Create API Key" klicken
4. Neuen Key kopieren

**Aktueller funktionierender Key:**
```
[DEIN_GOOGLE_API_KEY_HIER]
```

### Schritt 2: auth.json reparieren

**WO?** `~/.local/share/opencode/auth.json`

Der Google-Eintrag MUSS so aussehen:
```json
{
  "google": {
    "type": "api",
    "key": "[DEIN_GOOGLE_API_KEY_HIER]"
  }
}
```

**Kommando zum Prüfen:**
```bash
cat ~/.local/share/opencode/auth.json | grep -A3 google
```

### Schritt 3: opencode.json reparieren

**WO?** `~/.config/opencode/opencode.json`

**Problem:** Doppelte JSON-Keys entfernen
- Doppelte `gemini-3.1-pro-preview` Einträge
- Doppelte `gemini-3-flash-preview` Einträge

**Kommando zum Validieren:**
```bash
cat ~/.config/opencode/opencode.json | python3 -m json.tool > /dev/null && echo "✅ JSON valid"
```

### Schritt 4: Testen

```bash
# Test Gemini 2.5 Flash
opencode run "Say OK" --model google/gemini-2.5-flash

# Test Gemini 2.5 Pro
opencode run "Say OK" --model google/gemini-2.5-pro

# Test Gemini 3.1 Pro Preview
opencode run "Say OK" --model google/gemini-3.1-pro-preview

# Test Gemini 3 Pro Preview
opencode run "Say OK" --model google/gemini-3-pro-preview
```

---

## Verifizierte Modelle (Februar 2026)

| Model | Test Status | Kommando |
|-------|------------|----------|
| `google/gemini-2.5-flash` | ✅ OK | `opencode run "test" --model google/gemini-2.5-flash` |
| `google/gemini-2.5-pro` | ✅ OK | `opencode run "test" --model google/gemini-2.5-pro` |
| `google/gemini-3.1-pro-preview` | ✅ OK | `opencode run "test" --model google/gemini-3.1-pro-preview` |
| `google/gemini-3.1-pro-preview-customtools` | ✅ OK | `opencode run "test" --model google/gemini-3.1-pro-preview-customtools` |
| `google/gemini-3-pro-preview` | ✅ OK | `opencode run "test" --model google/gemini-3-pro-preview` |
| `google/gemini-3-flash-preview` | ✅ OK | `opencode run "test" --model google/gemini-3-flash-preview` |

---

## Konfiguration Files

### 1. auth.json (API Keys) - KRITISCH!

**Location:** `~/.local/share/opencode/auth.json`

Dies ist WO der Google API Key gespeichert werden muss!

```json
{
  "google": {
    "type": "api",
    "key": "[DEIN_GOOGLE_API_KEY_HIER]"
  }
}
```

**WICHTIG:** 
- Der Key MUSS in dieser Datei sein, NICHT nur in ~/.zshrc
- Die Datei muss valides JSON sein
- KEINE deutschen Texte oder Kommentare

### 2. opencode.json (Modell Konfiguration)

**Location:** `~/.config/opencode/opencode.json`

Enthält die Provider Konfiguration:

```json
{
  "provider": {
    "google": {
      "npm": "@ai-sdk/google",
      "models": {
        "gemini-2.5-flash": {
          "id": "gemini-2.5-flash",
          "name": "Gemini 2.5 Flash",
          "limit": { "context": 1048576, "output": 65536 }
        },
        "gemini-2.5-pro": {
          "id": "gemini-2.5-pro",
          "name": "Gemini 2.5 Pro",
          "limit": { "context": 1048576, "output": 65536 }
        },
        "gemini-3.1-pro-preview": {
          "id": "gemini-3.1-pro-preview",
          "name": "Gemini 3.1 Pro Preview",
          "limit": { "context": 2097152, "output": 65536 }
        },
        "gemini-3.1-pro-preview-customtools": {
          "id": "gemini-3.1-pro-preview",
          "name": "Gemini 3.1 Pro Preview (Custom Tools)",
          "limit": { "context": 2097152, "output": 65536 }
        },
        "gemini-3-pro-preview": {
          "id": "gemini-3-pro-preview",
          "name": "Gemini 3 Pro Preview",
          "limit": { "context": 2097152, "output": 65536 }
        },
        "gemini-3-flash-preview": {
          "id": "gemini-3-flash-preview",
          "name": "Gemini 3 Flash Preview",
          "limit": { "context": 1048576, "output": 65536 }
        }
      }
    }
  }
}
```

---

## API Endpoint Information

**WICHTIG:** Alle Gemini Modelle nutzen den gleichen v1beta Endpoint:

```
https://generativelanguage.googleapis.com/v1beta
```

Dies ist bereits in der opencode.json konfiguriert (durch das @ai-sdk/google Package).

### Modell-ID Unterschiede:

| Modellfamilie | API Modell ID |
|--------------|---------------|
| Gemini 2.5 | `gemini-2.5-flash`, `gemini-2.5-pro` |
| Gemini 3.0 | `gemini-3-flash-preview`, `gemini-3-pro-preview` |
| Gemini 3.1 | `gemini-3.1-pro-preview` |

**Wichtig:** Es gibt KEINE unterschiedlichen API-Aufrufe für verschiedene Modellfamilien. Alle nutzen den gleichen Endpoint!

---

## Model Configuration

### Default Modelle

| Setting | Value |
|---------|-------|
| Default Model | `google/gemini-3.1-pro-preview-customtools` |
| Small Model | `google/gemini-3-flash-preview` |
| Default Agent | `sisyphus` |

### Agent Configurations

| Agent | Model | Purpose |
|-------|-------|---------|
| **sisyphus** | `google/gemini-3.1-pro-preview-customtools` | Main Coder |
| **atlas** | `google/gemini-3.1-pro-preview` | Heavy Lifting |
| **prometheus** | `google/gemini-3.1-pro-preview` | Strategic Planning |
| **oracle** | `google/gemini-3.1-pro-preview-customtools` | Architecture Review |
| **quick** | `google/gemini-2.5-flash` | Quick Tasks |
| **explore** | `google/gemini-2.5-flash` | Code Discovery |
| **librarian** | `google/gemini-2.5-flash` | Documentation |

---

## Google Gemini Modelle

| Model ID | Name | Context | Output | Use Case |
|----------|------|---------|--------|----------|
| `gemini-2.5-flash` | **Gemini 2.5 Flash** | 1M | 64K | Fast, Cost-effective |
| `gemini-2.5-pro` | **Gemini 2.5 Pro** | 1M | 64K | Best Value Brain |
| `gemini-3.1-pro-preview` | Gemini 3.1 Pro | 2M | 64K | Advanced Reasoning |
| `gemini-3.1-pro-preview-customtools` | Gemini 3.1 Pro (Tools) | 2M | 64K | Code, Tools, Agentic |
| `gemini-3-pro-preview` | Gemini 3 Pro | 2M | 64K | Complex Multi-step |
| `gemini-3-flash-preview` | Gemini 3 Flash | 1M | 64K | Fast, Quick Tasks |
| `antigravity-gemini-3-flash` | Gemini 3 Flash (OAuth) | 1M | 64K | With OAuth |
| `antigravity-gemini-3-pro` | Gemini 3 Pro (OAuth) | 1M | 64K | With OAuth |

---

## Test Commands

### Schnelltest (alle Modelle):

```bash
# Test Gemini 2.5 Flash
opencode run "Say OK" --model google/gemini-2.5-flash

# Test Gemini 2.5 Pro
opencode run "Say OK" --model google/gemini-2.5-pro

# Test Gemini 3.1 Pro Preview
opencode run "Say OK" --model google/gemini-3.1-pro-preview

# Test Gemini 3 Pro Preview
opencode run "Say OK" --model google/gemini-3-pro-preview
```

### Modelle auflisten:

```bash
opencode models | grep google/
```

---

## Troubleshooting

### Fehler: "API key not valid"

**Ursache:** Der Key ist nicht in auth.json oder der Key ist gesperrt.

**Lösung:**
1. Neuen Key generieren: https://aistudio.google.com/app/apikey
2. In auth.json eintragen (siehe Schritt 2 oben)
3. Testen: `opencode run "test" --model google/gemini-2.5-flash`

### Fehler: "Failed to change directory"

**Ursache:** Falsches Kommando Format

**Lösung:**
```bash
opencode run "Nachricht" --model google/gemini-2.5-flash
```

### Fehler: "Unrecognized key: env"

**Ursache:** API key in opencode.json

**Lösung:** Key entfernen, nur in auth.json speichern

### Fehler: "Configuration is invalid"

**Ursache:** JSON Fehler in opencode.json

**Lösung:**
```bash
cat ~/.config/opencode/opencode.json | python3 -m json.tool
```

### Schritt-für-Schritt Fehlerbehebung:

**Schritt 1: Prüfe ob Google in auth.json vorhanden**
```bash
cat ~/.local/share/opencode/auth.json | grep -A2 google
```

Sollte anzeigen:
```json
"google": {
  "type": "api",
  "key": "[DEIN_GOOGLE_API_KEY_HIER]"
}
```

**Schritt 2: JSON validieren**
```bash
# opencode.json
cat ~/.config/opencode/opencode.json | python3 -m json.tool > /dev/null && echo "✅ JSON valid"

# auth.json
cat ~/.local/share/opencode/auth.json | python3 -m json.tool > /dev/null && echo "✅ auth.json valid"
```

**Schritt 3: Modell testen**
```bash
opencode run "Say OK" --model google/gemini-2.5-flash
```

Wenn "OK" zurückkommt -> Alles funktioniert!

---

## Alternative Providers

| Provider | Model | Context | Output |
|----------|-------|---------|--------|
| **NVIDIA NIM** | `qwen-3.5-397b` | 262K | 32K |
| **OpenCode ZEN** | `zen/big-pickle` | 200K | 128K |
| **XiaoMi** | `mimo-v2-flash` | 1M | 64K |
| **Streamlake** | `kat-coder-pro-v1` | 2M | 128K |

---

## Wichtige Hinweise

- **NEVER** API Keys in opencode.json speichern
- **ALWAYS** Keys in auth.json eintragen
- **REMEMBER** Beide Dateien müssen valides JSON sein
- Google Gemini nutzt v1beta Endpoint (bereits konfiguriert)
- Alle Gemini Modelle nutzen die gleiche API - keine unterschiedlichen Aufrufe nötig

---

## MCP Servers

### Local

| Server | Status |
|--------|--------|
| serena | ✅ Enabled |
| tavily | ✅ Enabled |
| canva | ✅ Enabled |
| context7 | ✅ Enabled |
| skyvern | ✅ Enabled |
| chrome-devtools | ✅ Enabled |
| singularity | ✅ Enabled |

### Remote

| Server | URL |
|--------|-----|
| linear | https://mcp.linear.app/sse |
| sin_social | https://sin-social.delqhi.com |
| sin_deep_research | https://sin-research.delqhi.com |
| sin_video_gen | https://sin-video.delqhi.com |

---

## Zusammenfassung - Checkliste

Falls du das nochmal machen musst:

- [ ] Neuen API Key bei Google AI Studio generieren
- [ ] ~/.local/share/opencode/auth.json prüfen (google Eintrag mit type: "api" und key: "API_KEY")
- [ ] Doppelte Keys in ~/.config/opencode/opencode.json entfernen
- [ ] JSON Validieren: `python3 -m json.tool`
- [ ] Testen: `opencode run "OK" --model google/gemini-2.5-flash`

---

**Letzte Prüfung:** 2026-02-22 - Alle Modelle funktionieren!

## KORREKTE KONFIGURATION - Das Wissen aus der Praxis

### Die WICHTIGSTE Lektion: Zwei verschiedene Syntaxen!

**KRITISCH:** OpenCode und oh-my-opencode nutzen UNTERSCHIEDLICHE Syntax fuer Umgebungsvariablen!

| Wo | Syntax | Beispiel |
|---|--------|---------|
| **opencode.json** (Basis) | `{env:VARIABLE}` | `{env:GEMINI_API_KEY}` |
| **oh-my-opencode.jsonc** | `${VARIABLE}` | `${GEMINI_API_KEY}` |

**ERROR:** Wenn du `${VAR}` in opencode.json verwendest, sendet es den Literal-String an die API!
**ERROR:** Wenn du `{env:VAR}` in oh-my-opencode.jsonc verwendest, funktioniert es nicht!

---

### Konfigurations-Hierarchie (Priority: Low -> High)

OpenCode laedt Konfigurationen in dieser Reihenfolge - hoehere ueberschreibt niedrigere:

| Ebene | Quelle | Zweck |
|-------|--------|-------|
| **1. Remote** | `.well-known/opencode` | Globale Org-Standards, MCP-Server |
| **2. Global** | `~/.config/opencode/opencode.json` | User-Praeferenzen, Fallback-Keys |
| **3. Custom** | `OPENCODE_CONFIG` env Variable | Profil-Overrides, CI/CD |
| **4. Projekt** | `./opencode.json` | Projekt-spezifische Agenten |
| **5. Inline** | `OPENCODE_CONFIG_CONTENT` env | Temporaere Runtime-Overrides |

---

### auth.json - Der FALLBACK fuer API Keys

**Location:** `~/.local/share/opencode/auth.json`

Dies ist das FALLBACK-System wenn keine env-Variable gesetzt ist:

```
Pruefung bei Start:
1. opencode.json -> {env:VAR} Token
2. .env Dateien + OS env vars
3. FALLBACK -> auth.json
```

**WICHTIG:** Alle API Keys MUESSEN in auth.json sein, NICHT in opencode.json!

---

### .env Auto-Loading

OpenCode laedt .env Dateien AUTOMATISCH:

| Ebene | Location | Override? |
|-------|----------|----------|
| **Global** | `~/.config/opencode/.env` | Nein (Fallback) |
| **Lokal** | `./.env` (neben opencode.json) | Ja (ueberschreibt Global) |

**WICHTIG:** .env wird GELADEN BEVOR die {env:VAR} Substitution passiert!

---

### OPENCODE_CONFIG_DIR - Isolierte Profile

Du kannst das Konfigurations-Verzeichnis wechseln:

```bash
# Standard
~/.config/opencode/

# Mit OPENCODE_CONFIG_DIR
export OPENCODE_CONFIG_DIR=~/.config/opencode/profiles/work
```

**Effekt:** Alle Konfigurationen werden aus dem neuen Verzeichnis geladen.

---

### Provider Umgebungsvariablen

| Provider | Variable | Modelle |
|----------|----------|--------|
| **Anthropic** | `ANTHROPIC_API_KEY` | Claude 3.5, Opus |
| **OpenAI** | `OPENAI_API_KEY` | GPT-4, GPT-5 |
| **Google Gemini** | `GEMINI_API_KEY` | Gemini 2.5, 3.0, 3.1 |
| **VertexAI** | `VERTEXAI_PROJECT`, `VERTEXAI_LOCATION` | Enterprise Gemini |
| **DeepSeek** | `DEEPSEEK_API_KEY` | DeepSeek Coder |
| **NVIDIA** | `NVIDIA_API_KEY` | Qwen, Llama, Mistral |
| **OpenCode Zen** | `OPENCODE_API_KEY` | Big Pickle, Uncensored |
| **MiniMax** | `MINIMAX_API_KEY` | M2.5 |
| **Kimi** | `KIMI_API_KEY` | K2.5 |

---

### Antigravity OAuth - Google ohne API Key

**Was ist das?** Plugin das Google OAuth statt API Keys nutzt.

**Setup:**
```json
{
  "plugin": ["opencode-antigravity-auth@latest"]
}
```

**Dann:**
```bash
opencode auth login
```

**Dateien:**
- `~/.config/opencode/antigravity-accounts.json` - OAuth Tokens
- `~/.config/opencode/antigravity.json` - Plugin Config

**Multi-Account:** Das Plugin kann automatisch zwischen Google-Accounts rotieren wenn Rate-Limit erreicht ist.

---

### Sicherheit - env-protection

**Problem:** Agenten koennten .env Dateien lesen und API Keys exfiltrieren!

**Loesung:** env-protection.js Plugin

```bash
# Plugin in .opencode/plugins/ laden
```

**Was es macht:**
- Blockiert Lese-Zugriff auf `.env` Dateien
- Erlaubt `.env.example` und `.env.template` (keine echten Keys)

**Permissions in opencode.json:**
```json
{
  "permission": {
    "edit": "deny",
    "bash": "ask",
    "webfetch": "allow",
    "mcp_*": "ask"
  }
}
```

---

### JSONC in oh-my-opencode

oh-my-opencode unterstuetzt JSONC (JSON with Comments):

| Feature | Syntax |
|---------|--------|
| Zeilenkommentar | `// Das ist ein Kommentar` |
| Blockkommentar | `/* Mehrzeilig */` |
| Trailing Comma | `{ "key": "value", }` |

**Priority:** .jsonc > .json (wenn beide existieren)

---

## Zusammenfassung - KORREKTE Setup-Schritte

```bash
# 1. API Key generieren (z.B. Google AI Studio)
# 2. NUR in auth.json eintragen (NICHT in opencode.json!)

# Pruefen:
cat ~/.local/share/opencode/auth.json | python3 -m json.tool

# 3. Modelle testen:
opencode run "OK" --model google/gemini-2.5-flash

# 4. Oder Antigravity OAuth nutzen:
opencode auth login  # Browser oeffnet sich
```

---

**Letzte Aktualisierung:** 2026-02-22 - KORREKTE Konfiguration dokumentiert