import { useState, useEffect } from 'react'
import { useParams, Link, useOutletContext } from 'react-router-dom'
import {
  ArrowLeft, Mail, Phone, ShieldCheck, Briefcase, User,
  Building2, CreditCard, GraduationCap, Edit3, X, Save, Loader,
  Wallet, Download, Plus, Trash2,
} from 'lucide-react'
import { staffApi, payrollApi } from '../services/api'
import { useSchool } from '../services/SchoolContext'
import './StaffDetail.css'

const MONTHS = [
  'January', 'February', 'March', 'April', 'May', 'June',
  'July', 'August', 'September', 'October', 'November', 'December',
]

function fmtCurrency(amt) {
  return new Intl.NumberFormat('en-IN', { style: 'currency', currency: 'INR', maximumFractionDigits: 0 }).format(amt || 0)
}

const roleLabels = {
  super_admin: 'Super Admin',
  school_admin: 'Principal',
  teacher: 'Teacher',
  registrar: 'Registrar',
}

const roleBadge = {
  teacher: 'info',
  school_admin: 'warning',
  super_admin: 'danger',
  registrar: 'success',
}

const staffTypeLabels = { teaching: 'Teaching', non_teaching: 'Non-Teaching' }

const DESIGNATION_OPTIONS = [
  'Principal', 'Pre Primary Teacher', 'Primary Teacher', 'Upper Primary Teacher',
  'Secondary Teacher', 'Receptionist', 'Registrar', 'Driver', 'Cleaning Staff',
  'Housekeeping Staff', 'Peon',
]

function Field({ label, value }) {
  return (
    <div className="sd-field">
      <span className="sd-field__label">{label}</span>
      <span className="sd-field__value">{value || <span className="sd-field__empty">—</span>}</span>
    </div>
  )
}

function StaffDetail() {
  const { id } = useParams()
  const { user } = useOutletContext() || {}
  const { currentSchool, currentYear } = useSchool()
  const isSuperAdmin = user?.role === 'super_admin'
  const [member, setMember] = useState(null)
  const [loading, setLoading] = useState(true)
  const [editing, setEditing] = useState(false)
  const [saving, setSaving] = useState(false)
  const [saveErr, setSaveErr] = useState('')

  const now = new Date()
  const [payMonth, setPayMonth] = useState(now.getMonth() + 1)
  const [payYear, setPayYear] = useState(now.getFullYear())
  const [payrollRow, setPayrollRow] = useState(null)
  const [payrollLoading, setPayrollLoading] = useState(false)
  const [leaves, setLeaves] = useState([])
  const [slipDownloading, setSlipDownloading] = useState(false)
  const [showLeaveModal, setShowLeaveModal] = useState(false)
  const [leaveForm, setLeaveForm] = useState({ leave_type: 'cl', start_date: '', end_date: '', reason: '' })
  const [leaveSaving, setLeaveSaving] = useState(false)
  const [leaveErr, setLeaveErr] = useState('')

  const [form, setForm] = useState({
    guardian_name: '',
    aadhar_number: '',
    education_qualification: '',
    professional_qualification: '',
    designation: '',
    salary: '',
    bank_name: '',
    bank_ifsc: '',
    bank_branch: '',
    bank_account_number: '',
    bank_account_holder: '',
    phone: '',
    staff_type: 'teaching',
    cl_quota_per_year: 7,
  })

  useEffect(() => {
    setLoading(true)
    staffApi.get(id)
      .then(m => {
        setMember(m)
        if (m.profile) {
          setForm({
            guardian_name: m.profile.guardian_name || '',
            aadhar_number: m.profile.aadhar_number || '',
            education_qualification: m.profile.education_qualification || '',
            professional_qualification: m.profile.professional_qualification || '',
            designation: m.profile.designation || '',
            salary: m.profile.salary != null ? String(m.profile.salary) : '',
            bank_name: m.profile.bank_name || '',
            bank_ifsc: m.profile.bank_ifsc || '',
            bank_branch: m.profile.bank_branch || '',
            bank_account_number: m.profile.bank_account_number || '',
            bank_account_holder: m.profile.bank_account_holder || '',
            phone: m.profile.phone || '',
            staff_type: m.profile.staff_type || 'teaching',
            cl_quota_per_year: m.profile.cl_quota_per_year || 7,
          })
        }
      })
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [id])

  function set(k, v) {
    setForm(f => ({ ...f, [k]: v }))
  }

  function loadLeaves() {
    if (!isSuperAdmin || !currentYear) return
    payrollApi.listLeaves({ user_id: id, academic_year_id: currentYear.id })
      .then(r => setLeaves(r.items || []))
      .catch(() => setLeaves([]))
  }

  function loadPayrollRow() {
    if (!isSuperAdmin || !currentSchool || !currentYear) return
    setPayrollLoading(true)
    payrollApi.computeMonth({ school_id: currentSchool.id, academic_year_id: currentYear.id, year: payYear, month: payMonth })
      .then(r => setPayrollRow((r.items || []).find(row => row.user_id === id) || null))
      .catch(() => setPayrollRow(null))
      .finally(() => setPayrollLoading(false))
  }

  useEffect(loadPayrollRow, [isSuperAdmin, currentSchool, currentYear, payYear, payMonth, id])
  useEffect(loadLeaves, [isSuperAdmin, currentYear, id])

  async function handleAddLeave(e) {
    e.preventDefault()
    setLeaveSaving(true); setLeaveErr('')
    try {
      await payrollApi.createLeave({ ...leaveForm, user_id: id, academic_year_id: currentYear.id })
      setShowLeaveModal(false)
      setLeaveForm({ leave_type: 'cl', start_date: '', end_date: '', reason: '' })
      loadLeaves()
      loadPayrollRow()
    } catch (err) {
      setLeaveErr(err.message || 'Failed to save leave')
    } finally {
      setLeaveSaving(false)
    }
  }

  async function handleDeleteLeave(leaveId) {
    if (!confirm('Delete this leave record?')) return
    try {
      await payrollApi.deleteLeave(leaveId)
      loadLeaves()
      loadPayrollRow()
    } catch (err) {
      alert(err.message)
    }
  }

  async function handleDownloadSlip() {
    if (!currentSchool || !currentYear) return
    setSlipDownloading(true)
    try {
      const blob = await payrollApi.downloadSlip({ school_id: currentSchool.id, academic_year_id: currentYear.id, user_id: id, year: payYear, month: payMonth })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `salary_slip_${id.substring(0, 8)}_${payYear}_${String(payMonth).padStart(2, '0')}.pdf`
      a.click()
      URL.revokeObjectURL(url)
    } catch (err) {
      alert(err.message)
    } finally {
      setSlipDownloading(false)
    }
  }

  async function handleSave() {
    setSaving(true)
    setSaveErr('')
    try {
      const body = { ...form, salary: form.salary ? parseInt(form.salary, 10) : 0 }
      await staffApi.upsertProfile(id, body)
      const updated = await staffApi.get(id)
      setMember(updated)
      setEditing(false)
    } catch (e) {
      setSaveErr(e.message || 'Failed to save')
    } finally {
      setSaving(false)
    }
  }

  if (loading) return <p className="loading-text">Loading...</p>
  if (!member) return <p className="empty-text">Staff member not found</p>

  const p = member.profile
  const initials = `${member.first_name?.[0] || ''}${member.last_name?.[0] || ''}`.toUpperCase()
  const isNonTeaching = p?.staff_type === 'non_teaching'

  return (
    <div className="sd-page">
      <Link to="/staff" className="back-link"><ArrowLeft size={18} /> Back to Staff</Link>

      {/* Hero */}
      <div className="sd-hero">
        <div className={`sd-hero__avatar ${isNonTeaching ? 'sd-hero__avatar--amber' : ''}`}>
          {initials || <User size={28} />}
        </div>
        <div className="sd-hero__info">
          <h1>{member.first_name} {member.last_name}</h1>
          <div className="sd-hero__badges">
            <span className={`badge badge--${roleBadge[member.role] || 'muted'}`}>
              {roleLabels[member.role] || member.role}
            </span>
            {p?.staff_type && (
              <span className={`sd-type-badge ${isNonTeaching ? 'sd-type-badge--amber' : 'sd-type-badge--blue'}`}>
                {staffTypeLabels[p.staff_type] || p.staff_type}
              </span>
            )}
            <span className={`sd-status-pill ${member.is_active ? 'sd-status-pill--active' : ''}`}>
              <span className={`sd-status-dot ${member.is_active ? 'sd-status-dot--active' : ''}`} />
              {member.is_active ? 'Active' : 'Inactive'}
            </span>
          </div>
          {p?.designation && <p className="sd-hero__designation">{p.designation}</p>}
          {p?.salary != null && (
            <p className="sd-hero__salary">
              ₹{Number(p.salary).toLocaleString('en-IN')} <span>/ month</span>
            </p>
          )}
        </div>
        <button className="btn btn--outline sd-edit-btn" onClick={() => setEditing(true)}>
          <Edit3 size={15} /> Edit Profile
        </button>
      </div>

      <div className="sd-grid">
        {/* Contact */}
        <div className="sd-card">
          <div className="sd-card__header">
            <Mail size={16} />
            <h3>Contact</h3>
          </div>
          <Field label="Email" value={member.email} />
          <Field label="Phone" value={p?.phone} />
          <Field label="Guardian / Spouse" value={p?.guardian_name} />
          <Field label="Aadhaar Number" value={p?.aadhar_number} />
        </div>

        {/* HR Details */}
        <div className="sd-card">
          <div className="sd-card__header">
            <Briefcase size={16} />
            <h3>HR Details</h3>
          </div>
          <Field label="Designation" value={p?.designation} />
          <Field label="Staff Type" value={p?.staff_type ? staffTypeLabels[p.staff_type] : null} />
          <Field label="Salary" value={p?.salary != null ? `₹${Number(p.salary).toLocaleString('en-IN')} / month` : null} />
          <Field label="CL Quota" value={p?.cl_quota_per_year != null ? `${p.cl_quota_per_year} / year` : null} />
          <Field label="Education" value={p?.education_qualification} />
          <Field label="Professional Qual." value={p?.professional_qualification} />
        </div>

        {/* Bank Details */}
        <div className="sd-card">
          <div className="sd-card__header">
            <CreditCard size={16} />
            <h3>Bank Details</h3>
          </div>
          <Field label="Bank Name" value={p?.bank_name} />
          <Field label="Account Holder" value={p?.bank_account_holder} />
          <Field label="Account Number" value={p?.bank_account_number} />
          <Field label="IFSC Code" value={p?.bank_ifsc} />
          <Field label="Branch" value={p?.bank_branch} />
        </div>

        {/* Account meta */}
        <div className="sd-card">
          <div className="sd-card__header">
            <ShieldCheck size={16} />
            <h3>Account</h3>
          </div>
          <Field label="Role" value={roleLabels[member.role] || member.role} />
          <Field label="Status" value={member.is_active ? 'Active' : 'Inactive'} />
          <Field
            label="Joined"
            value={member.created_at
              ? new Date(member.created_at).toLocaleDateString('en-IN', { day: 'numeric', month: 'long', year: 'numeric' })
              : null}
          />
          <div className="sd-field">
            <span className="sd-field__label">User ID</span>
            <span className="sd-field__value sd-field__value--mono">{member.id}</span>
          </div>
        </div>
      </div>

      {isSuperAdmin && (
        <div className="sd-card" style={{ marginTop: '20px' }}>
          <div className="sd-card__header">
            <Wallet size={16} />
            <h3>Payroll &amp; Leave</h3>
          </div>

          {!currentSchool || !currentYear ? (
            <p className="empty-text">Select a school and academic year first.</p>
          ) : (
            <>
              <div className="payroll-controls">
                <label className="form-field">
                  <span>Month</span>
                  <select value={payMonth} onChange={e => setPayMonth(parseInt(e.target.value, 10))}>
                    {MONTHS.map((m, i) => <option key={m} value={i + 1}>{m}</option>)}
                  </select>
                </label>
                <label className="form-field">
                  <span>Year</span>
                  <input type="number" value={payYear} onChange={e => setPayYear(parseInt(e.target.value, 10) || now.getFullYear())} />
                </label>
                <button className="btn btn--outline btn--sm" onClick={handleDownloadSlip} disabled={slipDownloading || !payrollRow}>
                  <Download size={14} /> {slipDownloading ? 'Downloading...' : 'Download Salary Slip'}
                </button>
              </div>

              {payrollLoading ? (
                <p className="empty-text">Loading...</p>
              ) : !payrollRow ? (
                <p className="empty-text">No salary on file, or no data for {MONTHS[payMonth - 1]} {payYear}.</p>
              ) : (
                <div className="sd-payroll-summary">
                  <Field label="Working Days" value={payrollRow.working_days_per_month} />
                  <Field label="Present Days" value={payrollRow.working_days_per_month - payrollRow.deducted_days} />
                  <Field label="CL Availed (Paid)" value={payrollRow.cl_paid_days} />
                  <Field label="CL Beyond Quota" value={payrollRow.cl_excess_days} />
                  <Field label="Unpaid Days" value={payrollRow.unpaid_days} />
                  <Field label="Deduction" value={fmtCurrency(payrollRow.deduction)} />
                  <Field label="Net Salary" value={fmtCurrency(payrollRow.net_salary)} />
                  <Field label="CL Balance (YTD)" value={`${payrollRow.cl_balance} / ${payrollRow.cl_quota_per_year}`} />
                </div>
              )}

              <div className="sd-card__header" style={{ marginTop: '18px', justifyContent: 'space-between' }}>
                <h3>Leave Records — {currentYear.name}</h3>
                <button className="btn btn--outline btn--sm" onClick={() => setShowLeaveModal(true)}>
                  <Plus size={14} /> Log Leave
                </button>
              </div>
              <table className="data-table">
                <thead>
                  <tr><th>Type</th><th>From</th><th>To</th><th>Reason</th><th></th></tr>
                </thead>
                <tbody>
                  {leaves.length === 0 ? (
                    <tr><td colSpan={5} className="data-table__empty">No leave records yet</td></tr>
                  ) : leaves.map(l => (
                    <tr key={l.id}>
                      <td><span className={`badge badge--${l.leave_type === 'cl' ? 'info' : 'danger'}`}>{l.leave_type === 'cl' ? 'CL' : 'Unpaid'}</span></td>
                      <td className="data-table__muted">{l.start_date}</td>
                      <td className="data-table__muted">{l.end_date}</td>
                      <td className="data-table__muted">{l.reason || '-'}</td>
                      <td><button className="btn btn--outline btn--sm" onClick={() => handleDeleteLeave(l.id)}><Trash2 size={14} /></button></td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </>
          )}
        </div>
      )}

      {showLeaveModal && (
        <div className="modal-overlay" onClick={() => setShowLeaveModal(false)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <h2>Log Leave for {member.first_name} {member.last_name}</h2>
            <form className="modal__form" onSubmit={handleAddLeave}>
              <div className="form-row">
                <label className="form-field">
                  <span>Leave Type *</span>
                  <select value={leaveForm.leave_type} onChange={e => setLeaveForm({ ...leaveForm, leave_type: e.target.value })}>
                    <option value="cl">Casual Leave (CL)</option>
                    <option value="unpaid">Unpaid</option>
                  </select>
                </label>
              </div>
              <div className="form-row">
                <label className="form-field">
                  <span>From *</span>
                  <input required type="date" value={leaveForm.start_date} onChange={e => setLeaveForm({ ...leaveForm, start_date: e.target.value })} />
                </label>
                <label className="form-field">
                  <span>To *</span>
                  <input required type="date" value={leaveForm.end_date} onChange={e => setLeaveForm({ ...leaveForm, end_date: e.target.value })} />
                </label>
              </div>
              <label className="form-field">
                <span>Reason</span>
                <input value={leaveForm.reason} onChange={e => setLeaveForm({ ...leaveForm, reason: e.target.value })} placeholder="Optional" />
              </label>
              {leaveErr && <p className="doc-msg doc-msg--error">{leaveErr}</p>}
              <div className="modal__actions">
                <button type="button" className="btn btn--outline" onClick={() => setShowLeaveModal(false)}>Cancel</button>
                <button type="submit" className="btn btn--primary" disabled={leaveSaving}>{leaveSaving ? 'Saving...' : 'Save'}</button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Edit Modal */}
      {editing && (
        <div className="sd-modal-overlay" onClick={e => e.target === e.currentTarget && setEditing(false)}>
          <div className="sd-modal">
            <div className="sd-modal__header">
              <h2>Edit Profile</h2>
              <button className="sd-modal__close" onClick={() => setEditing(false)}><X size={18} /></button>
            </div>

            <div className="sd-modal__body">
              <p className="sd-modal__section-title">Personal</p>
              <div className="sd-form-row">
                <label>Phone<input value={form.phone} onChange={e => set('phone', e.target.value)} placeholder="Mobile number" /></label>
                <label>Guardian / Spouse<input value={form.guardian_name} onChange={e => set('guardian_name', e.target.value)} placeholder="Name" /></label>
              </div>
              <div className="sd-form-row">
                <label>Aadhaar Number<input value={form.aadhar_number} onChange={e => set('aadhar_number', e.target.value)} placeholder="12-digit" /></label>
              </div>

              <p className="sd-modal__section-title">HR</p>
              <div className="sd-form-row">
                <label>Job Profile
                  <select
                    value={DESIGNATION_OPTIONS.includes(form.designation) ? form.designation : (form.designation ? 'Other' : '')}
                    onChange={e => set('designation', e.target.value === 'Other' ? '' : e.target.value)}
                  >
                    <option value="">Select...</option>
                    {DESIGNATION_OPTIONS.map(d => <option key={d} value={d}>{d}</option>)}
                    <option value="Other">Other (custom)</option>
                  </select>
                </label>
                <label>Staff Type
                  <select value={form.staff_type} onChange={e => set('staff_type', e.target.value)}>
                    <option value="teaching">Teaching</option>
                    <option value="non_teaching">Non-Teaching</option>
                  </select>
                </label>
              </div>
              {!DESIGNATION_OPTIONS.includes(form.designation) && (
                <div className="sd-form-row">
                  <label>Custom Job Profile<input value={form.designation} onChange={e => set('designation', e.target.value)} placeholder="e.g. Lab Assistant" /></label>
                </div>
              )}
              <div className="sd-form-row">
                <label>Salary (₹)<input type="number" value={form.salary} onChange={e => set('salary', e.target.value)} placeholder="Monthly salary" /></label>
                <label>CL Quota (per year)<input type="number" min="0" value={form.cl_quota_per_year} onChange={e => set('cl_quota_per_year', parseInt(e.target.value, 10) || 0)} /></label>
              </div>
              <div className="sd-form-row">
                <label>Education Qualification<input value={form.education_qualification} onChange={e => set('education_qualification', e.target.value)} placeholder="e.g. B.Ed" /></label>
                <label>Professional Qualification<input value={form.professional_qualification} onChange={e => set('professional_qualification', e.target.value)} placeholder="e.g. M.Ed" /></label>
              </div>

              <p className="sd-modal__section-title">Bank Details</p>
              <div className="sd-form-row">
                <label>Bank Name<input value={form.bank_name} onChange={e => set('bank_name', e.target.value)} placeholder="e.g. SBI" /></label>
                <label>Account Holder<input value={form.bank_account_holder} onChange={e => set('bank_account_holder', e.target.value)} /></label>
              </div>
              <div className="sd-form-row">
                <label>Account Number<input value={form.bank_account_number} onChange={e => set('bank_account_number', e.target.value)} /></label>
                <label>IFSC Code<input value={form.bank_ifsc} onChange={e => set('bank_ifsc', e.target.value)} placeholder="e.g. SBIN0001234" /></label>
              </div>
              <div className="sd-form-row">
                <label>Branch<input value={form.bank_branch} onChange={e => set('bank_branch', e.target.value)} /></label>
              </div>

              {saveErr && <p className="sd-modal__err">{saveErr}</p>}
            </div>

            <div className="sd-modal__footer">
              <button className="btn btn--ghost" onClick={() => setEditing(false)}>Cancel</button>
              <button className="btn btn--primary" onClick={handleSave} disabled={saving}>
                {saving ? <Loader size={15} className="spin" /> : <Save size={15} />}
                {saving ? 'Saving…' : 'Save Changes'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

export default StaffDetail
