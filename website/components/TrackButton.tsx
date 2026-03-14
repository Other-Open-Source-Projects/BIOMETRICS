import { type PropsWithChildren } from 'react'
import { trackEvent, type AnalyticsEvent } from '../lib/analytics'

type TrackButtonProps = PropsWithChildren<{
  href: string
  event: AnalyticsEvent
  label: string
  className?: string
  testID?: string
}>

export function TrackButton({ href, event, label, className, testID, children }: TrackButtonProps) {
  return (
    <a
      href={href}
      data-testid={testID}
      className={className ?? 'hero-cta'}
      onClick={() => {
        void trackEvent({ event, label })
      }}
    >
      {children}
    </a>
  )
}
