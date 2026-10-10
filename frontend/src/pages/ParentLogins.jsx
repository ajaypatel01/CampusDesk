import { useEffect, useMemo, useState } from 'react'
import { KeyRound, RotateCcw, Search } from 'lucide-react'
import { useSchool } from '../services/SchoolContext'
import { parentLoginsApi } from '../services/api'

// Admins set or reset any parent's password. A parent login is a mobile
// number; until the parent (or an admin) sets a password it is any linked
// child's first name + "@123".
function ParentLogins() {
  const { currentSchool } = useSchool()
  const [items, setItems] = useState([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const [q, setQ] = useState('')
  const [editing, setEditing] = useState(null)
  const [password, setPassword] = useState('')
  const [saving, setSaving] = useState(false)

  const load = () => {
    if (!currentSchool) return
    setLoading(true)
    parentLoginsApi.list({ school_id: currentSchool.id })
      .then(r => setItems(r.items || []))
      .catch(e => setError(e.message))
      .finally(() => setLoading(false))
  }
  useEffect(load, [currentSchool])

  const shown = useMemo(() => {
    const t = q.trim().toLowerCase()
    if (!t) return items
    return items.filter(p => p.phone.includes(t.replace(/\D/g, '') || '~') ||
      p.guardians.some(g => g.toLowerCase().includes(t)) ||
      p.children.some(c => c.name.toLowerCase().includes(t) || c.student_code.toLowerCase().includes(t)))
  }, [items, q])

  const save = async (p, reset) => {
    setSaving(true); setError(''); setNotice('')
    try {
      await parentLoginsApi.setPassword({ school_id: currentSchool.id, phone: p.phone, password: reset ? undefined : password, reset })
      setNotice(reset
        ? `${p.phone}: password reset to the default (child's first name@123).`
        : `${p.phone}: new password set. Share it with the parent.`)
      setEditing(null); setPassword('')
      load()
    } catch (e) {
      setError(e.message)
    } finally {
      setSaving(false)
    }
  }

  if (!currentSchool) return <p className="empty-text">Select a school first.</p>

  return (
    <div>
      <div className="page-header">
        <div>
          <h1>Parent Logins</h1>
          <p className="page-subtitle">Parents log in with their mobile number. Set a new password or reset it to the default (child&apos;s first name@123).</p>
        </div>
      </div>

      <div className="page-filters">
        <label className="filter-select" style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
          <Search size={14} />
          <input value={q} onChange={e => setQ(e.target.value)} placeholder="Mobile, parent or child name, student code" style={{ minWidth: '260px' }} />
        </label>
      </div>

      <div className="page-count">{loading ? 'Loading...' : `${shown.length} number${shown.length === 1 ? '' : 's'}`}</div>
      {error && <p className="doc-msg doc-msg--error">{error}</p>}
      {notice && <p className="doc-msg doc-msg--ok">{notice}</p>}

      <div className="table-card">
        <table className="data-table">
          <thead><tr><th>Mobile</th><th>Children</th><th>Parent</th><th>Password</th><th></th></tr></thead>
          <tbody>
            {!loading && shown.length === 0 ? (
              <tr><td colSpan={5} className="data-table__empty">No parent numbers found</td></tr>
            ) : shown.map(p => (
              <tr key={p.phone}>
                <td>{p.phone}</td>
                <td>{p.children.map(c => (
                  <div key={c.student_code}>{c.name} <span className="data-table__muted">{c.student_code}{c.class ? ` · ${c.class}` : ''}</span></div>
                ))}</td>
                <td className="data-table__muted">{p.guardians.join(', ') || '—'}{p.email && <div>{p.email}</div>}</td>
                <td>
                  {p.own_password
                    ? <span className="badge badge--info">Changed</span>
                    : <span className="badge badge--muted">Default</span>}
                  {!p.has_login && <div className="data-table__muted">Not logged in yet</div>}
                </td>
                <td style={{ whiteSpace: 'nowrap' }}>
                  <button className="btn btn--outline btn--sm" onClick={() => { setEditing(p); setPassword(''); setError(''); setNotice('') }} title="Set password"><KeyRound size={13} /> Set</button>
                  {p.own_password && (
                    <button className="btn btn--outline btn--sm" style={{ marginLeft: '6px' }} disabled={saving} onClick={() => save(p, true)} title="Reset to default"><RotateCcw size={13} /> Default</button>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {editing && (
        <div className="modal-overlay" onClick={() => setEditing(null)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <h2>Set password for {editing.phone}</h2>
            <p className="data-table__muted">{editing.children.map(c => c.name).join(', ')}. The parent is logged out on all devices and must use this password from now on.</p>
            <form className="modal__form" onSubmit={e => { e.preventDefault(); save(editing, false) }}>
              <label className="form-field"><span>New password *</span>
                <input required minLength={6} value={password} onChange={e => setPassword(e.target.value)} autoComplete="new-password" />
              </label>
              {error && <p className="doc-msg doc-msg--error">{error}</p>}
              <div className="modal__actions">
                <button type="button" className="btn btn--outline" onClick={() => setEditing(null)}>Cancel</button>
                <button type="submit" className="btn btn--primary" disabled={saving}>{saving ? 'Saving...' : 'Set password'}</button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}

export default ParentLogins
