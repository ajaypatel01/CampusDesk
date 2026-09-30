import { useState, useEffect } from 'react'
import { useOutletContext } from 'react-router-dom'
import { IndianRupee, Download } from 'lucide-react'
import { useSchool } from '../services/SchoolContext'
import { payrollApi } from '../services/api'
import './Payroll.css'

const MONTHS = [
  'January', 'February', 'March', 'April', 'May', 'June',
  'July', 'August', 'September', 'October', 'November', 'December',
]

function fmt(amt) {
  return new Intl.NumberFormat('en-IN', { style: 'currency', currency: 'INR', maximumFractionDigits: 0 }).format(amt || 0)
}

function downloadBlob(blob, filename) {
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url; a.download = filename; a.click()
  URL.revokeObjectURL(url)
}

// A teacher's own view of the payroll module -- same /payroll and
// /payroll/slip endpoints Payroll.jsx uses, but always scoped to their own
// user_id. The backend independently enforces that a non-admin caller can
// only ever pass their own id here (see payroll.canAccessUser), so this
// page can't be used to see anyone else's salary even if the request were
// tampered with client-side.
export default function MySalary() {
  const { user } = useOutletContext() || {}
  const { currentSchool, currentYear } = useSchool()
  const now = new Date()
  const [month, setMonth] = useState(now.getMonth() + 1)
  const [year, setYear] = useState(now.getFullYear())
  const [row, setRow] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [downloading, setDownloading] = useState(false)

  useEffect(() => {
    if (!currentSchool || !currentYear || !user?.id) return
    setLoading(true); setError('')
    payrollApi.computeMonth({ school_id: currentSchool.id, academic_year_id: currentYear.id, year, month, user_id: user.id })
      .then(r => setRow((r.items || [])[0] || null))
      .catch(err => setError(err.message))
      .finally(() => setLoading(false))
  }, [currentSchool, currentYear, year, month, user?.id])

  async function handleDownloadSlip() {
    if (!currentSchool || !currentYear || !user?.id) return
    setDownloading(true)
    try {
      const blob = await payrollApi.downloadSlip({ school_id: currentSchool.id, academic_year_id: currentYear.id, year, month, user_id: user.id })
      downloadBlob(blob, `salary_slip_${year}_${month}.pdf`)
    } catch (err) { alert(err.message) } finally { setDownloading(false) }
  }

  if (!currentSchool || !currentYear) {
    return <p className="empty-text">Select a school and academic year first.</p>
  }

  return (
    <div className="payroll-page">
      <div className="page-header">
        <div>
          <h1>My Salary</h1>
          <p className="page-subtitle">Your own salary and deductions, computed for the month below.</p>
        </div>
        <button className="btn btn--primary" onClick={handleDownloadSlip} disabled={downloading || !row}>
          <Download size={16} /> {downloading ? 'Downloading...' : 'Download Slip'}
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

      {error && <p className="doc-msg doc-msg--error">{error}</p>}

      {loading ? (
        <p className="loading-text">Loading...</p>
      ) : !row ? (
        <p className="empty-text">No salary on file for this month yet.</p>
      ) : (
        <>
          <div className="payroll-summary">
            <div className="payroll-stat"><IndianRupee size={16} /> <span>{fmt(row.monthly_salary)}</span><label>Gross Salary</label></div>
            <div className="payroll-stat payroll-stat--danger"><span>{fmt(row.deduction)}</span><label>Total Deductions</label></div>
            <div className="payroll-stat payroll-stat--green"><span>{fmt(row.net_salary)}</span><label>Net Payable</label></div>
          </div>

          <div className="table-card">
            <table className="data-table">
              <tbody>
                <tr><td>Basic Salary</td><td>{fmt(row.basic_salary)}</td></tr>
                <tr><td>HRA</td><td>{fmt(row.hra)}</td></tr>
                <tr><td>Special Allowance</td><td>{fmt(row.special_allowance)}</td></tr>
                {row.bonus > 0 && <tr><td>Bonus</td><td>{fmt(row.bonus)}</td></tr>}
                <tr><td className="data-table__muted">CL Paid Days</td><td className="data-table__muted">{row.cl_paid_days}</td></tr>
                {row.cl_excess_days > 0 && <tr><td>CL Excess Days</td><td className="payroll-negative">{row.cl_excess_days}</td></tr>}
                {row.unpaid_days > 0 && <tr><td>Unpaid Days</td><td className="payroll-negative">{row.unpaid_days}</td></tr>}
                {row.attendance_deduction > 0 && <tr><td>Attendance Deduction</td><td className="payroll-negative">- {fmt(row.attendance_deduction)}</td></tr>}
                {row.epf > 0 && <tr><td>EPF</td><td className="payroll-negative">- {fmt(row.epf)}</td></tr>}
                {row.esic > 0 && <tr><td>ESIC</td><td className="payroll-negative">- {fmt(row.esic)}</td></tr>}
                {row.tax > 0 && <tr><td>Tax</td><td className="payroll-negative">- {fmt(row.tax)}</td></tr>}
                {row.additional_deduction > 0 && <tr><td>{row.additional_deduction_label || 'Other Deduction'}</td><td className="payroll-negative">- {fmt(row.additional_deduction)}</td></tr>}
                <tr><td className="data-table__muted">CL Balance</td><td className="data-table__muted">{row.cl_balance} / {row.cl_quota_per_year}</td></tr>
              </tbody>
            </table>
          </div>
        </>
      )}
    </div>
  )
}
