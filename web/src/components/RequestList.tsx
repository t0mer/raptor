import type { CapturedRequest } from '../api'
import { methodColor, relativeTime, summarise } from '../lib'

interface Props {
  requests: CapturedRequest[]
  activeId: string | null
  onSelect: (r: CapturedRequest) => void
}

export function RequestList({ requests, activeId, onSelect }: Props) {
  if (requests.length === 0) {
    return (
      <div className="grid place-items-center h-full p-8 text-center">
        <div className="max-w-xs">
          <div className="text-sm text-muted">
            Waiting for requests… Send any HTTP request to this URL and it will appear here
            instantly.
          </div>
        </div>
      </div>
    )
  }

  return (
    <ul className="divide-y divide-border overflow-y-auto h-full">
      {requests.map((r) => {
        const active = r.uuid === activeId
        return (
          <li key={r.uuid}>
            <button
              onClick={() => onSelect(r)}
              className={`w-full text-left px-4 py-3 transition ${
                active ? 'bg-accent/10' : 'hover:bg-surface-2'
              }`}
            >
              <div className="flex items-center gap-2">
                <span
                  className={`text-[11px] font-semibold font-mono px-1.5 py-0.5 rounded border ${methodColor(
                    r.method,
                  )}`}
                >
                  {r.method || r.type.toUpperCase()}
                </span>
                <span className="text-xs text-muted ml-auto shrink-0">
                  {relativeTime(r.created_at)}
                </span>
              </div>
              <div className="mt-1 font-mono text-xs text-muted truncate">{summarise(r)}</div>
            </button>
          </li>
        )
      })}
    </ul>
  )
}
