import type { Group, Token } from '../api'
import { relativeTime, shortId } from '../lib'

interface Props {
  tokens: Token[]
  groups: Group[]
  activeId: string | null
  onSelect: (id: string) => void
  onOpenPanel: () => void
}

export function TokenList({ tokens, groups, activeId, onSelect, onOpenPanel }: Props) {
  const groupName = new Map(groups.map((g) => [g.id, g]))

  // Bucket tokens by group, preserving a stable "Ungrouped" bucket last.
  const buckets = new Map<string, Token[]>()
  for (const t of tokens) {
    const key = t.group_id && groupName.has(t.group_id) ? t.group_id : ''
    if (!buckets.has(key)) buckets.set(key, [])
    buckets.get(key)!.push(t)
  }
  const orderedKeys = [...groups.map((g) => g.id).filter((id) => buckets.has(id)), '']

  return (
    <div className="flex flex-col h-full">
      <nav className="flex-1 flex flex-col p-2 gap-3 overflow-y-auto">
        {tokens.length === 0 && (
          <div className="p-2 text-sm text-muted">
            No URLs yet. Create one with <span className="font-medium text-text">New URL</span>.
          </div>
        )}

        {orderedKeys.map((key) => {
          const items = buckets.get(key)
          if (!items || items.length === 0) return null
          const g = key ? groupName.get(key) : undefined
          return (
            <div key={key || 'ungrouped'}>
              {tokens.some((t) => t.group_id && groupName.has(t.group_id)) && (
                <div className="flex items-center gap-1.5 px-2 mb-1">
                  {g?.color && (
                    <span
                      className="w-2 h-2 rounded-full"
                      style={{ backgroundColor: g.color }}
                    />
                  )}
                  <span className="text-[11px] uppercase tracking-wide text-muted">
                    {g?.name ?? 'Ungrouped'}
                  </span>
                </div>
              )}
              <div className="flex flex-col gap-1">
                {items.map((t) => (
                  <TokenRow
                    key={t.uuid}
                    token={t}
                    active={t.uuid === activeId}
                    onSelect={() => onSelect(t.uuid)}
                  />
                ))}
              </div>
            </div>
          )
        })}
      </nav>

      <button
        onClick={onOpenPanel}
        className="text-sm text-muted hover:text-text border-t border-border px-4 py-2.5 text-left"
      >
        Control Panel →
      </button>
    </div>
  )
}

function TokenRow({
  token,
  active,
  onSelect,
}: {
  token: Token
  active: boolean
  onSelect: () => void
}) {
  return (
    <button
      onClick={onSelect}
      className={`text-left rounded-lg px-3 py-2.5 transition border ${
        active ? 'bg-accent/10 border-accent/30' : 'border-transparent hover:bg-surface-2'
      }`}
    >
      <div className="flex items-center gap-2">
        <span className="font-mono text-sm truncate text-text">
          {token.alias || shortId(token.uuid)}
        </span>
      </div>
      <div className="text-xs text-muted truncate">
        {token.description || token.url.replace(/^https?:\/\//, '')}
      </div>
      <div className="text-[11px] text-muted mt-0.5">
        {token.latest_request_at ? relativeTime(token.latest_request_at) : 'no requests'}
      </div>
    </button>
  )
}
