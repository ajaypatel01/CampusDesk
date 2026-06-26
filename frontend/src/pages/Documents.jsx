import { useState, useEffect } from 'react'
import { FileText, Download, Mail, MessageCircle } from 'lucide-react'
import { useSchool } from '../services/SchoolContext'
import { useConfig } from '../services/ConfigContext'
import { studentsApi, usersApi, documentsApi, academicApi } from '../services/api'
import './Documents.css'

function downloadBlob(blob, filename) {
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  a.click()
  URL.revokeObjectURL(url)
}

function Documents() {
  const { currentSchool, currentYear } = useSchool()
  const { whatsapp_enabled } = useConfig()
  const [tab, setTab] = useState('bonafide')

  // shared state
  const [students, setStudents] = useState([])
  const [staff, setStaff] = useState([])

  // bonafide state
  const [bStudentId, setBStudentId] = useState('')
  const [bEmail, setBEmail] = useState('')
  const [bPhone, setBPhone] = useState('')
  const [bDelivery, setBDelivery] = useState('download')
  const [bBusy, setBBusy] = useState(false)
  const [bMsg, setBMsg] = useState('')

  // TC state
  const [tcStudentId, setTcStudentId] = useState('')
  const [tcDate, setTcDate] = useState('')
  const [tcReason, setTcReason] = useState('')
  const [tcConduct, setTcConduct] = useState('Good')
  const [tcEmail, setTcEmail] = useState('')
  const [tcPhone, setTcPhone] = useState('')
  const [tcDelivery, setTcDelivery] = useState('download')
  const [tcBusy, setTcBusy] = useState(false)
  const [tcMsg, setTcMsg] = useState('')

  // Salary slip state
  const [ssUserId, setSsUserId] = useState('')
  const [ssMonth, setSsMonth] = useState('')
  const [ssYear, setSsYear] = useState(new Date().getFullYear())
  const [ssBasic, setSsBasic] = useState('')
  const [ssHra, setSsHra] = useState('')
  const [ssConveyance, setSsConveyance] = useState('')
  const [ssOtherAllow, setSsOtherAllow] = useState('')
  const [ssPf, setSsPf] = useState('')
  const [ssTds, setSsTds] = useState('')
  const [ssOtherDeduct, setSsOtherDeduct] = useState('')
  const [ssEmail, setSsEmail] = useState('')
  const [ssPhone, setSsPhone] = useState('')
  const [ssDelivery, setSsDelivery] = useState('download')
  const [ssBusy, setSsBusy] = useState(false)
  const [ssMsg, setSsMsg] = useState('')

  useEffect(() => {
    if (!currentSchool || !currentYear) return
    studentsApi.list({ school_id: currentSchool.id, limit: 500 })
      .then(r => setStudents(r.items || []))
      .catch(() => {})
    usersApi.list({ school_id: currentSchool.id })
      .then(r => setStaff(r.items || []))
      .catch(() => {})
  }, [currentSchool, currentYear])

  // --- Bonafide ---
  async function handleBonafide(e) {
    e.preventDefault()
    if (!bStudentId || !currentYear) return
    setBBusy(true); setBMsg('')
    try {
      if (bDelivery === 'download') {
        const blob = await documentsApi.downloadBonafide(bStudentId, currentYear.id)
        downloadBlob(blob, `bonafide_${bStudentId.slice(0,8)}.pdf`)
        setBMsg('Downloaded successfully.')
      } else if (bDelivery === 'email') {
        await documentsApi.emailBonafide({ student_id: bStudentId, academic_year_id: currentYear.id, recipient_email: bEmail, recipient_name: '' })
        setBMsg('Email sent successfully.')
      } else {
        await documentsApi.whatsappBonafide({ student_id: bStudentId, academic_year_id: currentYear.id, phone: bPhone })
        setBMsg('Sent via WhatsApp.')
      }
    } catch (err) { setBMsg('Error: ' + err.message) }
    setBBusy(false)
  }

  // --- TC ---
  async function handleTC(e) {
    e.preventDefault()
    if (!tcStudentId) return
    setTcBusy(true); setTcMsg('')
    try {
      const params = { student_id: tcStudentId }
      if (tcDate) params.date_of_leaving = tcDate
      if (tcReason) params.reason = tcReason
      if (tcConduct) params.conduct = tcConduct
      if (tcDelivery === 'download') {
        const blob = await documentsApi.downloadTC(params)
        downloadBlob(blob, `tc_${tcStudentId.slice(0,8)}.pdf`)
        setTcMsg('Downloaded successfully.')
      } else if (tcDelivery === 'email') {
        await documentsApi.emailTC({ ...params, recipient_email: tcEmail, recipient_name: '' })
        setTcMsg('Email sent successfully.')
      } else {
        await documentsApi.whatsappTC({ ...params, phone: tcPhone })
        setTcMsg('Sent via WhatsApp.')
      }
    } catch (err) { setTcMsg('Error: ' + err.message) }
    setTcBusy(false)
  }

  // --- Salary Slip ---
  async function handleSalarySlip(e) {
    e.preventDefault()
    const user = staff.find(u => u.id === ssUserId)
    const body = {
      user_id: ssUserId,
      employee_name: user ? `${user.first_name} ${user.last_name}` : '',
      designation: user?.role || '',
      department: user?.department || '',
      month: ssMonth,
      year: parseInt(ssYear, 10),
      basic_salary: parseInt(ssBasic, 10) || 0,
      hra: parseInt(ssHra, 10) || 0,
      conveyance_allowance: parseInt(ssConveyance, 10) || 0,
      other_allowances: parseInt(ssOtherAllow, 10) || 0,
      pf_deduction: parseInt(ssPf, 10) || 0,
      tds_deduction: parseInt(ssTds, 10) || 0,
      other_deductions: parseInt(ssOtherDeduct, 10) || 0,
    }
    setSsBusy(true); setSsMsg('')
    try {
      if (ssDelivery === 'download') {
        const blob = await documentsApi.downloadSalarySlip(body)
        downloadBlob(blob, `salary_${ssMonth}_${ssYear}.pdf`)
        setSsMsg('Downloaded successfully.')
      } else if (ssDelivery === 'email') {
        await documentsApi.emailSalarySlip({ ...body, recipient_email: ssEmail, recipient_name: body.employee_name })
        setSsMsg('Email sent successfully.')
      } else {
        await documentsApi.whatsappSalarySlip({ ...body, phone: ssPhone })
        setSsMsg('Sent via WhatsApp.')
      }
    } catch (err) { setSsMsg('Error: ' + err.message) }
    setSsBusy(false)
  }

  const months = ['January','February','March','April','May','June','July','August','September','October','November','December']

  if (!currentSchool || !currentYear) {
    return <p className="empty-text">Select a school and academic year first.</p>
  }

  return (
    <div className="documents-page">
      <div className="page-header">
        <div>
          <h1>Documents</h1>
          <p className="page-subtitle">Generate certificates, TC, and salary slips</p>
        </div>
      </div>

      <div className="docs-tabs">
        {[['bonafide','Bonafide Certificate'],['tc','Transfer Certificate'],['salary','Salary Slip']].map(([key, label]) => (
          <button key={key} className={`docs-tab ${tab === key ? 'docs-tab--active' : ''}`} onClick={() => setTab(key)}>
            <FileText size={16} /> {label}
          </button>
        ))}
      </div>

      {tab === 'bonafide' && (
        <div className="doc-card">
          <h2>Bonafide Certificate</h2>
          <form className="doc-form" onSubmit={handleBonafide}>
            <label className="form-field">
              <span>Student *</span>
              <select required value={bStudentId} onChange={e => setBStudentId(e.target.value)}>
                <option value="">Select student...</option>
                {students.map(s => <option key={s.id} value={s.id}>{s.first_name} {s.last_name} ({s.student_code})</option>)}
              </select>
            </label>
            <div className="doc-delivery">
              {[['download','Download'],['email','Email'],['whatsapp','WhatsApp']].filter(([v]) => v !== 'whatsapp' || whatsapp_enabled).map(([v, l]) => (
                <label key={v} className={`delivery-option ${bDelivery === v ? 'delivery-option--active' : ''}`}>
                  <input type="radio" name="b-delivery" value={v} checked={bDelivery === v} onChange={() => setBDelivery(v)} />
                  {v === 'download' ? <Download size={14} /> : v === 'email' ? <Mail size={14} /> : <MessageCircle size={14} />}
                  {l}
                </label>
              ))}
            </div>
            {bDelivery === 'email' && (
              <label className="form-field">
                <span>Recipient Email *</span>
                <input type="email" required value={bEmail} onChange={e => setBEmail(e.target.value)} placeholder="parent@example.com" />
              </label>
            )}
            {bDelivery === 'whatsapp' && (
              <label className="form-field">
                <span>WhatsApp Number * (with country code, e.g. 919876543210)</span>
                <input required value={bPhone} onChange={e => setBPhone(e.target.value)} placeholder="919876543210" />
              </label>
            )}
            {bMsg && <p className={`doc-msg ${bMsg.startsWith('Error') ? 'doc-msg--error' : 'doc-msg--ok'}`}>{bMsg}</p>}
            <button type="submit" className="btn btn--primary" disabled={bBusy}>{bBusy ? 'Processing...' : 'Generate'}</button>
          </form>
        </div>
      )}

      {tab === 'tc' && (
        <div className="doc-card">
          <h2>Transfer Certificate</h2>
          <form className="doc-form" onSubmit={handleTC}>
            <label className="form-field">
              <span>Student *</span>
              <select required value={tcStudentId} onChange={e => setTcStudentId(e.target.value)}>
                <option value="">Select student...</option>
                {students.map(s => <option key={s.id} value={s.id}>{s.first_name} {s.last_name} ({s.student_code})</option>)}
              </select>
            </label>
            <div className="form-row">
              <label className="form-field">
                <span>Date of Leaving</span>
                <input type="date" value={tcDate} onChange={e => setTcDate(e.target.value)} />
              </label>
              <label className="form-field">
                <span>Conduct</span>
                <select value={tcConduct} onChange={e => setTcConduct(e.target.value)}>
                  {['Good','Satisfactory','Excellent','Fair'].map(c => <option key={c}>{c}</option>)}
                </select>
              </label>
            </div>
            <label className="form-field">
              <span>Reason for Leaving</span>
              <input value={tcReason} onChange={e => setTcReason(e.target.value)} placeholder="e.g. Shifting to another city" />
            </label>
            <div className="doc-delivery">
              {[['download','Download'],['email','Email'],['whatsapp','WhatsApp']].filter(([v]) => v !== 'whatsapp' || whatsapp_enabled).map(([v, l]) => (
                <label key={v} className={`delivery-option ${tcDelivery === v ? 'delivery-option--active' : ''}`}>
                  <input type="radio" name="tc-delivery" value={v} checked={tcDelivery === v} onChange={() => setTcDelivery(v)} />
                  {v === 'download' ? <Download size={14} /> : v === 'email' ? <Mail size={14} /> : <MessageCircle size={14} />}
                  {l}
                </label>
              ))}
            </div>
            {tcDelivery === 'email' && (
              <label className="form-field">
                <span>Recipient Email *</span>
                <input type="email" required value={tcEmail} onChange={e => setTcEmail(e.target.value)} placeholder="parent@example.com" />
              </label>
            )}
            {tcDelivery === 'whatsapp' && (
              <label className="form-field">
                <span>WhatsApp Number *</span>
                <input required value={tcPhone} onChange={e => setTcPhone(e.target.value)} placeholder="919876543210" />
              </label>
            )}
            {tcMsg && <p className={`doc-msg ${tcMsg.startsWith('Error') ? 'doc-msg--error' : 'doc-msg--ok'}`}>{tcMsg}</p>}
            <button type="submit" className="btn btn--primary" disabled={tcBusy}>{tcBusy ? 'Processing...' : 'Generate'}</button>
          </form>
        </div>
      )}

      {tab === 'salary' && (
        <div className="doc-card">
          <h2>Salary Slip</h2>
          <form className="doc-form" onSubmit={handleSalarySlip}>
            <label className="form-field">
              <span>Staff Member *</span>
              <select required value={ssUserId} onChange={e => setSsUserId(e.target.value)}>
                <option value="">Select staff...</option>
                {staff.map(u => <option key={u.id} value={u.id}>{u.first_name} {u.last_name} ({u.role})</option>)}
              </select>
            </label>
            <div className="form-row">
              <label className="form-field">
                <span>Month *</span>
                <select required value={ssMonth} onChange={e => setSsMonth(e.target.value)}>
                  <option value="">Select month...</option>
                  {months.map(m => <option key={m}>{m}</option>)}
                </select>
              </label>
              <label className="form-field">
                <span>Year *</span>
                <input type="number" required min="2000" max="2099" value={ssYear} onChange={e => setSsYear(e.target.value)} />
              </label>
            </div>
            <p className="doc-section-label">Earnings (INR)</p>
            <div className="form-row">
              <label className="form-field"><span>Basic Salary</span><input type="number" min="0" value={ssBasic} onChange={e => setSsBasic(e.target.value)} /></label>
              <label className="form-field"><span>HRA</span><input type="number" min="0" value={ssHra} onChange={e => setSsHra(e.target.value)} /></label>
            </div>
            <div className="form-row">
              <label className="form-field"><span>Conveyance</span><input type="number" min="0" value={ssConveyance} onChange={e => setSsConveyance(e.target.value)} /></label>
              <label className="form-field"><span>Other Allowances</span><input type="number" min="0" value={ssOtherAllow} onChange={e => setSsOtherAllow(e.target.value)} /></label>
            </div>
            <p className="doc-section-label">Deductions (INR)</p>
            <div className="form-row">
              <label className="form-field"><span>PF</span><input type="number" min="0" value={ssPf} onChange={e => setSsPf(e.target.value)} /></label>
              <label className="form-field"><span>TDS</span><input type="number" min="0" value={ssTds} onChange={e => setSsTds(e.target.value)} /></label>
              <label className="form-field"><span>Other Deductions</span><input type="number" min="0" value={ssOtherDeduct} onChange={e => setSsOtherDeduct(e.target.value)} /></label>
            </div>
            <div className="doc-delivery">
              {[['download','Download'],['email','Email'],['whatsapp','WhatsApp']].filter(([v]) => v !== 'whatsapp' || whatsapp_enabled).map(([v, l]) => (
                <label key={v} className={`delivery-option ${ssDelivery === v ? 'delivery-option--active' : ''}`}>
                  <input type="radio" name="ss-delivery" value={v} checked={ssDelivery === v} onChange={() => setSsDelivery(v)} />
                  {v === 'download' ? <Download size={14} /> : v === 'email' ? <Mail size={14} /> : <MessageCircle size={14} />}
                  {l}
                </label>
              ))}
            </div>
            {ssDelivery === 'email' && (
              <label className="form-field">
                <span>Recipient Email *</span>
                <input type="email" required value={ssEmail} onChange={e => setSsEmail(e.target.value)} />
              </label>
            )}
            {ssDelivery === 'whatsapp' && (
              <label className="form-field">
                <span>WhatsApp Number *</span>
                <input required value={ssPhone} onChange={e => setSsPhone(e.target.value)} placeholder="919876543210" />
              </label>
            )}
            {ssMsg && <p className={`doc-msg ${ssMsg.startsWith('Error') ? 'doc-msg--error' : 'doc-msg--ok'}`}>{ssMsg}</p>}
            <button type="submit" className="btn btn--primary" disabled={ssBusy}>{ssBusy ? 'Processing...' : 'Generate'}</button>
          </form>
        </div>
      )}
    </div>
  )
}

export default Documents
