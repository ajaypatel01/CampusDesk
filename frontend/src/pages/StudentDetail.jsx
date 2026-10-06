import { useState, useEffect } from 'react'
import { useParams, Link, useOutletContext } from 'react-router-dom'
import { ArrowLeft, Edit2, Save, X, UserPlus, IndianRupee } from 'lucide-react'
import { studentsApi, guardiansApi, feesApi, academicApi } from '../services/api'
import { useSchool } from '../services/SchoolContext'
import CustomFieldsSection from '../components/CustomFieldsSection'
import './StudentDetail.css'
import { formatDate } from '../utils/date'

const FEE_EDITOR_ROLES = ['super_admin', 'school_admin', 'registrar']
const SCHOLAR_NO_EDITOR_ROLES = ['registrar', 'super_admin']

function StudentDetail() {
  const { id } = useParams()
  const { currentYear, currentSchool, academicYears } = useSchool()
  const { user } = useOutletContext() || {}
  const canEditFees = FEE_EDITOR_ROLES.includes(user?.role)
  const canEditScholarNo = SCHOLAR_NO_EDITOR_ROLES.includes(user?.role)
  const [student, setStudent] = useState(null)
  const [guardians, setGuardians] = useState([])
  const [feeSummary, setFeeSummary] = useState(null)
  const [grades, setGrades] = useState([])
  const [loading, setLoading] = useState(true)
  const [editing, setEditing] = useState(false)
  const [form, setForm] = useState({})
  const [saving, setSaving] = useState(false)
  const [showGuardianModal, setShowGuardianModal] = useState(false)
  const [guardianForm, setGuardianForm] = useState({ first_name: '', last_name: '', phone: '', email: '', relation: '', aadhar_number: '' })
  const [feeEditing, setFeeEditing] = useState(false)
  const [feeForm, setFeeForm] = useState({})
  const [feeSaving, setFeeSaving] = useState(false)
  const [sections, setSections] = useState([])
  const [sectionId, setSectionId] = useState(null)
  const [sectionEditing, setSectionEditing] = useState(false)
  const [sectionDraft, setSectionDraft] = useState('')
  const [sectionSaving, setSectionSaving] = useState(false)

  useEffect(() => {
    if (!currentSchool) return
    academicApi.listGrades(currentSchool.id)
      .then(res => setGrades(res.items || []))
      .catch(() => setGrades([]))
  }, [currentSchool])

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
      if (fee) {
        setFeeForm({
          tuition_fee: fee.tuition_fee, discount_amount: fee.discount_amount,
          discount_reason: fee.discount_reason || '', van_fee: fee.van_fee,
          previous_year_dues: fee.previous_year_dues, late_fee: fee.late_fee, grade_level_id: fee.grade_level_id,
          is_rte: fee.is_rte || false,
        })
      }
    }).catch(() => {})
      .finally(() => setLoading(false))
  }, [id, currentYear])

  useEffect(() => {
    if (!currentSchool || !currentYear) return
    academicApi.listSections({ school_id: currentSchool.id, academic_year_id: currentYear.id })
      .then(res => setSections(res.items || []))
      .catch(() => setSections([]))
    studentsApi.getSection(id, currentYear.id)
      .then(res => setSectionId(res.class_section_id || null))
      .catch(() => setSectionId(null))
  }, [id, currentSchool, currentYear])

  async function handleSaveSection() {
    setSectionSaving(true)
    try {
      await studentsApi.updateSection(id, { academic_year_id: currentYear.id, class_section_id: sectionDraft || null })
      setSectionId(sectionDraft || null)
      setSectionEditing(false)
    } catch (err) {
      alert(err.message)
    } finally {
      setSectionSaving(false)
    }
  }

  // Only the sections that belong to this student's current grade are
  // offered -- a section from a different grade would be rejected by the
  // backend (and wouldn't make sense) anyway.
  const sectionsForGrade = sections.filter(sec => sec.grade_level_id === feeSummary?.grade_level_id)
  const currentSectionName = sections.find(sec => sec.id === sectionId)?.name

  async function handleSaveFee() {
    if (!feeSummary?.account_id) return
    setFeeSaving(true)
    try {
      const body = {
        tuition_fee: parseInt(feeForm.tuition_fee, 10) || 0,
        discount_amount: parseInt(feeForm.discount_amount, 10) || 0,
        discount_reason: feeForm.discount_reason,
        van_fee: parseInt(feeForm.van_fee, 10) || 0,
        previous_year_dues: parseInt(feeForm.previous_year_dues, 10) || 0,
        late_fee: parseInt(feeForm.late_fee, 10) || 0,
        is_rte: !!feeForm.is_rte,
      }
      // Only send grade_level_id when it actually changed - the backend re-bases
      // tuition/van to the new grade's fee structure, which would silently
      // overwrite a custom tuition otherwise.
      if (feeForm.grade_level_id && feeForm.grade_level_id !== feeSummary.grade_level_id) {
        body.grade_level_id = feeForm.grade_level_id
      }
      await feesApi.updateAccount(feeSummary.account_id, body)
      const updated = await feesApi.studentSummary(id, currentYear.id)
      setFeeSummary(updated)
      setFeeForm({
        tuition_fee: updated.tuition_fee, discount_amount: updated.discount_amount,
        discount_reason: updated.discount_reason || '', van_fee: updated.van_fee,
        previous_year_dues: updated.previous_year_dues, late_fee: updated.late_fee, grade_level_id: updated.grade_level_id,
        is_rte: updated.is_rte || false,
      })
      setFeeEditing(false)
    } catch (err) {
      alert(err.message)
    } finally {
      setFeeSaving(false)
    }
  }

  function toISODate(val) {
    if (!val) return undefined
    return new Date(val).toISOString()
  }

  async function handleSave() {
    if (form.status === 'inactive' && (!form.tc_date || !form.tc_year)) {
      alert('TC Date and TC Year are required to mark a student inactive.')
      return
    }
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
        enrollment_number: form.enrollment_number,
        admission_class: form.admission_class, admission_year: form.admission_year,
        previous_school: form.previous_school,
        bank_name: form.bank_name, bank_ifsc: form.bank_ifsc,
        bank_account_number: form.bank_account_number,
        bank_holder_name: form.bank_holder_name, bank_branch: form.bank_branch,
        status: form.status,
        tc_date: toISODate(form.tc_date), tc_year: form.tc_year,
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
            <Field
              label="Scholar No"
              value={f.student_code}
              editing={editing && canEditScholarNo}
              onChange={v => setForm({ ...form, student_code: v })}
            />
            <Field label="First Name" value={f.first_name} editing={editing} onChange={v => setForm({ ...form, first_name: v })} />
            <Field label="Last Name" value={f.last_name} editing={editing} onChange={v => setForm({ ...form, last_name: v })} />
            <Field label="Gender" value={f.gender} editing={editing} onChange={v => setForm({ ...form, gender: v })} type="select" options={['', 'male', 'female']} />
            <Field label="Date of Birth" value={f.date_of_birth?.split('T')[0] || ''} editing={editing} onChange={v => setForm({ ...form, date_of_birth: v })} type="date" />
            <Field label="Phone" value={f.phone} editing={editing} onChange={v => setForm({ ...form, phone: v })} />
            <Field label="Email" value={f.email} editing={editing} onChange={v => setForm({ ...form, email: v })} />
            <Field label="Address" value={f.address} editing={editing} onChange={v => setForm({ ...form, address: v })} />
            <Field label="Caste" value={f.caste} editing={editing} onChange={v => setForm({ ...form, caste: v })} />
            <Field label="Category" value={f.category} editing={editing} onChange={v => setForm({ ...form, category: v })} />
            <Field label="Status" value={f.status} editing={editing} onChange={v => setForm({ ...form, status: v })} type="select" options={['active', 'inactive', 'graduated', 'transferred']} />
            {f.status === 'inactive' && (
              editing ? (
                <>
                  <div className="detail-field">
                    <span className="detail-field__label">TC Date *</span>
                    <input className="detail-field__input" type="date" value={f.tc_date?.split('T')[0] || ''} onChange={e => setForm({ ...form, tc_date: e.target.value })} />
                  </div>
                  <div className="detail-field">
                    <span className="detail-field__label">TC Year *</span>
                    <select className="detail-field__input" value={f.tc_year || ''} onChange={e => setForm({ ...form, tc_year: e.target.value })}>
                      <option value="">Select...</option>
                      {(academicYears || []).map(y => <option key={y.id} value={y.name}>{y.name}</option>)}
                    </select>
                  </div>
                </>
              ) : (
                <>
                  <Field label="TC Date" value={formatDate(f.tc_date)} editing={false} />
                  <Field label="TC Year" value={f.tc_year || '-'} editing={false} />
                </>
              )
            )}
          </div>
        </div>

        <div className="detail-card">
          <h3>Identity & Documents</h3>
          <div className="detail-fields">
            <Field label="Aadhar Number" value={f.aadhar_number} editing={editing} onChange={v => setForm({ ...form, aadhar_number: v })} />
            <Field label="Samagra ID" value={f.samagra_id} editing={editing} onChange={v => setForm({ ...form, samagra_id: v })} />
            <Field label="PEN Number" value={f.pen_number} editing={editing} onChange={v => setForm({ ...form, pen_number: v })} />
            <Field label="APAR ID" value={f.apar_id} editing={editing} onChange={v => setForm({ ...form, apar_id: v })} />
            <Field label="Enrollment No. (9th & 10th)" value={f.enrollment_number} editing={editing} onChange={v => setForm({ ...form, enrollment_number: v })} />
            <Field label="Previous School" value={f.previous_school} editing={editing} onChange={v => setForm({ ...form, previous_school: v })} />
            <Field label="Admission Date" value={f.admission_date?.split('T')[0] || ''} editing={editing} onChange={v => setForm({ ...form, admission_date: v })} type="date" />
            <Field label="Class of Admission" value={f.admission_class} editing={editing} onChange={v => setForm({ ...form, admission_class: v })} />
            <Field label="Admission Year" value={f.admission_year} editing={editing} onChange={v => setForm({ ...form, admission_year: v })} placeholder="e.g. 2023-24" />
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
              <div className="student-detail__actions">
                {canEditFees && (feeEditing ? (
                  <>
                    <button className="btn btn--outline btn--sm" onClick={() => { setFeeEditing(false); setFeeForm({ tuition_fee: feeSummary.tuition_fee, discount_amount: feeSummary.discount_amount, discount_reason: feeSummary.discount_reason || '', van_fee: feeSummary.van_fee, previous_year_dues: feeSummary.previous_year_dues, late_fee: feeSummary.late_fee, grade_level_id: feeSummary.grade_level_id, is_rte: feeSummary.is_rte || false }) }}><X size={14} /> Cancel</button>
                    <button className="btn btn--primary btn--sm" onClick={handleSaveFee} disabled={feeSaving}><Save size={14} /> {feeSaving ? 'Saving...' : 'Save'}</button>
                  </>
                ) : (
                  <button className="btn btn--outline btn--sm" onClick={() => setFeeEditing(true)}><Edit2 size={14} /> Edit Fees</button>
                ))}
                <Link to={`/fees`} className="btn btn--outline btn--sm">View All Fees</Link>
              </div>
            </div>
            <div className="detail-fields">
              {feeEditing ? (
                <div className="detail-field">
                  <span className="detail-field__label">Class</span>
                  <select className="detail-field__input" value={feeForm.grade_level_id || ''} onChange={e => setFeeForm({ ...feeForm, grade_level_id: e.target.value })}>
                    {grades.map(g => <option key={g.id} value={g.id}>{g.name}</option>)}
                  </select>
                </div>
              ) : (
                <Field label="Class" value={feeSummary.grade_level_name} editing={false} />
              )}
              {sectionEditing ? (
                <div className="detail-field">
                  <span className="detail-field__label">Section</span>
                  <div style={{ display: 'flex', gap: '6px', alignItems: 'center' }}>
                    <select className="detail-field__input" value={sectionDraft} onChange={e => setSectionDraft(e.target.value)}>
                      <option value="">Unassigned</option>
                      {sectionsForGrade.map(sec => <option key={sec.id} value={sec.id}>{sec.name}</option>)}
                    </select>
                    <button className="btn btn--primary btn--sm" onClick={handleSaveSection} disabled={sectionSaving}>{sectionSaving ? '...' : 'Save'}</button>
                    <button className="btn btn--outline btn--sm" onClick={() => setSectionEditing(false)}><X size={13} /></button>
                  </div>
                </div>
              ) : (
                <div className="detail-field">
                  <span className="detail-field__label">Section</span>
                  <span className="detail-field__value" style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
                    {currentSectionName || <span className="sd-field__empty">Unassigned</span>}
                    {canEditFees && (
                      <button className="btn btn--outline btn--sm" onClick={() => { setSectionDraft(sectionId || ''); setSectionEditing(true) }}><Edit2 size={12} /></button>
                    )}
                  </span>
                </div>
              )}
              {feeEditing && (
                <label className="form-field--checkbox" style={{ gridColumn: '1 / -1' }}>
                  <input type="checkbox" checked={!!feeForm.is_rte} onChange={e => setFeeForm({ ...feeForm, is_rte: e.target.checked })} />
                  <span>RTE (Right to Education) — waives tuition fee &amp; discount; van fee still applies if the student uses the bus</span>
                </label>
              )}
              <Field label="Tuition Fee" value={feeEditing && !feeForm.is_rte ? feeForm.tuition_fee : fmt(feeEditing ? 0 : feeSummary.tuition_fee)} editing={feeEditing && !feeForm.is_rte} type="number" onChange={v => setFeeForm({ ...feeForm, tuition_fee: v })} />
              <Field label="Discount" value={feeEditing ? feeForm.discount_amount : (feeSummary.discount_amount ? `${fmt(feeSummary.discount_amount)}${feeSummary.discount_reason ? ` (${feeSummary.discount_reason})` : ''}` : '-')} editing={feeEditing} type="number" onChange={v => setFeeForm({ ...feeForm, discount_amount: v })} />
              {feeEditing && (
                <Field label="Discount Reason" value={feeForm.discount_reason} editing={true} onChange={v => setFeeForm({ ...feeForm, discount_reason: v })} />
              )}
              <Field label="Net Tuition" value={fmt(feeSummary.net_tuition_fee)} editing={false} />
              <Field label="Van Fee" value={feeEditing ? feeForm.van_fee : fmt(feeSummary.van_fee)} editing={feeEditing} type="number" onChange={v => setFeeForm({ ...feeForm, van_fee: v })} />
              <Field label="Previous Dues" value={feeEditing ? feeForm.previous_year_dues : fmt(feeSummary.previous_year_dues)} editing={feeEditing} type="number" onChange={v => setFeeForm({ ...feeForm, previous_year_dues: v })} />
              <Field label="Late Fee" value={feeEditing ? feeForm.late_fee : fmt(feeSummary.late_fee)} editing={feeEditing} type="number" onChange={v => setFeeForm({ ...feeForm, late_fee: v })} />
              <Field label="Total Due" value={fmt(feeSummary.total_due)} editing={false} />
              <Field label="Total Paid" value={fmt(feeSummary.total_paid)} editing={false} />
              <Field label="Balance" value={fmt(feeSummary.balance_remaining)} editing={false} />
              {feeSummary.is_rte && <Field label="RTE Status" value="Yes - Right to Education" editing={false} />}
            </div>
            {feeSummary.payments && feeSummary.payments.length > 0 && (
              <div style={{ marginTop: '12px' }}>
                <h4 style={{ fontSize: '14px', marginBottom: '8px', color: 'var(--gray-600)' }}>Recent Payments</h4>
                <div className="table-card" style={{ overflowX: 'auto' }}>
                  <table className="data-table" style={{ fontSize: '13px' }}>
                  <thead><tr><th>Date</th><th>Type</th><th>Amount</th><th>Mode</th></tr></thead>
                  <tbody>
                    {feeSummary.payments.slice(0, 5).map(p => (
                      <tr key={p.id}>
                        <td>{formatDate(p.payment_date)}</td>
                        <td>{p.fee_type}</td>
                        <td style={{ color: 'var(--success-600)' }}>{fmt(p.amount)}</td>
                        <td>{p.payment_mode}</td>
                      </tr>
                    ))}
                  </tbody>
                  </table>
                </div>
              </div>
            )}
          </div>
        )}
      </div>

      <CustomFieldsSection entityType="student" entityId={student.id} schoolId={currentSchool?.id} user={user} />

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

function Field({ label, value, editing, onChange, type = 'text', options, placeholder }) {
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
        <input className="detail-field__input" type={type} value={value || ''} onChange={e => onChange(e.target.value)} placeholder={placeholder} />
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
