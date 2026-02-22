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
