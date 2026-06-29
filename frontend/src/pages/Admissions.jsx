import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { UserPlus, Search, Download } from 'lucide-react'
import { useSchool } from '../services/SchoolContext'
import { studentsApi, enrollmentsApi, academicApi } from '../services/api'
import './Admissions.css'

function fmt(dateStr) {
  if (!dateStr) return '—'
  return new Date(dateStr).toLocaleDateString('en-IN', { day: '2-digit', month: 'short', year: 'numeric' })
}

function statusBadge(status) {
  const map = { active: 'success', inactive: 'muted', graduated: 'info', transferred: 'warning', dropped: 'danger' }
  return `badge badge--${map[status] || 'muted'}`
}

function enrollBadge(status) {
  const map = { active: 'success', inactive: 'muted', completed: 'info', withdrawn: 'danger' }
  return `badge badge--${map[status] || 'muted'}`
}

export default function Admissions() {
  const { schools, currentSchool, setCurrentSchool, academicYears } = useSchool()
  const [selectedYearId, setSelectedYearId] = useState('')
  const [allYears, setAllYears] = useState([])
  const [rows, setRows] = useState([])
  const [loading, setLoading] = useState(false)
  const [search, setSearch] = useState('')
  const [statusFilter, setStatusFilter] = useState('')
  const [gradeFilter, setGradeFilter] = useState('')
  const [grades, setGrades] = useState([])
  const [stats, setStats] = useState({ total: 0, active: 0, girls: 0, boys: 0 })

  // Load all academic years across all schools for year-wise switching
  useEffect(() => {
    if (!currentSchool) return
    academicApi.listYears(currentSchool.id)
      .then(r => {
        const items = r.items || []
        setAllYears(items)
        if (!selectedYearId && items.length > 0) {
          const current = items.find(y => y.is_current) || items[0]
          setSelectedYearId(current.id)
        }
      })
      .catch(() => {})
  }, [currentSchool])

  // Load grades for filter
  useEffect(() => {
    if (!currentSchool) return
    academicApi.listGrades(currentSchool.id)
      .then(r => setGrades(r.items || []))
      .catch(() => {})
  }, [currentSchool])

  // Fetch students + enrollments and join them
  useEffect(() => {
    if (!currentSchool || !selectedYearId) return
    setLoading(true)

    Promise.all([
      studentsApi.list({ school_id: currentSchool.id, academic_year_id: selectedYearId, limit: 1000 }),
      enrollmentsApi.list({ school_id: currentSchool.id, academic_year_id: selectedYearId, limit: 1000 }),
    ]).then(([sRes, eRes]) => {
      const students = sRes.items || []
      const enrollments = eRes.items || []

      // Build enrollment lookup by student_id
      const enrollMap = {}
      enrollments.forEach(e => { enrollMap[e.student_id] = e })

      const joined = students.map(s => ({
        ...s,
        enrollment: enrollMap[s.id] || null,
      }))

      setRows(joined)
      setStats({
        total: joined.length,
        active: joined.filter(r => r.status === 'active').length,
        girls: joined.filter(r => r.gender === 'female' || r.gender === 'Female').length,
        boys: joined.filter(r => r.gender === 'male' || r.gender === 'Male').length,
      })
    }).catch(() => {}).finally(() => setLoading(false))
  }, [currentSchool, selectedYearId])

  const filtered = rows.filter(r => {
    if (statusFilter && r.status !== statusFilter) return false
    if (gradeFilter && r.grade_level_name !== gradeFilter) return false
    if (search) {
      const q = search.toLowerCase()
      if (
        !r.first_name?.toLowerCase().includes(q) &&
        !r.last_name?.toLowerCase().includes(q) &&
        !r.student_code?.toLowerCase().includes(q) &&
        !r.category?.toLowerCase().includes(q)
      ) return false
    }
    return true
  })

  function exportCSV() {
    const headers = ['Student Code', 'Name', 'Gender', 'Category', 'Class', 'Admission Date', 'Enrollment Date', 'Status', 'Enrollment Status']
    const csvRows = [headers.join(',')]
    filtered.forEach(r => {
      csvRows.push([
        r.student_code,
        `"${r.first_name} ${r.last_name}"`,
        r.gender,
        r.category,
        r.grade_level_name || '—',
        fmt(r.admission_date),
        fmt(r.enrollment?.enrollment_date),
        r.status,
        r.enrollment?.status || '—',
      ].join(','))
    })
    const blob = new Blob([csvRows.join('\n')], { type: 'text/csv' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `admissions_${selectedYearId}.csv`
    a.click()
    URL.revokeObjectURL(url)
  }

  if (!currentSchool) return <p className="empty-text">Select a school first.</p>

  return (
    <div className="admissions-page">
      <div className="page-header">
        <div>
          <h1>Admissions</h1>
          <p className="page-subtitle">Year-wise student admission records</p>
        </div>
        <div className="page-header__actions">
          <button className="btn btn--outline" onClick={exportCSV} disabled={filtered.length === 0}>
            <Download size={16} /> Export CSV
          </button>
          <Link to="/students" className="btn btn--primary">
            <UserPlus size={16} /> New Admission
          </Link>
        </div>
      </div>

      {/* Year + School selector row */}
      <div className="admissions-selectors">
        <div className="selector-group">
          <label>School</label>
          <select value={currentSchool.id} onChange={e => {
            const s = schools.find(s => s.id === e.target.value)
            if (s) { setCurrentSchool(s); setSelectedYearId(''); setRows([]) }
          }}>
            {schools.map(s => <option key={s.id} value={s.id}>{s.name}</option>)}
          </select>
        </div>
        <div className="selector-group">
          <label>Academic Year</label>
          <select value={selectedYearId} onChange={e => setSelectedYearId(e.target.value)}>
            {allYears.map(y => (
              <option key={y.id} value={y.id}>{y.name}{y.is_current ? ' (Current)' : ''}</option>
            ))}
          </select>
        </div>
      </div>

      {/* Stats bar */}
      <div className="admissions-stats">
        <div className="admission-stat"><span className="admission-stat__val">{stats.total}</span><span className="admission-stat__lbl">Total</span></div>
        <div className="admission-stat"><span className="admission-stat__val">{stats.active}</span><span className="admission-stat__lbl">Active</span></div>
        <div className="admission-stat"><span className="admission-stat__val">{stats.boys}</span><span className="admission-stat__lbl">Boys</span></div>
        <div className="admission-stat"><span className="admission-stat__val">{stats.girls}</span><span className="admission-stat__lbl">Girls</span></div>
      </div>

      {/* Filters */}
      <div className="admissions-filters">
        <div className="search-box">
          <Search size={16} />
          <input placeholder="Search name, code, category..." value={search} onChange={e => setSearch(e.target.value)} />
        </div>
        <select value={gradeFilter} onChange={e => setGradeFilter(e.target.value)}>
          <option value="">All Classes</option>
          {grades.map(g => <option key={g.id} value={g.name}>{g.name}</option>)}
        </select>
        <select value={statusFilter} onChange={e => setStatusFilter(e.target.value)}>
          <option value="">All Statuses</option>
          <option value="active">Active</option>
          <option value="inactive">Inactive</option>
          <option value="graduated">Graduated</option>
          <option value="transferred">Transferred</option>
          <option value="dropped">Dropped</option>
        </select>
      </div>

      {loading ? (
        <p className="loading-text">Loading admissions...</p>
      ) : filtered.length === 0 ? (
        <p className="empty-text">No admission records found for this year.</p>
      ) : (
        <div className="table-wrapper">
          <table className="data-table">
            <thead>
              <tr>
                <th>#</th>
                <th>Student Code</th>
                <th>Name</th>
                <th>Gender</th>
                <th>Category</th>
                <th>Class</th>
                <th>Admission Date</th>
                <th>Enrolled On</th>
                <th>Status</th>
                <th>Enrollment</th>
              </tr>
            </thead>
            <tbody>
              {filtered.map((r, i) => (
                <tr key={r.id}>
                  <td className="data-table__muted">{i + 1}</td>
                  <td className="data-table__mono">{r.student_code}</td>
                  <td>
                    <Link to={`/students/${r.id}`} className="table-link">
                      {r.first_name} {r.last_name}
                    </Link>
                  </td>
                  <td className="data-table__muted">{r.gender || '—'}</td>
                  <td className="data-table__muted">{r.category || '—'}</td>
                  <td>{r.grade_level_name || <span className="data-table__muted">—</span>}</td>
                  <td className="data-table__muted">{fmt(r.admission_date)}</td>
                  <td className="data-table__muted">{fmt(r.enrollment?.enrollment_date)}</td>
                  <td><span className={statusBadge(r.status)}>{r.status}</span></td>
                  <td>
                    {r.enrollment
                      ? <span className={enrollBadge(r.enrollment.status)}>{r.enrollment.status}</span>
                      : <span className="badge badge--muted">not enrolled</span>}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
