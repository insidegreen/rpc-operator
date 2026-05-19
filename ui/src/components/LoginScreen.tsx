import { useState } from 'react'
import { toast } from 'sonner'
import { whoami } from '../api'
import { parseKubeconfigToken, setToken, clearToken } from '../auth'

export function LoginScreen({
  onLoggedIn,
  onCancel,
  oidcEnabled,
}: {
  onLoggedIn: () => void
  /** F44: when set, renders a Cancel button that calls clearToken() + onCancel().
   *  Used in Mode C (anonymous read-only) to return to the read-only view without
   *  forcing the user to authenticate. Mode B (auth strict) omits it. */
  onCancel?: () => void
  /** F20b: when true, renders a prominent "Log in with SSO" button that
   *  redirects to /api/v1/auth/login (the PKCE flow). Token-paste stays
   *  available as a fallback for SA tokens, CI/CD bots, and IdP outages. */
  oidcEnabled?: boolean
}) {
  const [tokenText, setTokenText] = useState('')
  const [busy, setBusy] = useState(false)

  async function tryLogin(rawToken: string) {
    const trimmed = rawToken.trim()
    if (!trimmed) {
      toast.error('Token is empty')
      return
    }
    setBusy(true)
    setToken(trimmed)
    try {
      const r = await whoami()
      toast.success(`Logged in as ${r.user.name}`)
      onLoggedIn()
    } catch (e) {
      clearToken()
      toast.error('Login failed: ' + (e as Error).message)
    } finally {
      setBusy(false)
    }
  }

  async function onFile(file: File) {
    const text = await file.text()
    const r = parseKubeconfigToken(text)
    if ('error' in r) {
      toast.error('Kubeconfig error: ' + r.error)
      return
    }
    setTokenText(r.token)
    await tryLogin(r.token)
  }

  return (
    <div style={containerStyle}>
      <h2 style={{ margin: 0 }}>Login</h2>
      <p style={{ color: '#666', fontSize: 14, margin: '8px 0 24px' }}>
        {oidcEnabled
          ? 'Log in with single sign-on, or paste a Bearer token below as a fallback.'
          : 'Paste a Bearer token or upload a Kubeconfig file (only the token will be extracted; certificate-based auth is not supported).'}
      </p>

      {oidcEnabled && (
        <>
          <button
            onClick={() => { window.location.href = '/api/v1/auth/login' }}
            disabled={busy}
            style={ssoBtnStyle}
          >
            Log in with SSO
          </button>
          <div style={dividerStyle}>or use a Bearer token</div>
        </>
      )}

      <label style={{ fontSize: 13, color: '#444' }}>Bearer Token</label>
      <textarea
        value={tokenText}
        onChange={e => setTokenText(e.target.value)}
        placeholder="eyJhbGciOiJSUzI1NiIs…"
        style={textareaStyle}
        rows={6}
      />

      <div style={{ display: 'flex', gap: 12, marginTop: 12, alignItems: 'center' }}>
        <button onClick={() => tryLogin(tokenText)} disabled={busy} style={primaryBtn}>
          Log in
        </button>
        <label style={fileLabel}>
          Upload Kubeconfig
          <input
            type="file"
            accept=".yaml,.yml,.conf,application/yaml,text/yaml"
            onChange={e => e.target.files?.[0] && onFile(e.target.files[0])}
            style={{ display: 'none' }}
          />
        </label>
        {onCancel && (
          <button
            onClick={() => { clearToken(); onCancel() }}
            disabled={busy}
            style={cancelBtnStyle}
          >
            Back to read-only view
          </button>
        )}
      </div>
    </div>
  )
}

const containerStyle: React.CSSProperties = {
  maxWidth: 600,
  margin: '60px auto',
  padding: 24,
  border: '1px solid #ddd',
  borderRadius: 8,
  background: '#fff',
}
const textareaStyle: React.CSSProperties = {
  width: '100%',
  fontFamily: 'monospace',
  fontSize: 12,
  padding: 8,
  marginTop: 4,
}
const primaryBtn: React.CSSProperties = {
  padding: '8px 16px',
  background: '#3b82f6',
  color: '#fff',
  border: 'none',
  borderRadius: 4,
  cursor: 'pointer',
}
const fileLabel: React.CSSProperties = {
  padding: '8px 16px',
  background: '#eee',
  borderRadius: 4,
  cursor: 'pointer',
  fontSize: 14,
}
const cancelBtnStyle: React.CSSProperties = {
  padding: '8px 16px',
  background: 'transparent',
  color: '#666',
  border: '1px solid #ccc',
  borderRadius: 4,
  cursor: 'pointer',
  fontSize: 14,
  marginLeft: 'auto',
}
const ssoBtnStyle: React.CSSProperties = {
  display: 'block',
  width: '100%',
  padding: '12px 16px',
  background: '#3b82f6',
  color: '#fff',
  border: 'none',
  borderRadius: 4,
  cursor: 'pointer',
  fontSize: 15,
  fontWeight: 500,
  marginBottom: 16,
}
const dividerStyle: React.CSSProperties = {
  textAlign: 'center',
  color: '#999',
  fontSize: 12,
  margin: '8px 0 16px',
  textTransform: 'uppercase',
  letterSpacing: 1,
}
