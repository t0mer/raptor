import { useMemo } from 'react'
import type { CapturedRequest } from '../api'
import { api } from '../api'
import { copyText, methodColor, relativeTime } from '../lib'
import { CopyIcon, TrashIcon } from './icons'

interface Props {
  tokenId: string
  request: CapturedRequest
  onDelete: (rid: string) => void
}

function prettyBody(content: string, headers: Record<string, string[]> | null): string {
  const ct = headers?.['Content-Type']?.[0] ?? ''
  if (ct.includes('json') || (content.trim().startsWith('{') || content.trim().startsWith('['))) {
    try {
      return JSON.stringify(JSON.parse(content), null, 2)
    } catch {
      /* not valid JSON */
    }
  }
  return content
}

function KeyValues({ title, data }: { title: string; data: Record<string, string[]> | null }) {
  const entries = data ? Object.entries(data) : []
  if (entries.length === 0) return null
  return (
    <section>
      <h3 className="text-xs uppercase tracking-wide text-muted mb-2">{title}</h3>
      <div className="rounded-lg border border-border overflow-hidden">
        <table className="w-full text-sm">
          <tbody>
            {entries.map(([k, vals]) => (
              <tr key={k} className="border-b border-border last:border-0 align-top">
                <td className="px-3 py-2 font-mono text-muted w-1/3 break-all">{k}</td>
                <td className="px-3 py-2 font-mono break-all">{vals.join(', ')}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  )
}

export function RequestDetail({ tokenId, request, onDelete }: Props) {
  const body = useMemo(
    () => prettyBody(request.content, request.headers),
    [request.content, request.headers],
  )

  return (
    <div className="flex flex-col h-full overflow-y-auto">
      <div className="flex items-center gap-2 px-4 py-3 border-b border-border sticky top-0 bg-surface z-10">
        <span
          className={`text-xs font-semibold font-mono px-2 py-1 rounded border ${methodColor(
            request.method,
          )}`}
        >
          {request.method || request.type.toUpperCase()}
        </span>
        <span className="text-xs text-muted">{relativeTime(request.created_at)}</span>
        <div className="ml-auto flex items-center gap-1">
          <a
            href={api.rawURL(tokenId, request.uuid)}
            target="_blank"
            rel="noreferrer"
            className="text-xs px-2 py-1 rounded-lg hover:bg-surface-2 text-muted"
          >
            Raw
          </a>
          <button
            onClick={() => copyText(request.content)}
            className="p-1.5 rounded-lg hover:bg-surface-2 text-muted"
            aria-label="Copy body"
          >
            <CopyIcon width={16} height={16} />
          </button>
          <button
            onClick={() => onDelete(request.uuid)}
            className="p-1.5 rounded-lg hover:bg-err/10 text-err"
            aria-label="Delete request"
          >
            <TrashIcon width={16} height={16} />
          </button>
        </div>
      </div>

      <div className="p-4 space-y-5">
        <dl className="grid grid-cols-2 sm:grid-cols-4 gap-3 text-sm">
          <Meta label="IP" value={request.ip} />
          <Meta label="Host" value={request.hostname} />
          <Meta label="Size" value={`${request.size} B`} />
          <Meta label="ID" value={request.uuid.slice(0, 8)} mono />
        </dl>

        <section>
          <h3 className="text-xs uppercase tracking-wide text-muted mb-2">URL</h3>
          <div className="font-mono text-sm break-all rounded-lg border border-border bg-surface-2 px-3 py-2">
            {request.url}
          </div>
        </section>

        <KeyValues title="Query" data={request.query} />
        <KeyValues title="Headers" data={request.headers} />

        <section>
          <h3 className="text-xs uppercase tracking-wide text-muted mb-2">Body</h3>
          {body ? (
            <pre className="rounded-lg border border-border bg-surface-2 px-3 py-2 text-xs font-mono overflow-x-auto whitespace-pre-wrap break-words">
              {body}
            </pre>
          ) : (
            <div className="text-sm text-muted italic">empty</div>
          )}
        </section>

        {request.files && request.files.length > 0 && (
          <section>
            <h3 className="text-xs uppercase tracking-wide text-muted mb-2">Files</h3>
            <ul className="space-y-1">
              {request.files.map((f) => (
                <li key={f.id}>
                  <a
                    className="text-sm text-accent hover:underline font-mono"
                    href={`/api/v1/tokens/${tokenId}/requests/${request.uuid}/files/${f.id}`}
                  >
                    {f.filename || f.id} ({f.size} B)
                  </a>
                </li>
              ))}
            </ul>
          </section>
        )}
      </div>
    </div>
  )
}

function Meta({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div>
      <dt className="text-xs text-muted">{label}</dt>
      <dd className={`truncate ${mono ? 'font-mono text-sm' : ''}`}>{value || '—'}</dd>
    </div>
  )
}
