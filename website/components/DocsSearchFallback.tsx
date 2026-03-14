import { useState } from 'react'
import { trackEvent } from '../lib/analytics'
import { searchDocs, type SearchResult } from '../lib/search-index'

export function DocsSearchFallback() {
  const [query, setQuery] = useState('')
  const [results, setResults] = useState<SearchResult[]>([])
  const [didSearch, setDidSearch] = useState(false)

  return (
    <section className="search-fallback" data-testid="search-fallback">
      <label htmlFor="docs-search">Instant docs search</label>
      <form
        className="search-row"
        onSubmit={async (event) => {
          event.preventDefault()
          const q = query.trim()
          setDidSearch(true)
          if (!q) {
            setResults([])
            return
          }

          const matches = searchDocs(q, 10)
          setResults(matches)
          if (matches.length > 0) {
            await trackEvent({ event: 'docs_search_success', label: q })
          }
        }}
      >
        <input
          id="docs-search"
          type="search"
          value={query}
          data-testid="docs-search-input"
          placeholder="Search quickstart, API, security, runbook..."
          onChange={(event) => setQuery(event.target.value)}
        />
        <button type="submit" data-testid="docs-search-button">
          Search
        </button>
      </form>
      <p className="search-meta" aria-live="polite">
        {didSearch
          ? `${results.length} result${results.length === 1 ? '' : 's'} for "${query.trim()}"`
          : 'Search by endpoint, workflow, policy, or operating task.'}
      </p>
      {didSearch && results.length === 0 ? <p className="search-empty">No matching docs entry found.</p> : null}
      <ul className="search-results">
        {results.map((result) => (
          <li key={result.href}>
            <a href={result.href}>{result.title}</a>
            <p>{result.snippet}</p>
          </li>
        ))}
      </ul>
    </section>
  )
}
