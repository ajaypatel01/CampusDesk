import { useState, useEffect, useRef } from 'react'
import { useOutletContext } from 'react-router-dom'
import { Plus, ChevronRight, ChevronDown, Trash2, Paperclip } from 'lucide-react'
import { useSchool } from '../services/SchoolContext'
import { homeworkApi, academicApi, studentsApi, resultsApi, classMediaApi } from '../services/api'
import './Homework.css'
import { formatDate } from '../utils/date'

function Homework() {
  const { currentSchool, currentYear } = useSchool()
  const { user } = useOutletContext() || {}
  const isTeacher = user?.role === 'teacher'
  // A teacher's own sections this year (class or vice class teacher).
  const [mySections, setMySections] = useState([])
  const [sectionId, setSectionId] = useState('')
  const fileRef = useRef(null)
  const [grades, setGrades] = useState([])
  const [subjects, setSubjects] = useState([])
  const [selectedGrade, setSelectedGrade] = useState('')
  const [assignments, setAssignments] = useState([])
  const [loading, setLoading] = useState(false)
  const [expandedId, setExpandedId] = useState(null)
  const [submissions, setSubmissions] = useState({})

  const [showForm, setShowForm] = useState(false)
  const [form, setForm] = useState({
    title: '', description: '', subject_id: '', assigned_date: new Date().toISOString().split('T')[0], due_date: '',
  })

  // Submission modal
  const [subModal, setSubModal] = useState(null) // { assignmentId }
  const [students, setStudents] = useState([])
  const [subForm, setSubForm] = useState({ student_id: '', status: 'submitted', submitted_date: new Date().toISOString().split('T')[0], remarks: '' })

  useEffect(() => {
    if (!currentSchool) return
    academicApi.listGrades(currentSchool.id)
      .then(async r => {
        let g = r.items || []
        if (isTeacher) {
          // Teachers only see and set homework for their own class.
          const secs = ((await classMediaApi.sections(currentSchool.id).catch(() => ({ items: [] }))).items || []).filter(s => s.is_current_year)
          setMySections(secs)
          g = g.filter(gr => secs.some(s => s.grade_level_id === gr.id))
        }
        setGrades(g); if (g.length) setSelectedGrade(g[0].id)
      })
      .catch(() => {})
    studentsApi.list({ school_id: currentSchool.id, limit: 500 })
      .then(r => setStudents(r.items || [])).catch(() => {})
  }, [currentSchool, isTeacher])

  useEffect(() => {
    if (!currentSchool || !currentYear || !selectedGrade) return
    setLoading(true)
    homeworkApi.list({ school_id: currentSchool.id, academic_year_id: currentYear.id, grade_level_id: selectedGrade })
      .then(r => setAssignments(r.items || []))
      .catch(() => {})
      .finally(() => setLoading(false))
    resultsApi.listSubjects({ school_id: currentSchool.id, grade_level_id: selectedGrade })
      .then(r => setSubjects(r.items || [])).catch(() => {})
  }, [currentSchool, currentYear, selectedGrade])

  async function handleCreate(e) {
    e.preventDefault()
    try {
      const sectionsHere = mySections.filter(sec => sec.grade_level_id === selectedGrade)
      await homeworkApi.create({
        school_id: currentSchool.id, academic_year_id: currentYear.id, grade_level_id: selectedGrade,
        class_section_id: isTeacher ? (sectionId || sectionsHere[0]?.id) : undefined,
        title: form.title, description: form.description,
        subject_id: form.subject_id || undefined,
        assigned_date: form.assigned_date,
        due_date: form.due_date,
      }, fileRef.current?.files?.[0])
      if (fileRef.current) fileRef.current.value = 
      setShowForm(false)
      setForm({ title: '', description: '', subject_id: '', assigned_date: new Date().toISOString().split('T')[0], due_date: '' })
      homeworkApi.list({ school_id: currentSchool.id, academic_year_id: currentYear.id, grade_level_id: selectedGrade })
        .then(r => setAssignments(r.items || []))
    } catch (err) { alert(err.message) }
  }

  async function handleDelete(id) {
    if (!confirm('Delete this assignment?')) return
    try {
      await homeworkApi.delete(id)
      setAssignments(prev => prev.filter(a => a.id !== id))
    } catch (err) { alert(err.message) }
  }

  async function toggleExpand(id) {
    if (expandedId === id) { setExpandedId(null); return }
    setExpandedId(id)
    if (!submissions[id]) {
      const r = await homeworkApi.listSubmissions(id).catch(() => ({ items: [] }))
      setSubmissions(prev => ({ ...prev, [id]: r.items || [] }))
    }
  }

  async function handleAddSubmission(e) {
    e.preventDefault()
    try {
      await homeworkApi.upsertSubmission(subModal.assignmentId, subForm)
      const r = await homeworkApi.listSubmissions(subModal.assignmentId)
      setSubmissions(prev => ({ ...prev, [subModal.assignmentId]: r.items || [] }))
      setSubModal(null)
      setSubForm({ student_id: '', status: 'submitted', submitted_date: new Date().toISOString().split('T')[0], remarks: '' })
    } catch (err) { alert(err.message) }
  }

  function statusBadge(status) {
    const map = { submitted: 'success', late: 'warning', missing: 'danger', pending: 'muted' }
    return map[status] || 'muted'
  }

  function fmtDate(s) { return formatDate(s) }

  if (!currentSchool || !currentYear) return <p className="empty-text">Select a school and academic year first.</p>

  return (
    <div className="homework-page">
      <div className="page-header">
        <div>
          <h1>Homework Tracker</h1>
          <p className="page-subtitle">Assignments and submission tracking</p>
        </div>
        <button className="btn btn--primary" onClick={() => setShowForm(!showForm)}>
          <Plus size={16} /> New Assignment
        </button>
      </div>

      <div className="results-grade-bar">
        <span className="results-grade-label">Grade:</span>
        {grades.map(g => (
          <button key={g.id} className={`grade-chip ${selectedGrade === g.id ? 'grade-chip--active' : ''}`} onClick={() => setSelectedGrade(g.id)}>
            {g.name}
          </button>
        ))}
      </div>

      {showForm && (
        <form className="results-inline-form" onSubmit={handleCreate} style={{ marginBottom: '20px' }}>
          <div className="form-row">
            <label className="form-field"><span>Title *</span><input required value={form.title} onChange={e => setForm({ ...form, title: e.target.value })} placeholder="Assignment title" /></label>
            <label className="form-field">
              <span>Subject</span>
              <select value={form.subject_id} onChange={e => setForm({ ...form, subject_id: e.target.value })}>
                <option value="">All Subjects</option>
                {subjects.map(s => <option key={s.id} value={s.id}>{s.name}</option>)}
              </select>
            </label>
          </div>
          <label className="form-field">
            <span>Description</span>
            <textarea rows={2} value={form.description} onChange={e => setForm({ ...form, description: e.target.value })} />
          </label>
          <div className="form-row">
            <label className="form-field"><span>Assigned Date</span><input type="date" value={form.assigned_date} onChange={e => setForm({ ...form, assigned_date: e.target.value })} /></label>
            <label className="form-field"><span>Due Date *</span><input type="date" required value={form.due_date} onChange={e => setForm({ ...form, due_date: e.target.value })} /></label>
            {isTeacher && mySections.filter(sec => sec.grade_level_id === selectedGrade).length > 1 && (
              <label className="form-field"><span>Section *</span>
                <select value={sectionId} onChange={e => setSectionId(e.target.value)}>
                  {mySections.filter(sec => sec.grade_level_id === selectedGrade).map(sec => <option key={sec.id} value={sec.id}>{sec.grade_name} {sec.section_name}</option>)}
                </select>
              </label>
            )}
            <label className="form-field"><span>Attachment (image or PDF, optional)</span><input ref={fileRef} type="file" accept="application/pdf,image/*" /></label>
          </div>
          <div style={{ display: 'flex', gap: '8px', flexWrap: 'wrap' }}>
            <button type="submit" className="btn btn--primary">Create</button>
            <button type="button" className="btn btn--outline" onClick={() => setShowForm(false)}>Cancel</button>
          </div>
        </form>
      )}

      {loading ? <p className="loading-text">Loading...</p> : (
        <div className="hw-list">
          {assignments.length === 0 ? <p className="empty-text">No assignments for this grade yet.</p> : (
            assignments.map(a => {
              const sub = subjects.find(s => s.id === a.subject_id)
              const isExpanded = expandedId === a.id
              const isOverdue = new Date(a.due_date) < new Date()
              return (
                <div key={a.id} className="hw-item">
                  <div className="hw-item__header">
                    <div className="hw-item__info" onClick={() => toggleExpand(a.id)} style={{ cursor: 'pointer' }}>
                      {isExpanded ? <ChevronDown size={16} /> : <ChevronRight size={16} />}
                      <div>
                        <span className="hw-item__title">{a.title}</span>
                        <span className="hw-item__meta">
                          Due: {fmtDate(a.due_date)}
                          {isOverdue && <span className="badge badge--danger" style={{ marginLeft: '6px' }}>Overdue</span>}
                          {sub && <span className="badge badge--muted" style={{ marginLeft: '6px' }}>{sub.name}</span>}
                        </span>
                        {a.description && <span className="hw-item__desc">{a.description}</span>}
                        {a.attachment_url && (
                          <a href={a.attachment_url} target="_blank" rel="noreferrer" className="hw-item__desc" onClick={e => e.stopPropagation()} style={{ display: 'inline-flex', alignItems: 'center', gap: '4px' }}>
                            <Paperclip size={13} /> {a.attachment_name || 'Attachment'}
                          </a>
                        )}
                      </div>
                    </div>
                    <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
                      <button className="btn btn--outline btn--sm" onClick={() => setSubModal({ assignmentId: a.id })}>+ Submission</button>
                      <button className="btn btn--outline btn--sm" onClick={() => handleDelete(a.id)}><Trash2 size={14} /></button>
                    </div>
                  </div>

                  {isExpanded && (
                    <div className="hw-submissions">
                      {!submissions[a.id] ? <p className="loading-text">Loading...</p> : (
                        submissions[a.id].length === 0 ? <p className="empty-text" style={{ padding: '12px 16px' }}>No submissions recorded.</p> : (
                          <table className="data-table">
                            <thead><tr><th>Student</th><th>Status</th><th>Submitted</th><th>Remarks</th></tr></thead>
                            <tbody>
                              {submissions[a.id].map(s => (
                                <tr key={s.id}>
                                  <td className="data-table__muted">{s.student_id.slice(0, 8)}...</td>
                                  <td><span className={`badge badge--${statusBadge(s.status)}`}>{s.status}</span></td>
                                  <td className="data-table__muted">{s.submitted_date ? fmtDate(s.submitted_date) : '-'}</td>
                                  <td className="data-table__muted">{s.remarks || '-'}</td>
                                </tr>
                              ))}
                            </tbody>
                          </table>
                        )
                      )}
                    </div>
                  )}
                </div>
              )
            })
          )}
        </div>
      )}

      {subModal && (
        <div className="modal-overlay" onClick={() => setSubModal(null)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <h2>Record Submission</h2>
            <form className="modal__form" onSubmit={handleAddSubmission}>
              <label className="form-field">
                <span>Student *</span>
                <select required value={subForm.student_id} onChange={e => setSubForm({ ...subForm, student_id: e.target.value })}>
                  <option value="">Select student...</option>
                  {students.map(s => <option key={s.id} value={s.id}>{s.first_name} {s.last_name} ({s.student_code})</option>)}
                </select>
              </label>
              <div className="form-row">
                <label className="form-field">
                  <span>Status</span>
                  <select value={subForm.status} onChange={e => setSubForm({ ...subForm, status: e.target.value })}>
                    <option value="submitted">Submitted</option>
                    <option value="late">Late</option>
                    <option value="missing">Missing</option>
                    <option value="pending">Pending</option>
                  </select>
                </label>
                <label className="form-field">
                  <span>Submitted Date</span>
                  <input type="date" value={subForm.submitted_date} onChange={e => setSubForm({ ...subForm, submitted_date: e.target.value })} />
                </label>
              </div>
              <label className="form-field">
                <span>Remarks</span>
                <input value={subForm.remarks} onChange={e => setSubForm({ ...subForm, remarks: e.target.value })} placeholder="Optional" />
              </label>
              <div className="modal__actions">
                <button type="button" className="btn btn--outline" onClick={() => setSubModal(null)}>Cancel</button>
                <button type="submit" className="btn btn--primary">Save</button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}

export default Homework
