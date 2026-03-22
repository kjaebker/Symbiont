import { useState, type FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { setToken } from '@/api/client'

export default function Login() {
  const [tokenInput, setTokenInput] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const navigate = useNavigate()

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError('')
    setLoading(true)

    try {
      const res = await fetch('/api/system', {
        headers: { Authorization: `Bearer ${tokenInput.trim()}` },
      })

      if (res.ok) {
        setToken(tokenInput.trim())
        navigate('/')
      } else {
        setError('Invalid access token')
      }
    } catch {
      setError('Unable to reach the server')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-surface p-4">
      <div className="w-full max-w-sm">
        <div className="text-center mb-8">
          <div className="flex items-center justify-center gap-2 mb-3">
            <span className="h-3 w-3 rounded-full bg-primary animate-bio-pulse" />
            <h1 className="text-2xl font-bold text-on-surface tracking-tight">
              SYMBIONT
            </h1>
          </div>
          <p className="text-sm text-on-surface-dim uppercase tracking-widest">
            Subaquatic Controller Access Point
          </p>
        </div>

        <form
          onSubmit={handleSubmit}
          className="bg-surface-container rounded-2xl p-6 shadow-abyss space-y-5"
        >
          <div className="flex items-center gap-2 text-xs text-secondary uppercase tracking-widest font-medium">
            <span className="h-1.5 w-1.5 rounded-full bg-secondary" />
            Encrypted Channel Ready
          </div>

          <div>
            <label
              htmlFor="token"
              className="block text-xs text-on-surface-dim uppercase tracking-widest mb-2"
            >
              Access Credentials
            </label>
            <input
              id="token"
              type="password"
              value={tokenInput}
              onChange={(e) => setTokenInput(e.target.value)}
              placeholder="Enter access token"
              className="w-full bg-surface-container-highest rounded-xl px-4 py-3 text-on-surface placeholder:text-on-surface-faint outline-none focus:ring-1 focus:ring-primary/50 transition-fluid"
              autoFocus
            />
          </div>

          {error && (
            <p className="text-sm text-tertiary">{error}</p>
          )}

          <button
            type="submit"
            disabled={!tokenInput.trim() || loading}
            className="w-full py-3 rounded-xl font-semibold text-on-primary bg-gradient-to-r from-primary to-primary-container hover:shadow-glow-primary disabled:opacity-40 disabled:cursor-not-allowed transition-fluid"
          >
            {loading ? 'Connecting...' : 'Establish Connection'}
          </button>
        </form>
      </div>
    </div>
  )
}
