import { useCallback, useEffect, useState } from 'react'
import { api, type CapturedRequest, type Token, type TokenInput } from './api'
import { copyText } from './lib'
import { useTheme } from './useTheme'
import { Navbar } from './components/Navbar'
import { TokenList } from './components/TokenList'
import { RequestList } from './components/RequestList'
import { RequestDetail } from './components/RequestDetail'
import { SettingsDialog } from './components/SettingsDialog'
import { CopyIcon, SettingsIcon, TrashIcon } from './components/icons'

const ACTIVE_KEY = 'raptor-active'

function initialActive(): string | null {
  const hash = window.location.hash.replace(/^#\/?/, '')
  if (hash) return hash
  return localStorage.getItem(ACTIVE_KEY)
}

export default function App() {
  const { theme, toggle } = useTheme()
  const [tokens, setTokens] = useState<Token[]>([])
  const [activeId, setActiveId] = useState<string | null>(initialActive)
  const [requests, setRequests] = useState<CapturedRequest[]>([])
  const [selected, setSelected] = useState<CapturedRequest | null>(null)
  const [showSettings, setShowSettings] = useState(false)
  const [sidebarOpen, setSidebarOpen] = useState(false)
  const [copied, setCopied] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const activeToken = tokens.find((t) => t.uuid === activeId) ?? null

  const loadTokens = useCallback(async () => {
    try {
      const list = await api.listTokens()
      setTokens(list)
      setActiveId((cur) => cur ?? list[0]?.uuid ?? null)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'failed to load')
    }
  }, [])

  useEffect(() => {
    void loadTokens()
  }, [loadTokens])

  const loadRequests = useCallback(async (id: string) => {
    try {
      const page = await api.listRequests(id, 1, 100)
      setRequests(page.data ?? [])
    } catch {
      /* transient */
    }
  }, [])

  // Live stream + polling fallback for the active token.
  useEffect(() => {
    if (!activeId) return
    localStorage.setItem(ACTIVE_KEY, activeId)
    window.location.hash = `/${activeId}`
    setSelected(null)
    void loadRequests(activeId)

    const es = new EventSource(api.streamURL(activeId))
    es.addEventListener('request', (e) => {
      const r = JSON.parse((e as MessageEvent).data) as CapturedRequest
      setRequests((prev) => (prev.some((x) => x.uuid === r.uuid) ? prev : [r, ...prev]))
      setTokens((prev) =>
        prev.map((t) => (t.uuid === activeId ? { ...t, latest_request_at: r.created_at } : t)),
      )
    })

    const poll = setInterval(() => void loadRequests(activeId), 60_000)
    return () => {
      es.close()
      clearInterval(poll)
    }
  }, [activeId, loadRequests])

  async function handleCreate() {
    try {
      const tok = await api.createToken({})
      setTokens((prev) => [tok, ...prev])
      setActiveId(tok.uuid)
      setSidebarOpen(false)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'create failed')
    }
  }

  function handleSelectToken(id: string) {
    setActiveId(id)
    setSidebarOpen(false)
  }

  async function handleSaveSettings(body: TokenInput) {
    if (!activeToken) return
    const updated = await api.updateToken(activeToken.uuid, body)
    setTokens((prev) => prev.map((t) => (t.uuid === updated.uuid ? updated : t)))
  }

  async function handleDeleteToken() {
    if (!activeToken) return
    await api.deleteToken(activeToken.uuid)
    setTokens((prev) => prev.filter((t) => t.uuid !== activeToken.uuid))
    setShowSettings(false)
    setActiveId(null)
    setRequests([])
  }

  async function handleDeleteRequest(rid: string) {
    if (!activeToken) return
    await api.deleteRequest(activeToken.uuid, rid)
    setRequests((prev) => prev.filter((r) => r.uuid !== rid))
    setSelected((s) => (s?.uuid === rid ? null : s))
  }

  async function handleClear() {
    if (!activeToken) return
    await api.clearRequests(activeToken.uuid)
    setRequests([])
    setSelected(null)
  }

  async function copyURL() {
    if (!activeToken) return
    await copyText(activeToken.url)
    setCopied(true)
    setTimeout(() => setCopied(false), 1500)
  }

  return (
    <div className="h-screen flex flex-col">
      <Navbar
        theme={theme}
        onToggleTheme={toggle}
        onNewToken={handleCreate}
        onToggleSidebar={() => setSidebarOpen((o) => !o)}
      />

      {error && (
        <div className="bg-err/10 text-err text-sm px-4 py-2 border-b border-err/20">{error}</div>
      )}

      <div className="flex-1 flex overflow-hidden relative">
        {/* Sidebar: persistent on desktop, drawer on mobile */}
        <aside
          className={`absolute md:static z-30 h-full w-72 shrink-0 bg-surface border-r border-border flex flex-col transition-transform md:translate-x-0 ${
            sidebarOpen ? 'translate-x-0 shadow-xl' : '-translate-x-full'
          }`}
        >
          <TokenList tokens={tokens} activeId={activeId} onSelect={handleSelectToken} />
        </aside>
        {sidebarOpen && (
          <div
            className="absolute inset-0 z-20 bg-black/40 md:hidden"
            onClick={() => setSidebarOpen(false)}
          />
        )}

        {/* Main content */}
        {activeToken ? (
          <main className="flex-1 flex flex-col min-w-0">
            <TokenBar
              token={activeToken}
              copied={copied}
              onCopy={copyURL}
              onSettings={() => setShowSettings(true)}
              onClear={handleClear}
            />
            <div className="flex-1 flex min-h-0">
              {/* request list */}
              <div
                className={`w-full md:w-80 lg:w-96 shrink-0 border-r border-border min-h-0 ${
                  selected ? 'hidden md:block' : 'block'
                }`}
              >
                <RequestList
                  requests={requests}
                  activeId={selected?.uuid ?? null}
                  onSelect={setSelected}
                />
              </div>
              {/* detail */}
              <div className={`flex-1 min-w-0 ${selected ? 'block' : 'hidden md:block'}`}>
                {selected ? (
                  <div className="h-full flex flex-col">
                    <button
                      className="md:hidden text-sm text-accent px-4 py-2 text-left border-b border-border"
                      onClick={() => setSelected(null)}
                    >
                      ← Back to requests
                    </button>
                    <div className="flex-1 min-h-0">
                      <RequestDetail
                        tokenId={activeToken.uuid}
                        request={selected}
                        onDelete={handleDeleteRequest}
                      />
                    </div>
                  </div>
                ) : (
                  <div className="hidden md:grid place-items-center h-full text-sm text-muted">
                    Select a request to inspect it
                  </div>
                )}
              </div>
            </div>
          </main>
        ) : (
          <main className="flex-1 grid place-items-center p-8 text-center">
            <div className="max-w-sm">
              <h2 className="text-lg font-semibold mb-2">No URL selected</h2>
              <p className="text-sm text-muted mb-4">
                Create a unique URL and every request sent to it is captured and shown here in real
                time.
              </p>
              <button
                onClick={handleCreate}
                className="rounded-lg bg-accent text-accent-fg px-4 py-2 text-sm font-medium"
              >
                Create your first URL
              </button>
            </div>
          </main>
        )}
      </div>

      {showSettings && activeToken && (
        <SettingsDialog
          token={activeToken}
          onClose={() => setShowSettings(false)}
          onSave={handleSaveSettings}
          onDelete={handleDeleteToken}
        />
      )}
    </div>
  )
}

function TokenBar({
  token,
  copied,
  onCopy,
  onSettings,
  onClear,
}: {
  token: Token
  copied: boolean
  onCopy: () => void
  onSettings: () => void
  onClear: () => void
}) {
  return (
    <div className="flex items-center gap-2 px-4 h-12 border-b border-border bg-surface shrink-0">
      <code className="font-mono text-xs sm:text-sm truncate text-text">{token.url}</code>
      <button
        onClick={onCopy}
        className="p-1.5 rounded-lg hover:bg-surface-2 text-muted shrink-0"
        aria-label="Copy URL"
      >
        <CopyIcon width={16} height={16} />
      </button>
      {copied && <span className="text-xs text-ok shrink-0">copied</span>}
      <div className="ml-auto flex items-center gap-1 shrink-0">
        <a
          href={api.csvURL(token.uuid)}
          className="text-xs px-2 py-1 rounded-lg hover:bg-surface-2 text-muted"
        >
          CSV
        </a>
        <button
          onClick={onClear}
          className="p-1.5 rounded-lg hover:bg-surface-2 text-muted"
          aria-label="Clear all requests"
        >
          <TrashIcon width={16} height={16} />
        </button>
        <button
          onClick={onSettings}
          className="p-1.5 rounded-lg hover:bg-surface-2 text-muted"
          aria-label="URL settings"
        >
          <SettingsIcon width={16} height={16} />
        </button>
      </div>
    </div>
  )
}
