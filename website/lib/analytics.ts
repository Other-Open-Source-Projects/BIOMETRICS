export type AnalyticsEvent =
  | 'cta_click'
  | 'quickstart_step_complete'
  | 'copy_command'
  | 'docs_search_success'

export type AnalyticsPayload = {
  event: AnalyticsEvent
  label?: string
  path?: string
}

declare global {
  interface Window {
    plausible?: (event: string, options?: { props?: Record<string, string> }) => void
  }
}

export async function trackEvent(payload: AnalyticsPayload): Promise<void> {
  if (typeof window === 'undefined') {
    return
  }

  if (window.navigator.doNotTrack === '1') {
    return
  }

  const path = payload.path ?? window.location.pathname
  const body = {
    event: payload.event,
    label: payload.label ?? '',
    path,
    ts: new Date().toISOString()
  }

  if (window.plausible) {
    window.plausible(payload.event, {
      props: {
        label: body.label,
        path
      }
    })
  }

  const endpoint = process.env.NEXT_PUBLIC_ANALYTICS_ENDPOINT
  if (!endpoint) {
    return
  }

  try {
    await fetch(endpoint, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(body),
      keepalive: true
    })
  } catch {
    // Non-blocking analytics
  }
}
