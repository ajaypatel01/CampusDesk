import { useState, useEffect } from 'react'
import { Building, CalendarDays, Layers, Plus, Trash2, IndianRupee } from 'lucide-react'
import { useSchool } from '../services/SchoolContext'
import { schoolsApi, academicApi, feesApi } from '../services/api'
import './Settings.css'

function Settings() {
  const { currentSchool, setCurrentSchool, schools, academicYears, currentYear, setCurrentYear } = useSchool()
  const [activeTab, setActiveTab] = useState('school')
  const [grades, setGrades] = useState([])
  const [sections, setSections] = useState([])
  const [feeStructures, setFeeStructures] = useState([])

  const [showFeeStructureModal, setShowFeeStructureModal] = useState(false)
  const [feeForm, setFeeForm] = useState({ grade_level_id: '', tuition_fee_annual: '', van_fee_annual: '0', num_installments: '4' })

  const [showSchoolModal, setShowSchoolModal] = useState(false)
  const [schoolForm, setSchoolForm] = useState({ name: '', code: '', address: '', phone: '', email: '' })

  const [showYearModal, setShowYearModal] = useState(false)
  const [yearForm, setYearForm] = useState({ name: '', start_date: '', end_date: '', is_current: false })

  const [showGradeModal, setShowGradeModal] = useState(false)
  const [gradeForm, setGradeForm] = useState({ name: '', sort_order: '' })

  const [showSectionModal, setShowSectionModal] = useState(false)
  const [sectionForm, setSectionForm] = useState({ grade_level_id: '', name: '', capacity: '30' })

  const [saving, setSaving] = useState(false)

  useEffect(() => {
    if (!currentSchool) return
    academicApi.listGrades(currentSchool.id)
      .then(res => setGrades(res.items || []))
      .catch(() => setGrades([]))
  }, [currentSchool])

  useEffect(() => {
    if (!currentSchool || !currentYear) return
    academicApi.listSections({ school_id: currentSchool.id, academic_year_id: currentYear.id })
      .then(res => setSections(res.items || []))
      .catch(() => setSections([]))
    feesApi.listStructures({ school_id: currentSchool.id, academic_year_id: currentYear.id })
      .then(res => setFeeStructures(res.items || []))
      .catch(() => setFeeStructures([]))
  }, [currentSchool, currentYear])

  async function handleCreateSchool(e) {
    e.preventDefault()
    setSaving(true)
    try {
      const s = await schoolsApi.create(schoolForm)
      setCurrentSchool(s)
      setShowSchoolModal(false)
      setSchoolForm({ name: '', code: '', address: '', phone: '', email: '' })
      window.location.reload()
    } catch (err) { alert(err.message) }
    finally { setSaving(false) }
  }

  async function handleCreateYear(e) {
    e.preventDefault()
    setSaving(true)
    try {
      await academicApi.createYear({ ...yearForm, school_id: currentSchool.id })
      setShowYearModal(false)
      setYearForm({ name: '', start_date: '', end_date: '', is_current: false })
      window.location.reload()
    } catch (err) { alert(err.message) }
    finally { setSaving(false) }
  }

  async function handleCreateGrade(e) {
    e.preventDefault()
    setSaving(true)
    try {
      await academicApi.createGrade({ school_id: currentSchool.id, name: gradeForm.name, sort_order: parseInt(gradeForm.sort_order, 10) || 0 })
      setShowGradeModal(false)
      setGradeForm({ name: '', sort_order: '' })
      const res = await academicApi.listGrades(currentSchool.id)
      setGrades(res.items || [])
    } catch (err) { alert(err.message) }
    finally { setSaving(false) }
  }

  async function handleCreateSection(e) {
    e.preventDefault()
    setSaving(true)
    try {
      await academicApi.createSection({
        school_id: currentSchool.id, academic_year_id: currentYear.id,
        grade_level_id: sectionForm.grade_level_id, name: sectionForm.name,
        capacity: parseInt(sectionForm.capacity, 10) || 30,
      })
      setShowSectionModal(false)
      setSectionForm({ grade_level_id: '', name: '', capacity: '30' })
      const res = await academicApi.listSections({ school_id: currentSchool.id, academic_year_id: currentYear.id })
      setSections(res.items || [])
    } catch (err) { alert(err.message) }
    finally { setSaving(false) }
  }

  async function handleCreateFeeStructure(e) {
    e.preventDefault()
    setSaving(true)
    try {
      await feesApi.createStructure({
        school_id: currentSchool.id,
        academic_year_id: currentYear.id,
        grade_level_id: feeForm.grade_level_id,
        tuition_fee_annual: parseInt(feeForm.tuition_fee_annual, 10),
        num_installments: parseInt(feeForm.num_installments, 10) || 4,
        van_fee_annual: parseInt(feeForm.van_fee_annual, 10) || 0,
      })
      setShowFeeStructureModal(false)
      setFeeForm({ grade_level_id: '', tuition_fee_annual: '', van_fee_annual: '0', num_installments: '4' })
      const res = await feesApi.listStructures({ school_id: currentSchool.id, academic_year_id: currentYear.id })
      setFeeStructures(res.items || [])
    } catch (err) { alert(err.message) }
    finally { setSaving(false) }
  }

  function fmt(amt) {
    return new Intl.NumberFormat('en-IN', { style: 'currency', currency: 'INR', maximumFractionDigits: 0 }).format(amt || 0)
  }

  function gradeName(glId) {
    const g = grades.find(g => g.id === glId)
    return g ? g.name : glId
  }

  const tabs = [
    { id: 'school', label: 'School', icon: Building },
    { id: 'academic', label: 'Academic Years', icon: CalendarDays },
    { id: 'grades', label: 'Grades & Sections', icon: Layers },
    { id: 'fees', label: 'Fee Structures', icon: IndianRupee },
  ]

  return (
    <div className="settings-page">
      <div className="settings-page__header">
        <h1>Settings</h1>
        <p className="settings-page__subtitle">Configure school, academic years, and grades</p>
      </div>

      <div className="settings-page__layout">
        <div className="settings-page__tabs">
          {tabs.map(({ id, label, icon: Icon }) => (
            <button key={id} className={`settings-page__tab ${activeTab === id ? 'settings-page__tab--active' : ''}`} onClick={() => setActiveTab(id)}>
              <Icon size={18} /> {label}
            </button>
          ))}
        </div>

        <div className="settings-page__content">
          {activeTab === 'school' && (
            <div className="settings-section">
              <div className="settings-section__header">
                <div>
                  <h2>Schools</h2>
                  <p className="settings-section__desc">Manage registered schools</p>
                </div>
                <button className="btn btn--primary btn--sm" onClick={() => setShowSchoolModal(true)}>
                  <Plus size={16} /> Add School
                </button>
              </div>
              {schools.length === 0 ? (
                <p className="empty-text">No schools registered. Add one to get started.</p>
              ) : (
                <div className="settings-list">
                  {schools.map(s => (
                    <div key={s.id} className={`settings-list__item ${currentSchool?.id === s.id ? 'settings-list__item--active' : ''}`}>
                      <div>
                        <strong>{s.name}</strong>
                        <span className="settings-list__meta">{s.code}</span>
                      </div>
                      <div className="settings-list__details">
                        {s.email && <span>{s.email}</span>}
                        {s.phone && <span>{s.phone}</span>}
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          )}

          {activeTab === 'academic' && (
            <div className="settings-section">
              <div className="settings-section__header">
                <div>
                  <h2>Academic Years</h2>
                  <p className="settings-section__desc">Manage academic year periods</p>
                </div>
                {currentSchool && (
                  <button className="btn btn--primary btn--sm" onClick={() => setShowYearModal(true)}>
                    <Plus size={16} /> Add Year
                  </button>
                )}
              </div>
              {!currentSchool ? (
                <p className="empty-text">Select a school first</p>
              ) : academicYears.length === 0 ? (
                <p className="empty-text">No academic years. Create one to get started.</p>
              ) : (
                <div className="settings-list">
                  {academicYears.map(y => (
                    <div key={y.id} className={`settings-list__item ${currentYear?.id === y.id ? 'settings-list__item--active' : ''}`}>
                      <div>
                        <strong>{y.name}</strong>
                        {y.is_current && <span className="badge badge--success">Current</span>}
                      </div>
                      <span className="settings-list__meta">
                        {new Date(y.start_date).toLocaleDateString('en-IN')} - {new Date(y.end_date).toLocaleDateString('en-IN')}
                      </span>
                    </div>
                  ))}
                </div>
              )}
            </div>
          )}

          {activeTab === 'grades' && (
            <div className="settings-section">
              <div className="settings-section__header">
                <div>
                  <h2>Grade Levels</h2>
                  <p className="settings-section__desc">Manage grades and class sections</p>
                </div>
                {currentSchool && (
                  <div className="settings-section__actions">
                    <button className="btn btn--outline btn--sm" onClick={() => setShowGradeModal(true)}>
                      <Plus size={16} /> Add Grade
                    </button>
                    {currentYear && (
                      <button className="btn btn--primary btn--sm" onClick={() => setShowSectionModal(true)}>
                        <Plus size={16} /> Add Section
                      </button>
                    )}
                  </div>
                )}
              </div>
              {!currentSchool ? (
                <p className="empty-text">Select a school first</p>
              ) : grades.length === 0 ? (
                <p className="empty-text">No grades configured. Add grade levels to organize classes.</p>
              ) : (
                <div className="grades-list">
                  {grades.sort((a, b) => a.sort_order - b.sort_order).map(g => (
                    <div key={g.id} className="grade-item">
                      <div className="grade-item__header">
                        <strong>{g.name}</strong>
                        <span className="settings-list__meta">Order: {g.sort_order}</span>
                      </div>
                      <div className="grade-item__sections">
                        {sections.filter(s => s.grade_level_id === g.id).map(s => (
                          <span key={s.id} className="section-tag">
                            {s.name} ({s.capacity})
                          </span>
                        ))}
                        {sections.filter(s => s.grade_level_id === g.id).length === 0 && (
                          <span className="settings-list__meta">No sections for {currentYear?.name || 'current year'}</span>
                        )}
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          )}

          {activeTab === 'fees' && (
            <div className="settings-section">
              <div className="settings-section__header">
                <div>
                  <h2>Fee Structures</h2>
                  <p className="settings-section__desc">Annual fee per grade for {currentYear?.name || 'current year'}</p>
                </div>
                {currentSchool && currentYear && (
                  <button className="btn btn--primary btn--sm" onClick={() => setShowFeeStructureModal(true)}>
                    <Plus size={16} /> Add Fee Structure
                  </button>
                )}
              </div>
              {!currentSchool || !currentYear ? (
                <p className="empty-text">Select a school and academic year</p>
              ) : feeStructures.length === 0 ? (
                <p className="empty-text">No fee structures for {currentYear.name}. Add one per grade.</p>
              ) : (
                <table className="data-table" style={{ marginTop: '12px' }}>
                  <thead>
                    <tr>
                      <th>Grade</th>
                      <th>Annual Tuition</th>
                      <th>Installments</th>
                      <th>Per Installment</th>
                      <th>Van Fee (Annual)</th>
                    </tr>
                  </thead>
                  <tbody>
                    {feeStructures.sort((a, b) => {
                      const ga = grades.find(g => g.id === a.grade_level_id)
                      const gb = grades.find(g => g.id === b.grade_level_id)
                      return (ga?.sort_order || 0) - (gb?.sort_order || 0)
                    }).map(fs => (
                      <tr key={fs.id}>
                        <td><strong>{gradeName(fs.grade_level_id)}</strong></td>
                        <td>{fmt(fs.tuition_fee_annual)}</td>
                        <td>{fs.num_installments}</td>
                        <td>{fmt(Math.round(fs.tuition_fee_annual / fs.num_installments))}</td>
                        <td>{fs.van_fee_annual > 0 ? fmt(fs.van_fee_annual) : '-'}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
            </div>
          )}
        </div>
      </div>

      {showSchoolModal && (
        <div className="modal-overlay" onClick={() => setShowSchoolModal(false)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <h2>Add School</h2>
            <form className="modal__form" onSubmit={handleCreateSchool}>
              <div className="form-row">
                <label className="form-field"><span>Name *</span><input required value={schoolForm.name} onChange={e => setSchoolForm({ ...schoolForm, name: e.target.value })} /></label>
                <label className="form-field"><span>Code *</span><input required value={schoolForm.code} onChange={e => setSchoolForm({ ...schoolForm, code: e.target.value })} placeholder="e.g. DPS-01" /></label>
              </div>
              <label className="form-field"><span>Address</span><input value={schoolForm.address} onChange={e => setSchoolForm({ ...schoolForm, address: e.target.value })} /></label>
              <div className="form-row">
                <label className="form-field"><span>Phone</span><input value={schoolForm.phone} onChange={e => setSchoolForm({ ...schoolForm, phone: e.target.value })} /></label>
                <label className="form-field"><span>Email</span><input type="email" value={schoolForm.email} onChange={e => setSchoolForm({ ...schoolForm, email: e.target.value })} /></label>
              </div>
              <div className="modal__actions">
                <button type="button" className="btn btn--outline" onClick={() => setShowSchoolModal(false)}>Cancel</button>
                <button type="submit" className="btn btn--primary" disabled={saving}>{saving ? 'Creating...' : 'Add School'}</button>
              </div>
            </form>
          </div>
        </div>
      )}

      {showYearModal && (
        <div className="modal-overlay" onClick={() => setShowYearModal(false)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <h2>Add Academic Year</h2>
            <form className="modal__form" onSubmit={handleCreateYear}>
              <label className="form-field"><span>Name *</span><input required value={yearForm.name} onChange={e => setYearForm({ ...yearForm, name: e.target.value })} placeholder="e.g. 2024-25" /></label>
              <div className="form-row">
                <label className="form-field"><span>Start Date *</span><input type="date" required value={yearForm.start_date} onChange={e => setYearForm({ ...yearForm, start_date: e.target.value })} /></label>
                <label className="form-field"><span>End Date *</span><input type="date" required value={yearForm.end_date} onChange={e => setYearForm({ ...yearForm, end_date: e.target.value })} /></label>
              </div>
              <label className="form-field--checkbox">
                <input type="checkbox" checked={yearForm.is_current} onChange={e => setYearForm({ ...yearForm, is_current: e.target.checked })} />
                <span>Set as current academic year</span>
              </label>
              <div className="modal__actions">
                <button type="button" className="btn btn--outline" onClick={() => setShowYearModal(false)}>Cancel</button>
                <button type="submit" className="btn btn--primary" disabled={saving}>{saving ? 'Creating...' : 'Add Year'}</button>
              </div>
            </form>
          </div>
        </div>
      )}

      {showGradeModal && (
        <div className="modal-overlay" onClick={() => setShowGradeModal(false)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <h2>Add Grade Level</h2>
            <form className="modal__form" onSubmit={handleCreateGrade}>
              <div className="form-row">
                <label className="form-field"><span>Name *</span><input required value={gradeForm.name} onChange={e => setGradeForm({ ...gradeForm, name: e.target.value })} placeholder="e.g. Class 1, UKG" /></label>
                <label className="form-field"><span>Sort Order</span><input type="number" value={gradeForm.sort_order} onChange={e => setGradeForm({ ...gradeForm, sort_order: e.target.value })} placeholder="0" /></label>
              </div>
              <div className="modal__actions">
                <button type="button" className="btn btn--outline" onClick={() => setShowGradeModal(false)}>Cancel</button>
                <button type="submit" className="btn btn--primary" disabled={saving}>{saving ? 'Creating...' : 'Add Grade'}</button>
              </div>
            </form>
          </div>
        </div>
      )}

      {showSectionModal && (
        <div className="modal-overlay" onClick={() => setShowSectionModal(false)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <h2>Add Class Section</h2>
            <form className="modal__form" onSubmit={handleCreateSection}>
              <label className="form-field">
                <span>Grade Level *</span>
                <select required value={sectionForm.grade_level_id} onChange={e => setSectionForm({ ...sectionForm, grade_level_id: e.target.value })}>
                  <option value="">Select grade</option>
                  {grades.map(g => <option key={g.id} value={g.id}>{g.name}</option>)}
                </select>
              </label>
              <div className="form-row">
                <label className="form-field"><span>Section Name *</span><input required value={sectionForm.name} onChange={e => setSectionForm({ ...sectionForm, name: e.target.value })} placeholder="e.g. A, B, C" /></label>
                <label className="form-field"><span>Capacity</span><input type="number" value={sectionForm.capacity} onChange={e => setSectionForm({ ...sectionForm, capacity: e.target.value })} /></label>
              </div>
              <div className="modal__actions">
                <button type="button" className="btn btn--outline" onClick={() => setShowSectionModal(false)}>Cancel</button>
                <button type="submit" className="btn btn--primary" disabled={saving}>{saving ? 'Creating...' : 'Add Section'}</button>
              </div>
            </form>
          </div>
        </div>
      )}
      {showFeeStructureModal && (
        <div className="modal-overlay" onClick={() => setShowFeeStructureModal(false)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <h2>Add Fee Structure</h2>
            <form className="modal__form" onSubmit={handleCreateFeeStructure}>
              <label className="form-field">
                <span>Grade Level *</span>
                <select required value={feeForm.grade_level_id} onChange={e => setFeeForm({ ...feeForm, grade_level_id: e.target.value })}>
                  <option value="">Select grade</option>
                  {grades.map(g => <option key={g.id} value={g.id}>{g.name}</option>)}
                </select>
              </label>
              <div className="form-row">
                <label className="form-field"><span>Annual Tuition Fee (₹) *</span><input type="number" required min="1" value={feeForm.tuition_fee_annual} onChange={e => setFeeForm({ ...feeForm, tuition_fee_annual: e.target.value })} placeholder="e.g. 9000" /></label>
                <label className="form-field"><span>Number of Installments</span><input type="number" min="1" max="12" value={feeForm.num_installments} onChange={e => setFeeForm({ ...feeForm, num_installments: e.target.value })} /></label>
              </div>
              <label className="form-field"><span>Annual Van Fee (₹)</span><input type="number" min="0" value={feeForm.van_fee_annual} onChange={e => setFeeForm({ ...feeForm, van_fee_annual: e.target.value })} placeholder="0" /></label>
              <div className="modal__actions">
                <button type="button" className="btn btn--outline" onClick={() => setShowFeeStructureModal(false)}>Cancel</button>
                <button type="submit" className="btn btn--primary" disabled={saving}>{saving ? 'Creating...' : 'Add Fee Structure'}</button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}

export default Settings
