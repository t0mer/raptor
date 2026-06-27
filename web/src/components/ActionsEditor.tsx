import { useEffect, useState } from 'react'
import { api, type Action, type ActionInput, type TestActionResult } from '../api'
import { TrashIcon } from './icons'

interface Props {
  tokenId: string
  onClose: () => void
}

// Example parameters shown per action type to guide the editor.
const PARAM_HINTS: Record<string, string> = {
  set_variable: `{"name": "id", "value": "$request.content$"}`,
  modify_response: `{"status": "200", "content": "ok", "content_type": "text/plain"}`,
  conditions: `{"input": "$request.method$", "operator": "equals", "value": "POST", "action": "stop"}`,
  extract_json: `{"source": "$request.content$", "path": "data.id", "variable": "id"}`,
  extract_regex: `{"pattern": "#(\\\\d+)", "group": 1, "variable": "order"}`,
  http_request: `{"url": "https://example.com/hook", "method": "POST", "mode": "json", "body": "$request.content$"}`,
  script: `{"script": "respond('ok', 200, 'text/plain')"}`,
  dont_save: `{}`,
  stop: `{}`,
}

const field =
  'w-full rounded-lg border border-border bg-surface-2 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-accent/40'

export function ActionsEditor({ tokenId, onClose }: Props) {
  const [actions, setActions] = useState<Action[]>([])
  const [types, setTypes] = useState<string[]>([])
  const [editing, setEditing] = useState<Action | null>(null)
  const [error, setError] = useState<string | null>(null)

  const reload = () => api.listActions(tokenId).then(setActions).catch(() => {})
  useEffect(() => {
    void reload()
    api.listActionTypes().then(setTypes).catch(() => {})
  }, [tokenId]) // eslint-disable-line react-hooks/exhaustive-deps

  async function toggle(a: Action) {
    await api.updateAction(tokenId, a.uuid, { type: a.type, disabled: !a.disabled, parameters: a.parameters })
    void reload()
  }
  async function remove(a: Action) {
    await api.deleteAction(tokenId, a.uuid)
    void reload()
  }

  return (
    <div className="fixed inset-0 z-50 grid place-items-center bg-black/40 p-4" onClick={onClose}>
      <div
        className="w-full max-w-2xl max-h-[90vh] overflow-y-auto rounded-xl border border-border bg-surface shadow-xl"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="px-5 py-4 border-b border-border flex items-center">
          <h2 className="font-semibold">Custom Actions</h2>
          <span className="ml-2 text-xs text-muted">run in order on every request</span>
          <button onClick={onClose} className="ml-auto text-sm text-muted hover:text-text">
            Close
          </button>
        </div>

        <div className="p-5 space-y-4">
          {actions.length === 0 && (
            <p className="text-sm text-muted">
              No actions yet. Add one below — actions run top-to-bottom and share variables.
            </p>
          )}
          <ul className="space-y-2">
            {actions.map((a) => (
              <li
                key={a.uuid}
                className="flex items-center gap-3 rounded-lg border border-border px-3 py-2"
              >
                <span className="text-xs font-mono text-muted w-6">{a.position}</span>
                <div className="min-w-0">
                  <div className="text-sm font-medium truncate">{a.name || a.type}</div>
                  <div className="text-xs text-muted font-mono">{a.type}</div>
                </div>
                <label className="ml-auto flex items-center gap-1 text-xs text-muted">
                  <input type="checkbox" checked={!a.disabled} onChange={() => toggle(a)} />
                  enabled
                </label>
                <button onClick={() => setEditing(a)} className="text-xs text-accent hover:underline">
                  edit
                </button>
                <button
                  onClick={() => remove(a)}
                  className="p-1 rounded hover:bg-err/10 text-err"
                  aria-label="Delete action"
                >
                  <TrashIcon width={14} height={14} />
                </button>
              </li>
            ))}
          </ul>

          <ActionForm
            tokenId={tokenId}
            types={types}
            editing={editing}
            onSaved={() => {
              setEditing(null)
              setError(null)
              void reload()
            }}
            onError={setError}
            onCancelEdit={() => setEditing(null)}
          />
          {error && <div className="text-sm text-err">{error}</div>}
        </div>
      </div>
    </div>
  )
}

function ActionForm({
  tokenId,
  types,
  editing,
  onSaved,
  onError,
  onCancelEdit,
}: {
  tokenId: string
  types: string[]
  editing: Action | null
  onSaved: () => void
  onError: (m: string) => void
  onCancelEdit: () => void
}) {
  const [type, setType] = useState('set_variable')
  const [name, setName] = useState('')
  const [params, setParams] = useState(PARAM_HINTS['set_variable'])
  const [test, setTest] = useState<TestActionResult | null>(null)

  useEffect(() => {
    if (editing) {
      setType(editing.type)
      setName(editing.name)
      setParams(JSON.stringify(editing.parameters ?? {}, null, 2))
    }
  }, [editing])

  function onTypeChange(t: string) {
    setType(t)
    if (!editing) setParams(PARAM_HINTS[t] ?? '{}')
  }

  function buildBody(): ActionInput | null {
    let parameters: Record<string, unknown> = {}
    try {
      parameters = params.trim() ? JSON.parse(params) : {}
    } catch {
      onError('Parameters must be valid JSON')
      return null
    }
    return { type, name, parameters }
  }

  async function save() {
    const body = buildBody()
    if (!body) return
    try {
      if (editing) await api.updateAction(tokenId, editing.uuid, body)
      else await api.createAction(tokenId, body)
      onSaved()
      setName('')
    } catch (e) {
      onError(e instanceof Error ? e.message : 'save failed')
    }
  }

  async function runTest() {
    const body = buildBody()
    if (!body) return
    try {
      setTest(await api.testAction(tokenId, body))
    } catch (e) {
      onError(e instanceof Error ? e.message : 'test failed')
    }
  }

  return (
    <div className="rounded-lg border border-border p-3 space-y-3 bg-surface-2/40">
      <div className="text-xs uppercase tracking-wide text-muted">
        {editing ? 'Edit action' : 'Add action'}
      </div>
      <div className="grid grid-cols-2 gap-3">
        <select className={field} value={type} onChange={(e) => onTypeChange(e.target.value)}>
          {(types.length ? types : [type]).map((t) => (
            <option key={t} value={t}>
              {t}
            </option>
          ))}
        </select>
        <input
          className={field}
          placeholder="Name (optional)"
          value={name}
          onChange={(e) => setName(e.target.value)}
        />
      </div>
      <textarea
        className={`${field} font-mono h-28 resize-y`}
        value={params}
        onChange={(e) => setParams(e.target.value)}
        spellCheck={false}
      />
      <div className="flex gap-2">
        <button
          onClick={save}
          className="text-sm px-4 py-2 rounded-lg bg-accent text-accent-fg font-medium"
        >
          {editing ? 'Save' : 'Add action'}
        </button>
        <button onClick={runTest} className="text-sm px-4 py-2 rounded-lg hover:bg-surface-2 border border-border">
          Test
        </button>
        {editing && (
          <button onClick={onCancelEdit} className="text-sm px-4 py-2 rounded-lg hover:bg-surface-2">
            Cancel
          </button>
        )}
      </div>

      {test && (
        <div className="rounded-lg border border-border bg-surface px-3 py-2 text-xs space-y-1">
          <div className="text-muted uppercase tracking-wide">Test result</div>
          {test.output && <pre className="whitespace-pre-wrap font-mono">{test.output}</pre>}
          {test.error && <div className="text-err font-mono">error: {test.error}</div>}
          <div className="text-muted">
            response {test.response.status} · vars: {Object.keys(test.variables).join(', ') || '—'}
          </div>
        </div>
      )}
    </div>
  )
}
