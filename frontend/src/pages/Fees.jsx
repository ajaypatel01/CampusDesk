import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { Search, ChevronLeft, ChevronRight, Filter, X, Download } from 'lucide-react'
import { useSchool } from '../services/SchoolContext'
import { feesApi, academicApi } from '../services/api'
import './Fees.css'

function Fees() {
  const { currentSchool, currentYear } = useSchool()
  const [accounts, setAccounts] = useState([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [summary, setSummary] = useState(null)
  const [grades, setGrades] = useState([])

  const [search, setSearch] = useState('')
  const [gradeFilter, setGradeFilter] = useState('')
  const [paymentStatus, setPaymentStatus] = useState('')
  const [offset, setOffset] = useState(0)
  const limit = 20

  useEffect(() => {
    if (!currentSchool) return
    academicApi.listGrades(currentSchool.id)
      .then(res => setGrades(res.items || []))
      .catch(() => setGrades([]))
  }, [currentSchool])

  useEffect(() => {
    if (!currentSchool || !currentYear) { setLoading(false); return }
    setLoading(true)
    Promise.all([
      feesApi.listAccounts({
        school_id: currentSchool.id,
        academic_year_id: currentYear.id,
        search: search || undefined,
        grade_level: gradeFilter || undefined,
        payment_status: paymentStatus || undefined,
        limit,
        offset,
      }),
      feesApi.schoolSummary({ school_id: currentSchool.id, academic_year_id: currentYear.id }),
    ])
      .then(([accRes, sumRes]) => {
        setAccounts(accRes.items || [])
        setTotal(accRes.total || 0)
        setSummary(sumRes)
      })
      .catch(() => { setAccounts([]); setTotal(0) })
      .finally(() => setLoading(false))
  }, [currentSchool, currentYear, search, gradeFilter, paymentStatus, offset])

  const activeFilterCount = [gradeFilter, paymentStatus].filter(Boolean).length

  function clearFilters() {
    setGradeFilter('')
    setPaymentStatus('')
    setSearch('')
    setOffset(0)
  }

  const [exporting, setExporting] = useState(false)

  function fmt(amt) {
    return new Intl.NumberFormat('en-IN', { style: 'currency', currency: 'INR', maximumFractionDigits: 0 }).format(amt)
  }

  async function exportCSV() {
    setExporting(true)
    try {
      const res = await feesApi.listAccounts({
        school_id: currentSchool.id,
        academic_year_id: currentYear.id,
        limit: 5000,
        offset: 0,
      })
      const rows = res.items || []
      const headers = [
        'Student Name', 'Student Code', 'Grade',
        'Tuition Fee', 'Discount', 'Van Fee', 'Previous Dues',
        'Total Due', 'Total Paid', 'Balance', 'RTE', 'Status',
      ]
      const csvRows = rows.map(a => {
        const status = a.balance_remaining <= 0 ? 'Paid' : a.total_paid > 0 ? 'Partial' : 'Unpaid'
        return [
          `"${a.student_name}"`,
          a.student_code,
          `"${a.grade_level_name || ''}"`,
          a.tuition_fee,
          a.discount_amount,
          a.van_fee,
          a.previous_year_dues,
          a.total_due,
          a.total_paid,
          a.balance_remaining,
          a.is_rte ? 'Yes' : 'No',
          status,
        ].join(',')
      })
      const csv = [headers.join(','), ...csvRows].join('\n')
      const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `fee-sheet-${currentYear.name.replace(/\s+/g, '-')}.csv`
      a.click()
      URL.revokeObjectURL(url)
    } catch (err) {
      alert('Export failed: ' + err.message)
    } finally {
      setExporting(false)
    }
  }

  if (!currentSchool || !currentYear) {
    return <p className="empty-text">Select a school and academic year first.</p>
  }

  return (
    <div className="fees-page">
      <div className="page-header">
        <div>
          <h1>Fee Management</h1>
          <p className="page-subtitle">{currentYear.name} - Fee accounts and payments</p>
        </div>
        <button className="btn btn--outline" onClick={exportCSV} disabled={exporting}>
          <Download size={16} /> {exporting ? 'Exporting...' : 'Export CSV'}
        </button>
      </div>

      {summary && (
        <div className="fees-page__summary">
          <div className="fee-summary-card">
            <span className="fee-summary-card__label">Total Students</span>
            <span className="fee-summary-card__value">{summary.total_students}</span>
          </div>
          <div className="fee-summary-card">
            <span className="fee-summary-card__label">RTE Students</span>
            <span className="fee-summary-card__value">{summary.rte_students}</span>
          </div>
          <div className="fee-summary-card fee-summary-card--green">
            <span className="fee-summary-card__label">Total Collected</span>
            <span className="fee-summary-card__value">{fmt(summary.total_collected)}</span>
          </div>
          <div className="fee-summary-card fee-summary-card--red">
            <span className="fee-summary-card__label">Outstanding</span>
            <span className="fee-summary-card__value">{fmt(summary.total_outstanding)}</span>
          </div>
          <div className="fee-summary-card">
            <span className="fee-summary-card__label">Total Discount</span>
            <span className="fee-summary-card__value">{fmt(summary.total_discount)}</span>
          </div>
        </div>
      )}

      <div className="page-filters">
        <div className="filter-search">
          <Search size={18} />
          <input
            type="text" placeholder="Search by student name or code..."
            value={search} onChange={e => { setSearch(e.target.value); setOffset(0) }}
          />
        </div>
        <div className="filter-select">
          <Filter size={16} />
          <select value={paymentStatus} onChange={e => { setPaymentStatus(e.target.value); setOffset(0) }}>
            <option value="">All Payment Status</option>
            <option value="paid">Fully Paid</option>
            <option value="due">Balance Due</option>
            <option value="partial">Partial Paid</option>
          </select>
        </div>
        {grades.length > 0 && (
          <div className="filter-select">
            <select value={gradeFilter} onChange={e => { setGradeFilter(e.target.value); setOffset(0) }}>
              <option value="">All Grades</option>
              {grades.map(g => <option key={g.id} value={g.name}>{g.name}</option>)}
            </select>
          </div>
        )}
        {activeFilterCount > 0 && (
          <button className="filter-clear" onClick={clearFilters}>
            <X size={14} /> Clear ({activeFilterCount})
          </button>
        )}
      </div>

      <div className="page-count">Showing {accounts.length} of {total} fee accounts</div>

      <div className="table-card">
        {loading ? <p className="loading-text">Loading...</p> : (
          <table className="data-table">
            <thead>
              <tr>
                <th>Student</th>
                <th>Code</th>
                <th>Grade</th>
                <th>Tuition</th>
                <th>Discount</th>
                <th>Van Fee</th>
                <th>Prev Dues</th>
                <th>Total Due</th>
                <th>Paid</th>
                <th>Balance</th>
                <th>RTE</th>
              </tr>
            </thead>
            <tbody>
              {accounts.length === 0 ? (
                <tr><td colSpan={11} className="data-table__empty">No fee accounts found</td></tr>
              ) : accounts.map(a => (
                <tr key={a.id}>
                  <td>
                    <Link to={`/fees/${a.id}`} className="data-table__link">{a.student_name}</Link>
                  </td>
                  <td className="data-table__muted">{a.student_code}</td>
                  <td className="data-table__muted">{a.grade_level_name || '-'}</td>
                  <td>{fmt(a.tuition_fee)}</td>
                  <td className="data-table__muted">{a.discount_amount ? fmt(a.discount_amount) : '-'}</td>
                  <td>{a.van_fee ? fmt(a.van_fee) : '-'}</td>
                  <td>{a.previous_year_dues ? fmt(a.previous_year_dues) : '-'}</td>
                  <td className="fees-page__amount">{fmt(a.total_due)}</td>
                  <td className="fees-page__paid">{fmt(a.total_paid)}</td>
                  <td className={`fees-page__balance ${a.balance_remaining > 0 ? 'fees-page__balance--due' : ''}`}>
                    {fmt(a.balance_remaining)}
                  </td>
                  <td>{a.is_rte ? <span className="badge badge--info">RTE</span> : '-'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
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

export default Fees
