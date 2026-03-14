import { useMemo, useState } from 'react'
import { trackEvent } from '../lib/analytics'

type CommandTabsProps = {
  macos: string
  linux: string
  windowsWSL: string
}

type Platform = 'macos' | 'linux' | 'windows-wsl'
type CopyState = 'idle' | 'copied' | 'failed'

const platformLabels: Record<Platform, string> = {
  macos: 'macOS',
  linux: 'Linux',
  'windows-wsl': 'Windows (WSL)'
}

async function copyWithFallback(text: string): Promise<boolean> {
  try {
    if (typeof navigator !== 'undefined' && navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(text)
      return true
    }
  } catch {
    // fall through to legacy copy path
  }

  if (typeof document === 'undefined') {
    return false
  }

  const textArea = document.createElement('textarea')
  textArea.value = text
  textArea.setAttribute('readonly', '')
  textArea.style.position = 'absolute'
  textArea.style.left = '-9999px'
  document.body.appendChild(textArea)
  textArea.select()

  let copied = false
  try {
    copied = document.execCommand('copy')
  } finally {
    document.body.removeChild(textArea)
  }

  return copied
}

export function CommandTabs({ macos, linux, windowsWSL }: CommandTabsProps) {
  const [active, setActive] = useState<Platform>('macos')
  const [copyState, setCopyState] = useState<CopyState>('idle')

  const activeCommand = useMemo(() => {
    if (active === 'macos') return macos
    if (active === 'linux') return linux
    return windowsWSL
  }, [active, linux, macos, windowsWSL])

  return (
    <div className="command-tabs" data-testid="command-tabs">
      <div className="platform-tabs" role="tablist" aria-label="platform tabs">
        {(['macos', 'linux', 'windows-wsl'] as const).map((platform) => (
          <button
            key={platform}
            role="tab"
            type="button"
            className={platform === active ? 'active' : ''}
            aria-selected={platform === active}
            data-testid={`tab-${platform}`}
            onClick={() => setActive(platform)}
          >
            {platformLabels[platform]}
          </button>
        ))}
      </div>
      <pre className="command-block" data-testid="command-block">
        <code>{activeCommand}</code>
      </pre>
      <div className="copy-row">
        <button
          type="button"
          className={`copy-button${copyState === 'copied' ? ' copy-success' : ''}${copyState === 'failed' ? ' copy-failed' : ''}`}
          data-testid="copy-command"
          onClick={async () => {
            const copied = await copyWithFallback(activeCommand)
            setCopyState(copied ? 'copied' : 'failed')
            if (copied) {
              await trackEvent({ event: 'copy_command', label: active })
            }
            window.setTimeout(() => setCopyState('idle'), 1500)
          }}
        >
          {copyState === 'copied' ? 'Copied' : copyState === 'failed' ? 'Copy failed' : 'Copy command'}
        </button>
        <span className="copy-status" role="status" aria-live="polite">
          {copyState === 'copied' ? 'Command copied to clipboard.' : copyState === 'failed' ? 'Clipboard access is blocked in this browser context.' : ''}
        </span>
      </div>
    </div>
  )
}
