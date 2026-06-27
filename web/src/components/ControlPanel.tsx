import { useState } from 'react'
import type { Group, Token } from '../api'
import { relativeTime, shortId } from '../lib'
import { TrashIcon } from './icons'

interface Props {
  tokens: Token[]
  groups: Group[]
  onOpenToken: (id: string) => void
  onDeleteToken: (id: string) => void
  onAssignGroup: (tokenId: string, groupId: string) => void
  onCreateGroup: (name: string) => void
  onDeleteGroup: (id: string) => void
}

export function ControlPanel({
  tokens,
  groups,
  onOpenToken,
  onDeleteToken,
  onAssignGroup,
  onCreateGroup,
  onDeleteGroup,
}: Props) {
  const [newGroup, setNewGroup] = useState('')

  return (
    <div className="flex-1 overflow-y-auto p-4 sm:p-6 max-w-5xl mx-auto w-full">
      <h1 className="text-xl font-semibold mb-1">Control Panel</h1>
      <p className="text-sm text-muted mb-6">Manage every capture URL and group in one place.</p>

      {/* Groups */}
      <section className="mb-8">
        <h2 className="text-sm uppercase tracking-wide text-muted mb-2">Groups</h2>
        <div className="flex flex-wrap items-center gap-2 mb-3">
          {groups.length === 0 && <span className="text-sm text-muted">No groups yet.</span>}
          {groups.map((g) => (
            <span
              key={g.id}
              className="inline-flex items-center gap-1.5 rounded-full border border-border bg-surface-2 pl-3 pr-1.5 py-1 text-sm"
            >
              {g.color && (
                <span className="w-2 h-2 rounded-full" style={{ backgroundColor: g.color }} />
              )}
              {g.name}
              <button
                onClick={() => onDeleteGroup(g.id)}
                className="p-0.5 rounded hover:bg-err/10 text-err"
                aria-label={`Delete group ${g.name}`}
              >
                <TrashIcon width={13} height={13} />
              </button>
            </span>
          ))}
        </div>
        <form
          className="flex gap-2"
          onSubmit={(e) => {
            e.preventDefault()
            const name = newGroup.trim()
            if (name) {
              onCreateGroup(name)
              setNewGroup('')
            }
          }}
        >
          <input
            value={newGroup}
            onChange={(e) => setNewGroup(e.target.value)}
            placeholder="New group name"
            className="rounded-lg border border-border bg-surface-2 px-3 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-accent/40"
          />
          <button className="rounded-lg bg-accent text-accent-fg px-3 py-1.5 text-sm font-medium">
            Add group
          </button>
        </form>
      </section>

      {/* URLs table */}
      <section>
        <h2 className="text-sm uppercase tracking-wide text-muted mb-2">URLs ({tokens.length})</h2>
        <div className="rounded-lg border border-border overflow-x-auto">
          <table className="w-full text-sm min-w-[640px]">
            <thead>
              <tr className="text-left text-muted border-b border-border">
                <th className="px-3 py-2 font-medium">URL</th>
                <th className="px-3 py-2 font-medium">Description</th>
                <th className="px-3 py-2 font-medium">Group</th>
                <th className="px-3 py-2 font-medium">Last activity</th>
                <th className="px-3 py-2"></th>
              </tr>
            </thead>
            <tbody>
              {tokens.map((t) => (
                <tr key={t.uuid} className="border-b border-border last:border-0">
                  <td className="px-3 py-2">
                    <button
                      onClick={() => onOpenToken(t.uuid)}
                      className="font-mono text-accent hover:underline"
                    >
                      {t.alias || shortId(t.uuid)}
                    </button>
                  </td>
                  <td className="px-3 py-2 text-muted truncate max-w-[16rem]">
                    {t.description || '—'}
                  </td>
                  <td className="px-3 py-2">
                    <select
                      value={t.group_id ?? ''}
                      onChange={(e) => onAssignGroup(t.uuid, e.target.value)}
                      className="rounded-md border border-border bg-surface-2 px-2 py-1 text-xs"
                    >
                      <option value="">Ungrouped</option>
                      {groups.map((g) => (
                        <option key={g.id} value={g.id}>
                          {g.name}
                        </option>
                      ))}
                    </select>
                  </td>
                  <td className="px-3 py-2 text-muted">
                    {t.latest_request_at ? relativeTime(t.latest_request_at) : '—'}
                  </td>
                  <td className="px-3 py-2 text-right">
                    <button
                      onClick={() => onDeleteToken(t.uuid)}
                      className="p-1.5 rounded-lg hover:bg-err/10 text-err"
                      aria-label="Delete URL"
                    >
                      <TrashIcon width={15} height={15} />
                    </button>
                  </td>
                </tr>
              ))}
              {tokens.length === 0 && (
                <tr>
                  <td colSpan={5} className="px-3 py-6 text-center text-muted">
                    No URLs yet.
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </section>
    </div>
  )
}
