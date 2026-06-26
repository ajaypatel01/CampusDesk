import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { Users, IndianRupee, UserCog, GraduationCap, TrendingUp, AlertCircle } from 'lucide-react'
import { useSchool } from '../services/SchoolContext'
import { studentsApi, usersApi, feesApi } from '../services/api'
import './Dashboard.css'

function Dashboard() {
  const { currentSchool, currentYear } = useSchool()
  const [stats, setStats] = useState({ students: 0, teachers: 0, collected: 0, outstanding: 0, byGrade: [] })
  const [recentStudents, setRecentStudents] = useState([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (!currentSchool) return
    setLoading(true)

    const promises = [
      studentsApi.list({ school_id: currentSchool.id, limit: 5 }).catch(() => ({ items: [], total: 0 })),
      usersApi.list({ school_id: currentSchool.id, limit: 100 }).catch(() => ({ items: [], total: 0 })),
    ]

    if (currentYear) {
      promises.push(
        feesApi.schoolSummary({ school_id: currentSchool.id, academic_year_id: currentYear.id }).catch(() => null)
      )
    }

    Promise.all(promises).then(([studentRes, userRes, feeRes]) => {
      const teachers = (userRes.items || []).filter(u => u.role === 'teacher')
      setRecentStudents(studentRes.items || [])
      setStats({
        students: studentRes.total || 0,
        teachers: teachers.length,
        collected: feeRes?.total_collected || 0,
        outstanding: feeRes?.total_outstanding || 0,
        byGrade: feeRes?.by_grade || [],
      })
      setLoading(false)
    })
  }, [currentSchool, currentYear])

  function formatCurrency(amt) {
    return new Intl.NumberFormat('en-IN', { style: 'currency', currency: 'INR', maximumFractionDigits: 0 }).format(amt)
  }

  if (!currentSchool) {
    return (
      <div className="dashboard__empty">
        <GraduationCap size={48} />
        <h2>Welcome to CampusDesk</h2>
        <p>Create a school in Settings to get started.</p>
        <Link to="/settings" className="btn btn--primary">Go to Settings</Link>
      </div>
    )
  }

  return (
    <div className="dashboard">
      <div className="dashboard__header">
        <div>
          <h1>{currentSchool.name}</h1>
          <p className="dashboard__subtitle">
            {currentYear ? `Academic Year: ${currentYear.name}` : 'No academic year configured'}
          </p>
        </div>
      </div>

      <div className="dashboard__stats">
        <div className="stat-card">
          <div className="stat-card__header">
            <div className="stat-card__icon stat-card__icon--primary"><Users size={20} /></div>
          </div>
          <div className="stat-card__value">{loading ? '...' : stats.students}</div>
          <div className="stat-card__label">Total Students</div>
        </div>
        <div className="stat-card">
          <div className="stat-card__header">
            <div className="stat-card__icon stat-card__icon--success"><UserCog size={20} /></div>
          </div>
          <div className="stat-card__value">{loading ? '...' : stats.teachers}</div>
          <div className="stat-card__label">Teachers</div>
        </div>
        <div className="stat-card">
          <div className="stat-card__header">
            <div className="stat-card__icon stat-card__icon--info"><IndianRupee size={20} /></div>
          </div>
          <div className="stat-card__value">{loading ? '...' : formatCurrency(stats.collected)}</div>
          <div className="stat-card__label">Fee Collected</div>
        </div>
        <div className="stat-card">
          <div className="stat-card__header">
            <div className="stat-card__icon stat-card__icon--danger"><AlertCircle size={20} /></div>
          </div>
          <div className="stat-card__value">{loading ? '...' : formatCurrency(stats.outstanding)}</div>
          <div className="stat-card__label">Outstanding</div>
        </div>
      </div>

      <div className="dashboard__grid">
        <div className="dashboard__section">
          <div className="section-header">
            <h2>Recent Students</h2>
            <Link to="/students" className="section-header__link">View all</Link>
          </div>
          {loading ? (
            <p className="loading-text">Loading...</p>
          ) : recentStudents.length === 0 ? (
            <p className="empty-text">No students yet. <Link to="/students">Add one</Link></p>
          ) : (
            <table className="data-table">
              <thead>
                <tr>
                  <th>Code</th>
                  <th>Name</th>
                  <th>Status</th>
                  <th>Gender</th>
                </tr>
              </thead>
              <tbody>
                {recentStudents.map(s => (
                  <tr key={s.id}>
                    <td className="data-table__muted">{s.student_code}</td>
                    <td>
                      <Link to={`/students/${s.id}`} className="data-table__link">
                        {s.first_name} {s.last_name}
                      </Link>
                    </td>
                    <td><span className={`badge badge--${s.status === 'active' ? 'success' : 'muted'}`}>{s.status}</span></td>
                    <td className="data-table__muted">{s.gender || '-'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>

        <div className="dashboard__sidebar-section">
          <div className="section-header">
            <h2>Fee by Grade</h2>
          </div>
          {stats.byGrade.length === 0 ? (
            <p className="empty-text">No fee data available</p>
          ) : (
            <div className="grade-fee-list">
              {stats.byGrade.map(g => (
                <div key={g.grade_level_id} className="grade-fee-item">
                  <div className="grade-fee-item__header">
                    <span className="grade-fee-item__name">{g.grade_level_name}</span>
                    <span className="grade-fee-item__count">{g.student_count} students</span>
                  </div>
                  <div className="grade-fee-item__bar">
                    <div
                      className="grade-fee-item__fill"
                      style={{ width: `${g.total_due ? (g.total_collected / g.total_due * 100) : 0}%` }}
                    />
                  </div>
                  <div className="grade-fee-item__amounts">
                    <span>{formatCurrency(g.total_collected)}</span>
                    <span className="grade-fee-item__due">of {formatCurrency(g.total_due)}</span>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

export default Dashboard
