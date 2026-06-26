import { useState, useEffect, useMemo } from 'react'
import { Link } from 'react-router-dom'
import { Plus, Search, Mail, Filter, X, ArrowUpDown, LayoutGrid, List, Users, UserCog, ShieldCheck, Clock } from 'lucide-react'
import { useSchool } from '../services/SchoolContext'
import { usersApi } from '../services/api'
import './Teachers.css'

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

const sortOptions = [
  { value: '', label: 'Default' },
  { value: 'name_asc', label: 'Name A-Z' },
  { value: 'name_desc', label: 'Name Z-A' },
  { value: 'role_asc', label: 'Role A-Z' },
  { value: 'role_desc', label: 'Role Z-A' },
  { value: 'created_asc', label: 'Oldest First' },
  { value: 'created_desc', label: 'Newest First' },
]

function Teachers() {
  const { currentSchool } = useSchool()
  const [users, setUsers] = useState([])
  const [loading, setLoading] = useState(true)
  const [search, setSearch] = useState('')
  const [roleFilter, setRoleFilter] = useState('')
  const [activeFilter, setActiveFilter] = useState('')
  const [sortBy, setSortBy] = useState('')
  const [viewMode, setViewMode] = useState('table')
  const [showModal, setShowModal] = useState(false)
  const [form, setForm] = useState({
    first_name: '', last_name: '', email: '', password: '', role: 'teacher',
  })
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    if (!currentSchool) return
    setLoading(true)
    usersApi.list({ school_id: currentSchool.id, limit: 100 })
      .then(res => setUsers(res.items || []))
      .catch(() => setUsers([]))
      .finally(() => setLoading(false))
  }, [currentSchool])

  const displayUsers = useMemo(() => {
    let result = users
    if (search) {
      const q = search.toLowerCase()
      result = result.filter(u => {
        const name = `${u.first_name} ${u.last_name}`.toLowerCase()
        return name.includes(q) || u.email.toLowerCase().includes(q)
      })
    }
    if (roleFilter) result = result.filter(u => u.role === roleFilter)
    if (activeFilter === 'active') result = result.filter(u => u.is_active)
    if (activeFilter === 'inactive') result = result.filter(u => !u.is_active)
    if (sortBy) {
      const [field, dir] = sortBy.split('_')
      result = [...result].sort((a, b) => {
        let va, vb
        if (field === 'name') {
          va = `${a.first_name} ${a.last_name}`.toLowerCase()
          vb = `${b.first_name} ${b.last_name}`.toLowerCase()
        } else if (field === 'role') {
          va = a.role; vb = b.role
        } else if (field === 'created') {
          va = a.created_at || ''; vb = b.created_at || ''
        }
        if (va < vb) return dir === 'asc' ? -1 : 1
        if (va > vb) return dir === 'asc' ? 1 : -1
        return 0
      })
    }
    return result
  }, [users, search, roleFilter, activeFilter, sortBy])

  const stats = useMemo(() => {
    const s = { total: users.length, active: 0, teachers: 0, admins: 0 }
    users.forEach(u => {
      if (u.is_active) s.active++
      if (u.role === 'teacher') s.teachers++
      if (u.role === 'school_admin' || u.role === 'super_admin') s.admins++
    })
    return s
  }, [users])

  const activeFilterCount = [roleFilter, activeFilter].filter(Boolean).length

  function clearFilters() {
    setSearch('')
    setRoleFilter('')
    setActiveFilter('')
    setSortBy('')
  }

  async function handleCreate(e) {
    e.preventDefault()
    setSaving(true)
    try {
      await usersApi.create({ ...form, school_id: currentSchool.id })
      setShowModal(false)
      setForm({ first_name: '', last_name: '', email: '', password: '', role: 'teacher' })
      const res = await usersApi.list({ school_id: currentSchool.id, limit: 100 })
      setUsers(res.items || [])
    } catch (err) {
      alert(err.message)
    } finally {
      setSaving(false)
    }
  }

  if (!currentSchool) return <p className="empty-text">Select a school first.</p>

  return (
    <div className="teachers-page">
      <div className="page-header">
        <div>
          <h1>Teachers & Staff</h1>
          <p className="page-subtitle">Manage school users and roles</p>
        </div>
        <button className="btn btn--primary" onClick={() => setShowModal(true)}>
          <Plus size={18} /> Add User
        </button>
      </div>

      <div className="teachers-stats">
        <div className="tstat-card">
          <div className="tstat-card__icon tstat-card__icon--primary"><Users size={18} /></div>
          <div className="tstat-card__body">
            <span className="tstat-card__value">{stats.total}</span>
            <span className="tstat-card__label">Total Users</span>
          </div>
        </div>
        <div className="tstat-card">
          <div className="tstat-card__icon tstat-card__icon--success"><ShieldCheck size={18} /></div>
          <div className="tstat-card__body">
            <span className="tstat-card__value">{stats.active}</span>
            <span className="tstat-card__label">Active</span>
          </div>
        </div>
        <div className="tstat-card">
          <div className="tstat-card__icon tstat-card__icon--info"><UserCog size={18} /></div>
          <div className="tstat-card__body">
            <span className="tstat-card__value">{stats.teachers}</span>
            <span className="tstat-card__label">Teachers</span>
          </div>
        </div>
        <div className="tstat-card">
          <div className="tstat-card__icon tstat-card__icon--warning"><ShieldCheck size={18} /></div>
          <div className="tstat-card__body">
            <span className="tstat-card__value">{stats.admins}</span>
            <span className="tstat-card__label">Admins</span>
          </div>
        </div>
      </div>

      <div className="page-filters">
        <div className="filter-search">
          <Search size={18} />
          <input type="text" placeholder="Search by name or email..." value={search} onChange={e => setSearch(e.target.value)} />
        </div>
        <div className="filter-select">
          <Filter size={16} />
          <select value={roleFilter} onChange={e => setRoleFilter(e.target.value)}>
            <option value="">All Roles</option>
            <option value="teacher">Teacher</option>
            <option value="school_admin">School Admin</option>
            <option value="registrar">Registrar</option>
            <option value="parent">Parent</option>
            <option value="super_admin">Super Admin</option>
          </select>
        </div>
        <div className="filter-select">
          <select value={activeFilter} onChange={e => setActiveFilter(e.target.value)}>
            <option value="">All Users</option>
            <option value="active">Active Only</option>
            <option value="inactive">Inactive Only</option>
          </select>
        </div>
        <div className="filter-select">
          <ArrowUpDown size={16} />
          <select value={sortBy} onChange={e => setSortBy(e.target.value)}>
            {sortOptions.map(o => <option key={o.value} value={o.value}>{o.label}</option>)}
          </select>
        </div>
        {activeFilterCount > 0 && (
          <button className="filter-clear" onClick={clearFilters}>
            <X size={14} /> Clear ({activeFilterCount})
          </button>
        )}
        <div className="view-toggle">
          <button className={`view-toggle__btn ${viewMode === 'table' ? 'view-toggle__btn--active' : ''}`} onClick={() => setViewMode('table')}><List size={16} /></button>
          <button className={`view-toggle__btn ${viewMode === 'grid' ? 'view-toggle__btn--active' : ''}`} onClick={() => setViewMode('grid')}><LayoutGrid size={16} /></button>
        </div>
      </div>

      <div className="page-count">Showing {displayUsers.length} of {users.length} users</div>

      {loading ? <p className="loading-text">Loading...</p> : displayUsers.length === 0 ? (
        <p className="empty-text">No users found</p>
      ) : viewMode === 'table' ? (
        <div className="table-card">
          <table className="data-table">
            <thead>
              <tr>
                <th>User</th>
                <th>Email</th>
                <th>Role</th>
                <th>Status</th>
                <th>Joined</th>
              </tr>
            </thead>
            <tbody>
              {displayUsers.map(u => (
                <tr key={u.id}>
                  <td>
                    <Link to={`/teachers/${u.id}`} className="teacher-table__user">
                      <div className="teacher-table__avatar">{u.first_name[0]}{u.last_name[0]}</div>
                      <span className="data-table__link">{u.first_name} {u.last_name}</span>
                    </Link>
                  </td>
                  <td className="data-table__muted">{u.email}</td>
                  <td><span className={`badge badge--${roleBadge[u.role] || 'muted'}`}>{roleLabels[u.role] || u.role}</span></td>
                  <td>
                    <span className={`teacher-status-dot ${u.is_active ? 'teacher-status-dot--active' : ''}`} />
                    <span className={u.is_active ? '' : 'data-table__muted'}>{u.is_active ? 'Active' : 'Inactive'}</span>
                  </td>
                  <td className="data-table__muted">{u.created_at ? new Date(u.created_at).toLocaleDateString('en-IN') : '-'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <div className="teachers-grid">
          {displayUsers.map(u => (
            <Link key={u.id} to={`/teachers/${u.id}`} className="teacher-card">
              <div className="teacher-card__top">
                <div className="teacher-card__avatar">{u.first_name[0]}{u.last_name[0]}</div>
                <div>
                  <h3>{u.first_name} {u.last_name}</h3>
                  <span className={`badge badge--${roleBadge[u.role] || 'muted'}`}>
                    {roleLabels[u.role] || u.role}
                  </span>
                </div>
              </div>
              <div className="teacher-card__info">
                <div className="teacher-card__info-item">
                  <Mail size={14} />
                  <span>{u.email}</span>
                </div>
                <div className="teacher-card__info-item">
                  <span className={`teacher-status-dot ${u.is_active ? 'teacher-status-dot--active' : ''}`} />
                  <span className={`teacher-card__status ${u.is_active ? 'teacher-card__status--active' : ''}`}>
                    {u.is_active ? 'Active' : 'Inactive'}
                  </span>
                </div>
                <div className="teacher-card__info-item">
                  <Clock size={14} />
                  <span>Joined {u.created_at ? new Date(u.created_at).toLocaleDateString('en-IN') : '-'}</span>
                </div>
              </div>
            </Link>
          ))}
        </div>
      )}

      {showModal && (
        <div className="modal-overlay" onClick={() => setShowModal(false)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <h2>Add User</h2>
            <form className="modal__form" onSubmit={handleCreate}>
              <div className="form-row">
                <label className="form-field">
                  <span>First Name *</span>
                  <input required value={form.first_name} onChange={e => setForm({ ...form, first_name: e.target.value })} />
                </label>
                <label className="form-field">
                  <span>Last Name *</span>
                  <input required value={form.last_name} onChange={e => setForm({ ...form, last_name: e.target.value })} />
                </label>
              </div>
              <label className="form-field">
                <span>Email *</span>
                <input type="email" required value={form.email} onChange={e => setForm({ ...form, email: e.target.value })} />
              </label>
              <label className="form-field">
                <span>Password *</span>
                <input type="password" required minLength={6} value={form.password} onChange={e => setForm({ ...form, password: e.target.value })} />
              </label>
              <label className="form-field">
                <span>Role</span>
                <select value={form.role} onChange={e => setForm({ ...form, role: e.target.value })}>
                  <option value="teacher">Teacher</option>
                  <option value="school_admin">School Admin</option>
                  <option value="registrar">Registrar</option>
                  <option value="parent">Parent</option>
                </select>
              </label>
              <div className="modal__actions">
                <button type="button" className="btn btn--outline" onClick={() => setShowModal(false)}>Cancel</button>
                <button type="submit" className="btn btn--primary" disabled={saving}>
                  {saving ? 'Creating...' : 'Add User'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}

export default Teachers
