import { useState } from 'react'
import { api } from '../api'
import { BoltIcon } from './icons'

interface Props {
  bootstrapped: boolean
  allowRegistration: boolean
  onAuthed: () => void
  onClose?: () => void // when set, render as a dismissible modal
}

const field =
  'w-full rounded-lg border border-border bg-surface-2 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-accent/40'

// Login handles sign-in and registration. When the instance has no users yet it
// forces "create admin"; otherwise it offers a login form with a toggle to
// register (when registration is enabled).
export function Login({ bootstrapped, allowRegistration, onAuthed, onClose }: Props) {
  const firstRun = !bootstrapped
  const [mode, setMode] = useState<'login' | 'register'>(firstRun ? 'register' : 'login')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const registering = mode === 'register'

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    setBusy(true)
    setError(null)
    try {
      if (registering) await api.register(email, password)
      else await api.login(email, password)
      onAuthed()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'failed')
    } finally {
      setBusy(false)
    }
  }

  const card = (
    <form
      onSubmit={submit}
      className="w-full max-w-sm rounded-xl border border-border bg-surface shadow-xl p-6 space-y-4"
      onClick={(e) => e.stopPropagation()}
    >
      <div className="flex items-center gap-2 justify-center">
        <span className="grid place-items-center w-9 h-9 rounded-lg bg-accent text-accent-fg">
          <BoltIcon width={20} height={20} />
        </span>
        <span className="text-xl font-semibold tracking-tight">Raptor</span>
      </div>
      <h1 className="text-center text-sm text-muted">
        {firstRun
          ? 'Create the first admin account'
          : registering
            ? 'Create an account'
            : 'Sign in'}
      </h1>

      <div>
        <label className="text-xs uppercase tracking-wide text-muted">Email</label>
        <input
          className={field}
          type="email"
          autoComplete="username"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          autoFocus
          required
        />
      </div>
      <div>
        <label className="text-xs uppercase tracking-wide text-muted">Password</label>
        <input
          className={field}
          type="password"
          autoComplete={registering ? 'new-password' : 'current-password'}
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          required
        />
        {registering && <p className="text-xs text-muted mt-1">At least 8 characters.</p>}
      </div>

      {error && <div className="text-sm text-err">{error}</div>}

      <button
        type="submit"
        disabled={busy}
        className="w-full rounded-lg bg-accent text-accent-fg px-4 py-2 text-sm font-medium disabled:opacity-50"
      >
        {busy ? 'Please wait…' : firstRun ? 'Create admin' : registering ? 'Register' : 'Sign in'}
      </button>

      {/* Toggle between sign-in and register (not during first-run bootstrap). */}
      {!firstRun && (
        <div className="text-center text-xs text-muted">
          {registering ? (
            <button type="button" className="hover:text-text" onClick={() => setMode('login')}>
              Already have an account? Sign in
            </button>
          ) : allowRegistration ? (
            <button type="button" className="hover:text-text" onClick={() => setMode('register')}>
              Need an account? Register
            </button>
          ) : null}
        </div>
      )}

      {onClose && (
        <button
          type="button"
          onClick={onClose}
          className="w-full text-center text-xs text-muted hover:text-text"
        >
          Continue without an account
        </button>
      )}
    </form>
  )

  if (onClose) {
    return (
      <div className="fixed inset-0 z-50 grid place-items-center bg-black/40 p-4" onClick={onClose}>
        {card}
      </div>
    )
  }
  return <div className="min-h-screen grid place-items-center bg-bg p-4">{card}</div>
}
