type SuccessChecklistProps = {
  items: string[]
}

export function SuccessChecklist({ items }: SuccessChecklistProps) {
  return (
    <ul className="success-checklist" data-testid="success-checklist">
      {items.map((item) => (
        <li key={item}>{item}</li>
      ))}
    </ul>
  )
}
