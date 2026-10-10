import { useState } from 'react'
import { KeyRound } from 'lucide-react'
import { usersApi, setToken } from '../services/api'

// Change your own password. Every other session is logged out; this one
// keeps working with the token the server sends back.
function ChangePassword() {
  const [form, setForm] = useState({ current: '', next: '', confirm: '' })
  const [saving, setSaving] = useState(false)
  const [msg, setMsg] = useState('')
  const [err, setErr] = useState('')

  async function submit(e) {
    e.preventDefault()
    setMsg(''); setErr('')
    if (form.next !== form.confirm) { setErr('The new passwords do not match'); return }
    setSaving(true)
    try {
      const res = await usersApi.changePassword(form.current, form.next)
      setToken(res.token)
      setForm({ current: '', next: '', confirm: '' })
      setMsg('Password changed. You have been signed out on other devices.')
    } catch (e2) {
      setErr(e2.message || 'Could not change the password')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div style={{ marginTop: '24px' }}>
      <h3 style={{ display: 'flex', alignItems: 'center', gap: '8px', fontSize: '15px', marginBottom: '8px' }}><KeyRound size={16} /> Change Password</h3>
      {err && <p className="sd-modal__err">{err}</p>}
      {msg && <p className="empty-text" style={{ color: 'var(--success)' }}>{msg}</p>}
      <form onSubmit={submit} style={{ display: 'grid', gap: '10px', maxWidth: '360px' }}>
        <label className="form-field"><span>Current password</span>
          <input type="password" autoComplete="current-password" required value={form.current} onChange={e => setForm({ ...form, current: e.target.value })} /></label>
        <label className="form-field"><span>New password (at least 6 characters)</span>
          <input type="password" autoComplete="new-password" required minLength={6} value={form.next} onChange={e => setForm({ ...form, next: e.target.value })} /></label>
        <label className="form-field"><span>Confirm new password</span>
          <input type="password" autoComplete="new-password" required minLength={6} value={form.confirm} onChange={e => setForm({ ...form, confirm: e.target.value })} /></label>
        <div><button type="submit" className="btn btn--primary" disabled={saving}>{saving ? 'Saving...' : 'Change password'}</button></div>
      </form>
    </div>
  )
}

export default ChangePassword
