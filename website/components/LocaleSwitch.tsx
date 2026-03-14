export function LocaleSwitch({ dePath, enPath }: { dePath: string; enPath: string }) {
  return (
    <nav className="locale-switch" aria-label="locale switch" data-testid="locale-switch">
      <a href={enPath} data-testid="switch-en">
        EN
      </a>
      <span>/</span>
      <a href={dePath} data-testid="switch-de">
        DE
      </a>
    </nav>
  )
}
