import { useEffect, useState } from 'react'
import { ChevronLeft, ChevronRight, X } from 'lucide-react'
import { useSchool } from '../services/SchoolContext'
import { auditApi, usersApi } from '../services/api'
import useSessionState from '../hooks/useSessionState'
import { todayIST } from '../utils/date'
import './ActivityLog.css'

// What each table is called on screen. Anything missing falls back to its
// name with underscores turned into spaces.
const RECORD_TYPES = {
  students: 'Student', guardians: 'Guardian', student_guardians: 'Student–guardian link',
  enrollments: 'Class enrolment', exam_marks: 'Exam mark', exam_mark_components: 'Exam mark field',
  exams: 'Exam', subjects: 'Subject', subject_mark_components: 'Subject field',
  exam_subject_mark_formats: 'Exam marks format', marksheet_total_overrides: 'Marksheet total',
  report_card_details: 'Report card details', discipline_grades: 'Discipline grade',
  fee_payments: 'Fee payment', student_fee_accounts: 'Fee account', fee_structures: 'Fee structure',
  fee_installment_plans: 'Fee instalment', users: 'User account', staff_profiles: 'Staff profile',
  staff_leaves: 'Staff leave', class_sections: 'Class section', class_section_vice_teachers: 'Vice class teacher',
  grade_levels: 'Class', academic_years: 'Academic year', schools: 'School',
  attendance_records: 'Attendance', homework_assignments: 'Homework', homework_submissions: 'Homework submission',
  broadcasts: 'Broadcast', tc_records: 'TC record', vouchers: 'Voucher', vans: 'Van', van_routes: 'Van route',
  student_van_assignments: 'Van assignment', books: 'Book', book_lists: 'Book list', book_list_items: 'Book list item',
  student_book_receipts: 'Book receipt', rte_quotas: 'RTE quota', permission_overrides: 'Permission',
  custom_field_definitions: 'Custom field', custom_field_values: 'Custom field value',
}
const ACTIONS = { INSERT: 'Added', UPDATE: 'Changed', DELETE: 'Deleted' }
const ACTION_BADGE = { INSERT: 'success', UPDATE: 'info', DELETE: 'danger' }
// Bookkeeping columns not worth showing as "changes".
const HIDDEN_FIELDS = new Set(['id', 'created_at', 'updated_at'])

function recordType(table) {
  return RECORD_TYPES[table] || table.replace(/_/g, ' ')
}

function fieldLabel(key) {
  return key.replace(/_id$/, '').replace(/_/g, ' ')
}

function showValue(v) {
  if (v === null || v === undefined || v === '') return '—'
  if (typeof v === 'boolean') return v ? 'yes' : 'no'
  if (typeof v === 'object') return JSON.stringify(v)
  const s = String(v)
  // Dates and timestamps: just the date (and time if there is one).
  if (/^\d{4}-\d{2}-\d{2}T00:00:00(Z|[+-]00:00)?$/.test(s)) return s.slice(0, 10)
  if (/^[0-9a-f]{8}-[0-9a-f]{4}-/.test(s)) return s.slice(0, 8) + '…'
  return s.length > 60 ? s.slice(0, 60) + '…' : s
}

function formatWhen(iso) {
  return new Date(iso).toLocaleString('en-IN', {
    timeZone: 'Asia/Kolkata', day: '2-digit', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit',
  })
}

function Changes({ entry }) {
  const before = entry.old_data || {}
  const after = entry.new_data || {}
  const keys = Object.keys(entry.action === 'DELETE' ? before : after).filter(k => !HIDDEN_FIELDS.has(k))
  if (keys.length === 0) return <span className="data-table__muted">—</span>
  return (
    <ul className="activity-changes">
      {keys.map(k => (
        <li key={k}>
          <span className="activity-changes__field">{fieldLabel(k)}:</span>{' '}
          {entry.action === 'UPDATE'
            ? <><span className="activity-changes__old">{showValue(before[k])}</span> → <span className="activity-changes__new">{showValue(after[k])}</span></>
            : showValue(entry.action === 'DELETE' ? before[k] : after[k])}
        </li>
      ))}
    </ul>
  )
}

function ActivityLog() {
  const { currentSchool } = useSchool()
  const [from, setFrom] = useSessionState('activity.from', todayIST())
  const [to, setTo] = useSessionState('activity.to', todayIST())
  const [action, setAction] = useSessionState('activity.action', '')
  const [table, setTable] = useSessionState('activity.table', '')
  const [userId, setUserId] = useSessionState('activity.user', '')
  const [offset, setOffset] = useSessionState('activity.offset', 0)
  const [entries, setEntries] = useState([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [users, setUsers] = useState([])
  const limit = 50

  useEffect(() => {
    if (!currentSchool) return
    usersApi.list({ school_id: currentSchool.id, limit: 200 })
      .then(res => setUsers(res.items || []))
      .catch(() => setUsers([]))
  }, [currentSchool])

  useEffect(() => {
    if (!currentSchool) return
    let cancelled = false
    setLoading(true); setError('')
    auditApi.list({
      school_id: currentSchool.id, from: from || undefined, to: to || undefined,
      action: action || undefined, table: table || undefined, user_id: userId || undefined, limit, offset,
    })
      .then(res => { if (!cancelled) { setEntries(res.items || []); setTotal(res.total || 0) } })
      .catch(err => { if (!cancelled) { setEntries([]); setTotal(0); setError(err.message) } })
      .finally(() => { if (!cancelled) setLoading(false) })
    return () => { cancelled = true }
  }, [currentSchool, from, to, action, table, userId, offset])

  function setFilter(setter) {
    return e => { setter(e.target.value); setOffset(0) }
  }

  function clearFilters() {
    setFrom(todayIST()); setTo(todayIST()); setAction(''); setTable(''); setUserId(''); setOffset(0)
  }

  if (!currentSchool) return <p className="empty-text">Select a school first.</p>

  return (
    <div className="activity-page">
      <div className="page-header">
        <div>
          <h1>Activity Log</h1>
          <p className="page-subtitle">Every record added, changed or deleted in {currentSchool.name}, and who did it</p>
        </div>
      </div>

      <div className="page-filters">
        <label className="activity-date">
          <span>From</span>
          <input type="date" value={from} max={to || undefined} onChange={setFilter(setFrom)} />
        </label>
        <label className="activity-date">
          <span>To</span>
          <input type="date" value={to} min={from || undefined} onChange={setFilter(setTo)} />
        </label>
        <div className="filter-select">
          <select value={action} onChange={setFilter(setAction)} aria-label="Action">
            <option value="">All actions</option>
            {Object.entries(ACTIONS).map(([v, l]) => <option key={v} value={v}>{l}</option>)}
          </select>
        </div>
        <div className="filter-select">
          <select value={table} onChange={setFilter(setTable)} aria-label="Record type">
            <option value="">All records</option>
            {Object.entries(RECORD_TYPES).sort((a, b) => a[1].localeCompare(b[1])).map(([v, l]) => <option key={v} value={v}>{l}</option>)}
          </select>
        </div>
        <div className="filter-select">
          <select value={userId} onChange={setFilter(setUserId)} aria-label="Person">
            <option value="">Everyone</option>
            {users.map(u => <option key={u.id} value={u.id}>{u.first_name} {u.last_name} ({u.role})</option>)}
          </select>
        </div>
        <button className="filter-clear" onClick={clearFilters}><X size={14} /> Today, everyone</button>
      </div>

      <div className="page-count">{loading ? 'Loading...' : `${total} change${total === 1 ? '' : 's'}`}</div>
      {error && <p className="doc-msg doc-msg--error">{error}</p>}

      <div className="table-card">
        <table className="data-table activity-table">
          <thead>
            <tr><th>When (IST)</th><th>Who</th><th>Action</th><th>Record</th><th>Details</th><th>Device</th></tr>
          </thead>
          <tbody>
            {!loading && entries.length === 0 ? (
              <tr><td colSpan={6} className="data-table__empty">No changes for these filters</td></tr>
            ) : entries.map(e => (
              <tr key={e.id}>
                <td className="activity-when">{formatWhen(e.at)}</td>
                <td>
                  {e.user_name || e.user_email
                    ? <><div>{e.user_name || e.user_email}</div><div className="data-table__muted">{e.user_role}</div></>
                    : <span className="data-table__muted">System</span>}
                </td>
                <td><span className={`badge badge--${ACTION_BADGE[e.action]}`}>{ACTIONS[e.action]}</span></td>
                <td>{recordType(e.table)}</td>
                <td><Changes entry={e} /></td>
                <td className="data-table__muted activity-device" title={e.user_agent}>{e.ip}{e.request ? <div>{e.request.replace('/api/v1', '')}</div> : null}</td>
              </tr>
            ))}
          </tbody>
        </table>
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
    </div>
  )
}

export default ActivityLog
