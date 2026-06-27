import type { CapturedRequest } from './api'

// methodColor maps an HTTP method to Tailwind text/border classes for badges.
export function methodColor(method: string): string {
  switch (method.toUpperCase()) {
    case 'GET':
      return 'text-ok border-ok/40 bg-ok/10'
    case 'POST':
      return 'text-accent border-accent/40 bg-accent/10'
    case 'PUT':
    case 'PATCH':
      return 'text-warn border-warn/40 bg-warn/10'
    case 'DELETE':
      return 'text-err border-err/40 bg-err/10'
    default:
      return 'text-muted border-border bg-surface-2'
  }
}

export function relativeTime(iso: string): string {
  const then = new Date(iso).getTime()
  const secs = Math.max(0, Math.floor((Date.now() - then) / 1000))
  if (secs < 5) return 'just now'
  if (secs < 60) return `${secs}s ago`
  const mins = Math.floor(secs / 60)
  if (mins < 60) return `${mins}m ago`
  const hrs = Math.floor(mins / 60)
  if (hrs < 24) return `${hrs}h ago`
  return new Date(iso).toLocaleString()
}

export function shortId(id: string): string {
  return id.slice(0, 8)
}

// summarise returns a one-line preview of a request for the list.
export function summarise(r: CapturedRequest): string {
  if (r.content) return r.content.replace(/\s+/g, ' ').slice(0, 80)
  const q = r.query ? Object.keys(r.query) : []
  if (q.length) return `?${q.join(', ')}`
  return r.url
}

export async function copyText(text: string): Promise<void> {
  try {
    await navigator.clipboard.writeText(text)
  } catch {
    /* clipboard unavailable */
  }
}
