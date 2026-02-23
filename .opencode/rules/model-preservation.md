# MODEL PRESERVATION RULES V12.0 (UNLIMITED HYBRID MATRIX)

**CRITICAL RULE - PERMANENT ENFORCEMENT**

The system is now optimized for the **Hybrid-Hybrid Architecture (V12.0)** to eliminate all quota lockouts.

## 🛡️ THE STRATEGY: ANTIGRAVITY PLUGIN FOR ALL

To bypass quota lockouts and ensure seamless operation, we utilize the Antigravity Plugin for BOTH Gemini and Claude models:

1. **All Models (Gemini & Claude)**:
   - Always use the `google` provider (Provider: `google`).
   - This uses the Antigravity Plugin for optimal performance and quota management.
   - Sequence: Gemini 3.1 Pro -> Gemini 3 Pro -> Gemini 3 Flash -> Claude.

2. **Claude Models (Architecture Only)**:
   - Reserved for Oracle and Momus via Antigravity Plugin (Provider: `google`).
   - Suffix mapping: `claude-3-opus@20240229` and `claude-3-5-sonnet-v2@20241022`.

## 🤖 THE EXACT MODEL ALLOCATION MATRIX (V12.0)

| Agent | Primary (Antigravity) | Fallback 1 | Fallback 2 (Safety) |
|-------|-------------------|----------------------|----------------------|
| **Sisyphus** | `google/gemini-3.1-pro-preview` | `google/gemini-3-pro-preview` | `nvidia-nim/qwen3.5-397b-a17b` |
| **Prometheus** | `google/gemini-3.1-pro-preview` | `google/gemini-3-flash-preview` | `opencode/glm-5-free` |
| **Oracle** | `google/antigravity-claude-opus-4-6-thinking` | `google/gemini-3.1-pro-preview` | `nvidia-nim/qwen3.5-397b-a17b` |
| **Momus** | `google/antigravity-claude-sonnet-4-6` | `google/gemini-3-flash-preview` | `opencode/glm-5-free` |
| **Atlas** | `google/gemini-3.1-pro-preview` | `google/gemini-3-flash-preview` | `nvidia-nim/qwen3.5-397b-a17b` |
| **Sisyphus-Jr** | `google/gemini-3-flash-preview` | `google/gemini-3-pro-preview` | `opencode/glm-5-free` |

## 🚫 ABSOLUTE PROHIBITIONS

1. **NO DATE SUFFIXES**: Never add suffixes like `-05120` to Gemini IDs.
2. **NO TIMEOUTS**: Timeouts in any configuration file are strictly forbidden.
3. **NO REMOVAL**: Never remove models from config due to 403/429 errors.
4. **NO PLURAL**: The JSON key is `"command"`, NOT `"commands"`.

---
**Version:** 12.0
**Status:** ACTIVE - PERMANENT ENFORCEMENT
