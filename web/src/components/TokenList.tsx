import type { Token } from '../api'
import { relativeTime, shortId } from '../lib'

interface Props {
  tokens: Token[]
  activeId: string | null
  onSelect: (id: string) => void
}

export function TokenList({ tokens, activeId, onSelect }: Props) {
  if (tokens.length === 0) {
    return (
      <div className="p-4 text-sm text-muted">
        No URLs yet. Create one with <span className="font-medium text-text">New URL</span>.
      </div>
    )
  }

  return (
    <nav className="flex flex-col p-2 gap-1 overflow-y-auto">
      {tokens.map((t) => {
        const active = t.uuid === activeId
        return (
          <button
            key={t.uuid}
            onClick={() => onSelect(t.uuid)}
            className={`text-left rounded-lg px-3 py-2.5 transition border ${
              active
                ? 'bg-accent/10 border-accent/30'
                : 'border-transparent hover:bg-surface-2'
            }`}
          >
            <div className="flex items-center gap-2">
              <span className="font-mono text-sm truncate text-text">
                {t.alias || shortId(t.uuid)}
              </span>
              {t.actions && (
                <span className="text-[10px] uppercase tracking-wide text-accent">actions</span>
              )}
            </div>
            <div className="text-xs text-muted truncate">
              {t.description || t.url.replace(/^https?:\/\//, '')}
            </div>
            <div className="text-[11px] text-muted mt-0.5">
              {t.latest_request_at ? relativeTime(t.latest_request_at) : 'no requests'}
            </div>
          </button>
        )
      })}
    </nav>
  )
}
