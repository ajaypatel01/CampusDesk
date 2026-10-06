import { useState, useEffect, useMemo, useCallback, useRef } from 'react'
import { useOutletContext } from 'react-router-dom'
import { Search, Download, X, ChevronLeft, ChevronRight, RefreshCw } from 'lucide-react'
import { useSchool } from '../services/SchoolContext'
import { feesApi, academicApi } from '../services/api'
import './Ledger.css'
import { formatDate, todayIST, dateKey } from '../utils/date'

const modeColor = {
  cash: 'ledger-mode--cash',
  online: 'ledger-mode--online',
  upi: 'ledger-mode--online',
  cheque: 'ledger-mode--cheque',
}

function fmt(d) {
  if (!d) return '—'
  return formatDate(d)
}

function fmtAmt(n) {
  return '₹' + Number(n).toLocaleString('en-IN', { maximumFractionDigits: 0 })
}

function Ledger() {
  const { user } = useOutletContext() || {}
  const isRegistrar = user?.role === 'registrar'
  const { currentSchool, currentYear } = useSchool()
  const [years, setYears] = useState([])
  const [selectedYearId, setSelectedYearId] = useState('')

  // Date range filter (default: today)
  const today = todayIST()
  const [fromDate, setFromDate] = useState(today)
  const [toDate, setToDate] = useState(today)
  const [mode, setMode] = useState('day') // day | range | month | all

  const [payments, setPayments] = useState([])
  const [loading, setLoading] = useState(false) // first load for this school/year
  const [refreshing, setRefreshing] = useState(false)
  const [loadError, setLoadError] = useState('')
  const [updatedAt, setUpdatedAt] = useState(null)
  const [search, setSearch] = useState('')

  // Load academic years for this school's dropdown
  useEffect(() => {
    if (!currentSchool) return
    academicApi.listYears(currentSchool.id)
      .then(res => setYears(res.items || []))
      .catch(() => setYears([]))
  }, [currentSchool])

  // Keep the ledger's year selection in sync with the school/year picker in the header —
  // it previously always defaulted to whichever year was flagged current in the database,
  // ignoring what was actually selected up top.
  useEffect(() => {
    if (currentYear) setSelectedYearId(currentYear.id)
  }, [currentYear])

  // All of the school's payments for the year in one request. If it fails the
  // ledger says so -- it never shows a partial list as if it were complete.
  const requestId = useRef(0)
  const loadPayments = useCallback(async ({ background = false } = {}) => {
    if (!currentSchool || !selectedYearId) return
    const id = ++requestId.current
    if (background) setRefreshing(true)
    else { setLoading(true); setPayments([]) }
    try {
      const res = await feesApi.listLedgerPayments({ school_id: currentSchool.id, academic_year_id: selectedYearId })
      if (id !== requestId.current) return
      setPayments(res.items || [])
      setLoadError('')
      setUpdatedAt(new Date())
    } catch (err) {
      if (id !== requestId.current) return
      setLoadError(err.message || 'Could not load payments')
    } finally {
      if (id === requestId.current) { setLoading(false); setRefreshing(false) }
    }
  }, [currentSchool, selectedYearId])

  useEffect(() => { loadPayments() }, [loadPayments])

  // Coming back to this tab (e.g. after recording a fee elsewhere) refreshes it.
  useEffect(() => {
    function onVisible() {
      if (document.visibilityState === 'visible') loadPayments({ background: true })
    }
    document.addEventListener('visibilitychange', onVisible)
    return () => document.removeEventListener('visibilitychange', onVisible)
  }, [loadPayments])

  // Date range helpers
  function setModeDay() {
    setMode('day')
    setFromDate(today)
    setToDate(today)
  }
  function setModeMonth() {
    setMode('month')
    const d = new Date()
    setFromDate(`${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-01`)
    const last = new Date(d.getFullYear(), d.getMonth() + 1, 0)
    setToDate(`${last.getFullYear()}-${String(last.getMonth() + 1).padStart(2, '0')}-${String(last.getDate()).padStart(2, '0')}`)
  }
  function setModeAll() {
    setMode('all')
    setFromDate('')
    setToDate('')
  }

  const filtered = useMemo(() => {
    let result = payments.filter(p => !p.voided)

    // payment_date arrives as "2026-10-06T00:00:00Z"; compare just the date so
    // payments on the To date itself are included.
    if (fromDate) result = result.filter(p => dateKey(p.payment_date) >= fromDate)
    if (toDate) result = result.filter(p => dateKey(p.payment_date) <= toDate)

    if (search) {
      const q = search.toLowerCase()
      result = result.filter(p =>
        p.student_name?.toLowerCase().includes(q) ||
        p.student_code?.toLowerCase().includes(q) ||
        p.grade?.toLowerCase().includes(q) ||
        (p.reference_number || '').toLowerCase().includes(q)
      )
    }

    return result.sort((a, b) => a.payment_date > b.payment_date ? -1 : a.payment_date < b.payment_date ? 1 : 0)
  }, [payments, fromDate, toDate, search])

  const summary = useMemo(() => {
    const s = { total: 0, cash: 0, online: 0, other: 0, count: filtered.length }
    filtered.forEach(p => {
      s.total += p.amount
      const m = (p.payment_mode || '').toLowerCase()
      if (m === 'cash') s.cash += p.amount
      else if (m === 'online' || m === 'upi') s.online += p.amount
      else s.other += p.amount
    })
    return s
  }, [filtered])

  function exportCSV() {
    const headers = ['#', 'Date', 'Student Name', 'Code', 'Class', 'Fee Type', 'Amount', 'Mode', 'Reference', 'Notes']
    const rows = filtered.map((p, i) => [
      i + 1,
      formatDate(p.payment_date),
      `"${p.student_name}"`,
      p.student_code,
      `"${p.grade || ''}"`,
      p.fee_type || '',
      p.amount,
      p.payment_mode || '',
      p.reference_number || '',
      `"${p.notes || ''}"`,
    ].join(','))
    const csv = [headers.join(','), ...rows, `,,,,,,${summary.total},,Total,`].join('\n')
    const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' })
    const url = URL.createObjectURL(blob)
    const label = mode === 'day' ? fromDate : mode === 'month' ? fromDate.slice(0, 7) : 'all'
    const a = document.createElement('a'); a.href = url; a.download = `ledger-${label}.csv`; a.click()
    URL.revokeObjectURL(url)
  }

  if (!currentSchool) return <p className="empty-text">Select a school first.</p>

  return (
    <div className="ledger-page">
      <div className="page-header">
        <div>
          <h1>Daily Fee Ledger</h1>
          <p className="page-subtitle">Fee collection register — day-wise payment log</p>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: '8px', flexWrap: 'wrap' }}>
          {updatedAt && !loading && (
            <span className="data-table__muted" style={{ fontSize: '12px' }}>
              Updated {updatedAt.toLocaleTimeString('en-IN', { hour: '2-digit', minute: '2-digit' })}
            </span>
          )}
          <button className="btn btn--outline" onClick={() => loadPayments({ background: true })} disabled={loading || refreshing} title="Load the latest payments">
            <RefreshCw size={16} /> {refreshing ? 'Refreshing...' : 'Refresh'}
          </button>
          <button className="btn btn--outline" onClick={exportCSV} disabled={filtered.length === 0}>
            <Download size={16} /> Export CSV
          </button>
        </div>
      </div>

      {/* Year + Date controls */}
      <div className="ledger-controls">
        <select
          className="ledger-year-select"
          value={selectedYearId}
          onChange={e => setSelectedYearId(e.target.value)}
        >
          {years.map(y => <option key={y.id} value={y.id}>{y.name}{y.is_current ? ' (Current)' : ''}</option>)}
        </select>

        <div className="ledger-mode-tabs">
          <button className={`ledger-tab ${mode === 'day' ? 'ledger-tab--active' : ''}`} onClick={setModeDay}>Today</button>
          <button className={`ledger-tab ${mode === 'month' ? 'ledger-tab--active' : ''}`} onClick={setModeMonth}>This Month</button>
          <button className={`ledger-tab ${mode === 'range' ? 'ledger-tab--active' : ''}`} onClick={() => setMode('range')}>Custom Range</button>
          <button className={`ledger-tab ${mode === 'all' ? 'ledger-tab--active' : ''}`} onClick={setModeAll}>All</button>
        </div>

        {(mode === 'day' || mode === 'range') && (
          <div className="ledger-date-inputs">
            <input
              type="date"
              value={fromDate}
              onChange={e => { setFromDate(e.target.value); if (mode === 'day') setToDate(e.target.value) }}
            />
            {mode === 'range' && (
              <>
                <span className="ledger-date-sep">to</span>
                <input type="date" value={toDate} onChange={e => setToDate(e.target.value)} />
              </>
            )}
          </div>
        )}
      </div>

      {/* Summary bar — registrars only see per-payment rows below, not the summed totals */}
      {!isRegistrar && (
        <div className="ledger-summary">
          <div className="lsumm"><span className="lsumm__val">{summary.count}</span><span className="lsumm__lbl">Payments</span></div>
          <div className="lsumm lsumm--green"><span className="lsumm__val">{fmtAmt(summary.total)}</span><span className="lsumm__lbl">Total Collected</span></div>
          <div className="lsumm"><span className="lsumm__val">{fmtAmt(summary.cash)}</span><span className="lsumm__lbl">Cash</span></div>
          <div className="lsumm"><span className="lsumm__val">{fmtAmt(summary.online)}</span><span className="lsumm__lbl">Online / UPI</span></div>
          {summary.other > 0 && <div className="lsumm"><span className="lsumm__val">{fmtAmt(summary.other)}</span><span className="lsumm__lbl">Other</span></div>}
        </div>
      )}

      <div className="page-filters">
        <div className="filter-search">
          <Search size={18} />
          <input
            placeholder="Search name, code, class..."
            value={search}
            onChange={e => setSearch(e.target.value)}
          />
        </div>
        {search && <button className="filter-clear" onClick={() => setSearch('')}><X size={14} /> Clear</button>}
      </div>

      <div className="page-count">
        {loading ? 'Loading payments...' : `${filtered.length} payment${filtered.length !== 1 ? 's' : ''}`}
      </div>

      {loadError && (
        <div className="doc-msg doc-msg--error" style={{ display: 'flex', alignItems: 'center', gap: '10px', flexWrap: 'wrap' }}>
          <span>Couldn&apos;t load payments: {loadError}.{payments.length > 0 ? ' Showing the last loaded list.' : ''}</span>
          <button className="btn btn--outline btn--sm" onClick={() => loadPayments({ background: payments.length > 0 })}>Try again</button>
        </div>
      )}

      {loading ? (
        <p className="loading-text">Loading fee payments...</p>
      ) : loadError && payments.length === 0 ? null : filtered.length === 0 ? (
        <p className="empty-text">No payments found for this period</p>
      ) : (
        <div className="table-card">
          <table className="data-table">
            <thead>
              <tr>
                <th>#</th>
                <th>Date</th>
                <th>Student</th>
                <th>Class</th>
                <th>Fee Type</th>
                <th>Mode</th>
                <th>Reference</th>
                <th style={{textAlign:'right'}}>Amount</th>
              </tr>
            </thead>
            <tbody>
              {filtered.map((p, i) => (
                <tr key={p.id}>
                  <td className="data-table__muted">{i + 1}</td>
                  <td className="data-table__muted ledger-date">{fmt(p.payment_date)}</td>
                  <td>
                    <div className="ledger-student">{p.student_name}</div>
                    <div className="data-table__muted ledger-code">{p.student_code}</div>
                  </td>
                  <td className="data-table__muted">{p.grade || '—'}</td>
                  <td className="data-table__muted">{p.fee_type || '—'}</td>
                  <td>
                    <span className={`ledger-mode ${modeColor[(p.payment_mode || '').toLowerCase()] || ''}`}>
                      {p.payment_mode || '—'}
                    </span>
                  </td>
                  <td className="data-table__muted">{p.reference_number || '—'}</td>
                  <td style={{textAlign:'right'}} className="ledger-amount">{fmtAmt(p.amount)}</td>
                </tr>
              ))}
            </tbody>
            <tfoot>
              <tr className="ledger-total-row">
                <td colSpan={7} style={{textAlign:'right', fontWeight:600}}>Total</td>
                <td style={{textAlign:'right'}} className="ledger-amount ledger-amount--total">{fmtAmt(summary.total)}</td>
              </tr>
            </tfoot>
          </table>
        </div>
      )}
    </div>
  )
}

export default Ledger
