import { BoltIcon, MenuIcon, MoonIcon, PlusIcon, SunIcon } from './icons'
import type { Theme } from '../useTheme'

interface Props {
  theme: Theme
  onToggleTheme: () => void
  onNewToken: () => void
  onToggleSidebar: () => void
}

export function Navbar({ theme, onToggleTheme, onNewToken, onToggleSidebar }: Props) {
  return (
    <header className="flex items-center gap-3 border-b border-border bg-surface px-4 h-14 shrink-0">
      <button
        className="md:hidden p-2 -ml-2 rounded-lg hover:bg-surface-2 text-muted"
        onClick={onToggleSidebar}
        aria-label="Toggle token list"
      >
        <MenuIcon />
      </button>

      <div className="flex items-center gap-2 font-semibold tracking-tight">
        <span className="grid place-items-center w-8 h-8 rounded-lg bg-accent text-accent-fg">
          <BoltIcon width={18} height={18} />
        </span>
        <span className="text-lg">Raptor</span>
        <span className="hidden sm:inline text-xs text-muted font-normal">webhook inspector</span>
      </div>

      <div className="ml-auto flex items-center gap-2">
        <button
          onClick={onNewToken}
          className="inline-flex items-center gap-1.5 rounded-lg bg-accent text-accent-fg px-3 py-1.5 text-sm font-medium hover:opacity-90 transition"
        >
          <PlusIcon width={16} height={16} />
          <span className="hidden sm:inline">New URL</span>
        </button>
        <button
          onClick={onToggleTheme}
          className="p-2 rounded-lg hover:bg-surface-2 text-muted"
          aria-label={theme === 'dark' ? 'Switch to light mode' : 'Switch to dark mode'}
        >
          {theme === 'dark' ? <SunIcon /> : <MoonIcon />}
        </button>
      </div>
    </header>
  )
}
