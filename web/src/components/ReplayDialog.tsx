import { useState } from 'react'
import { api } from '../api'

interface Props {
  tokenId: string
  query: string
  onClose: () => void
}

const field =
  'w-full rounded-lg border border-border bg-surface-2 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-accent/40'

// ReplayDialog re-delivers the current (optionally filtered) request subset to a
// target URL.
export function ReplayDialog({ tokenId, query, onClose }: Props) {
  const [target, setTarget] = useState('')
  const [busy, setBusy] = useState(false)
  const [result, setResult] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  async function run() {
    if (!target.trim()) {
      setError('target URL is required')
      return
    }
    setBusy(true)
    setError(null)
    try {
      const r = await api.replay(tokenId, target.trim(), query)
      setResult(`Replayed ${r.replayed} request(s), ${r.failed} failed.`)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'replay failed')
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="fixed inset-0 z-50 grid place-items-center bg-black/40 p-4" onClick={onClose}>
      <div
        className="w-full max-w-md rounded-xl border border-border bg-surface shadow-xl"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="px-5 py-4 border-b border-border">
          <h2 className="font-semibold">Replay requests</h2>
          <p className="text-xs text-muted mt-0.5">
            Re-delivers {query ? 'the filtered' : 'all'} captured requests to a target URL
            (preserving method, body and headers; credentials are stripped).
          </p>
        </div>
        <div className="p-5 space-y-3">
          <input
            className={field}
            placeholder="https://your-service.example/webhook"
            value={target}
            onChange={(e) => setTarget(e.target.value)}
            autoFocus
          />
          {query && <div className="text-xs text-muted font-mono">filter: {query}</div>}
          {result && <div className="text-sm text-ok">{result}</div>}
          {error && <div className="text-sm text-err">{error}</div>}
        </div>
        <div className="flex justify-end gap-2 px-5 py-4 border-t border-border">
          <button onClick={onClose} className="text-sm px-4 py-2 rounded-lg hover:bg-surface-2">
            Close
          </button>
          <button
            onClick={run}
            disabled={busy}
            className="text-sm px-4 py-2 rounded-lg bg-accent text-accent-fg font-medium disabled:opacity-50"
          >
            {busy ? 'Replaying…' : 'Replay'}
          </button>
        </div>
      </div>
    </div>
  )
}
