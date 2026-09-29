import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { Plus, Search, Filter, ChevronLeft, ChevronRight, X, ArrowUpDown, Download, Upload, ArrowUpRight } from 'lucide-react'
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

  const [showImportModal, setShowImportModal] = useState(false)
  const [importFile, setImportFile] = useState(null)
  const [importing, setImporting] = useState(false)
  const [importResult, setImportResult] = useState(null)
  const [importError, setImportError] = useState('')
  const [saving, setSaving] = useState(false)
  const limit = 20

  // Promotion/demotion: select students on the current page, then move them
  // all to a target grade (and, usually, a target academic year) at once.
  const [selected, setSelected] = useState(() => new Set())
  const [allYears, setAllYears] = useState([])
  const [sections, setSections] = useState([])
  const [showPromoteModal, setShowPromoteModal] = useState(false)
  const [promoteForm, setPromoteForm] = useState({
    to_academic_year_id: '', to_grade_level_id: '', class_section_id: '',
    carry_forward_dues: true, carry_forward_discount: true, carry_forward_van_fee: true,
  })
  const [promoting, setPromoting] = useState(false)
  const [promoteResult, setPromoteResult] = useState(null)
  const [promoteError, setPromoteError] = useState('')

  useEffect(() => {
    if (!currentSchool) return
    academicApi.listGrades(currentSchool.id)
      .then(res => setGrades(res.items || []))
      .catch(() => setGrades([]))
    academicApi.listYears(currentSchool.id)
      .then(res => setAllYears(res.items || res || []))
      .catch(() => setAllYears([]))
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

  async function handleDownloadTemplate() {
    try {
      const blob = await studentsApi.downloadImportTemplate()
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url; a.download = 'student_import_template.xlsx'; a.click()
      URL.revokeObjectURL(url)
    } catch (err) {
      alert(err.message)
    }
  }

  async function handleImport(e) {
    e.preventDefault()
    if (!importFile) return
    setImporting(true); setImportError(''); setImportResult(null)
    try {
      const result = await studentsApi.importStudents(currentSchool.id, importFile)
      setImportResult(result)
      // Reload the roster so newly-imported students show up right away.
      const res = await studentsApi.list({ school_id: currentSchool.id, limit, offset: 0 })
      setStudents(res.items || []); setTotal(res.total || 0); setOffset(0)
    } catch (err) {
      setImportError(err.message)
    } finally {
      setImporting(false)
    }
  }

  function closeImportModal() {
    setShowImportModal(false)
    setImportFile(null)
    setImportResult(null)
    setImportError('')
  }

  function toggleSelected(id) {
    setSelected(prev => {
      const next = new Set(prev)
      next.has(id) ? next.delete(id) : next.add(id)
      return next
    })
  }

  function toggleSelectAll() {
    setSelected(prev => prev.size === students.length ? new Set() : new Set(students.map(s => s.id)))
  }

  function openPromoteModal() {
    setPromoteError(''); setPromoteResult(null)
    setPromoteForm({
      to_academic_year_id: '', to_grade_level_id: '', class_section_id: '',
      carry_forward_dues: true, carry_forward_discount: true, carry_forward_van_fee: true,
    })
    setShowPromoteModal(true)
  }

  // Sections are per academic-year, so refetch whenever the target year
  // changes; the dropdown below filters these down to the target grade.
  useEffect(() => {
    if (!currentSchool || !promoteForm.to_academic_year_id) { setSections([]); return }
    academicApi.listSections({ school_id: currentSchool.id, academic_year_id: promoteForm.to_academic_year_id })
      .then(res => setSections(res.items || []))
      .catch(() => setSections([]))
  }, [currentSchool, promoteForm.to_academic_year_id])

  const sectionsForTargetGrade = sections.filter(s => s.grade_level_id === promoteForm.to_grade_level_id)

  // Snapshot names at the moment of promotion so the result list can show
  // who succeeded/failed even after the roster reloads underneath it.
  const nameById = Object.fromEntries(students.map(s => [s.id, `${s.first_name} ${s.last_name}`]))

  async function handlePromote(e) {
    e.preventDefault()
    if (!promoteForm.to_academic_year_id || !promoteForm.to_grade_level_id) return
    setPromoting(true); setPromoteError(''); setPromoteResult(null)
    try {
      const result = await studentsApi.bulkPromote({
        student_ids: Array.from(selected),
        from_academic_year_id: currentYear?.id || undefined,
        ...promoteForm,
      })
      setPromoteResult({ ...result, nameById })
      setSelected(new Set())
      // Reload so a same-year move shows its new grade immediately.
      const res = await studentsApi.list({ school_id: currentSchool.id, academic_year_id: currentYear?.id || undefined, limit, offset })
      setStudents(res.items || []); setTotal(res.total || 0)
    } catch (err) {
      setPromoteError(err.message || 'Promotion failed')
    } finally {
      setPromoting(false)
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
        <div style={{ display: 'flex', gap: '8px', flexWrap: 'wrap' }}>
          {selected.size > 0 && (
            <button className="btn btn--outline" onClick={openPromoteModal}>
              <ArrowUpRight size={16} /> Promote/Move Grade ({selected.size})
            </button>
          )}
          <button className="btn btn--outline" onClick={handleDownloadTemplate}>
            <Download size={16} /> Download Template
          </button>
          <button className="btn btn--outline" onClick={() => setShowImportModal(true)}>
            <Upload size={16} /> Bulk Import
          </button>
          <button className="btn btn--primary" onClick={() => setShowModal(true)}>
            <Plus size={18} /> Add Student
          </button>
        </div>
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
                <th style={{ width: '32px' }}>
                  <input type="checkbox" checked={students.length > 0 && selected.size === students.length} onChange={toggleSelectAll} />
                </th>
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
                <tr><td colSpan={8} className="data-table__empty">No students found</td></tr>
              ) : students.map(s => (
                <tr key={s.id}>
                  <td><input type="checkbox" checked={selected.has(s.id)} onChange={() => toggleSelected(s.id)} /></td>
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
                  <span>Gender *</span>
                  <select required value={form.gender} onChange={e => setForm({ ...form, gender: e.target.value })}>
                    <option value="">Select</option>
                    <option value="male">Male</option>
                    <option value="female">Female</option>
                  </select>
                </label>
                <label className="form-field">
                  <span>Date of Birth *</span>
                  <input required type="date" value={form.date_of_birth} onChange={e => setForm({ ...form, date_of_birth: e.target.value })} />
                </label>
                <label className="form-field">
                  <span>Admission Date</span>
                  <input type="date" value={form.admission_date} onChange={e => setForm({ ...form, admission_date: e.target.value })} />
                </label>
              </div>
              <div className="form-row">
                <label className="form-field">
                  <span>Phone *</span>
                  <input required value={form.phone} onChange={e => setForm({ ...form, phone: e.target.value })} />
                </label>
                <label className="form-field">
                  <span>Email *</span>
                  <input required type="email" value={form.email} onChange={e => setForm({ ...form, email: e.target.value })} />
                </label>
              </div>
              <div className="form-row">
                <label className="form-field">
                  <span>Caste *</span>
                  <input required value={form.caste} onChange={e => setForm({ ...form, caste: e.target.value })} />
                </label>
                <label className="form-field">
                  <span>Category *</span>
                  <input required value={form.category} onChange={e => setForm({ ...form, category: e.target.value })} placeholder="e.g. General, OBC, SC, ST" />
                </label>
                <label className="form-field">
                  <span>Aadhar Number</span>
                  <input value={form.aadhar_number} onChange={e => setForm({ ...form, aadhar_number: e.target.value })} maxLength={12} />
                </label>
              </div>
              <label className="form-field">
                <span>Address *</span>
                <textarea required rows={2} value={form.address} onChange={e => setForm({ ...form, address: e.target.value })} />
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

      {showImportModal && (
        <div className="modal-overlay" onClick={closeImportModal}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <h2>Bulk Import Students</h2>
            <p className="empty-text">
              Download the template, fill in one row per student, then upload it here.
              Each row is validated the same way as adding a student one at a time --
              a bad row (e.g. a duplicate Student Code) won't block the rest of the file.
            </p>

            {!importResult ? (
              <form className="modal__form" onSubmit={handleImport}>
                {importError && <p className="doc-msg doc-msg--error">{importError}</p>}
                <label className="form-field">
                  <span>Filled-in Template (.xlsx) *</span>
                  <input
                    type="file" accept=".xlsx" required
                    onChange={e => setImportFile(e.target.files?.[0] || null)}
                  />
                </label>
                <div className="modal__actions">
                  <button type="button" className="btn btn--outline" onClick={closeImportModal}>Cancel</button>
                  <button type="submit" className="btn btn--primary" disabled={importing || !importFile}>
                    {importing ? 'Importing...' : 'Upload & Import'}
                  </button>
                </div>
              </form>
            ) : (
              <div>
                <p className={`doc-msg ${importResult.failed > 0 ? 'doc-msg--error' : 'doc-msg--ok'}`}>
                  {importResult.succeeded} of {importResult.total} row{importResult.total === 1 ? '' : 's'} imported successfully
                  {importResult.failed > 0 ? `, ${importResult.failed} failed.` : '.'}
                </p>
                {importResult.failed > 0 && (
                  <div className="table-card" style={{ maxHeight: '260px', overflowY: 'auto' }}>
                    <table className="data-table">
                      <thead><tr><th>Row</th><th>Status</th><th>Detail</th></tr></thead>
                      <tbody>
                        {importResult.results.map(r => (
                          <tr key={r.row_number}>
                            <td>{r.row_number}</td>
                            <td>
                              {r.success
                                ? <span className="badge badge--success">Imported</span>
                                : <span className="badge badge--danger">Failed</span>}
                            </td>
                            <td className="data-table__muted">{r.success ? r.student_code : r.error}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                )}
                <div className="modal__actions">
                  <button type="button" className="btn btn--primary" onClick={closeImportModal}>Done</button>
                </div>
              </div>
            )}
          </div>
        </div>
      )}

      {showPromoteModal && (
        <div className="modal-overlay" onClick={() => setShowPromoteModal(false)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <h2>Promote / Move Grade</h2>
            <p className="empty-text">
              Moving {selected.size} student{selected.size === 1 ? '' : 's'} to a grade. If the target academic
              year is different from {currentYear?.name || 'the current year'}, this is a promotion (or demotion) --
              a new fee account is created for that year, carrying forward whatever you check below. If it's the
              same year, this just repoints their existing fee account at the new grade.
            </p>

            {!promoteResult ? (
              <form className="modal__form" onSubmit={handlePromote}>
                {promoteError && <p className="doc-msg doc-msg--error">{promoteError}</p>}
                <div className="form-row">
                  <label className="form-field">
                    <span>Target Academic Year *</span>
                    <select required value={promoteForm.to_academic_year_id} onChange={e => setPromoteForm(f => ({ ...f, to_academic_year_id: e.target.value }))}>
                      <option value="">Select...</option>
                      {allYears.map(y => <option key={y.id} value={y.id}>{y.name}</option>)}
                    </select>
                  </label>
                  <label className="form-field">
                    <span>Target Grade *</span>
                    <select required value={promoteForm.to_grade_level_id} onChange={e => setPromoteForm(f => ({ ...f, to_grade_level_id: e.target.value, class_section_id: '' }))}>
                      <option value="">Select...</option>
                      {grades.map(g => <option key={g.id} value={g.id}>{g.name}</option>)}
                    </select>
                  </label>
                </div>
                <label className="form-field">
                  <span>Section (optional)</span>
                  <select
                    value={promoteForm.class_section_id}
                    onChange={e => setPromoteForm(f => ({ ...f, class_section_id: e.target.value }))}
                    disabled={!promoteForm.to_grade_level_id}
                  >
                    <option value="">Don&apos;t assign a section yet</option>
                    {sectionsForTargetGrade.map(s => <option key={s.id} value={s.id}>{s.name}</option>)}
                  </select>
                  {promoteForm.to_grade_level_id && sectionsForTargetGrade.length === 0 && (
                    <span className="doc-msg" style={{ marginTop: '4px' }}>
                      No sections set up yet for this grade/year -- add one under Settings first if you want to assign one now.
                    </span>
                  )}
                </label>
                <label className="form-field--checkbox">
                  <input type="checkbox" checked={promoteForm.carry_forward_discount} onChange={e => setPromoteForm(f => ({ ...f, carry_forward_discount: e.target.checked }))} />
                  <span>Carry forward any discount</span>
                </label>
                <label className="form-field--checkbox">
                  <input type="checkbox" checked={promoteForm.carry_forward_van_fee} onChange={e => setPromoteForm(f => ({ ...f, carry_forward_van_fee: e.target.checked }))} />
                  <span>Carry forward van fee</span>
                </label>
                <label className="form-field--checkbox">
                  <input type="checkbox" checked={promoteForm.carry_forward_dues} onChange={e => setPromoteForm(f => ({ ...f, carry_forward_dues: e.target.checked }))} />
                  <span>Carry forward any outstanding balance as previous year dues</span>
                </label>
                <p className="empty-text" style={{ marginTop: '-4px' }}>
                  These only apply when moving to a new academic year -- a same-year grade change just repoints
                  the existing account, nothing is duplicated.
                </p>
                <div className="modal__actions">
                  <button type="button" className="btn btn--outline" onClick={() => setShowPromoteModal(false)}>Cancel</button>
                  <button type="submit" className="btn btn--primary" disabled={promoting}>
                    {promoting ? 'Moving...' : `Move ${selected.size} Student${selected.size === 1 ? '' : 's'}`}
                  </button>
                </div>
              </form>
            ) : (
              <div>
                <p className={`doc-msg ${promoteResult.failed > 0 ? 'doc-msg--error' : 'doc-msg--ok'}`}>
                  {promoteResult.succeeded} of {promoteResult.total} student{promoteResult.total === 1 ? '' : 's'} moved successfully
                  {promoteResult.failed > 0 ? `, ${promoteResult.failed} failed.` : '.'}
                </p>
                <div className="table-card" style={{ maxHeight: '260px', overflowY: 'auto' }}>
                  <table className="data-table">
                    <thead><tr><th>Student</th><th>Status</th><th>Detail</th></tr></thead>
                    <tbody>
                      {promoteResult.results.map(r => (
                        <tr key={r.student_id}>
                          <td>{promoteResult.nameById[r.student_id] || r.student_id}</td>
                          <td>
                            {r.success
                              ? <span className="badge badge--success">Moved</span>
                              : <span className="badge badge--danger">Failed</span>}
                          </td>
                          <td className="data-table__muted">{r.error || '-'}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
                <div className="modal__actions">
                  <button type="button" className="btn btn--primary" onClick={() => setShowPromoteModal(false)}>Done</button>
                </div>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  )
}

export default Students
