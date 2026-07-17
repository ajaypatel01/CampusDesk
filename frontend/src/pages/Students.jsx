import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { Plus, Search, Filter, ChevronLeft, ChevronRight, X, ArrowUpDown } from 'lucide-react'
import { useSchool } from '../services/SchoolContext'
import { studentsApi, academicApi } from '../services/api'
import SortHeader from '../components/SortHeader'
import './Students.css'

function Students() {
  const { currentSchool, currentYear } = useSchool()
  const [students, setStudents] = useState([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [grades, setGrades] = useState([])

  const [search, setSearch] = useState('')
  const [statusFilter, setStatusFilter] = useState('')
  const [categoryFilter, setCategoryFilter] = useState('')
  const [gradeFilter, setGradeFilter] = useState('')
  const [paymentStatus, setPaymentStatus] = useState('')
  const [sortBy, setSortBy] = useState('name')
  const [sortOrder, setSortOrder] = useState('asc')
  const [offset, setOffset] = useState(0)

  const [showModal, setShowModal] = useState(false)
  const [form, setForm] = useState({
    student_code: '', first_name: '', last_name: '', gender: '', date_of_birth: '',
    phone: '', email: '', address: '', admission_date: '', caste: '', category: '',
    aadhar_number: '', status: 'active',
  })
  const [saving, setSaving] = useState(false)
  const limit = 20

  useEffect(() => {
    if (!currentSchool) return
    academicApi.listGrades(currentSchool.id)
      .then(res => setGrades(res.items || []))
      .catch(() => setGrades([]))
  }, [currentSchool])

  useEffect(() => {
    if (!currentSchool) return
    setLoading(true)
    studentsApi.list({
      school_id: currentSchool.id,
      search: search || undefined,
      status: statusFilter || undefined,
      category: categoryFilter || undefined,
      grade_level: gradeFilter || undefined,
      payment_status: paymentStatus || undefined,
      academic_year_id: currentYear?.id || undefined,
      sort_by: sortBy || undefined,
      sort_order: sortOrder || undefined,
      limit,
      offset,
    })
      .then(res => { setStudents(res.items || []); setTotal(res.total || 0) })
      .catch(() => { setStudents([]); setTotal(0) })
      .finally(() => setLoading(false))
  }, [currentSchool, currentYear, search, statusFilter, categoryFilter, gradeFilter, paymentStatus, sortBy, sortOrder, offset])

  function handleSort(field) {
    if (sortBy === field) {
      setSortOrder(d => d === 'asc' ? 'desc' : 'asc')
    } else {
      setSortBy(field)
      setSortOrder('asc')
    }
    setOffset(0)
  }

  const activeFilterCount = [statusFilter, categoryFilter, gradeFilter, paymentStatus].filter(Boolean).length

  function clearFilters() {
    setStatusFilter('')
    setCategoryFilter('')
    setGradeFilter('')
    setPaymentStatus('')
    setSearch('')
    setSortBy('name')
    setSortOrder('asc')
    setOffset(0)
  }

  function toISODate(val) {
    if (!val) return undefined
    return new Date(val).toISOString()
  }

  async function handleCreate(e) {
    e.preventDefault()
    setSaving(true)
    try {
      const body = {
        ...form,
        school_id: currentSchool.id,
        date_of_birth: toISODate(form.date_of_birth),
        admission_date: toISODate(form.admission_date),
      }
      await studentsApi.create(body)
      setShowModal(false)
      setForm({ student_code: '', first_name: '', last_name: '', gender: '', date_of_birth: '', phone: '', email: '', address: '', admission_date: '', caste: '', category: '', aadhar_number: '', status: 'active' })
      setOffset(0)
    } catch (err) {
      alert(err.message)
    } finally {
      setSaving(false)
    }
  }

  if (!currentSchool) return <p className="empty-text">Select a school first.</p>

  return (
    <div className="students-page">
      <div className="page-header">
        <div>
          <h1>Students</h1>
          <p className="page-subtitle">Manage student records</p>
        </div>
        <button className="btn btn--primary" onClick={() => setShowModal(true)}>
          <Plus size={18} /> Add Student
        </button>
      </div>

      <div className="page-filters">
        <div className="filter-search">
          <Search size={18} />
          <input
            type="text" placeholder="Search by name or code..."
            value={search} onChange={e => { setSearch(e.target.value); setOffset(0) }}
          />
        </div>
        <div className="filter-select">
          <Filter size={16} />
          <select value={statusFilter} onChange={e => { setStatusFilter(e.target.value); setOffset(0) }}>
            <option value="">All Status</option>
            <option value="active">Active</option>
            <option value="inactive">Inactive</option>
            <option value="graduated">Graduated</option>
            <option value="transferred">Transferred</option>
          </select>
        </div>
        <div className="filter-select">
          <select value={categoryFilter} onChange={e => { setCategoryFilter(e.target.value); setOffset(0) }}>
            <option value="">All Category</option>
            <option value="General">General</option>
            <option value="OBC">OBC</option>
            <option value="SC">SC</option>
            <option value="ST">ST</option>
          </select>
        </div>
        {grades.length > 0 && (
          <div className="filter-select">
            <select value={gradeFilter} onChange={e => { setGradeFilter(e.target.value); setOffset(0) }}>
              <option value="">All Grades</option>
              {grades.map(g => <option key={g.id} value={g.name}>{g.name}</option>)}
            </select>
          </div>
        )}
        {currentYear && (
          <div className="filter-select">
            <select value={paymentStatus} onChange={e => { setPaymentStatus(e.target.value); setOffset(0) }}>
              <option value="">All Payment</option>
              <option value="paid">Fully Paid</option>
              <option value="due">Balance Due</option>
              <option value="partial">Partial Paid</option>
              <option value="unpaid">Unpaid</option>
            </select>
          </div>
        )}
        {activeFilterCount > 0 && (
          <button className="filter-clear" onClick={clearFilters}>
            <X size={14} /> Clear ({activeFilterCount})
          </button>
        )}
      </div>

      <div className="page-count">Showing {students.length} of {total} students</div>

      <div className="table-card">
        {loading ? <p className="loading-text">Loading...</p> : (
          <table className="data-table">
            <thead>
              <tr>
                <SortHeader label="Code" field="student_code" sortField={sortBy} sortDir={sortOrder} onSort={handleSort} />
                <SortHeader label="Name" field="name" sortField={sortBy} sortDir={sortOrder} onSort={handleSort} />
                <th>Gender</th>
                <th>Phone</th>
                <th>Status</th>
                <SortHeader label="Grade" field="class" sortField={sortBy} sortDir={sortOrder} onSort={handleSort} />
                <SortHeader label="Admission Date" field="admission_date" sortField={sortBy} sortDir={sortOrder} onSort={handleSort} />
              </tr>
            </thead>
            <tbody>
              {students.length === 0 ? (
                <tr><td colSpan={7} className="data-table__empty">No students found</td></tr>
              ) : students.map(s => (
                <tr key={s.id}>
                  <td className="data-table__muted">{s.student_code}</td>
                  <td>
                    <Link to={`/students/${s.id}`} className="data-table__link">
                      {s.first_name} {s.last_name}
                    </Link>
                  </td>
                  <td className="data-table__muted">{s.gender || '-'}</td>
                  <td className="data-table__muted">{s.phone || '-'}</td>
                  <td><span className={`badge badge--${s.status === 'active' ? 'success' : 'muted'}`}>{s.status}</span></td>
                  <td className="data-table__muted">{s.grade_level_name || '-'}</td>
                  <td className="data-table__muted">{s.admission_date ? new Date(s.admission_date).toLocaleDateString('en-IN') : '-'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      {total > limit && (
        <div className="pagination">
          <button className="pagination__btn" disabled={offset === 0} onClick={() => setOffset(Math.max(0, offset - limit))}>
            <ChevronLeft size={16} /> Prev
          </button>
          <span className="pagination__info">Page {Math.floor(offset / limit) + 1} of {Math.ceil(total / limit)}</span>
          <button className="pagination__btn" disabled={offset + limit >= total} onClick={() => setOffset(offset + limit)}>
            Next <ChevronRight size={16} />
          </button>
        </div>
      )}

      {showModal && (
        <div className="modal-overlay" onClick={() => setShowModal(false)}>
          <div className="modal modal--wide" onClick={e => e.stopPropagation()}>
            <h2>Add Student</h2>
            <form className="modal__form" onSubmit={handleCreate}>
              <div className="form-row">
                <label className="form-field">
                  <span>Student Code *</span>
                  <input required value={form.student_code} onChange={e => setForm({ ...form, student_code: e.target.value })} placeholder="e.g. STU001" />
                </label>
                <label className="form-field">
                  <span>First Name *</span>
                  <input required value={form.first_name} onChange={e => setForm({ ...form, first_name: e.target.value })} />
                </label>
                <label className="form-field">
                  <span>Last Name *</span>
                  <input required value={form.last_name} onChange={e => setForm({ ...form, last_name: e.target.value })} />
                </label>
              </div>
              <div className="form-row">
                <label className="form-field">
                  <span>Gender</span>
                  <select value={form.gender} onChange={e => setForm({ ...form, gender: e.target.value })}>
                    <option value="">Select</option>
                    <option value="male">Male</option>
                    <option value="female">Female</option>
                    <option value="other">Other</option>
                  </select>
                </label>
                <label className="form-field">
                  <span>Date of Birth</span>
                  <input type="date" value={form.date_of_birth} onChange={e => setForm({ ...form, date_of_birth: e.target.value })} />
                </label>
                <label className="form-field">
                  <span>Admission Date</span>
                  <input type="date" value={form.admission_date} onChange={e => setForm({ ...form, admission_date: e.target.value })} />
                </label>
              </div>
              <div className="form-row">
                <label className="form-field">
                  <span>Phone</span>
                  <input value={form.phone} onChange={e => setForm({ ...form, phone: e.target.value })} />
                </label>
                <label className="form-field">
                  <span>Email</span>
                  <input type="email" value={form.email} onChange={e => setForm({ ...form, email: e.target.value })} />
                </label>
              </div>
              <div className="form-row">
                <label className="form-field">
                  <span>Caste</span>
                  <input value={form.caste} onChange={e => setForm({ ...form, caste: e.target.value })} />
                </label>
                <label className="form-field">
                  <span>Category</span>
                  <input value={form.category} onChange={e => setForm({ ...form, category: e.target.value })} placeholder="e.g. General, OBC, SC, ST" />
                </label>
                <label className="form-field">
                  <span>Aadhar Number</span>
                  <input value={form.aadhar_number} onChange={e => setForm({ ...form, aadhar_number: e.target.value })} maxLength={12} />
                </label>
              </div>
              <label className="form-field">
                <span>Address</span>
                <textarea rows={2} value={form.address} onChange={e => setForm({ ...form, address: e.target.value })} />
              </label>
              <div className="modal__actions">
                <button type="button" className="btn btn--outline" onClick={() => setShowModal(false)}>Cancel</button>
                <button type="submit" className="btn btn--primary" disabled={saving}>
                  {saving ? 'Saving...' : 'Add Student'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}

export default Students
