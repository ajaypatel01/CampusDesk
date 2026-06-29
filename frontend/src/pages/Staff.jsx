import { useState, useEffect, useMemo } from 'react'
import { Link } from 'react-router-dom'
import { Search, Filter, X, Users, GraduationCap, Briefcase, Phone, ChevronRight } from 'lucide-react'
import { useSchool } from '../services/SchoolContext'
import { staffApi } from '../services/api'
import './Staff.css'

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
}

const staffTypeLabel = { teaching: 'Teaching', non_teaching: 'Non-Teaching' }
const staffTypeBadge = { teaching: 'staff-type--teaching', non_teaching: 'staff-type--non-teaching' }

function Staff() {
  const { currentSchool } = useSchool()
  const [staff, setStaff] = useState([])
  const [loading, setLoading] = useState(true)
  const [search, setSearch] = useState('')
  const [typeFilter, setTypeFilter] = useState('')
  const [designFilter, setDesignFilter] = useState('')

  useEffect(() => {
    if (!currentSchool) return
    setLoading(true)
    staffApi.list({ school_id: currentSchool.id, limit: 500 })
      .then(res => setStaff(res.items || []))
      .catch(() => setStaff([]))
      .finally(() => setLoading(false))
  }, [currentSchool])

  const designations = useMemo(() => {
    const set = new Set()
    staff
      .filter(m => !typeFilter || m.profile?.staff_type === typeFilter)
      .forEach(m => { if (m.profile?.designation) set.add(m.profile.designation) })
    return [...set].sort()
  }, [staff, typeFilter])

  const filtered = useMemo(() => {
    let result = staff
    if (search) {
      const q = search.toLowerCase()
      result = result.filter(m => {
        const name = `${m.first_name} ${m.last_name}`.toLowerCase()
        const desig = (m.profile?.designation || '').toLowerCase()
        const phone = (m.profile?.phone || '').toLowerCase()
        return name.includes(q) || desig.includes(q) || phone.includes(q)
      })
    }
    if (typeFilter) result = result.filter(m => m.profile?.staff_type === typeFilter)
    if (designFilter) result = result.filter(m => m.profile?.designation === designFilter)
    return result
  }, [staff, search, typeFilter, designFilter])

  const stats = useMemo(() => {
    const s = { total: staff.length, teaching: 0, nonTeaching: 0, totalSalary: 0 }
    staff.forEach(m => {
      if (m.profile?.staff_type === 'non_teaching') s.nonTeaching++
      else s.teaching++
      s.totalSalary += m.profile?.salary || 0
    })
    return s
  }, [staff])

  const activeFilterCount = [typeFilter, designFilter].filter(Boolean).length

  function clearFilters() {
    setTypeFilter('')
    setDesignFilter('')
    setSearch('')
  }

  if (!currentSchool) return <p className="empty-text">Select a school first.</p>

  return (
    <div className="staff-page">
      <div className="page-header">
        <div>
          <h1>Staff</h1>
          <p className="page-subtitle">All staff members with their profiles</p>
        </div>
      </div>

      <div className="staff-stats">
        <div className="sstat-card sstat-card--clickable" onClick={() => setTypeFilter('')}>
          <div className="sstat-card__icon sstat-card__icon--primary"><Users size={18} /></div>
          <div className="sstat-card__body">
            <span className="sstat-card__value">{stats.total}</span>
            <span className="sstat-card__label">Total Staff</span>
          </div>
        </div>
        <div className={`sstat-card sstat-card--clickable ${typeFilter === 'teaching' ? 'sstat-card--active' : ''}`} onClick={() => setTypeFilter(t => t === 'teaching' ? '' : 'teaching')}>
          <div className="sstat-card__icon sstat-card__icon--info"><GraduationCap size={18} /></div>
          <div className="sstat-card__body">
            <span className="sstat-card__value">{stats.teaching}</span>
            <span className="sstat-card__label">Teaching</span>
          </div>
        </div>
        <div className={`sstat-card sstat-card--clickable ${typeFilter === 'non_teaching' ? 'sstat-card--active' : ''}`} onClick={() => setTypeFilter(t => t === 'non_teaching' ? '' : 'non_teaching')}>
          <div className="sstat-card__icon sstat-card__icon--warning"><Briefcase size={18} /></div>
          <div className="sstat-card__body">
            <span className="sstat-card__value">{stats.nonTeaching}</span>
            <span className="sstat-card__label">Non-Teaching</span>
          </div>
        </div>
        <div className="sstat-card">
          <div className="sstat-card__icon sstat-card__icon--success"><span className="sstat-rupee">₹</span></div>
          <div className="sstat-card__body">
            <span className="sstat-card__value">₹{stats.totalSalary.toLocaleString('en-IN')}</span>
            <span className="sstat-card__label">Total Salary</span>
          </div>
        </div>
      </div>

      <div className="page-filters">
        <div className="filter-search">
          <Search size={18} />
          <input
            type="text"
            placeholder="Search name, designation, phone..."
            value={search}
            onChange={e => setSearch(e.target.value)}
          />
        </div>
        <div className="filter-select">
          <Filter size={16} />
          <select value={typeFilter} onChange={e => { setTypeFilter(e.target.value); setDesignFilter('') }}>
            <option value="">All Types</option>
            <option value="teaching">Teaching</option>
            <option value="non_teaching">Non-Teaching</option>
          </select>
        </div>
        <div className="filter-select">
          <select value={designFilter} onChange={e => setDesignFilter(e.target.value)}>
            <option value="">All Designations</option>
            {designations.map(d => <option key={d} value={d}>{d}</option>)}
          </select>
        </div>
        {activeFilterCount > 0 && (
          <button className="filter-clear" onClick={clearFilters}>
            <X size={14} /> Clear ({activeFilterCount})
          </button>
        )}
      </div>

      <div className="page-count">Showing {filtered.length} of {staff.length} staff</div>

      {loading ? (
        <p className="loading-text">Loading...</p>
      ) : filtered.length === 0 ? (
        <p className="empty-text">No staff found</p>
      ) : (
        <div className="table-card">
          <table className="data-table">
            <thead>
              <tr>
                <th>Name</th>
                <th>Type</th>
                <th>Designation</th>
                <th>Role</th>
                <th>Phone</th>
                <th>Qualification</th>
                <th>Salary</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {filtered.map(m => (
                <tr key={m.id}>
                  <td>
                    <Link to={`/staff/${m.id}`} className="staff-table__user">
                      <div className={`staff-table__avatar ${m.profile?.staff_type === 'non_teaching' ? 'staff-table__avatar--non-teaching' : ''}`}>
                        {m.first_name[0]}{m.last_name[0]}
                      </div>
                      <div>
                        <div className="data-table__link">{m.first_name} {m.last_name}</div>
                        <div className="data-table__muted staff-email">{m.email}</div>
                      </div>
                    </Link>
                  </td>
                  <td>
                    {m.profile?.staff_type ? (
                      <span className={`staff-type-badge ${staffTypeBadge[m.profile.staff_type] || ''}`}>
                        {staffTypeLabel[m.profile.staff_type] || m.profile.staff_type}
                      </span>
                    ) : <span className="data-table__muted">—</span>}
                  </td>
                  <td>{m.profile?.designation || <span className="data-table__muted">—</span>}</td>
                  <td>
                    <span className={`badge badge--${roleBadge[m.role] || 'muted'}`}>
                      {roleLabels[m.role] || m.role}
                    </span>
                  </td>
                  <td>
                    {m.profile?.phone ? (
                      <span className="staff-phone"><Phone size={13} /> {m.profile.phone}</span>
                    ) : <span className="data-table__muted">—</span>}
                  </td>
                  <td className="data-table__muted">
                    {m.profile?.education_qualification || '—'}
                  </td>
                  <td>
                    {m.profile?.salary > 0
                      ? <span className="staff-salary">₹{m.profile.salary.toLocaleString('en-IN')}</span>
                      : <span className="data-table__muted">—</span>}
                  </td>
                  <td>
                    <Link to={`/staff/${m.id}`} className="data-table__action">
                      <ChevronRight size={16} />
                    </Link>
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

export default Staff
