import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { Check, X, Download, ExternalLink } from 'lucide-react'
import { useSchool } from '../services/SchoolContext'
import { studentsApi, academicApi } from '../services/api'
import './UdiseChecklist.css'

// The fields UDISE+ needs to complete a student's APAAR/enrollment profile.
// Every one of these already exists on the student record in CampusDesk --
// this page never talks to UDISE+ itself, it just tells staff who still
// needs data filled in here before they go complete it on the government site.
const REQUIRED_FIELDS = [
  { key: 'aadhar_number', label: 'Aadhar Number' },
  { key: 'date_of_birth', label: 'DOB' },
  { key: 'gender', label: 'Gender' },
  { key: 'category', label: 'Category' },
  { key: 'apar_id', label: 'APAAR ID' },
  { key: 'pen_number', label: 'PEN Number' },
]

function isFilled(v) {
  return v !== null && v !== undefined && String(v).trim() !== ''
}

function missingFields(student) {
  return REQUIRED_FIELDS.filter(f => !isFilled(student[f.key]))
}

function UdiseChecklist() {
  const navigate = useNavigate()
  const { currentSchool, currentYear } = useSchool()
  const [grades, setGrades] = useState([])
  const [selectedGrade, setSelectedGrade] = useState('')
  const [students, setStudents] = useState([])
  const [loading, setLoading] = useState(false)
  const [onlyIncomplete, setOnlyIncomplete] = useState(false)

  useEffect(() => {
    if (!currentSchool) return
    academicApi.listGrades(currentSchool.id)
      .then(res => {
        const items = res.items || []
        setGrades(items)
        if (items.length > 0) setSelectedGrade(prev => prev || items[0].name)
      })
      .catch(() => setGrades([]))
  }, [currentSchool])

  useEffect(() => {
    if (!currentSchool || !currentYear || !selectedGrade) return
    setLoading(true)
    studentsApi.list({
      school_id: currentSchool.id,
      academic_year_id: currentYear.id,
      grade_level: selectedGrade,
      status: 'active',
      limit: 1000,
    })
      .then(res => setStudents(res.items || []))
      .catch(() => setStudents([]))
      .finally(() => setLoading(false))
  }, [currentSchool, currentYear, selectedGrade])

  const rows = students.map(s => ({ student: s, missing: missingFields(s) }))
  const visibleRows = onlyIncomplete ? rows.filter(r => r.missing.length > 0) : rows
  const completeCount = rows.filter(r => r.missing.length === 0).length

  function exportCSV() {
    const headers = ['Scholar No', 'Name', ...REQUIRED_FIELDS.map(f => f.label), 'Missing']
    const csvRows = rows.map(({ student: s, missing }) => [
      s.student_code,
      `"${s.first_name} ${s.last_name}"`,
      ...REQUIRED_FIELDS.map(f => (isFilled(s[f.key]) ? 'Yes' : 'No')),
      `"${missing.map(f => f.label).join(', ')}"`,
    ].join(','))
    const csv = [headers.join(','), ...csvRows].join('\n')
    const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url; a.download = `udise-checklist-${selectedGrade || 'grade'}.csv`; a.click()
    URL.revokeObjectURL(url)
  }

  if (!currentSchool) return <p className="empty-text">Select a school first.</p>

  return (
    <div className="udise-page">
      <div className="page-header">
        <div>
          <h1>UDISE+ / APAAR Checklist</h1>
          <p className="page-subtitle">
            Which students are missing data needed to complete their APAAR/enrollment profile on UDISE+.
            This only checks CampusDesk records — the profile itself is still completed by staff on the
            UDISE+ portal directly.
          </p>
        </div>
        <div className="udise-header-actions">
          <a
            className="btn btn--outline"
            href="https://sdms.udiseplus.gov.in"
            target="_blank" rel="noreferrer"
          >
            <ExternalLink size={16} /> Open UDISE+
          </a>
          <button className="btn btn--outline" onClick={exportCSV} disabled={rows.length === 0}>
            <Download size={16} /> Export CSV
          </button>
        </div>
      </div>

      <div className="udise-filters">
        <label className="form-field">
          <span>Class</span>
          <select value={selectedGrade} onChange={e => setSelectedGrade(e.target.value)}>
            {grades.map(g => <option key={g.id} value={g.name}>{g.name}</option>)}
          </select>
        </label>
        <label className="form-field--checkbox">
          <input type="checkbox" checked={onlyIncomplete} onChange={e => setOnlyIncomplete(e.target.checked)} />
          <span>Show incomplete only</span>
        </label>
      </div>

      {loading ? <p className="loading-text">Loading...</p> : students.length === 0 ? (
        <p className="empty-text">No active students in this class for {currentYear?.name || 'the current year'}.</p>
      ) : (
        <>
          <div className="page-count">
            {completeCount} of {rows.length} students complete for {selectedGrade}
          </div>
          <div className="table-card udise-table-wrap">
            <table className="data-table">
              <thead>
                <tr>
                  <th>Scholar No</th>
                  <th>Name</th>
                  {REQUIRED_FIELDS.map(f => <th key={f.key}>{f.label}</th>)}
                  <th></th>
                </tr>
              </thead>
              <tbody>
                {visibleRows.length === 0 ? (
                  <tr><td colSpan={REQUIRED_FIELDS.length + 3} className="data-table__empty">All students complete</td></tr>
                ) : visibleRows.map(({ student: s, missing }) => (
                  <tr key={s.id}>
                    <td>{s.student_code}</td>
                    <td>{s.first_name} {s.last_name}</td>
                    {REQUIRED_FIELDS.map(f => (
                      <td key={f.key}>
                        {isFilled(s[f.key])
                          ? <Check size={16} className="udise-ok" />
                          : <X size={16} className="udise-missing" />}
                      </td>
                    ))}
                    <td>
                      {missing.length > 0 && (
                        <button className="btn btn--outline btn--sm" onClick={() => navigate(`/students/${s.id}`)}>
                          Complete
                        </button>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </>
      )}
    </div>
  )
}

export default UdiseChecklist
