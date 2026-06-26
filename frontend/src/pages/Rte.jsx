import { useState, useEffect } from 'react'
import { Plus, Trash2, Users, BarChart2, Settings } from 'lucide-react'
import { useSchool } from '../services/SchoolContext'
import { rteApi, academicApi } from '../services/api'
import './Rte.css'

function Rte() {
  const { currentSchool, currentYear } = useSchool()
  const [tab, setTab] = useState('summary')
  const [summary, setSummary] = useState(null)
  const [students, setStudents] = useState([])
  const [quotas, setQuotas] = useState([])
  const [grades, setGrades] = useState([])
  const [loading, setLoading] = useState(false)

  const [showQuotaModal, setShowQuotaModal] = useState(false)
  const [quotaForm, setQuotaForm] = useState({ grade_level_id: '', total_seats: '', govt_reimbursement_per_student: '', notes: '' })
  const [saving, setSaving] = useState(false)

  function fmt(n) {
    return new Intl.NumberFormat('en-IN', { style: 'currency', currency: 'INR', maximumFractionDigits: 0 }).format(n || 0)
  }

  function gradeName(id) {
    const g = grades.find(g => g.id === id)
    return g ? g.name : id
  }

  useEffect(() => {
    if (!currentSchool || !currentYear) return
    setLoading(true)
    const params = { school_id: currentSchool.id, academic_year_id: currentYear.id }
    Promise.all([
      rteApi.getSummary(params).catch(() => null),
      rteApi.listStudents(params).catch(() => ({ items: [] })),
      rteApi.listQuotas(params).catch(() => ({ items: [] })),
      academicApi.listGrades(currentSchool.id).catch(() => ({ items: [] })),
    ]).then(([sum, studs, qts, gds]) => {
      setSummary(sum)
      setStudents(studs.items || [])
      setQuotas(qts.items || [])
      setGrades(gds.items || [])
    }).finally(() => setLoading(false))
  }, [currentSchool, currentYear])

  async function handleSaveQuota(e) {
    e.preventDefault()
    setSaving(true)
    try {
      await rteApi.upsertQuota({
        school_id: currentSchool.id,
        academic_year_id: currentYear.id,
        grade_level_id: quotaForm.grade_level_id,
        total_seats: parseInt(quotaForm.total_seats, 10) || 0,
        govt_reimbursement_per_student: parseInt(quotaForm.govt_reimbursement_per_student, 10) || 0,
        notes: quotaForm.notes,
      })
      setShowQuotaModal(false)
      setQuotaForm({ grade_level_id: '', total_seats: '', govt_reimbursement_per_student: '', notes: '' })
      const res = await rteApi.listQuotas({ school_id: currentSchool.id, academic_year_id: currentYear.id })
      setQuotas(res.items || [])
      const sum = await rteApi.getSummary({ school_id: currentSchool.id, academic_year_id: currentYear.id }).catch(() => null)
      setSummary(sum)
    } catch (err) { alert(err.message) }
    finally { setSaving(false) }
  }

  async function handleDeleteQuota(id) {
    if (!confirm('Delete this RTE quota?')) return
    try {
      await rteApi.deleteQuota(id)
      setQuotas(prev => prev.filter(q => q.id !== id))
    } catch (err) { alert(err.message) }
  }

  if (!currentSchool || !currentYear) return (
    <div className="rte-page"><p className="empty-text">Select a school and academic year first.</p></div>
  )

  return (
    <div className="rte-page">
      <div className="page-header">
        <div>
          <h1>RTE Management</h1>
          <p className="page-subtitle">Right to Education — quotas, students, and reimbursements for {currentYear.name}</p>
        </div>
      </div>

      <div className="rte-tabs">
        {[['summary', 'Summary', BarChart2], ['students', 'RTE Students', Users], ['quotas', 'Grade Quotas', Settings]].map(([key, label, Icon]) => (
          <button key={key} className={`rte-tab ${tab === key ? 'rte-tab--active' : ''}`} onClick={() => setTab(key)}>
            <Icon size={15} /> {label}
          </button>
        ))}
      </div>

      {loading && <p className="empty-text">Loading...</p>}

      {!loading && tab === 'summary' && (
        <div>
          {!summary ? (
            <p className="empty-text">No RTE data for {currentYear.name}. Set up grade quotas first.</p>
          ) : (
            <>
              <div className="rte-summary-cards">
                <div className="rte-stat-card">
                  <div className="rte-stat-card__label">Total RTE Seats</div>
                  <div className="rte-stat-card__value">{summary.total_seats || 0}</div>
                </div>
                <div className="rte-stat-card">
                  <div className="rte-stat-card__label">Students Enrolled</div>
                  <div className="rte-stat-card__value rte-stat-card__value--primary">{summary.students_enrolled || 0}</div>
                </div>
                <div className="rte-stat-card">
                  <div className="rte-stat-card__label">Seats Remaining</div>
                  <div className="rte-stat-card__value">{(summary.total_seats || 0) - (summary.students_enrolled || 0)}</div>
                </div>
                <div className="rte-stat-card">
                  <div className="rte-stat-card__label">Total Reimbursement</div>
                  <div className="rte-stat-card__value rte-stat-card__value--success">{fmt(summary.total_reimbursement)}</div>
                </div>
              </div>

              {summary.by_grade && summary.by_grade.length > 0 && (
                <div className="table-card" style={{ marginTop: '20px' }}>
                  <table className="data-table">
                    <thead>
                      <tr>
                        <th>Grade</th>
                        <th>Total Seats</th>
                        <th>Enrolled</th>
                        <th>Remaining</th>
                        <th>Reimbursement/Student</th>
                        <th>Total Reimbursement</th>
                      </tr>
                    </thead>
                    <tbody>
                      {summary.by_grade.map((g, i) => (
                        <tr key={i}>
                          <td><strong>{g.grade_level_name}</strong></td>
                          <td>{g.total_seats}</td>
                          <td>{g.students_enrolled}</td>
                          <td>{g.total_seats - g.students_enrolled}</td>
                          <td>{fmt(g.govt_reimbursement_per_student)}</td>
                          <td>{fmt(g.govt_reimbursement_per_student * g.students_enrolled)}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </>
          )}
        </div>
      )}

      {!loading && tab === 'students' && (
        <div>
          <div className="table-card">
            <table className="data-table">
              <thead>
                <tr>
                  <th>Student</th>
                  <th>Code</th>
                  <th>Class</th>
                  <th>Category</th>
                </tr>
              </thead>
              <tbody>
                {students.length === 0 ? (
                  <tr><td colSpan={4} className="data-table__empty">No RTE students found for {currentYear.name}</td></tr>
                ) : students.map(s => (
                  <tr key={s.id}>
                    <td><strong>{s.first_name} {s.last_name}</strong></td>
                    <td className="data-table__muted">{s.student_code}</td>
                    <td>{s.grade_level_name || '-'}</td>
                    <td>{s.category || '-'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {!loading && tab === 'quotas' && (
        <div>
          <div className="rte-section-header">
            <h2>RTE Quotas by Grade</h2>
            <button className="btn btn--primary" onClick={() => setShowQuotaModal(true)}>
              <Plus size={16} /> Set Quota
            </button>
          </div>
          <div className="table-card">
            <table className="data-table">
              <thead>
                <tr>
                  <th>Grade</th>
                  <th>Total Seats</th>
                  <th>Govt. Reimbursement/Student</th>
                  <th>Notes</th>
                  <th></th>
                </tr>
              </thead>
              <tbody>
                {quotas.length === 0 ? (
                  <tr><td colSpan={5} className="data-table__empty">No quotas set. Add grade-wise RTE quotas.</td></tr>
                ) : quotas.map(q => (
                  <tr key={q.id}>
                    <td><strong>{gradeName(q.grade_level_id)}</strong></td>
                    <td>{q.total_seats}</td>
                    <td>{fmt(q.govt_reimbursement_per_student)}</td>
                    <td className="data-table__muted">{q.notes || '-'}</td>
                    <td>
                      <button className="btn btn--outline btn--sm" onClick={() => handleDeleteQuota(q.id)}>
                        <Trash2 size={13} />
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {showQuotaModal && (
        <div className="modal-overlay" onClick={() => setShowQuotaModal(false)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <h2>Set RTE Quota</h2>
            <p className="modal-hint">If a quota for the selected grade already exists, it will be updated.</p>
            <form className="modal__form" onSubmit={handleSaveQuota}>
              <label className="form-field">
                <span>Grade Level *</span>
                <select required value={quotaForm.grade_level_id} onChange={e => setQuotaForm({ ...quotaForm, grade_level_id: e.target.value })}>
                  <option value="">Select grade...</option>
                  {grades.map(g => <option key={g.id} value={g.id}>{g.name}</option>)}
                </select>
              </label>
              <div className="form-row">
                <label className="form-field"><span>Total RTE Seats *</span><input type="number" required min="0" value={quotaForm.total_seats} onChange={e => setQuotaForm({ ...quotaForm, total_seats: e.target.value })} /></label>
                <label className="form-field"><span>Govt. Reimbursement/Student (₹)</span><input type="number" min="0" value={quotaForm.govt_reimbursement_per_student} onChange={e => setQuotaForm({ ...quotaForm, govt_reimbursement_per_student: e.target.value })} /></label>
              </div>
              <label className="form-field"><span>Notes</span><textarea rows={2} value={quotaForm.notes} onChange={e => setQuotaForm({ ...quotaForm, notes: e.target.value })} /></label>
              <div className="modal__actions">
                <button type="button" className="btn btn--outline" onClick={() => setShowQuotaModal(false)}>Cancel</button>
                <button type="submit" className="btn btn--primary" disabled={saving}>{saving ? 'Saving...' : 'Save Quota'}</button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}

export default Rte
