import { useEffect, useState } from 'react'
import { api, type Schedule, type ScheduleInput, type ScheduleRun } from '../api'
import { relativeTime } from '../lib'
import { TrashIcon } from './icons'

const field =
  'w-full rounded-lg border border-border bg-surface-2 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-accent/40'
const label = 'text-xs uppercase tracking-wide text-muted'

function statusColor(s: string): string {
  if (s === 'ok') return 'text-ok border-ok/40 bg-ok/10'
  if (s === 'alert') return 'text-err border-err/40 bg-err/10'
  if (s === 'error') return 'text-warn border-warn/40 bg-warn/10'
  return 'text-muted border-border bg-surface-2'
}

export function SchedulesView() {
  const [schedules, setSchedules] = useState<Schedule[]>([])
  const [error, setError] = useState<string | null>(null)
  const [runs, setRuns] = useState<Record<string, ScheduleRun[]>>({})

  const reload = () => api.listSchedules().then(setSchedules).catch(() => {})
  useEffect(() => {
    void reload()
  }, [])

  async function runNow(s: Schedule) {
    await api.runSchedule(s.uuid)
    void reload()
    if (runs[s.uuid]) {
      setRuns((r) => ({ ...r, [s.uuid]: [] }))
      const list = await api.listScheduleRuns(s.uuid)
      setRuns((r) => ({ ...r, [s.uuid]: list }))
    }
  }
  async function toggleRuns(s: Schedule) {
    if (runs[s.uuid]) {
      setRuns((r) => {
        const next = { ...r }
        delete next[s.uuid]
        return next
      })
      return
    }
    const list = await api.listScheduleRuns(s.uuid)
    setRuns((r) => ({ ...r, [s.uuid]: list }))
  }
  async function remove(s: Schedule) {
    await api.deleteSchedule(s.uuid)
    void reload()
  }
  async function toggle(s: Schedule) {
    await api.updateSchedule(s.uuid, { enabled: !s.enabled })
    void reload()
  }

  return (
    <div className="flex-1 overflow-y-auto p-4 sm:p-6 max-w-4xl mx-auto w-full">
      <h1 className="text-xl font-semibold mb-1">Schedules</h1>
      <p className="text-sm text-muted mb-6">
        Run a URL (or a token's action chain) on a cron interval and alert on status, keyword,
        uptime or SSL expiry.
      </p>

      <ScheduleForm
        onSaved={() => {
          setError(null)
          void reload()
        }}
        onError={setError}
      />
      {error && <div className="text-sm text-err mt-2">{error}</div>}

      <ul className="mt-6 space-y-2">
        {schedules.length === 0 && <li className="text-sm text-muted">No schedules yet.</li>}
        {schedules.map((s) => (
          <li key={s.uuid} className="rounded-lg border border-border p-3">
            <div className="flex items-center gap-3">
              <div className="min-w-0">
                <div className="font-medium truncate">{s.name || s.target_url || s.uuid}</div>
                <div className="text-xs text-muted font-mono truncate">
                  {s.cron} · {s.run_actions ? 'run actions' : s.target_url}
                </div>
              </div>
              {s.last_status && (
                <span className={`text-[11px] font-mono px-1.5 py-0.5 rounded border ${statusColor(s.last_status)}`}>
                  {s.last_status}
                </span>
              )}
              <div className="ml-auto flex items-center gap-2 text-xs">
                {s.next_run && <span className="text-muted hidden sm:inline">next {relativeTime(s.next_run)}</span>}
                <label className="flex items-center gap-1 text-muted">
                  <input type="checkbox" checked={s.enabled} onChange={() => toggle(s)} /> on
                </label>
                <button onClick={() => runNow(s)} className="text-accent hover:underline">
                  run
                </button>
                <button onClick={() => toggleRuns(s)} className="text-muted hover:text-text">
                  history
                </button>
                <button onClick={() => remove(s)} className="p-1 rounded hover:bg-err/10 text-err" aria-label="Delete">
                  <TrashIcon width={14} height={14} />
                </button>
              </div>
            </div>
            {s.last_message && <div className="text-xs text-muted mt-1">{s.last_message}</div>}
            {runs[s.uuid] && (
              <div className="mt-2 space-y-1">
                {runs[s.uuid].length === 0 && <div className="text-xs text-muted">No runs yet.</div>}
                {runs[s.uuid].map((run) => (
                  <div key={run.id} className="text-xs font-mono flex items-center gap-2">
                    <span className={`px-1 rounded border ${statusColor(run.status)}`}>{run.status}</span>
                    <span className="text-muted">{relativeTime(run.created_at)}</span>
                    <span className="truncate">{run.message}</span>
                  </div>
                ))}
              </div>
            )}
          </li>
        ))}
      </ul>
    </div>
  )
}

function ScheduleForm({
  onSaved,
  onError,
}: {
  onSaved: () => void
  onError: (m: string) => void
}) {
  const [form, setForm] = useState<ScheduleInput>({
    name: '',
    cron: '*/5 * * * *',
    target_url: '',
    method: 'GET',
    keyword: '',
    notify_url: '',
    enabled: true,
    ssl_days: 14,
  })

  function set<K extends keyof ScheduleInput>(k: K, v: ScheduleInput[K]) {
    setForm((f) => ({ ...f, [k]: v }))
  }

  async function add() {
    if (!form.cron) {
      onError('cron is required')
      return
    }
    try {
      await api.createSchedule(form)
      setForm((f) => ({ ...f, name: '', target_url: '', keyword: '', notify_url: '' }))
      onSaved()
    } catch (e) {
      onError(e instanceof Error ? e.message : 'create failed')
    }
  }

  return (
    <div className="rounded-lg border border-border p-4 space-y-3 bg-surface-2/40">
      <div className="text-xs uppercase tracking-wide text-muted">New schedule</div>
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
        <div>
          <label className={label}>Name</label>
          <input className={field} value={form.name ?? ''} onChange={(e) => set('name', e.target.value)} />
        </div>
        <div>
          <label className={label}>Cron</label>
          <input className={field} value={form.cron ?? ''} onChange={(e) => set('cron', e.target.value)} />
        </div>
        <div>
          <label className={label}>Check SSL</label>
          <select
            className={field}
            value={form.check_ssl ? 'yes' : 'no'}
            onChange={(e) => set('check_ssl', e.target.value === 'yes')}
          >
            <option value="no">no</option>
            <option value="yes">yes</option>
          </select>
        </div>
      </div>
      <div>
        <label className={label}>Target URL</label>
        <input
          className={field}
          placeholder="https://example.com/health"
          value={form.target_url ?? ''}
          onChange={(e) => set('target_url', e.target.value)}
        />
      </div>
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
        <div>
          <label className={label}>Keyword (alert if missing)</label>
          <input className={field} value={form.keyword ?? ''} onChange={(e) => set('keyword', e.target.value)} />
        </div>
        <div>
          <label className={label}>Notify URL (Shoutrrr)</label>
          <input
            className={field}
            placeholder="slack://… / discord://… / ntfy://…"
            value={form.notify_url ?? ''}
            onChange={(e) => set('notify_url', e.target.value)}
          />
        </div>
      </div>
      <button onClick={add} className="text-sm px-4 py-2 rounded-lg bg-accent text-accent-fg font-medium">
        Add schedule
      </button>
    </div>
  )
}
