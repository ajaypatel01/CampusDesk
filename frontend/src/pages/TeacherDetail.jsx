import { useState, useEffect } from 'react'
import { useParams, useNavigate, Link, useOutletContext } from 'react-router-dom'
import { ArrowLeft, Mail, ShieldCheck, Clock, BookOpen, Users, Edit2, Save, X, Trash2, Plus } from 'lucide-react'
import { useSchool } from '../services/SchoolContext'
import { usersApi, academicApi } from '../services/api'
import './TeacherDetail.css'
import { formatDate } from '../utils/date'

const roleLabels = {
  super_admin: 'Super Admin',
  school_admin: 'School Admin',
  teacher: 'Teacher',
  registrar: 'Registrar',
  parent: 'Parent',
}

const roleBadge = {
  teacher: 'info',
  school_admin: 'warning',
  super_admin: 'danger',
  registrar: 'success',
  parent: 'muted',
}

function TeacherDetail() {
  const { id } = useParams()
  const navigate = useNavigate()
  const { user: viewer } = useOutletContext() || {}
  // Same roles the backend lets touch a section's class teacher (BlockRoles
  // teacher/parent on PUT /class-sections/{id}) -- registrar included.
  const canAssignSections = viewer && viewer.role !== 'teacher' && viewer.role !== 'parent'
  const { currentSchool, currentYear } = useSchool()
  const [user, setUser] = useState(null)
  const [sections, setSections] = useState([])
  const [grades, setGrades] = useState([])
  const [loading, setLoading] = useState(true)
  const [editing, setEditing] = useState(false)
  const [form, setForm] = useState({})
  const [saving, setSaving] = useState(false)
  const [deleting, setDeleting] = useState(false)
  const [showAssignModal, setShowAssignModal] = useState(false)
  const [assignGradeId, setAssignGradeId] = useState('')
  const [assignSectionId, setAssignSectionId] = useState('')
  const [assignSaving, setAssignSaving] = useState(false)
  const [assignErr, setAssignErr] = useState('')
  const [unassigningId, setUnassigningId] = useState(null)

  useEffect(() => {
    setLoading(true)
    usersApi.get(id)
      .then(u => { setUser(u); setForm(u) })
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [id])

  async function handleSave() {
    setSaving(true)
    try {
      const updated = await usersApi.update(id, {
        first_name: form.first_name,
        last_name: form.last_name,
        email: form.email,
        is_active: form.is_active,
      })
      setUser(updated)
      setEditing(false)
    } catch (err) {
      alert(err.message)
    } finally {
      setSaving(false)
    }
  }

  async function handleDelete() {
    if (!window.confirm(`Delete ${user.first_name} ${user.last_name}'s account? This cannot be undone.`)) return
    setDeleting(true)
    try {
      await usersApi.remove(id)
      navigate('/teachers')
    } catch (err) {
      alert(err.message)
      setDeleting(false)
    }
  }

  function reloadSections() {
    if (!currentSchool || !currentYear) return
    academicApi.listSections({ school_id: currentSchool.id, academic_year_id: currentYear.id })
      .then(res => setSections(res.items || [])).catch(() => {})
  }

  useEffect(() => {
    if (!currentSchool || !currentYear) return
    Promise.all([
      academicApi.listSections({ school_id: currentSchool.id, academic_year_id: currentYear.id }),
      academicApi.listGrades(currentSchool.id),
    ]).then(([secRes, gradeRes]) => {
      setSections(secRes.items || [])
      setGrades(gradeRes.items || [])
    }).catch(() => {})
  }, [currentSchool, currentYear])

  async function handleAssignSection() {
    if (!assignSectionId) return
    const section = sections.find(s => s.id === assignSectionId)
    if (!section) return
    if (section.homeroom_teacher_id && section.homeroom_teacher_id !== user.id) {
      if (!window.confirm(`This section already has a class teacher. Reassign it to ${user.first_name} ${user.last_name} instead?`)) return
    }
    setAssignSaving(true); setAssignErr('')
    try {
      await academicApi.updateSection(section.id, {
        name: section.name, capacity: section.capacity, homeroom_teacher_id: user.id,
      })
      reloadSections()
      setShowAssignModal(false)
      setAssignGradeId(''); setAssignSectionId('')
    } catch (err) {
      setAssignErr(err.message)
    } finally {
      setAssignSaving(false)
    }
  }

  async function handleUnassignSection(section) {
    const gradeName = grades.find(g => g.id === section.grade_level_id)?.name || 'Grade'
    if (!window.confirm(`Remove ${user.first_name} ${user.last_name} as class teacher of ${gradeName} ${section.name}?`)) return
    setUnassigningId(section.id)
    try {
      await academicApi.updateSection(section.id, {
        name: section.name, capacity: section.capacity, homeroom_teacher_id: null,
      })
      reloadSections()
    } catch (err) {
      alert(err.message)
    } finally {
      setUnassigningId(null)
    }
  }

  if (loading) return <p className="loading-text">Loading...</p>
  if (!user) return <p className="empty-text">User not found</p>

  const assignedSections = sections.filter(s => s.homeroom_teacher_id === user.id)
  const gradeMap = Object.fromEntries(grades.map(g => [g.id, g.name]))
  // Sections available to assign: not already this teacher's own.
  const assignableSections = sections.filter(s => s.grade_level_id === assignGradeId && s.homeroom_teacher_id !== user.id)
  const f = editing ? form : user

  return (
    <div className="teacher-detail">
      <Link to="/teachers" className="back-link"><ArrowLeft size={18} /> Back to Teachers</Link>

      <div className="teacher-detail__hero">
        <div className="teacher-detail__avatar">
          {f.first_name?.[0]}{f.last_name?.[0]}
        </div>
        <div className="teacher-detail__hero-info">
          {editing ? (
            <div className="teacher-detail__name-edit">
              <input className="detail-field__input" value={form.first_name || ''} placeholder="First name" onChange={e => setForm({ ...form, first_name: e.target.value })} />
              <input className="detail-field__input" value={form.last_name || ''} placeholder="Last name" onChange={e => setForm({ ...form, last_name: e.target.value })} />
            </div>
          ) : (
            <h1>{user.first_name} {user.last_name}</h1>
          )}
          <div className="teacher-detail__hero-meta">
            <span className={`badge badge--${roleBadge[user.role] || 'muted'}`}>
              {roleLabels[user.role] || user.role}
            </span>
            {editing ? (
              <select className="detail-field__input" value={form.is_active ? 'active' : 'inactive'} onChange={e => setForm({ ...form, is_active: e.target.value === 'active' })}>
                <option value="active">Active</option>
                <option value="inactive">Inactive</option>
              </select>
            ) : (
              <span className={`teacher-status-pill ${user.is_active ? 'teacher-status-pill--active' : ''}`}>
                <span className={`teacher-status-dot ${user.is_active ? 'teacher-status-dot--active' : ''}`} />
                {user.is_active ? 'Active' : 'Inactive'}
              </span>
            )}
          </div>
        </div>
        <div className="teacher-detail__hero-actions">
          {editing ? (
            <>
              <button className="btn btn--outline btn--sm" onClick={() => { setEditing(false); setForm(user) }}><X size={16} /> Cancel</button>
              <button className="btn btn--primary btn--sm" onClick={handleSave} disabled={saving}>
                <Save size={16} /> {saving ? 'Saving...' : 'Save'}
              </button>
            </>
          ) : (
            <>
              <button className="btn btn--outline btn--sm" onClick={() => setEditing(true)}><Edit2 size={16} /> Edit</button>
              <button className="btn btn--danger btn--sm" onClick={handleDelete} disabled={deleting}>
                <Trash2 size={16} /> {deleting ? 'Deleting...' : 'Delete'}
              </button>
            </>
          )}
        </div>
      </div>

      <div className="teacher-detail__grid">
        <div className="td-card">
          <h3>Contact Information</h3>
          <div className="td-card__fields">
            <div className="td-field">
              <Mail size={16} className="td-field__icon" />
              <div>
                <span className="td-field__label">Email</span>
                {editing ? (
                  <input className="detail-field__input" type="email" value={form.email || ''} onChange={e => setForm({ ...form, email: e.target.value })} />
                ) : (
                  <span className="td-field__value">{user.email}</span>
                )}
              </div>
            </div>
            <div className="td-field">
              <ShieldCheck size={16} className="td-field__icon" />
              <div>
                <span className="td-field__label">Role</span>
                <span className="td-field__value">{roleLabels[user.role] || user.role}</span>
              </div>
            </div>
            <div className="td-field">
              <Clock size={16} className="td-field__icon" />
              <div>
                <span className="td-field__label">Created</span>
                <span className="td-field__value">
                  {formatDate(user.created_at)}
                </span>
              </div>
            </div>
            <div className="td-field">
              <Clock size={16} className="td-field__icon" />
              <div>
                <span className="td-field__label">Last Updated</span>
                <span className="td-field__value">
                  {formatDate(user.updated_at)}
                </span>
              </div>
            </div>
          </div>
        </div>

        <div className="td-card">
          <div className="td-card__header">
            <h3><BookOpen size={18} /> Class Assignments</h3>
            <div style={{ display: 'flex', alignItems: 'center', gap: '10px', flexWrap: 'wrap' }}>
              {currentYear && <span className="td-card__year">{currentYear.name}</span>}
              {canAssignSections && currentYear && (
                <button className="btn btn--outline btn--sm" onClick={() => { setShowAssignModal(true); setAssignGradeId(''); setAssignSectionId(''); setAssignErr('') }}>
                  <Plus size={14} /> Assign Section
                </button>
              )}
            </div>
          </div>
          {!currentYear ? (
            <p className="empty-text">Select an academic year to see assignments</p>
          ) : assignedSections.length === 0 ? (
            <div className="td-empty-assign">
              <Users size={32} />
              <p>No class sections assigned as homeroom teacher{canAssignSections ? ' -- use "Assign Section" above to add one' : ''}</p>
            </div>
          ) : (
            <div className="td-sections">
              {assignedSections.map(s => (
                <div key={s.id} className="td-section-card">
                  <div className="td-section-card__grade">{gradeMap[s.grade_level_id] || 'Grade'}</div>
                  <div className="td-section-card__info">
                    <span className="td-section-card__name">Section {s.name}</span>
                    <span className="td-section-card__cap">{s.capacity} students capacity</span>
                  </div>
                  {canAssignSections && (
                    <button
                      className="btn-icon" title="Remove as class teacher of this section"
                      onClick={() => handleUnassignSection(s)} disabled={unassigningId === s.id}
                    >
                      <X size={14} />
                    </button>
                  )}
                </div>
              ))}
            </div>
          )}
        </div>

        {showAssignModal && (
          <div className="modal-overlay" onClick={() => setShowAssignModal(false)}>
            <div className="modal" onClick={e => e.stopPropagation()}>
              <h2>Assign {user.first_name} as Class Teacher</h2>
              <div className="modal__form">
                <label className="form-field">
                  <span>Grade *</span>
                  <select value={assignGradeId} onChange={e => { setAssignGradeId(e.target.value); setAssignSectionId('') }}>
                    <option value="">Select grade</option>
                    {grades.map(g => <option key={g.id} value={g.id}>{g.name}</option>)}
                  </select>
                </label>
                <label className="form-field">
                  <span>Section *</span>
                  <select value={assignSectionId} onChange={e => setAssignSectionId(e.target.value)} disabled={!assignGradeId}>
                    <option value="">Select section</option>
                    {assignableSections.map(s => {
                      const currentTeacher = s.homeroom_teacher_id ? 'assigned' : 'unassigned'
                      return <option key={s.id} value={s.id}>{s.name} ({currentTeacher})</option>
                    })}
                  </select>
                  {assignGradeId && assignableSections.length === 0 && (
                    <span className="settings-list__meta">No sections for this grade yet -- add one in Settings → Grades &amp; Sections first.</span>
                  )}
                </label>
                {assignErr && <p className="doc-msg doc-msg--error">{assignErr}</p>}
                <div className="modal__actions">
                  <button type="button" className="btn btn--outline" onClick={() => setShowAssignModal(false)}>Cancel</button>
                  <button type="button" className="btn btn--primary" onClick={handleAssignSection} disabled={!assignSectionId || assignSaving}>
                    {assignSaving ? 'Assigning...' : 'Assign'}
                  </button>
                </div>
              </div>
            </div>
          </div>
        )}

        <div className="td-card td-card--full">
          <h3>Account Details</h3>
          <div className="td-meta-grid">
            <div className="td-meta-item">
              <span className="td-meta-item__label">User ID</span>
              <span className="td-meta-item__value td-meta-item__value--mono">{user.id}</span>
            </div>
            {user.school_id && (
              <div className="td-meta-item">
                <span className="td-meta-item__label">School ID</span>
                <span className="td-meta-item__value td-meta-item__value--mono">{user.school_id}</span>
              </div>
            )}
            <div className="td-meta-item">
              <span className="td-meta-item__label">Account Status</span>
              <span className="td-meta-item__value">
                {user.is_active ? 'Active - Can log in' : 'Inactive - Login disabled'}
              </span>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

export default TeacherDetail
