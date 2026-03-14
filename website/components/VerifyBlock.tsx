import { type PropsWithChildren } from 'react'

type VerifyBlockProps = PropsWithChildren<{
  title: string
  expected: string
  testID?: string
}>

export function VerifyBlock({ title, expected, testID, children }: VerifyBlockProps) {
  return (
    <section className="verify-block" data-testid={testID}>
      <h4>{title}</h4>
      <div className="verify-body">{children}</div>
      <p className="verify-expected">
        <strong>Expected output:</strong> <code>{expected}</code>
      </p>
    </section>
  )
}
