#!/usr/bin/env bash
set -euo pipefail

# 🚨 INTERACTIVE OPENCODE.JSON GENERATOR
# This script interactively creates ~/.config/opencode/opencode.json

echo "START: Interactive OpenCode.json Generator"
echo "======================================"
echo ""

# Step 1: Get NVIDIA API Key
echo "STEP: Schritt 1: NVIDIA API Key"
echo "----------------------------"
read -r -p "Hast du bereits einen NVIDIA API Key? (y/n): " has_key

if [ "$has_key" != "y" ]; then
    echo ""
    echo "OPEN: Öffne https://build.nvidia.com/ in deinem Browser"
    echo "   1. Einloggen oder Account erstellen"
    echo "   2. Auf 'API Keys' klicken"
    echo "   3. 'Create New API Key' klicken"
    echo "   4. Key kopieren (beginnt mit nvapi-...)"
    echo ""
    read -r -p "Drück Enter wenn du den Key kopiert hast: "
fi

echo ""
read -r -s -p "Füge deinen NVIDIA API Key ein (wird nicht angezeigt): " nvidia_key
echo ""

# Validate key format
if [[ ! "$nvidia_key" =~ ^nvapi- ]]; then
    echo "ERROR: Fehler: Key muss mit 'nvapi-' beginnen!"
    exit 1
fi

echo "SUCCESS: Key validiert!"
echo ""

echo "PLUGIN: Schritt 1b: Optionale Plugins"
echo "------------------------------------"
read -r -p "Optional: 'oh-my-opencode' Plugin aktivieren? (y/N): " enable_omoc
enable_omoc="${enable_omoc:-N}"
enable_omoc_lower="$(printf '%s' "${enable_omoc}" | tr '[:upper:]' '[:lower:]')"

plugins_json='[]'
if [[ "${enable_omoc_lower}" == "y" || "${enable_omoc_lower}" == "yes" ]]; then
    plugins_json='["oh-my-opencode"]'
    echo "INFO: oh-my-opencode ist aktiviert (optional)."
else
    echo "INFO: oh-my-opencode ist deaktiviert (Standard)."
fi

echo ""

# Step 2: Create directory
echo "DIR: Schritt 2: Verzeichnis erstellen"
echo "------------------------------------"
mkdir -p ~/.config/opencode
echo "SUCCESS: Verzeichnis erstellt: ~/.config/opencode"
echo ""

# Step 3: Generate opencode.json
echo "CONFIG:  Schritt 3: opencode.json generieren"
echo "----------------------------------------"

cat > ~/.config/opencode/opencode.json << EOF
{
  "\$schema": "https://opencode.ai/config.json",
  "model": "nvidia-nim/qwen-3.5-397b",
  "default_agent": "sisyphus",
  "plugin": ${plugins_json},
  "provider": {
    "nvidia-nim": {
      "npm": "@ai-sdk/openai-compatible",
      "name": "NVIDIA NIM (Qwen 3.5)",
      "options": {
        "baseURL": "https://integrate.api.nvidia.com/v1"
      },
      "models": {
        "qwen-3.5-397b": {
          "name": "Qwen 3.5 397B (NVIDIA NIM)",
          "id": "qwen/qwen3.5-397b-a17b",
          "limit": {
            "context": 262144,
            "output": 32768
          }
        }
      }
    }
  }
}
EOF

echo "SUCCESS: opencode.json erstellt!"
echo ""

echo "ENV: Schritt 4: NVIDIA_API_KEY setzen"
echo "-----------------------------------------------"

rc_file="${HOME}/.zshrc"
if [[ "${SHELL:-}" == */bash ]]; then
    rc_file="${HOME}/.bashrc"
fi

read -r -p "NVIDIA_API_KEY dauerhaft in ${rc_file} speichern? (y/N): " persist
persist="${persist:-N}"
persist_lower="$(printf '%s' "${persist}" | tr '[:upper:]' '[:lower:]')"

if [[ "${persist_lower}" == "y" || "${persist_lower}" == "yes" ]]; then
    if ! grep -q "export NVIDIA_API_KEY" "${rc_file}" 2>/dev/null; then
        {
            echo ""
            echo "# NVIDIA NIM Configuration (added by BIOMETRICS setup)"
            echo "export NVIDIA_API_KEY=\"${nvidia_key}\""
        } >> "${rc_file}"
        echo "SUCCESS: NVIDIA_API_KEY zu ${rc_file} hinzugefügt"
    else
        echo "WARNING:  NVIDIA_API_KEY ist bereits in ${rc_file}"
    fi
else
    echo "INFO: NVIDIA_API_KEY wurde NICHT in ein Shell-Profil geschrieben."
    echo "      Setze es in deiner aktuellen Shell bevor du opencode nutzt:"
    echo "      export NVIDIA_API_KEY=\"nvapi-...\""
fi

echo ""

# Step 5: Verification
echo "SUCCESS: VERIFIKATION"
echo "--------------"
echo ""
echo "Deine Konfiguration wurde erstellt!"
echo ""
echo "Nächste Schritte:"
echo "1. Shell neu laden (oder neues Terminal)"
echo "2. Testen: opencode models"
echo "3. Optional: oh-my-opencode nutzen (siehe docs/OPENCODE.md)"
echo ""
echo "DOCS: Vollständige Anleitung: docs/OPENCODE.md"
echo ""

read -r -p "Shell jetzt neu laden? (y/n): " reload

if [ "$reload" = "y" ]; then
    exec "${SHELL:-/bin/zsh}"
fi
