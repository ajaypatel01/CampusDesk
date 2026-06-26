import { useState, useEffect } from 'react'
import { Send, ChevronDown, ChevronRight, Users, CheckCircle, XCircle, AlertTriangle } from 'lucide-react'
import { useSchool } from '../services/SchoolContext'
import { useConfig } from '../services/ConfigContext'
import { broadcastsApi, academicApi } from '../services/api'
import './Broadcasts.css'

function Broadcasts() {
  const { currentSchool, currentYear } = useSchool()
  const { whatsapp_enabled } = useConfig()
  const [broadcasts, setBroadcasts] = useState([])
  const [loading, setLoading] = useState(true)
  const [grades, setGrades] = useState([])
  const [expandedId, setExpandedId] = useState(null)
  const [recipients, setRecipients] = useState({})

  const [form, setForm] = useState({
    title: '', message: '', target: 'all_parents',
    grade_level_id: '', phones: '', is_template: false,
    template_name: '', body_params: '',
  })
  const [sending, setSending] = useState(false)
  const [sendMsg, setSendMsg] = useState('')

  function loadBroadcasts() {
    if (!currentSchool) return
    broadcastsApi.list(currentSchool.id)
      .then(r => setBroadcasts(r.items || []))
      .catch(() => {})
      .finally(() => setLoading(false))
  }

  useEffect(() => {
    if (!currentSchool) return
    loadBroadcasts()
    academicApi.listGrades(currentSchool.id)
      .then(r => setGrades(r.items || []))
      .catch(() => {})
  }, [currentSchool])

  async function handleSend(e) {
    e.preventDefault()
    if (!currentSchool) return
    setSending(true); setSendMsg('')
    try {
      const body = {
        school_id: currentSchool.id,
        academic_year_id: currentYear?.id || '',
        title: form.title,
        message: form.message,
        target: form.target,
        grade_level_id: form.target === 'grade' ? form.grade_level_id : undefined,
        phones: form.target === 'manual' ? form.phones.split('\n').map(p => p.trim()).filter(Boolean) : undefined,
        is_template: form.is_template,
        template_name: form.is_template ? form.template_name : undefined,
        body_params: form.is_template && form.body_params ? form.body_params.split('\n').map(p => p.trim()).filter(Boolean) : undefined,
      }
      await broadcastsApi.send(body)
      setSendMsg('Broadcast queued — messages are being sent.')
      setForm({ title: '', message: '', target: 'all_parents', grade_level_id: '', phones: '', is_template: false, template_name: '', body_params: '' })
      setTimeout(loadBroadcasts, 2000)
    } catch (err) { setSendMsg('Error: ' + err.message) }
    setSending(false)
  }

  async function toggleExpand(id) {
    if (expandedId === id) { setExpandedId(null); return }
    setExpandedId(id)
    if (!recipients[id]) {
      try {
        const r = await broadcastsApi.listRecipients(id)
        setRecipients(prev => ({ ...prev, [id]: r.items || [] }))
      } catch {}
    }
  }

  function fmtDate(s) {
    return new Date(s).toLocaleString('en-IN', { day: '2-digit', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit' })
  }

  if (!currentSchool) return <p className="empty-text">Select a school first.</p>

  if (!whatsapp_enabled) {
    return (
      <div className="broadcasts-page">
        <div className="page-header"><div><h1>WhatsApp Broadcasts</h1></div></div>
        <div className="wa-disabled-banner">
          <AlertTriangle size={20} />
          <span>WhatsApp is not configured for this installation. Contact your administrator to set up the WhatsApp Business API credentials.</span>
        </div>
      </div>
    )
  }

  return (
    <div className="broadcasts-page">
      <div className="page-header">
        <div>
          <h1>WhatsApp Broadcasts</h1>
          <p className="page-subtitle">Send announcements to parents, staff, or specific groups</p>
        </div>
      </div>

      <div className="broadcasts-layout">
        {/* Compose Panel */}
        <div className="broadcast-compose">
          <h2>New Broadcast</h2>
          <form className="broadcast-form" onSubmit={handleSend}>
            <label className="form-field">
              <span>Title *</span>
              <input required value={form.title} onChange={e => setForm({ ...form, title: e.target.value })} placeholder="e.g. Holiday Notice" />
            </label>

            <label className="form-field">
              <span>Send To *</span>
              <select value={form.target} onChange={e => setForm({ ...form, target: e.target.value })}>
                <option value="all_parents">All Parents</option>
                <option value="staff">All Staff</option>
                <option value="grade">Grade / Class</option>
                <option value="manual">Manual Phone List</option>
              </select>
            </label>

            {form.target === 'grade' && (
              <label className="form-field">
                <span>Grade *</span>
                <select required value={form.grade_level_id} onChange={e => setForm({ ...form, grade_level_id: e.target.value })}>
                  <option value="">Select grade...</option>
                  {grades.map(g => <option key={g.id} value={g.id}>{g.name}</option>)}
                </select>
              </label>
            )}

            {form.target === 'manual' && (
              <label className="form-field">
                <span>Phone Numbers (one per line, with country code)</span>
                <textarea rows={4} value={form.phones} onChange={e => setForm({ ...form, phones: e.target.value })} placeholder={'919876543210\n918765432109'} />
              </label>
            )}

            <label className="broadcast-toggle">
              <input type="checkbox" checked={form.is_template} onChange={e => setForm({ ...form, is_template: e.target.checked })} />
              <span>Use Meta Template Message</span>
            </label>

            {form.is_template ? (
              <>
                <label className="form-field">
                  <span>Template Name *</span>
                  <input required value={form.template_name} onChange={e => setForm({ ...form, template_name: e.target.value })} placeholder="e.g. holiday_notice" />
                </label>
                <label className="form-field">
                  <span>Body Parameters (one per line, for {'{{'}'1{'}}'}, {'{{'}'2{'}}'}...)</span>
                  <textarea rows={3} value={form.body_params} onChange={e => setForm({ ...form, body_params: e.target.value })} placeholder={'Param 1\nParam 2'} />
                </label>
              </>
            ) : (
              <label className="form-field">
                <span>Message *</span>
                <textarea required rows={4} value={form.message} onChange={e => setForm({ ...form, message: e.target.value })} placeholder="Type your announcement here..." />
              </label>
            )}

            {sendMsg && <p className={`doc-msg ${sendMsg.startsWith('Error') ? 'doc-msg--error' : 'doc-msg--ok'}`}>{sendMsg}</p>}

            <button type="submit" className="btn btn--primary" disabled={sending}>
              <Send size={16} /> {sending ? 'Sending...' : 'Send Broadcast'}
            </button>
          </form>
        </div>

        {/* History Panel */}
        <div className="broadcast-history">
          <h2>Broadcast History</h2>
          {loading ? <p className="loading-text">Loading...</p> : (
            broadcasts.length === 0 ? <p className="empty-text">No broadcasts sent yet.</p> : (
              <div className="broadcast-list">
                {broadcasts.map(b => (
                  <div key={b.id} className="broadcast-item">
                    <div className="broadcast-item__header" onClick={() => toggleExpand(b.id)}>
                      <div className="broadcast-item__info">
                        <span className="broadcast-item__title">{b.title}</span>
                        <span className="broadcast-item__meta">
                          <Users size={12} /> {b.total_count} recipients · {fmtDate(b.created_at)}
                        </span>
                      </div>
                      <div className="broadcast-item__stats">
                        <span className="broadcast-stat broadcast-stat--ok"><CheckCircle size={12} /> {b.sent_count}</span>
                        {b.failed_count > 0 && <span className="broadcast-stat broadcast-stat--fail"><XCircle size={12} /> {b.failed_count}</span>}
                        <span className={`badge badge--${b.status === 'done' ? 'success' : b.status === 'failed' ? 'danger' : 'muted'}`}>{b.status}</span>
                        {expandedId === b.id ? <ChevronDown size={16} /> : <ChevronRight size={16} />}
                      </div>
                    </div>
                    <p className="broadcast-item__message">{b.message || (b.is_template ? `Template: ${b.template_name}` : '')}</p>

                    {expandedId === b.id && (
                      <div className="broadcast-recipients">
                        {!recipients[b.id] ? <p className="loading-text">Loading...</p> : (
                          recipients[b.id].length === 0 ? <p className="empty-text">No recipient data.</p> : (
                            <table className="data-table">
                              <thead><tr><th>Phone</th><th>Name</th><th>Status</th><th>Error</th></tr></thead>
                              <tbody>
                                {recipients[b.id].map(rec => (
                                  <tr key={rec.id}>
                                    <td className="data-table__muted">{rec.phone}</td>
                                    <td>{rec.name || '-'}</td>
                                    <td><span className={`badge badge--${rec.status === 'sent' ? 'success' : 'danger'}`}>{rec.status}</span></td>
                                    <td className="data-table__muted">{rec.error_message || '-'}</td>
                                  </tr>
                                ))}
                              </tbody>
                            </table>
                          )
                        )}
                      </div>
                    )}
                  </div>
                ))}
              </div>
            )
          )}
        </div>
      </div>
    </div>
  )
}

export default Broadcasts
