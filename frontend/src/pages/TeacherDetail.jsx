import { useState, useEffect } from 'react'
import { useParams, useNavigate, Link } from 'react-router-dom'
import { ArrowLeft, Mail, ShieldCheck, Clock, BookOpen, Users, Edit2, Save, X, Trash2 } from 'lucide-react'
import { useSchool } from '../services/SchoolContext'
import { usersApi, academicApi } from '../services/api'
import './TeacherDetail.css'

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
  const { currentSchool, currentYear } = useSchool()
  const [user, setUser] = useState(null)
  const [sections, setSections] = useState([])
  const [grades, setGrades] = useState([])
  const [loading, setLoading] = useState(true)
  const [editing, setEditing] = useState(false)
  const [form, setForm] = useState({})
  const [saving, setSaving] = useState(false)
  const [deleting, setDeleting] = useState(false)

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

  if (loading) return <p className="loading-text">Loading...</p>
  if (!user) return <p className="empty-text">User not found</p>

  const assignedSections = sections.filter(s => s.homeroom_teacher_id === user.id)
  const gradeMap = Object.fromEntries(grades.map(g => [g.id, g.name]))
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
                  {user.created_at ? new Date(user.created_at).toLocaleDateString('en-IN', { day: 'numeric', month: 'long', year: 'numeric' }) : '-'}
                </span>
              </div>
            </div>
            <div className="td-field">
              <Clock size={16} className="td-field__icon" />
              <div>
                <span className="td-field__label">Last Updated</span>
                <span className="td-field__value">
                  {user.updated_at ? new Date(user.updated_at).toLocaleDateString('en-IN', { day: 'numeric', month: 'long', year: 'numeric' }) : '-'}
                </span>
              </div>
            </div>
          </div>
        </div>

        <div className="td-card">
          <div className="td-card__header">
            <h3><BookOpen size={18} /> Class Assignments</h3>
            {currentYear && <span className="td-card__year">{currentYear.name}</span>}
          </div>
          {!currentYear ? (
            <p className="empty-text">Select an academic year to see assignments</p>
          ) : assignedSections.length === 0 ? (
            <div className="td-empty-assign">
              <Users size={32} />
              <p>No class sections assigned as homeroom teacher</p>
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
                </div>
              ))}
            </div>
          )}
        </div>

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
