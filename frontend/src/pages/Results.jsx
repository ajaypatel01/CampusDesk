import { useState, useEffect } from 'react'
import { Plus, Trash2, Download, BookOpen, ClipboardList, BarChart2 } from 'lucide-react'
import { useSchool } from '../services/SchoolContext'
import { resultsApi, academicApi, studentsApi } from '../services/api'
import './Results.css'

function downloadBlob(blob, filename) {
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url; a.download = filename; a.click()
  URL.revokeObjectURL(url)
}

function Results() {
  const { currentSchool, currentYear } = useSchool()
  const [tab, setTab] = useState('subjects')
  const [grades, setGrades] = useState([])
  const [selectedGrade, setSelectedGrade] = useState('')

  // Subjects
  const [subjects, setSubjects] = useState([])
  const [showSubjectForm, setShowSubjectForm] = useState(false)
  const [subjectForm, setSubjectForm] = useState({ name: '', code: '', max_marks: 100, passing_marks: 33, sort_order: 0 })

  // Exams
  const [exams, setExams] = useState([])
  const [showExamForm, setShowExamForm] = useState(false)
  const [examForm, setExamForm] = useState({ name: '', exam_date: '', weight_percent: 100 })

  // Marks
  const [selectedExamId, setSelectedExamId] = useState('')
  const [selectedStudentId, setSelectedStudentId] = useState('')
  const [students, setStudents] = useState([])
  const [markSubjects, setMarkSubjects] = useState([])
  const [marks, setMarks] = useState({}) // subjectId → { marks_obtained, is_absent }
  const [markSaving, setMarkSaving] = useState(false)
  const [markMsg, setMarkMsg] = useState('')

  // Marksheet
  const [msExamId, setMsExamId] = useState('')
  const [msStudentId, setMsStudentId] = useState('')
  const [marksheet, setMarksheet] = useState(null)
  const [msLoading, setMsLoading] = useState(false)

  useEffect(() => {
    if (!currentSchool) return
    academicApi.listGrades(currentSchool.id)
      .then(r => { const g = r.items || []; setGrades(g); if (g.length) setSelectedGrade(g[0].id) })
      .catch(() => {})
  }, [currentSchool])

  useEffect(() => {
    if (!currentSchool || !currentYear || !selectedGrade) return
    resultsApi.listSubjects({ school_id: currentSchool.id, grade_level_id: selectedGrade })
      .then(r => setSubjects(r.items || [])).catch(() => {})
    resultsApi.listExams({ school_id: currentSchool.id, academic_year_id: currentYear.id, grade_level_id: selectedGrade })
      .then(r => setExams(r.items || [])).catch(() => {})
    studentsApi.list({ school_id: currentSchool.id, limit: 500 })
      .then(r => setStudents(r.items || [])).catch(() => {})
  }, [currentSchool, currentYear, selectedGrade])

  useEffect(() => {
    if (!selectedGrade) return
    resultsApi.listSubjects({ school_id: currentSchool?.id, grade_level_id: selectedGrade })
      .then(r => setMarkSubjects(r.items || [])).catch(() => {})
  }, [selectedGrade, currentSchool])

  async function handleAddSubject(e) {
    e.preventDefault()
    try {
      await resultsApi.createSubject({ school_id: currentSchool.id, grade_level_id: selectedGrade, ...subjectForm })
      setShowSubjectForm(false)
      setSubjectForm({ name: '', code: '', max_marks: 100, passing_marks: 33, sort_order: 0 })
      resultsApi.listSubjects({ school_id: currentSchool.id, grade_level_id: selectedGrade }).then(r => setSubjects(r.items || []))
    } catch (err) { alert(err.message) }
  }

  async function handleDeleteSubject(id) {
    if (!confirm('Delete this subject?')) return
    try {
      await resultsApi.deleteSubject(id)
      setSubjects(prev => prev.filter(s => s.id !== id))
    } catch (err) { alert(err.message) }
  }

  async function handleAddExam(e) {
    e.preventDefault()
    try {
      await resultsApi.createExam({
        school_id: currentSchool.id, academic_year_id: currentYear.id, grade_level_id: selectedGrade,
        name: examForm.name, exam_date: examForm.exam_date || undefined,
        weight_percent: parseInt(examForm.weight_percent, 10) || 100,
      })
      setShowExamForm(false)
      setExamForm({ name: '', exam_date: '', weight_percent: 100 })
      resultsApi.listExams({ school_id: currentSchool.id, academic_year_id: currentYear.id, grade_level_id: selectedGrade })
        .then(r => setExams(r.items || []))
    } catch (err) { alert(err.message) }
  }

  async function handleSaveMarks(e) {
    e.preventDefault()
    if (!selectedExamId || !selectedStudentId) return
    setMarkSaving(true); setMarkMsg('')
    try {
      const marksArr = markSubjects.map(sub => ({
        exam_id: selectedExamId,
        student_id: selectedStudentId,
        subject_id: sub.id,
        marks_obtained: parseFloat(marks[sub.id]?.marks_obtained || 0),
        max_marks: sub.max_marks,
        is_absent: marks[sub.id]?.is_absent || false,
        remarks: '',
      }))
      await resultsApi.bulkUpsertMarks(marksArr)
      setMarkMsg('Marks saved successfully.')
    } catch (err) { setMarkMsg('Error: ' + err.message) }
    setMarkSaving(false)
  }

  async function loadMarksheet() {
    if (!msExamId || !msStudentId) return
    setMsLoading(true); setMarksheet(null)
    try {
      const ms = await resultsApi.getMarksheet(msExamId, msStudentId)
      setMarksheet(ms)
    } catch (err) { alert(err.message) }
    setMsLoading(false)
  }

  async function downloadMarksheet() {
    if (!msExamId || !msStudentId) return
    try {
      const blob = await resultsApi.downloadMarksheet(msExamId, msStudentId)
      downloadBlob(blob, `marksheet.pdf`)
    } catch (err) { alert(err.message) }
  }

  if (!currentSchool || !currentYear) return <p className="empty-text">Select a school and academic year first.</p>

  return (
    <div className="results-page">
      <div className="page-header">
        <div>
          <h1>Results & Marksheets</h1>
          <p className="page-subtitle">Manage subjects, exams, marks, and download marksheets</p>
        </div>
      </div>

      <div className="results-grade-bar">
        <span className="results-grade-label">Grade:</span>
        {grades.map(g => (
          <button key={g.id} className={`grade-chip ${selectedGrade === g.id ? 'grade-chip--active' : ''}`} onClick={() => setSelectedGrade(g.id)}>
            {g.name}
          </button>
        ))}
      </div>

      <div className="docs-tabs">
        {[['subjects','Subjects', BookOpen], ['exams','Exams', ClipboardList], ['marks','Enter Marks', Plus], ['marksheet','Marksheet', BarChart2]].map(([key, label, Icon]) => (
          <button key={key} className={`docs-tab ${tab === key ? 'docs-tab--active' : ''}`} onClick={() => setTab(key)}>
            <Icon size={16} /> {label}
          </button>
        ))}
      </div>

      {/* Subjects Tab */}
      {tab === 'subjects' && (
        <div className="results-section">
          <div className="results-section__header">
            <h2>Subjects for {grades.find(g => g.id === selectedGrade)?.name || '—'}</h2>
            <button className="btn btn--primary" onClick={() => setShowSubjectForm(!showSubjectForm)}>
              <Plus size={16} /> Add Subject
            </button>
          </div>
          {showSubjectForm && (
            <form className="results-inline-form" onSubmit={handleAddSubject}>
              <div className="form-row">
                <label className="form-field"><span>Name *</span><input required value={subjectForm.name} onChange={e => setSubjectForm({ ...subjectForm, name: e.target.value })} placeholder="e.g. Mathematics" /></label>
                <label className="form-field"><span>Code</span><input value={subjectForm.code} onChange={e => setSubjectForm({ ...subjectForm, code: e.target.value })} placeholder="MATH" /></label>
              </div>
              <div className="form-row">
                <label className="form-field"><span>Max Marks</span><input type="number" min="1" value={subjectForm.max_marks} onChange={e => setSubjectForm({ ...subjectForm, max_marks: parseInt(e.target.value) })} /></label>
                <label className="form-field"><span>Passing Marks</span><input type="number" min="1" value={subjectForm.passing_marks} onChange={e => setSubjectForm({ ...subjectForm, passing_marks: parseInt(e.target.value) })} /></label>
                <label className="form-field"><span>Sort Order</span><input type="number" min="0" value={subjectForm.sort_order} onChange={e => setSubjectForm({ ...subjectForm, sort_order: parseInt(e.target.value) })} /></label>
              </div>
              <div style={{ display: 'flex', gap: '8px' }}>
                <button type="submit" className="btn btn--primary">Save</button>
                <button type="button" className="btn btn--outline" onClick={() => setShowSubjectForm(false)}>Cancel</button>
              </div>
            </form>
          )}
          <div className="table-card">
            <table className="data-table">
              <thead><tr><th>Name</th><th>Code</th><th>Max Marks</th><th>Passing</th><th></th></tr></thead>
              <tbody>
                {subjects.length === 0 ? (
                  <tr><td colSpan={5} className="data-table__empty">No subjects yet</td></tr>
                ) : subjects.map(s => (
                  <tr key={s.id}>
                    <td>{s.name}</td>
                    <td className="data-table__muted">{s.code || '-'}</td>
                    <td>{s.max_marks}</td>
                    <td>{s.passing_marks}</td>
                    <td><button className="btn btn--outline btn--sm" onClick={() => handleDeleteSubject(s.id)}><Trash2 size={14} /></button></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Exams Tab */}
      {tab === 'exams' && (
        <div className="results-section">
          <div className="results-section__header">
            <h2>Exams</h2>
            <button className="btn btn--primary" onClick={() => setShowExamForm(!showExamForm)}>
              <Plus size={16} /> Add Exam
            </button>
          </div>
          {showExamForm && (
            <form className="results-inline-form" onSubmit={handleAddExam}>
              <div className="form-row">
                <label className="form-field"><span>Exam Name *</span><input required value={examForm.name} onChange={e => setExamForm({ ...examForm, name: e.target.value })} placeholder="e.g. Unit Test 1" /></label>
                <label className="form-field"><span>Exam Date</span><input type="date" value={examForm.exam_date} onChange={e => setExamForm({ ...examForm, exam_date: e.target.value })} /></label>
                <label className="form-field"><span>Weight %</span><input type="number" min="1" max="100" value={examForm.weight_percent} onChange={e => setExamForm({ ...examForm, weight_percent: e.target.value })} /></label>
              </div>
              <div style={{ display: 'flex', gap: '8px' }}>
                <button type="submit" className="btn btn--primary">Save</button>
                <button type="button" className="btn btn--outline" onClick={() => setShowExamForm(false)}>Cancel</button>
              </div>
            </form>
          )}
          <div className="table-card">
            <table className="data-table">
              <thead><tr><th>Exam Name</th><th>Date</th><th>Weight</th><th>Published</th></tr></thead>
              <tbody>
                {exams.length === 0 ? (
                  <tr><td colSpan={4} className="data-table__empty">No exams yet</td></tr>
                ) : exams.map(e => (
                  <tr key={e.id}>
                    <td>{e.name}</td>
                    <td className="data-table__muted">{e.exam_date ? new Date(e.exam_date).toLocaleDateString('en-IN') : '-'}</td>
                    <td>{e.weight_percent}%</td>
                    <td><span className={`badge badge--${e.is_published ? 'success' : 'muted'}`}>{e.is_published ? 'Published' : 'Draft'}</span></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Enter Marks Tab */}
      {tab === 'marks' && (
        <div className="results-section">
          <h2>Enter Marks</h2>
          <div className="form-row" style={{ marginBottom: '16px' }}>
            <label className="form-field">
              <span>Exam *</span>
              <select value={selectedExamId} onChange={e => setSelectedExamId(e.target.value)}>
                <option value="">Select exam...</option>
                {exams.map(ex => <option key={ex.id} value={ex.id}>{ex.name}</option>)}
              </select>
            </label>
            <label className="form-field">
              <span>Student *</span>
              <select value={selectedStudentId} onChange={e => setSelectedStudentId(e.target.value)}>
                <option value="">Select student...</option>
                {students.map(s => <option key={s.id} value={s.id}>{s.first_name} {s.last_name} ({s.student_code})</option>)}
              </select>
            </label>
          </div>
          {selectedExamId && selectedStudentId && markSubjects.length > 0 && (
            <form onSubmit={handleSaveMarks}>
              <div className="table-card">
                <table className="data-table">
                  <thead><tr><th>Subject</th><th>Max Marks</th><th>Passing</th><th>Obtained</th><th>Absent</th></tr></thead>
                  <tbody>
                    {markSubjects.map(sub => (
                      <tr key={sub.id}>
                        <td>{sub.name}</td>
                        <td className="data-table__muted">{sub.max_marks}</td>
                        <td className="data-table__muted">{sub.passing_marks}</td>
                        <td>
                          <input
                            type="number" min="0" max={sub.max_marks} step="0.5"
                            className="marks-input"
                            disabled={marks[sub.id]?.is_absent}
                            value={marks[sub.id]?.marks_obtained || ''}
                            onChange={e => setMarks(prev => ({ ...prev, [sub.id]: { ...prev[sub.id], marks_obtained: e.target.value } }))}
                          />
                        </td>
                        <td>
                          <input
                            type="checkbox"
                            checked={marks[sub.id]?.is_absent || false}
                            onChange={e => setMarks(prev => ({ ...prev, [sub.id]: { ...prev[sub.id], is_absent: e.target.checked } }))}
                          />
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
              {markMsg && <p className={`doc-msg ${markMsg.startsWith('Error') ? 'doc-msg--error' : 'doc-msg--ok'}`} style={{ marginTop: '12px' }}>{markMsg}</p>}
              <button type="submit" className="btn btn--primary" style={{ marginTop: '12px' }} disabled={markSaving}>
                {markSaving ? 'Saving...' : 'Save Marks'}
              </button>
            </form>
          )}
          {markSubjects.length === 0 && selectedGrade && <p className="empty-text">No subjects found for this grade. Add subjects first.</p>}
        </div>
      )}

      {/* Marksheet Tab */}
      {tab === 'marksheet' && (
        <div className="results-section">
          <h2>Student Marksheet</h2>
          <div className="form-row" style={{ marginBottom: '16px' }}>
            <label className="form-field">
              <span>Exam</span>
              <select value={msExamId} onChange={e => setMsExamId(e.target.value)}>
                <option value="">Select exam...</option>
                {exams.map(ex => <option key={ex.id} value={ex.id}>{ex.name}</option>)}
              </select>
            </label>
            <label className="form-field">
              <span>Student</span>
              <select value={msStudentId} onChange={e => setMsStudentId(e.target.value)}>
                <option value="">Select student...</option>
                {students.map(s => <option key={s.id} value={s.id}>{s.first_name} {s.last_name} ({s.student_code})</option>)}
              </select>
            </label>
          </div>
          <div style={{ display: 'flex', gap: '10px', marginBottom: '20px' }}>
            <button className="btn btn--primary" onClick={loadMarksheet} disabled={!msExamId || !msStudentId || msLoading}>
              {msLoading ? 'Loading...' : 'View Marksheet'}
            </button>
            {marksheet && (
              <button className="btn btn--outline" onClick={downloadMarksheet}>
                <Download size={16} /> Download PDF
              </button>
            )}
          </div>

          {marksheet && (
            <div className="marksheet-preview">
              <div className="marksheet-header">
                <h3>{marksheet.school_name}</h3>
                <p>{marksheet.exam_name} · {marksheet.academic_year}</p>
                <p><strong>{marksheet.student_name}</strong> · {marksheet.student_code} · {marksheet.grade_level_name}</p>
              </div>
              <table className="data-table">
                <thead><tr><th>Subject</th><th>Max</th><th>Pass</th><th>Obtained</th><th>%</th><th>Grade</th><th>Status</th></tr></thead>
                <tbody>
                  {(marksheet.rows || []).map((row, i) => (
                    <tr key={i}>
                      <td>{row.subject_name}</td>
                      <td className="data-table__muted">{row.max_marks}</td>
                      <td className="data-table__muted">{row.passing_marks}</td>
                      <td>{row.is_absent ? 'Absent' : row.marks_obtained}</td>
                      <td className="data-table__muted">{row.is_absent ? '-' : row.percentage?.toFixed(1) + '%'}</td>
                      <td><span className="badge badge--muted">{row.grade}</span></td>
                      <td>
                        <span className={`badge badge--${row.status === 'Pass' ? 'success' : row.status === 'Fail' ? 'danger' : 'muted'}`}>
                          {row.status}
                        </span>
                      </td>
                    </tr>
                  ))}
                </tbody>
                <tfoot>
                  <tr className="marksheet-total">
                    <td colSpan={3}><strong>Total / Result</strong></td>
                    <td><strong>{marksheet.total_obtained?.toFixed(1)} / {marksheet.total_max}</strong></td>
                    <td><strong>{marksheet.percentage?.toFixed(1)}%</strong></td>
                    <td><span className="badge badge--muted">{marksheet.overall_grade}</span></td>
                    <td><span className={`badge badge--${marksheet.result === 'Pass' ? 'success' : 'danger'}`}>{marksheet.result}</span></td>
                  </tr>
                </tfoot>
              </table>
              <p className="marksheet-cgpa">CGPA: <strong>{marksheet.cgpa?.toFixed(2)}</strong></p>
            </div>
          )}
        </div>
      )}
    </div>
  )
}

export default Results
