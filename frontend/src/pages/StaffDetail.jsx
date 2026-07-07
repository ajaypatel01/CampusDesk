import { useState, useEffect } from 'react'
import { useParams, Link } from 'react-router-dom'
import {
  ArrowLeft, Mail, Phone, ShieldCheck, Briefcase, User,
  Building2, CreditCard, GraduationCap, Edit3, X, Save, Loader
} from 'lucide-react'
import { staffApi } from '../services/api'
import './StaffDetail.css'

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
  const [member, setMember] = useState(null)
  const [loading, setLoading] = useState(true)
  const [editing, setEditing] = useState(false)
  const [saving, setSaving] = useState(false)
  const [saveErr, setSaveErr] = useState('')

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
          })
        }
      })
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [id])

  function set(k, v) {
    setForm(f => ({ ...f, [k]: v }))
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
                <label>Designation<input value={form.designation} onChange={e => set('designation', e.target.value)} placeholder="e.g. Head Teacher" /></label>
                <label>Staff Type
                  <select value={form.staff_type} onChange={e => set('staff_type', e.target.value)}>
                    <option value="teaching">Teaching</option>
                    <option value="non_teaching">Non-Teaching</option>
                  </select>
                </label>
              </div>
              <div className="sd-form-row">
                <label>Salary (₹)<input type="number" value={form.salary} onChange={e => set('salary', e.target.value)} placeholder="Monthly salary" /></label>
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
