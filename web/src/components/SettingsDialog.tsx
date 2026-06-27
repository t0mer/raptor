import { useState } from 'react'
import type { Group, Token, TokenInput } from '../api'

interface Props {
  token: Token
  groups: Group[]
  onClose: () => void
  onSave: (body: TokenInput) => Promise<void>
  onDelete: () => Promise<void>
}

const field = 'w-full rounded-lg border border-border bg-surface-2 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-accent/40'
const label = 'text-xs uppercase tracking-wide text-muted'

export function SettingsDialog({ token, groups, onClose, onSave, onDelete }: Props) {
  const [form, setForm] = useState<TokenInput>({
    alias: token.alias ?? '',
    description: token.description,
    default_status: token.default_status,
    default_content_type: token.default_content_type,
    default_content: token.default_content,
    timeout: token.timeout,
    expiry: token.expiry,
    request_limit: token.request_limit,
    redirect: token.redirect,
    cors: token.cors,
    group_id: token.group_id ?? '',
  })
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  function set<K extends keyof TokenInput>(k: K, v: TokenInput[K]) {
    setForm((f) => ({ ...f, [k]: v }))
  }

  async function handleSave() {
    setSaving(true)
    setError(null)
    try {
      await onSave(form)
      onClose()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'save failed')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div
      className="fixed inset-0 z-50 grid place-items-center bg-black/40 p-4"
      onClick={onClose}
    >
      <div
        className="w-full max-w-lg max-h-[90vh] overflow-y-auto rounded-xl border border-border bg-surface shadow-xl"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="px-5 py-4 border-b border-border">
          <h2 className="font-semibold">URL settings & default response</h2>
        </div>

        <div className="p-5 space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className={label}>Alias</label>
              <input
                className={field}
                value={form.alias ?? ''}
                onChange={(e) => set('alias', e.target.value)}
                placeholder="optional"
              />
            </div>
            <div>
              <label className={label}>Default status</label>
              <input
                className={field}
                type="number"
                value={form.default_status ?? 200}
                onChange={(e) => set('default_status', Number(e.target.value))}
              />
            </div>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className={label}>Description</label>
              <input
                className={field}
                value={form.description ?? ''}
                onChange={(e) => set('description', e.target.value)}
              />
            </div>
            <div>
              <label className={label}>Group</label>
              <select
                className={field}
                value={form.group_id ?? ''}
                onChange={(e) => set('group_id', e.target.value)}
              >
                <option value="">Ungrouped</option>
                {groups.map((g) => (
                  <option key={g.id} value={g.id}>
                    {g.name}
                  </option>
                ))}
              </select>
            </div>
          </div>

          <div>
            <label className={label}>Content type</label>
            <input
              className={field}
              value={form.default_content_type ?? ''}
              onChange={(e) => set('default_content_type', e.target.value)}
            />
          </div>

          <div>
            <label className={label}>Response body</label>
            <textarea
              className={`${field} font-mono h-28 resize-y`}
              value={form.default_content ?? ''}
              onChange={(e) => set('default_content', e.target.value)}
            />
          </div>

          <div className="grid grid-cols-3 gap-4">
            <div>
              <label className={label}>Timeout</label>
              <input
                className={field}
                type="number"
                value={form.timeout ?? 0}
                onChange={(e) => set('timeout', Number(e.target.value))}
              />
            </div>
            <div>
              <label className={label}>Expiry (s)</label>
              <input
                className={field}
                type="number"
                value={form.expiry ?? 0}
                onChange={(e) => set('expiry', Number(e.target.value))}
              />
            </div>
            <div>
              <label className={label}>Req limit</label>
              <input
                className={field}
                type="number"
                value={form.request_limit ?? 0}
                onChange={(e) => set('request_limit', Number(e.target.value))}
              />
            </div>
          </div>

          <div>
            <label className={label}>Redirect URL</label>
            <input
              className={field}
              value={form.redirect ?? ''}
              onChange={(e) => set('redirect', e.target.value)}
              placeholder="leave empty to return the body"
            />
          </div>

          <label className="flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              checked={form.cors ?? false}
              onChange={(e) => set('cors', e.target.checked)}
            />
            Send permissive CORS headers
          </label>

          {error && <div className="text-sm text-err">{error}</div>}
        </div>

        <div className="flex items-center gap-2 px-5 py-4 border-t border-border">
          <button
            onClick={onDelete}
            className="text-sm text-err px-3 py-2 rounded-lg hover:bg-err/10"
          >
            Delete URL
          </button>
          <div className="ml-auto flex gap-2">
            <button onClick={onClose} className="text-sm px-4 py-2 rounded-lg hover:bg-surface-2">
              Cancel
            </button>
            <button
              onClick={handleSave}
              disabled={saving}
              className="text-sm px-4 py-2 rounded-lg bg-accent text-accent-fg font-medium hover:opacity-90 disabled:opacity-50"
            >
              {saving ? 'Saving…' : 'Save'}
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
