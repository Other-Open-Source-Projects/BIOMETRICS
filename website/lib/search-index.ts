export type SearchEntry = {
  title: string
  href: string
  snippet: string
  tags: string[]
}

export type SearchResult = {
  title: string
  href: string
  snippet: string
}

const entries: SearchEntry[] = [
  {
    title: 'Quickstart',
    href: '/quickstart',
    snippet: 'Install BIOMETRICS and run your first orchestrator run in under 10 minutes.',
    tags: ['quickstart', 'install', 'first run', 'onboarding']
  },
  {
    title: 'Install Hub',
    href: '/install',
    snippet: 'Choose your operating system and follow verified install steps.',
    tags: ['install', 'macos', 'linux', 'windows', 'wsl']
  },
  {
    title: 'Docs Home',
    href: '/docs',
    snippet: 'Start by intent with tutorials, how-to guides, references, and explanations.',
    tags: ['docs', 'diataxis', 'tutorials', 'how-to', 'reference']
  },
  {
    title: 'Tutorial: First Successful Run',
    href: '/docs/tutorials/first-successful-run',
    snippet: 'Run the first successful BIOMETRICS flow with readiness and evidence checks.',
    tags: ['tutorial', 'first run', 'evidence', 'readiness']
  },
  {
    title: 'How-to: Run Control Actions',
    href: '/docs/how-to/run-control-actions',
    snippet: 'Safe operator workflow for pause, resume, and cancel actions.',
    tags: ['how-to', 'pause', 'resume', 'cancel', 'operations']
  },
  {
    title: 'Reference: API and Contracts',
    href: '/docs/reference/api',
    snippet: 'OpenAPI and contract references for the BIOMETRICS control plane.',
    tags: ['api', 'openapi', 'contracts', 'schemas']
  },
  {
    title: 'Reference: CLI',
    href: '/docs/reference/cli',
    snippet: 'CLI entrypoints and operational command references.',
    tags: ['cli', 'commands', 'onboard', 'controlplane']
  },
  {
    title: 'Security Defaults',
    href: '/docs/security/defaults',
    snippet: 'Hardening-by-default guidance for secrets, filesystem, and network policies.',
    tags: ['security', 'defaults', 'hardening', 'policy']
  },
  {
    title: 'Runbook: Production Operations',
    href: '/docs/runbooks/production-operations',
    snippet: 'Operational best practices for run control, recovery, and evidence capture.',
    tags: ['runbook', 'production', 'operations', 'recovery']
  },
  {
    title: 'Explanation: Control-plane Architecture',
    href: '/docs/explanations/control-plane-architecture',
    snippet: 'Architecture boundaries, governance layers, and scaling tradeoffs.',
    tags: ['architecture', 'control plane', 'governance', 'explanation']
  }
]

export function searchDocs(query: string, limit = 8): SearchResult[] {
  const normalized = query.trim().toLowerCase()
  if (!normalized) {
    return []
  }

  const tokens = normalized.split(/\s+/).filter(Boolean)
  return entries
    .map((entry) => {
      const title = entry.title.toLowerCase()
      const snippet = entry.snippet.toLowerCase()
      const tags = entry.tags.join(' ').toLowerCase()
      const href = entry.href.toLowerCase()

      let score = 0
      for (const token of tokens) {
        if (title.includes(token)) score += 4
        if (title.startsWith(token)) score += 2
        if (tags.includes(token)) score += 3
        if (snippet.includes(token)) score += 2
        if (href.includes(token)) score += 1
      }

      return { entry, score }
    })
    .filter((candidate) => candidate.score > 0)
    .sort((left, right) => right.score - left.score)
    .slice(0, limit)
    .map((candidate) => ({
      title: candidate.entry.title,
      href: candidate.entry.href,
      snippet: candidate.entry.snippet
    }))
}
