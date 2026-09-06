import { useState, useEffect, useMemo } from 'react'
import { Plus, Trash2, IndianRupee } from 'lucide-react'
import { useSchool } from '../services/SchoolContext'
import { payrollApi, staffApi } from '../services/api'
import './Payroll.css'

const MONTHS = [
  'January', 'February', 'March', 'April', 'May', 'June',
  'July', 'August', 'September', 'October', 'November', 'December',
]

function fmt(amt) {
  return new Intl.NumberFormat('en-IN', { style: 'currency', currency: 'INR', maximumFractionDigits: 0 }).format(amt || 0)
}

function Payroll() {
  const { currentSchool, currentYear } = useSchool()
  const now = new Date()
  const [month, setMonth] = useState(now.getMonth() + 1)
  const [year, setYear] = useState(now.getFullYear())

  const [staff, setStaff] = useState([])
  const [rows, setRows] = useState([])
  const [loading, setLoading] = useState(true)
  const [leaves, setLeaves] = useState([])

  const [showForm, setShowForm] = useState(false)
  const [form, setForm] = useState({ user_id: '', leave_type: 'cl', start_date: '', end_date: '', reason: '' })
  const [saving, setSaving] = useState(false)
  const [err, setErr] = useState('')

  useEffect(() => {
    if (!currentSchool) return
    staffApi.list({ school_id: currentSchool.id, limit: 500 }).then(r => setStaff(r.items || [])).catch(() => setStaff([]))
  }, [currentSchool])

  function loadPayroll() {
    if (!currentSchool || !currentYear) return
    setLoading(true)
    payrollApi.computeMonth({ school_id: currentSchool.id, academic_year_id: currentYear.id, year, month })
      .then(r => setRows(r.items || []))
      .catch(() => setRows([]))
      .finally(() => setLoading(false))
  }

  function loadLeaves() {
    if (!currentSchool || !currentYear) return
    payrollApi.listLeaves({ school_id: currentSchool.id, academic_year_id: currentYear.id })
      .then(r => setLeaves(r.items || []))
      .catch(() => setLeaves([]))
  }

  useEffect(loadPayroll, [currentSchool, currentYear, year, month])
  useEffect(loadLeaves, [currentSchool, currentYear])

  const staffByID = useMemo(() => Object.fromEntries(staff.map(s => [s.id, s])), [staff])

  const totals = useMemo(() => rows.reduce((acc, r) => ({
    salary: acc.salary + r.monthly_salary,
    deduction: acc.deduction + r.deduction,
    net: acc.net + r.net_salary,
  }), { salary: 0, deduction: 0, net: 0 }), [rows])

  async function handleAddLeave(e) {
    e.preventDefault()
    setSaving(true); setErr('')
    try {
      await payrollApi.createLeave({ ...form, academic_year_id: currentYear.id })
      setShowForm(false)
      setForm({ user_id: '', leave_type: 'cl', start_date: '', end_date: '', reason: '' })
      loadLeaves()
      loadPayroll()
    } catch (e2) {
      setErr(e2.message || 'Failed to save leave')
    }
    setSaving(false)
  }

  async function handleDeleteLeave(id) {
    if (!confirm('Delete this leave record?')) return
    try {
      await payrollApi.deleteLeave(id)
      loadLeaves()
      loadPayroll()
    } catch (e) { alert(e.message) }
  }

  if (!currentSchool || !currentYear) {
    return <p className="empty-text">Select a school and academic year first.</p>
  }

  return (
    <div className="payroll-page">
      <div className="page-header">
        <div>
          <h1>Payroll & Leave</h1>
          <p className="page-subtitle">Salary auto-calculated from leave taken against each teacher's CL quota</p>
        </div>
        <button className="btn btn--primary" onClick={() => setShowForm(true)}>
          <Plus size={16} /> Log Leave
        </button>
      </div>

      <div className="payroll-controls">
        <label className="form-field">
          <span>Month</span>
          <select value={month} onChange={e => setMonth(parseInt(e.target.value, 10))}>
            {MONTHS.map((m, i) => <option key={m} value={i + 1}>{m}</option>)}
          </select>
        </label>
        <label className="form-field">
          <span>Year</span>
          <input type="number" value={year} onChange={e => setYear(parseInt(e.target.value, 10) || now.getFullYear())} />
        </label>
      </div>

      <div className="payroll-summary">
        <div className="payroll-stat"><IndianRupee size={16} /> <span>{fmt(totals.salary)}</span><label>Total Salary</label></div>
        <div className="payroll-stat payroll-stat--danger"><span>{fmt(totals.deduction)}</span><label>Total Deductions</label></div>
        <div className="payroll-stat payroll-stat--green"><span>{fmt(totals.net)}</span><label>Net Payable</label></div>
      </div>

      <div className="table-card">
        <table className="data-table">
          <thead>
            <tr>
              <th>Staff</th><th>Designation</th><th>Salary</th><th>CL Paid</th><th>CL Excess</th>
              <th>Unpaid</th><th>Deduction</th><th>Net Salary</th><th>CL Balance</th>
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <tr><td colSpan={9} className="data-table__empty">Loading...</td></tr>
            ) : rows.length === 0 ? (
              <tr><td colSpan={9} className="data-table__empty">No staff with salary on file for this school</td></tr>
            ) : rows.map(r => (
              <tr key={r.user_id}>
                <td>{r.first_name} {r.last_name}</td>
                <td className="data-table__muted">{r.designation || '-'}</td>
                <td>{fmt(r.monthly_salary)}</td>
                <td>{r.cl_paid_days}</td>
                <td className={r.cl_excess_days > 0 ? 'payroll-negative' : ''}>{r.cl_excess_days}</td>
                <td className={r.unpaid_days > 0 ? 'payroll-negative' : ''}>{r.unpaid_days}</td>
                <td className="payroll-negative">{r.deduction > 0 ? `- ${fmt(r.deduction)}` : fmt(0)}</td>
                <td className="payroll-net">{fmt(r.net_salary)}</td>
                <td>{r.cl_balance} / {r.cl_quota_per_year}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <h2 className="payroll-section-title">Leave Records — {currentYear.name}</h2>
      <div className="table-card">
        <table className="data-table">
          <thead>
            <tr><th>Staff</th><th>Type</th><th>From</th><th>To</th><th>Reason</th><th></th></tr>
          </thead>
          <tbody>
            {leaves.length === 0 ? (
              <tr><td colSpan={6} className="data-table__empty">No leave records yet</td></tr>
            ) : leaves.map(l => {
              const s = staffByID[l.user_id]
              return (
                <tr key={l.id}>
                  <td>{s ? `${s.first_name} ${s.last_name}` : l.user_id}</td>
                  <td><span className={`badge badge--${l.leave_type === 'cl' ? 'info' : 'danger'}`}>{l.leave_type === 'cl' ? 'CL' : 'Unpaid'}</span></td>
                  <td className="data-table__muted">{l.start_date}</td>
                  <td className="data-table__muted">{l.end_date}</td>
                  <td className="data-table__muted">{l.reason || '-'}</td>
                  <td><button className="btn btn--outline btn--sm" onClick={() => handleDeleteLeave(l.id)}><Trash2 size={14} /></button></td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>

      {showForm && (
        <div className="modal-overlay" onClick={() => setShowForm(false)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <h2>Log Leave</h2>
            <form className="modal__form" onSubmit={handleAddLeave}>
              <label className="form-field">
                <span>Staff Member *</span>
                <select required value={form.user_id} onChange={e => setForm({ ...form, user_id: e.target.value })}>
                  <option value="">Select...</option>
                  {staff.map(s => <option key={s.id} value={s.id}>{s.first_name} {s.last_name}</option>)}
                </select>
              </label>
              <div className="form-row">
                <label className="form-field">
                  <span>Leave Type *</span>
                  <select value={form.leave_type} onChange={e => setForm({ ...form, leave_type: e.target.value })}>
                    <option value="cl">Casual Leave (CL)</option>
                    <option value="unpaid">Unpaid</option>
                  </select>
                </label>
              </div>
              <div className="form-row">
                <label className="form-field">
                  <span>From *</span>
                  <input required type="date" value={form.start_date} onChange={e => setForm({ ...form, start_date: e.target.value })} />
                </label>
                <label className="form-field">
                  <span>To *</span>
                  <input required type="date" value={form.end_date} onChange={e => setForm({ ...form, end_date: e.target.value })} />
                </label>
              </div>
              <label className="form-field">
                <span>Reason</span>
                <input value={form.reason} onChange={e => setForm({ ...form, reason: e.target.value })} placeholder="Optional" />
              </label>
              {err && <p className="doc-msg doc-msg--error">{err}</p>}
              <div className="modal__actions">
                <button type="button" className="btn btn--outline" onClick={() => setShowForm(false)}>Cancel</button>
                <button type="submit" className="btn btn--primary" disabled={saving}>{saving ? 'Saving...' : 'Save'}</button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}

export default Payroll
