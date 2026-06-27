import { useMemo } from 'react'
import type { CapturedRequest } from '../api'
import { api } from '../api'
import { badgeLabel, checkColor, copyText, methodColor, relativeTime } from '../lib'
import { CopyIcon, TrashIcon } from './icons'

interface Props {
  tokenId: string
  request: CapturedRequest
  onDelete: (rid: string) => void
}

function prettyBody(content: string, headers: Record<string, string[]> | null): string {
  const ct = headers?.['Content-Type']?.[0] ?? ''
  if (ct.includes('json') || content.trim().startsWith('{') || content.trim().startsWith('[')) {
    try {
      return JSON.stringify(JSON.parse(content), null, 2)
    } catch {
      /* not valid JSON */
    }
  }
  return content
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <section>
      <h3 className="text-xs uppercase tracking-wide text-muted mb-2">{title}</h3>
      {children}
    </section>
  )
}

function KeyValues({ title, data }: { title: string; data: Record<string, string[]> | null }) {
  const entries = data ? Object.entries(data) : []
  if (entries.length === 0) return null
  return (
    <Section title={title}>
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
    </Section>
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
            badgeLabel(request),
          )}`}
        >
          {badgeLabel(request)}
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
        {request.type === 'email' ? (
          <EmailView request={request} body={body} />
        ) : request.type === 'dns' ? (
          <DNSView request={request} />
        ) : (
          <WebView request={request} body={body} />
        )}

        {request.files && request.files.length > 0 && (
          <Section title="Files">
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
          </Section>
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

function WebView({ request, body }: { request: CapturedRequest; body: string }) {
  return (
    <>
      <dl className="grid grid-cols-2 sm:grid-cols-4 gap-3 text-sm">
        <Meta label="IP" value={request.ip} />
        <Meta label="Host" value={request.hostname} />
        <Meta label="Size" value={`${request.size} B`} />
        <Meta label="ID" value={request.uuid.slice(0, 8)} mono />
      </dl>
      <Section title="URL">
        <div className="font-mono text-sm break-all rounded-lg border border-border bg-surface-2 px-3 py-2">
          {request.url}
        </div>
      </Section>
      <KeyValues title="Query" data={request.query} />
      <KeyValues title="Headers" data={request.headers} />
      <Section title="Body">
        {body ? (
          <pre className="rounded-lg border border-border bg-surface-2 px-3 py-2 text-xs font-mono overflow-x-auto whitespace-pre-wrap break-words">
            {body}
          </pre>
        ) : (
          <div className="text-sm text-muted italic">empty</div>
        )}
      </Section>
    </>
  )
}

function EmailView({ request, body }: { request: CapturedRequest; body: string }) {
  const isHTML = /<[a-z!/]/i.test(request.content)
  return (
    <>
      <dl className="grid grid-cols-1 sm:grid-cols-2 gap-3 text-sm">
        <Meta label="From" value={request.sender ?? ''} mono />
        <Meta label="To" value={request.destinations ?? ''} mono />
        <Meta label="Subject" value={request.subject ?? ''} />
        <Meta label="Message-ID" value={request.message_id ?? ''} mono />
      </dl>

      {request.checks && Object.keys(request.checks).length > 0 && (
        <Section title="Authentication">
          <div className="flex flex-wrap gap-2">
            {Object.entries(request.checks).map(([k, v]) => (
              <span
                key={k}
                className={`text-xs font-mono px-2 py-1 rounded border ${checkColor(v)}`}
              >
                {k.toUpperCase()}: {v}
              </span>
            ))}
          </div>
        </Section>
      )}

      <Section title="Body">
        {isHTML ? (
          // Rendered in a fully sandboxed iframe (no scripts, no same-origin),
          // so attacker-controlled email HTML cannot execute.
          <iframe
            title="email body"
            sandbox=""
            srcDoc={request.content}
            className="w-full h-96 rounded-lg border border-border bg-white"
          />
        ) : body ? (
          <pre className="rounded-lg border border-border bg-surface-2 px-3 py-2 text-xs font-mono overflow-x-auto whitespace-pre-wrap break-words">
            {body}
          </pre>
        ) : (
          <div className="text-sm text-muted italic">empty</div>
        )}
      </Section>

      {request.text_content && isHTML && (
        <Section title="Plain text">
          <pre className="rounded-lg border border-border bg-surface-2 px-3 py-2 text-xs font-mono overflow-x-auto whitespace-pre-wrap break-words">
            {request.text_content}
          </pre>
        </Section>
      )}

      <KeyValues title="Headers" data={request.headers} />
    </>
  )
}

function DNSView({ request }: { request: CapturedRequest }) {
  const q = request.query ?? {}
  return (
    <>
      <dl className="grid grid-cols-2 sm:grid-cols-4 gap-3 text-sm">
        <Meta label="Type" value={request.method} mono />
        <Meta label="Class" value={q['class']?.[0] ?? 'IN'} mono />
        <Meta label="Client IP" value={request.ip} mono />
        <Meta label="ID" value={request.uuid.slice(0, 8)} mono />
      </dl>
      <Section title="Query name">
        <div className="font-mono text-sm break-all rounded-lg border border-border bg-surface-2 px-3 py-2">
          {request.hostname || request.content}
        </div>
      </Section>
    </>
  )
}
