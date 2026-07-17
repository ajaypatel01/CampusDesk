import { useState, useEffect } from 'react'
import { useParams, Link } from 'react-router-dom'
import { ArrowLeft, Edit2, Save, X, UserPlus, IndianRupee } from 'lucide-react'
import { studentsApi, guardiansApi, feesApi } from '../services/api'
import { useSchool } from '../services/SchoolContext'
import './StudentDetail.css'

function StudentDetail() {
  const { id } = useParams()
  const { currentYear } = useSchool()
  const [student, setStudent] = useState(null)
  const [guardians, setGuardians] = useState([])
  const [feeSummary, setFeeSummary] = useState(null)
  const [loading, setLoading] = useState(true)
  const [editing, setEditing] = useState(false)
  const [form, setForm] = useState({})
  const [saving, setSaving] = useState(false)
  const [showGuardianModal, setShowGuardianModal] = useState(false)
  const [guardianForm, setGuardianForm] = useState({ first_name: '', last_name: '', phone: '', email: '', relation: '', aadhar_number: '' })

  function fmt(amt) {
    return new Intl.NumberFormat('en-IN', { style: 'currency', currency: 'INR', maximumFractionDigits: 0 }).format(amt || 0)
  }

  useEffect(() => {
    setLoading(true)
    const promises = [
      studentsApi.get(id),
      guardiansApi.list(id).catch(() => ({ items: [] })),
    ]
    if (currentYear) {
      promises.push(feesApi.studentSummary(id, currentYear.id).catch(() => null))
    }
    Promise.all(promises).then(([s, g, fee]) => {
      setStudent(s)
      setForm(s)
      setGuardians(g.items || [])
      setFeeSummary(fee || null)
    }).catch(() => {})
      .finally(() => setLoading(false))
  }, [id, currentYear])

  function toISODate(val) {
    if (!val) return undefined
    return new Date(val).toISOString()
  }

  async function handleSave() {
    setSaving(true)
    try {
      const updated = await studentsApi.update(id, {
        student_code: form.student_code,
        first_name: form.first_name,
        last_name: form.last_name,
        gender: form.gender, date_of_birth: toISODate(form.date_of_birth),
        phone: form.phone, email: form.email, address: form.address,
        admission_date: toISODate(form.admission_date), caste: form.caste, category: form.category,
        aadhar_number: form.aadhar_number, samagra_id: form.samagra_id,
        pen_number: form.pen_number, apar_id: form.apar_id,
        previous_school: form.previous_school,
        bank_name: form.bank_name, bank_ifsc: form.bank_ifsc,
        bank_account_number: form.bank_account_number,
        bank_holder_name: form.bank_holder_name, bank_branch: form.bank_branch,
        status: form.status,
      })
      setStudent(updated)
      setEditing(false)
    } catch (err) {
      alert(err.message)
    } finally {
      setSaving(false)
    }
  }

  async function handleAddGuardian(e) {
    e.preventDefault()
    try {
      const g = await guardiansApi.create(guardianForm)
      await guardiansApi.link({ student_id: id, guardian_id: g.id, is_primary: guardians.length === 0 })
      const res = await guardiansApi.list(id)
      setGuardians(res.items || [])
      setShowGuardianModal(false)
      setGuardianForm({ first_name: '', last_name: '', phone: '', email: '', relation: '', aadhar_number: '' })
    } catch (err) {
      alert(err.message)
    }
  }

  if (loading) return <p className="loading-text">Loading...</p>
  if (!student) return <p className="empty-text">Student not found</p>

  const f = editing ? form : student

  return (
    <div className="student-detail">
      <Link to="/students" className="back-link"><ArrowLeft size={18} /> Back to Students</Link>

      <div className="student-detail__top">
        <div className="student-detail__avatar">
          {student.first_name[0]}{student.last_name[0]}
        </div>
        <div className="student-detail__title">
          <h1>{student.first_name} {student.last_name}</h1>
          <span className="student-detail__code">{student.student_code}</span>
          <span className={`badge badge--${student.status === 'active' ? 'success' : 'muted'}`}>{student.status}</span>
        </div>
        <div className="student-detail__actions">
          {editing ? (
            <>
              <button className="btn btn--outline btn--sm" onClick={() => { setEditing(false); setForm(student) }}><X size={16} /> Cancel</button>
              <button className="btn btn--primary btn--sm" onClick={handleSave} disabled={saving}>
                <Save size={16} /> {saving ? 'Saving...' : 'Save'}
              </button>
            </>
          ) : (
            <button className="btn btn--outline btn--sm" onClick={() => setEditing(true)}><Edit2 size={16} /> Edit</button>
          )}
        </div>
      </div>

      <div className="student-detail__grid">
        <div className="detail-card">
          <h3>Personal Information</h3>
          <div className="detail-fields">
            <Field label="First Name" value={f.first_name} editing={editing} onChange={v => setForm({ ...form, first_name: v })} />
            <Field label="Last Name" value={f.last_name} editing={editing} onChange={v => setForm({ ...form, last_name: v })} />
            <Field label="Gender" value={f.gender} editing={editing} onChange={v => setForm({ ...form, gender: v })} type="select" options={['', 'male', 'female', 'other']} />
            <Field label="Date of Birth" value={f.date_of_birth?.split('T')[0] || ''} editing={editing} onChange={v => setForm({ ...form, date_of_birth: v })} type="date" />
            <Field label="Phone" value={f.phone} editing={editing} onChange={v => setForm({ ...form, phone: v })} />
            <Field label="Email" value={f.email} editing={editing} onChange={v => setForm({ ...form, email: v })} />
            <Field label="Address" value={f.address} editing={editing} onChange={v => setForm({ ...form, address: v })} />
            <Field label="Caste" value={f.caste} editing={editing} onChange={v => setForm({ ...form, caste: v })} />
            <Field label="Category" value={f.category} editing={editing} onChange={v => setForm({ ...form, category: v })} />
            <Field label="Status" value={f.status} editing={editing} onChange={v => setForm({ ...form, status: v })} type="select" options={['active', 'inactive', 'graduated', 'transferred']} />
          </div>
        </div>

        <div className="detail-card">
          <h3>Identity & Documents</h3>
          <div className="detail-fields">
            <Field label="Aadhar Number" value={f.aadhar_number} editing={editing} onChange={v => setForm({ ...form, aadhar_number: v })} />
            <Field label="Samagra ID" value={f.samagra_id} editing={editing} onChange={v => setForm({ ...form, samagra_id: v })} />
            <Field label="PEN Number" value={f.pen_number} editing={editing} onChange={v => setForm({ ...form, pen_number: v })} />
            <Field label="APAR ID" value={f.apar_id} editing={editing} onChange={v => setForm({ ...form, apar_id: v })} />
            <Field label="Previous School" value={f.previous_school} editing={editing} onChange={v => setForm({ ...form, previous_school: v })} />
            <Field label="Admission Date" value={f.admission_date?.split('T')[0] || ''} editing={editing} onChange={v => setForm({ ...form, admission_date: v })} type="date" />
          </div>
        </div>

        <div className="detail-card">
          <h3>Bank Details</h3>
          <div className="detail-fields">
            <Field label="Bank Name" value={f.bank_name} editing={editing} onChange={v => setForm({ ...form, bank_name: v })} />
            <Field label="IFSC Code" value={f.bank_ifsc} editing={editing} onChange={v => setForm({ ...form, bank_ifsc: v })} />
            <Field label="Account Number" value={f.bank_account_number} editing={editing} onChange={v => setForm({ ...form, bank_account_number: v })} />
            <Field label="Account Holder" value={f.bank_holder_name} editing={editing} onChange={v => setForm({ ...form, bank_holder_name: v })} />
            <Field label="Branch" value={f.bank_branch} editing={editing} onChange={v => setForm({ ...form, bank_branch: v })} />
          </div>
        </div>

        <div className="detail-card">
          <div className="detail-card__header">
            <h3>Guardians</h3>
            <button className="btn btn--outline btn--sm" onClick={() => setShowGuardianModal(true)}>
              <UserPlus size={14} /> Add
            </button>
          </div>
          {guardians.length === 0 ? (
            <p className="empty-text">No guardians linked</p>
          ) : (
            <div className="guardian-list">
              {guardians.map(g => (
                <div key={g.id} className="guardian-item">
                  <div className="guardian-item__avatar">{g.first_name[0]}{g.last_name[0]}</div>
                  <div>
                    <div className="guardian-item__name">{g.first_name} {g.last_name}</div>
                    <div className="guardian-item__meta">
                      {g.relation && <span>{g.relation}</span>}
                      {g.phone && <span>{g.phone}</span>}
                    </div>
                  </div>
                  {g.is_primary && <span className="badge badge--info">Primary</span>}
                </div>
              ))}
            </div>
          )}
        </div>

        {feeSummary && (
          <div className="detail-card">
            <div className="detail-card__header">
              <h3><IndianRupee size={16} /> Fee Summary ({feeSummary.academic_year_id ? currentYear?.name : ''})</h3>
              <Link to={`/fees`} className="btn btn--outline btn--sm">View All Fees</Link>
            </div>
            <div className="detail-fields">
              <Field label="Class" value={feeSummary.grade_level_name} editing={false} />
              <Field label="Tuition Fee" value={fmt(feeSummary.tuition_fee)} editing={false} />
              <Field label="Discount" value={feeSummary.discount_amount ? `${fmt(feeSummary.discount_amount)}${feeSummary.discount_reason ? ` (${feeSummary.discount_reason})` : ''}` : '-'} editing={false} />
              <Field label="Net Tuition" value={fmt(feeSummary.net_tuition_fee)} editing={false} />
              <Field label="Van Fee" value={fmt(feeSummary.van_fee)} editing={false} />
              <Field label="Previous Dues" value={fmt(feeSummary.previous_year_dues)} editing={false} />
              <Field label="Total Due" value={fmt(feeSummary.total_due)} editing={false} />
              <Field label="Total Paid" value={fmt(feeSummary.total_paid)} editing={false} />
              <Field label="Balance" value={fmt(feeSummary.balance_remaining)} editing={false} />
              {feeSummary.is_rte && <Field label="RTE Status" value="Yes - Right to Education" editing={false} />}
            </div>
            {feeSummary.payments && feeSummary.payments.length > 0 && (
              <div style={{ marginTop: '12px' }}>
                <h4 style={{ fontSize: '14px', marginBottom: '8px', color: 'var(--gray-600)' }}>Recent Payments</h4>
                <table className="data-table" style={{ fontSize: '13px' }}>
                  <thead><tr><th>Date</th><th>Type</th><th>Amount</th><th>Mode</th></tr></thead>
                  <tbody>
                    {feeSummary.payments.slice(0, 5).map(p => (
                      <tr key={p.id}>
                        <td>{new Date(p.payment_date).toLocaleDateString('en-IN')}</td>
                        <td>{p.fee_type}</td>
                        <td style={{ color: 'var(--success-600)' }}>{fmt(p.amount)}</td>
                        <td>{p.payment_mode}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        )}
      </div>

      {showGuardianModal && (
        <div className="modal-overlay" onClick={() => setShowGuardianModal(false)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <h2>Add Guardian</h2>
            <form className="modal__form" onSubmit={handleAddGuardian}>
              <div className="form-row">
                <label className="form-field">
                  <span>First Name *</span>
                  <input required value={guardianForm.first_name} onChange={e => setGuardianForm({ ...guardianForm, first_name: e.target.value })} />
                </label>
                <label className="form-field">
                  <span>Last Name *</span>
                  <input required value={guardianForm.last_name} onChange={e => setGuardianForm({ ...guardianForm, last_name: e.target.value })} />
                </label>
              </div>
              <div className="form-row">
                <label className="form-field">
                  <span>Relation</span>
                  <input value={guardianForm.relation} onChange={e => setGuardianForm({ ...guardianForm, relation: e.target.value })} placeholder="e.g. Father, Mother" />
                </label>
                <label className="form-field">
                  <span>Phone</span>
                  <input value={guardianForm.phone} onChange={e => setGuardianForm({ ...guardianForm, phone: e.target.value })} />
                </label>
              </div>
              <label className="form-field">
                <span>Aadhar Number</span>
                <input value={guardianForm.aadhar_number} onChange={e => setGuardianForm({ ...guardianForm, aadhar_number: e.target.value })} maxLength={12} />
              </label>
              <div className="modal__actions">
                <button type="button" className="btn btn--outline" onClick={() => setShowGuardianModal(false)}>Cancel</button>
                <button type="submit" className="btn btn--primary">Add Guardian</button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}

function Field({ label, value, editing, onChange, type = 'text', options }) {
  if (editing) {
    if (type === 'select') {
      return (
        <div className="detail-field">
          <span className="detail-field__label">{label}</span>
          <select className="detail-field__input" value={value || ''} onChange={e => onChange(e.target.value)}>
            {options.map(o => <option key={o} value={o}>{o || 'Select'}</option>)}
          </select>
        </div>
      )
    }
    return (
      <div className="detail-field">
        <span className="detail-field__label">{label}</span>
        <input className="detail-field__input" type={type} value={value || ''} onChange={e => onChange(e.target.value)} />
      </div>
    )
  }
  return (
    <div className="detail-field">
      <span className="detail-field__label">{label}</span>
      <span className="detail-field__value">{value || '-'}</span>
    </div>
  )
}

export default StudentDetail
