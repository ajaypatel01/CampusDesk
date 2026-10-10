import { useState } from 'react'
import { Mail } from 'lucide-react'
import { usersApi } from '../services/api'

// Parents add an email and confirm it with an emailed 6-digit code; only
// then is Change password shown.
function EmailVerify({ onVerified }) {
  const [email, setEmail] = useState('')
  const [code, setCode] = useState('')
  const [sent, setSent] = useState(false)
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState('')

  async function submit(e) {
    e.preventDefault(); setErr(''); setBusy(true)
    try {
      if (!sent) { await usersApi.requestEmailVerification(email); setSent(true) }
      else onVerified(await usersApi.confirmEmailVerification(email, code))
    } catch (e2) { setErr(e2.message) } finally { setBusy(false) }
  }

  return (
    <div style={{ marginTop: '24px' }}>
      <h3 style={{ display: 'flex', alignItems: 'center', gap: '8px', fontSize: '15px', marginBottom: '8px' }}><Mail size={16} /> Add your email</h3>
      <p className="empty-text">Add and verify your email to be able to change your password.</p>
      {err && <p className="sd-modal__err">{err}</p>}
      <form onSubmit={submit} style={{ display: 'grid', gap: '10px', maxWidth: '360px' }}>
        <label className="form-field"><span>Email</span>
          <input type="email" required value={email} disabled={sent} onChange={e => setEmail(e.target.value)} /></label>
        {sent && <label className="form-field"><span>6-digit code sent to your email</span>
          <input inputMode="numeric" maxLength={6} required value={code} onChange={e => setCode(e.target.value.replace(/\D/g, ''))} /></label>}
        <div style={{ display: 'flex', gap: '8px' }}>
          <button type="submit" className="btn btn--primary" disabled={busy}>{busy ? 'Please wait...' : sent ? 'Verify' : 'Send code'}</button>
          {sent && <button type="button" className="btn btn--outline" onClick={() => { setSent(false); setCode('') }}>Change email</button>}
        </div>
      </form>
    </div>
  )
}

export default EmailVerify
