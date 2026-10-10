import { useEffect, useRef, useState } from 'react'
import { FolderOpen, Upload, Download, Trash2 } from 'lucide-react'
import { studentDocsApi } from '../services/api'
import { formatDate } from '../utils/date'

const TYPES = [
  ['bank_passbook', 'Bank Passbook'],
  ['aadhar_card', 'Aadhaar Card'],
  ['samagra_id', 'Samagra ID'],
  ['caste_certificate', 'Caste Certificate'],
  ['birth_certificate', 'Birth Certificate'],
  ['income_certificate', 'Income Certificate'],
  ['domicile_certificate', 'Domicile Certificate'],
  ['transfer_certificate', 'Transfer Certificate'],
  ['marksheet', 'Marksheet'],
  ['photo', 'Photo'],
  ['other', 'Other'],
]

// Details asked for per type; these are saved to the student's profile too.
const FIELDS = {
  bank_passbook: [
    ['bank_account_number', 'Account number', true],
    ['bank_ifsc', 'IFSC', true],
    ['bank_name', 'Bank name'],
    ['bank_branch', 'Branch'],
    ['bank_holder_name', 'Account holder'],
  ],
  aadhar_card: [['aadhar_number', 'Aadhaar number', true]],
  samagra_id: [['samagra_id', 'Samagra ID', true]],
  caste_certificate: [['caste', 'Caste']],
  birth_certificate: [['date_of_birth', 'Date of birth', false, 'date']],
}

// A short, safe summary of what was saved with a document.
function summary(d) {
  const m = d.meta || {}
  if (d.doc_type === 'bank_passbook') return [m.name, m.ifsc, m.account_last4 && `A/c ••••${m.account_last4}`].filter(Boolean).join(' · ')
  if (d.doc_type === 'aadhar_card') return m.aadhaar_last4 ? `•••• •••• ${m.aadhaar_last4}` : ''
  return m.samagra_id || m.caste || m.date_of_birth || d.title || ''
}

// Scans/photos of a student's papers, with the details that go on the
// student's profile (passbook, Aadhaar, Samagra, caste, birth certificate).
function StudentDocuments({ studentId, student, onProfileChanged }) {
  const [docs, setDocs] = useState([])
  const [type, setType] = useState('bank_passbook')
  const [values, setValues] = useState({})
  const [title, setTitle] = useState('')
  const [busy, setBusy] = useState(false)
  const [msg, setMsg] = useState('')
  const [err, setErr] = useState('')
  const file = useRef(null)

  const load = () => studentDocsApi.list(studentId).then(r => setDocs(r.items || [])).catch(e => setErr(e.message))
  useEffect(() => { load() }, [studentId]) // eslint-disable-line react-hooks/exhaustive-deps

  // Pre-fill the details from the profile whenever the type changes.
  useEffect(() => {
    const v = {}
    for (const [k] of FIELDS[type] || []) v[k] = student?.[k] ? String(student[k]).slice(0, k === 'date_of_birth' ? 10 : undefined) : ''
    setValues(v)
  }, [type, student])

  async function upload(e) {
    e.preventDefault(); setErr(''); setMsg('')
    const f = file.current?.files?.[0]
    if (!f) { setErr('Choose a file'); return }
    setBusy(true)
    try {
      const res = await studentDocsApi.upload(studentId, type, f, values, title)
      setMsg(res.updated_fields?.length ? 'Uploaded; the student\'s details were updated.' : 'Uploaded.')
      if (file.current) file.current.value = ''
      setTitle('')
      load()
      if (res.updated_fields?.length) onProfileChanged()
    } catch (e2) { setErr(e2.message) } finally { setBusy(false) }
  }

  async function remove(d) {
    if (!window.confirm(`Delete this ${d.doc_type_label}?`)) return
    try { await studentDocsApi.remove(d.id); load() } catch (e) { setErr(e.message) }
  }

  return (
    <div className="detail-card">
      <h3 style={{ display: 'flex', alignItems: 'center', gap: '6px' }}><FolderOpen size={16} /> Documents</h3>

      <form onSubmit={upload} style={{ display: 'grid', gap: '10px', marginBottom: '14px' }}>
        <label className="form-field" style={{ margin: 0 }}><span>Document</span>
          <select value={type} onChange={e => setType(e.target.value)}>
            {TYPES.map(([v, l]) => <option key={v} value={v}>{l}</option>)}
          </select>
        </label>
        {(FIELDS[type] || []).map(([k, label, required, kind]) => (
          <label key={k} className="form-field" style={{ margin: 0 }}><span>{label}{required ? ' *' : ''}</span>
            <input type={kind || 'text'} required={!!required} value={values[k] || ''}
              onChange={e => setValues({ ...values, [k]: k === 'bank_ifsc' ? e.target.value.toUpperCase() : e.target.value })} />
          </label>
        ))}
        {!FIELDS[type] && (
          <label className="form-field" style={{ margin: 0 }}><span>Title / note</span>
            <input value={title} maxLength={120} onChange={e => setTitle(e.target.value)} /></label>
        )}
        <label className="form-field" style={{ margin: 0 }}><span>File (PDF or photo, up to 10 MB)</span>
          <input ref={file} type="file" accept="application/pdf,image/*,.heic" /></label>
        {err && <p className="sd-modal__err" style={{ margin: 0 }}>{err}</p>}
        {msg && <p className="empty-text" style={{ color: 'var(--success)', margin: 0 }}>{msg}</p>}
        <div><button className="btn btn--primary btn--sm" disabled={busy}><Upload size={14} /> {busy ? 'Uploading...' : 'Upload'}</button></div>
      </form>

      {docs.length === 0 ? <p className="empty-text">No documents uploaded yet.</p> : (
        <ul style={{ listStyle: 'none', margin: 0, padding: 0, display: 'flex', flexDirection: 'column', gap: '8px' }}>
          {docs.map(d => (
            <li key={d.id} style={{ display: 'flex', alignItems: 'center', gap: '10px', fontSize: '0.875rem' }}>
              <span className="badge badge--muted">{d.doc_type_label}</span>
              <span style={{ flex: 1, minWidth: 0 }}>
                {summary(d) || d.file_name}
                <span className="data-table__muted" style={{ display: 'block', fontSize: '0.78rem' }}>
                  {formatDate(d.created_at)}{d.uploaded_by_name ? ` · ${d.uploaded_by_name}` : ''}
                </span>
              </span>
              {d.url && <a className="btn btn--outline btn--sm" href={d.url} target="_blank" rel="noreferrer" aria-label={`Open ${d.doc_type_label}`}><Download size={14} /></a>}
              <button className="btn btn--outline btn--sm" onClick={() => remove(d)} aria-label={`Delete ${d.doc_type_label}`}><Trash2 size={14} /></button>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}

export default StudentDocuments
