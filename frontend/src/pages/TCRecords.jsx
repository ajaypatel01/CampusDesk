import { useState, useEffect } from 'react'
import { Search, Download, Plus, X, FileText } from 'lucide-react'
import { useSchool } from '../services/SchoolContext'
import { tcRecordsApi } from '../services/api'
import './TCRecords.css'

function fmt(d) {
  if (!d) return '—'
  return new Date(d).toLocaleDateString('en-IN')
}

function TCRecords() {
  const { currentSchool } = useSchool()
  const [records, setRecords] = useState([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [search, setSearch] = useState('')
  const [showModal, setShowModal] = useState(false)
  const [form, setForm] = useState({
    scholar_number: '', student_name: '', father_name: '', mother_name: '',
    dob: '', caste: '', category: '', date_of_admission: '', application_date: '',
    issue_date: '', class_passed: '', pen_number: '', apar_id: '', samagra_id: '',
    new_school: '', dice_code: '', remark: '',
  })
  const [saving, setSaving] = useState(false)

  function load() {
    if (!currentSchool) return
    setLoading(true)
    tcRecordsApi.list({ school_id: currentSchool.id, limit: 500 })
      .then(res => { setRecords(res.items || []); setTotal(res.total || 0) })
      .catch(() => setRecords([]))
      .finally(() => setLoading(false))
  }

  useEffect(() => { load() }, [currentSchool])

  const filtered = search
    ? records.filter(r => {
        const q = search.toLowerCase()
        return r.student_name.toLowerCase().includes(q) ||
          (r.scholar_number || '').toLowerCase().includes(q) ||
          (r.father_name || '').toLowerCase().includes(q) ||
          (r.class_passed || '').toLowerCase().includes(q) ||
          (r.new_school || '').toLowerCase().includes(q)
      })
    : records

  function exportCSV() {
    const headers = ['S.No','Scholar No','Student Name','Father Name','Mother Name','DOB','Caste','Category','Admission Date','Application Date','Issue Date','Class Passed','PEN No','APAR ID','Samagra ID','New School','DICE Code','Remark']
    const rows = filtered.map((r, i) => [
      i + 1, r.scholar_number || '', `"${r.student_name}"`,
      `"${r.father_name || ''}"`, `"${r.mother_name || ''}"`,
      r.dob ? new Date(r.dob).toLocaleDateString('en-IN') : '',
      r.caste || '', r.category || '',
      r.date_of_admission ? new Date(r.date_of_admission).toLocaleDateString('en-IN') : '',
      r.application_date ? new Date(r.application_date).toLocaleDateString('en-IN') : '',
      r.issue_date ? new Date(r.issue_date).toLocaleDateString('en-IN') : '',
      r.class_passed || '', r.pen_number || '', r.apar_id || '', r.samagra_id || '',
      `"${r.new_school || ''}"`, r.dice_code || '', `"${r.remark || ''}"`,
    ].join(','))
    const csv = [headers.join(','), ...rows].join('\n')
    const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a'); a.href = url; a.download = 'tc-records.csv'; a.click()
    URL.revokeObjectURL(url)
  }

  async function handleCreate(e) {
    e.preventDefault()
    setSaving(true)
    try {
      const payload = { ...form, school_id: currentSchool.id }
      Object.keys(payload).forEach(k => { if (payload[k] === '') payload[k] = null })
      payload.student_name = form.student_name
      await tcRecordsApi.create(payload)
      setShowModal(false)
      setForm({ scholar_number:'',student_name:'',father_name:'',mother_name:'',dob:'',caste:'',category:'',date_of_admission:'',application_date:'',issue_date:'',class_passed:'',pen_number:'',apar_id:'',samagra_id:'',new_school:'',dice_code:'',remark:'' })
      load()
    } catch (err) { alert(err.message) }
    finally { setSaving(false) }
  }

  if (!currentSchool) return <p className="empty-text">Select a school first.</p>

  return (
    <div className="tc-page">
      <div className="page-header">
        <div>
          <h1>TC Records</h1>
          <p className="page-subtitle">Transfer Certificate register — {total} records</p>
        </div>
        <div className="tc-header-actions">
          <button className="btn btn--outline" onClick={exportCSV}><Download size={16} /> Export CSV</button>
          <button className="btn btn--primary" onClick={() => setShowModal(true)}><Plus size={16} /> Add TC</button>
        </div>
      </div>

      <div className="page-filters">
        <div className="filter-search">
          <Search size={18} />
          <input placeholder="Search name, scholar no, school..." value={search} onChange={e => setSearch(e.target.value)} />
        </div>
        {search && (
          <button className="filter-clear" onClick={() => setSearch('')}><X size={14} /> Clear</button>
        )}
      </div>

      <div className="page-count">Showing {filtered.length} of {total} TC records</div>

      {loading ? <p className="loading-text">Loading...</p> : filtered.length === 0 ? (
        <p className="empty-text">No TC records found</p>
      ) : (
        <div className="table-card tc-table-wrap">
          <table className="data-table">
            <thead>
              <tr>
                <th>#</th>
                <th>Scholar No</th>
                <th>Student Name</th>
                <th>Father / Mother</th>
                <th>Class</th>
                <th>DOB</th>
                <th>Caste / Category</th>
                <th>Issue Date</th>
                <th>New School</th>
                <th>Remark</th>
              </tr>
            </thead>
            <tbody>
              {filtered.map((r, i) => (
                <tr key={r.id}>
                  <td className="data-table__muted">{i + 1}</td>
                  <td>{r.scholar_number || <span className="data-table__muted">—</span>}</td>
                  <td className="tc-name">
                    <FileText size={14} className="tc-icon" />
                    {r.student_name}
                  </td>
                  <td className="data-table__muted">
                    <div>{r.father_name || '—'}</div>
                    <div className="tc-mother">{r.mother_name || ''}</div>
                  </td>
                  <td>{r.class_passed || <span className="data-table__muted">—</span>}</td>
                  <td className="data-table__muted">{fmt(r.dob)}</td>
                  <td className="data-table__muted">
                    {r.caste || '—'} / {r.category || '—'}
                  </td>
                  <td>{fmt(r.issue_date)}</td>
                  <td className="tc-school">{r.new_school || <span className="data-table__muted">—</span>}</td>
                  <td className="data-table__muted">{r.remark || '—'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {showModal && (
        <div className="modal-overlay" onClick={() => setShowModal(false)}>
          <div className="modal modal--wide" onClick={e => e.stopPropagation()}>
            <h2>Add TC Record</h2>
            <form className="modal__form" onSubmit={handleCreate}>
              <div className="form-row">
                <label className="form-field"><span>Scholar Number</span><input value={form.scholar_number} onChange={e => setForm({...form, scholar_number: e.target.value})} /></label>
                <label className="form-field"><span>Student Name *</span><input required value={form.student_name} onChange={e => setForm({...form, student_name: e.target.value})} /></label>
              </div>
              <div className="form-row">
                <label className="form-field"><span>Father's Name</span><input value={form.father_name} onChange={e => setForm({...form, father_name: e.target.value})} /></label>
                <label className="form-field"><span>Mother's Name</span><input value={form.mother_name} onChange={e => setForm({...form, mother_name: e.target.value})} /></label>
              </div>
              <div className="form-row">
                <label className="form-field"><span>Date of Birth</span><input type="date" value={form.dob} onChange={e => setForm({...form, dob: e.target.value})} /></label>
                <label className="form-field"><span>Class Passed</span><input value={form.class_passed} onChange={e => setForm({...form, class_passed: e.target.value})} /></label>
              </div>
              <div className="form-row">
                <label className="form-field"><span>Caste</span><input value={form.caste} onChange={e => setForm({...form, caste: e.target.value})} /></label>
                <label className="form-field"><span>Category</span><input value={form.category} onChange={e => setForm({...form, category: e.target.value})} /></label>
              </div>
              <div className="form-row">
                <label className="form-field"><span>Date of Admission</span><input type="date" value={form.date_of_admission} onChange={e => setForm({...form, date_of_admission: e.target.value})} /></label>
                <label className="form-field"><span>Application Date</span><input type="date" value={form.application_date} onChange={e => setForm({...form, application_date: e.target.value})} /></label>
                <label className="form-field"><span>Issue Date</span><input type="date" value={form.issue_date} onChange={e => setForm({...form, issue_date: e.target.value})} /></label>
              </div>
              <div className="form-row">
                <label className="form-field"><span>PEN Number</span><input value={form.pen_number} onChange={e => setForm({...form, pen_number: e.target.value})} /></label>
                <label className="form-field"><span>APAR ID</span><input value={form.apar_id} onChange={e => setForm({...form, apar_id: e.target.value})} /></label>
                <label className="form-field"><span>Samagra ID</span><input value={form.samagra_id} onChange={e => setForm({...form, samagra_id: e.target.value})} /></label>
              </div>
              <label className="form-field"><span>New School</span><input value={form.new_school} onChange={e => setForm({...form, new_school: e.target.value})} /></label>
              <div className="form-row">
                <label className="form-field"><span>DICE Code</span><input value={form.dice_code} onChange={e => setForm({...form, dice_code: e.target.value})} /></label>
                <label className="form-field"><span>Remark</span><input value={form.remark} onChange={e => setForm({...form, remark: e.target.value})} /></label>
              </div>
              <div className="modal__actions">
                <button type="button" className="btn btn--outline" onClick={() => setShowModal(false)}>Cancel</button>
                <button type="submit" className="btn btn--primary" disabled={saving}>{saving ? 'Saving...' : 'Save TC'}</button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}

export default TCRecords
