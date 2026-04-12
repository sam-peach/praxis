import { FormEvent, useState } from 'react'
import { LogoWordmark } from './Logo'
import { colors, shadow, radius, font } from '../theme'

interface Props {
  onLogin: (username: string, password: string) => Promise<void>
}

export default function LoginPage({ onLogin }: Props) {
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [loading,  setLoading]  = useState(false)
  const [error,    setError]    = useState<string | null>(null)

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    setLoading(true)
    try {
      await onLogin(username, password)
    } catch {
      setError('Invalid username or password.')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="login-bg" style={overlay}>
      <div className="fade-up" style={card}>

        {/* Brand */}
        <div style={{ marginBottom: 32 }}>
          <LogoWordmark size={52} />
        </div>

        {error && (
          <div style={errorBox}>{error}</div>
        )}

        <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
          <label style={labelStyle}>
            Username
            <input
              className="field-input"
              type="text"
              value={username}
              onChange={e => setUsername(e.target.value)}
              autoComplete="username"
              required
              autoFocus
            />
          </label>
          <label style={labelStyle}>
            Password
            <input
              className="field-input"
              type="password"
              value={password}
              onChange={e => setPassword(e.target.value)}
              autoComplete="current-password"
              required
            />
          </label>
          <button type="submit" style={submitBtn} disabled={loading}>
            {loading ? 'Signing in…' : 'Sign in'}
          </button>
        </form>

      </div>
    </div>
  )
}

const overlay: React.CSSProperties = {
  minHeight:      '100vh',
  display:        'flex',
  alignItems:     'center',
  justifyContent: 'center',
  fontFamily:     font.body,
  padding:        24,
}

const card: React.CSSProperties = {
  background:   colors.surface,
  borderRadius: radius.xl,
  padding:      '40px 36px',
  width:        '100%',
  maxWidth:     380,
  boxShadow:    shadow.login,
}

const errorBox: React.CSSProperties = {
  background:   colors.errorBg,
  color:        colors.errorText,
  border:       `1px solid ${colors.errorBorder}`,
  padding:      '10px 14px',
  borderRadius: radius.md,
  fontSize:     14,
  marginBottom: 16,
}

const labelStyle: React.CSSProperties = {
  display:       'flex',
  flexDirection: 'column',
  gap:           6,
  fontSize:      13,
  fontWeight:    500,
  color:         colors.text,
}

const submitBtn: React.CSSProperties = {
  marginTop:     8,
  padding:       '11px',
  background:    colors.brand,
  color:         '#fff',
  border:        'none',
  borderRadius:  radius.md,
  cursor:        'pointer',
  fontSize:      14,
  fontWeight:    600,
  letterSpacing: '0.01em',
  fontFamily:    font.body,
}
