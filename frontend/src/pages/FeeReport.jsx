import { useState, useEffect, useMemo } from 'react'
import { useSchool } from '../services/SchoolContext'
import { feesApi, academicApi } from '../services/api'
import './FeeReport.css'

function fmt(amt) {
  return new Intl.NumberFormat('en-IN', { style: 'currency', currency: 'INR', maximumFractionDigits: 0 }).format(amt || 0)
}

function FeeReport() {
  const { currentSchool, currentYear, academicYears } = useSchool()
  const [grades, setGrades] = useState([])
  const [gradeFilter, setGradeFilter] = useState('')

  const [sheet, setSheet] = useState(null)
  const [loadingSheet, setLoadingSheet] = useState(true)

  const [currentSummary, setCurrentSummary] = useState(null)
  const [previousSummary, setPreviousSummary] = useState(null)
  const [previousYear, setPreviousYear] = useState(null)
  const [loadingCompare, setLoadingCompare] = useState(true)

  useEffect(() => {
    if (!currentSchool) return
    academicApi.listGrades(currentSchool.id).then(r => setGrades(r.items || [])).catch(() => setGrades([]))
  }, [currentSchool])

  // Classwise pending: current year vs the academic year immediately before it
  useEffect(() => {
    if (!currentSchool || !currentYear || academicYears.length === 0) return
    setLoadingCompare(true)
    const sorted = [...academicYears].sort((a, b) => new Date(b.start_date) - new Date(a.start_date))
    const idx = sorted.findIndex(y => y.id === currentYear.id)
    const prev = idx >= 0 ? sorted[idx + 1] : null
    setPreviousYear(prev || null)

    const requests = [feesApi.schoolSummary({ school_id: currentSchool.id, academic_year_id: currentYear.id })]
    if (prev) requests.push(feesApi.schoolSummary({ school_id: currentSchool.id, academic_year_id: prev.id }))

    Promise.all(requests)
      .then(([cur, prevRes]) => {
        setCurrentSummary(cur)
        setPreviousSummary(prevRes || null)
      })
      .catch(() => { setCurrentSummary(null); setPreviousSummary(null) })
      .finally(() => setLoadingCompare(false))
  }, [currentSchool, currentYear, academicYears])

  // Installment sheet for the current year
  useEffect(() => {
    if (!currentSchool || !currentYear) return
    setLoadingSheet(true)
    feesApi.installmentSheet({
      school_id: currentSchool.id,
      academic_year_id: currentYear.id,
      grade_level_id: gradeFilter || undefined,
    })
      .then(setSheet)
      .catch(() => setSheet(null))
      .finally(() => setLoadingSheet(false))
  }, [currentSchool, currentYear, gradeFilter])

  // Quarterly fee schedule: each installment's planned amount is bucketed by the
  // quarter its due date falls in (Apr-Jun, Jul-Sep, Oct-Dec, Jan-Mar), split into
  // what's already been collected against it vs. still pending.
  const QUARTER_LABELS = ['Q1 (Apr-Jun)', 'Q2 (Jul-Sep)', 'Q3 (Oct-Dec)', 'Q4 (Jan-Mar)']
  function quarterOf(dueDate) {
    const month = new Date(dueDate).getMonth() // 0=Jan
    if (month >= 3 && month <= 5) return 0
    if (month >= 6 && month <= 8) return 1
    if (month >= 9 && month <= 11) return 2
    return 3
  }
  const quarterData = useMemo(() => {
    const buckets = QUARTER_LABELS.map(label => ({ label, collected: 0, pending: 0 }))
    let unscheduled = 0
    for (const row of sheet?.items || []) {
      const datedCells = row.installments.filter(c => c.due_date && c.planned_amount > 0)
      if (datedCells.length === 0) {
        // No installment plan/due-dates configured for this student's fee structure —
        // there's nothing to bucket by quarter, so fall back to their real balance
        // rather than silently reporting zero.
        unscheduled += Math.max(row.balance, 0)
        continue
      }
      for (const cell of datedCells) {
        const q = quarterOf(cell.due_date)
        buckets[q].collected += Math.min(cell.paid_amount, cell.planned_amount)
        buckets[q].pending += Math.max(cell.planned_amount - cell.paid_amount, 0)
      }
    }
    return { buckets, unscheduled }
  }, [sheet])
  const quarterMax = Math.max(1, quarterData.unscheduled, ...quarterData.buckets.map(b => b.collected + b.pending))

  const gradeComparison = useMemo(() => {
    if (!currentSummary) return []
    const prevByGrade = Object.fromEntries((previousSummary?.by_grade || []).map(g => [g.grade_level_name, g]))
    return (currentSummary.by_grade || []).map(g => ({
      name: g.grade_level_name,
      studentCount: g.student_count,
      currentDue: g.total_due,
      currentPaid: g.total_collected,
      currentPending: g.outstanding,
      previousPending: prevByGrade[g.grade_level_name]?.outstanding ?? null,
    }))
  }, [currentSummary, previousSummary])

  if (!currentSchool || !currentYear) {
    return <p className="empty-text">Select a school and academic year first.</p>
  }

  return (
    <div className="fee-report-page">
      <div className="page-header">
        <div>
          <h1>Fee Report</h1>
          <p className="page-subtitle">{currentYear.name} — pending fees by class and installment status per student</p>
        </div>
      </div>

      {/* Classwise pending: current vs last year */}
      <h2 className="fee-report-section-title">Pending Fees by Class — {currentYear.name} vs {previousYear ? previousYear.name : 'previous year'}</h2>
      <div className="table-card">
        <table className="data-table">
          <thead>
            <tr>
              <th>Class</th><th>Students</th><th>Total Due ({currentYear.name})</th>
              <th>Collected ({currentYear.name})</th><th>Pending ({currentYear.name})</th>
              <th>Pending ({previousYear ? previousYear.name : '—'})</th>
            </tr>
          </thead>
          <tbody>
            {loadingCompare ? (
              <tr><td colSpan={6} className="data-table__empty">Loading...</td></tr>
            ) : gradeComparison.length === 0 ? (
              <tr><td colSpan={6} className="data-table__empty">No fee data for this year</td></tr>
            ) : gradeComparison.map(g => (
              <tr key={g.name}>
                <td>{g.name}</td>
                <td className="data-table__muted">{g.studentCount}</td>
                <td>{fmt(g.currentDue)}</td>
                <td>{fmt(g.currentPaid)}</td>
                <td className="fee-report-negative">{fmt(g.currentPending)}</td>
                <td className={g.previousPending > 0 ? 'fee-report-negative' : 'data-table__muted'}>
                  {g.previousPending == null ? '—' : fmt(g.previousPending)}
                </td>
              </tr>
            ))}
          </tbody>
          {gradeComparison.length > 0 && (
            <tfoot>
              <tr className="fee-report-total-row">
                <td>Total</td>
                <td>{gradeComparison.reduce((s, g) => s + g.studentCount, 0)}</td>
                <td>{fmt(gradeComparison.reduce((s, g) => s + g.currentDue, 0))}</td>
                <td>{fmt(gradeComparison.reduce((s, g) => s + g.currentPaid, 0))}</td>
                <td className="fee-report-negative">{fmt(gradeComparison.reduce((s, g) => s + g.currentPending, 0))}</td>
                <td className="fee-report-negative">{fmt(gradeComparison.reduce((s, g) => s + (g.previousPending || 0), 0))}</td>
              </tr>
            </tfoot>
          )}
        </table>
      </div>

      {/* Quarterly chart */}
      <h2 className="fee-report-section-title">Fee to Be Collected by Quarter — {currentYear.name}</h2>
      {loadingSheet ? (
        <p className="empty-text">Loading...</p>
      ) : (
        <div className="fee-quarter-chart">
          {quarterData.buckets.map(b => {
            const total = b.collected + b.pending
            return (
              <div key={b.label} className="fee-quarter-bar-col">
                <div className="fee-quarter-bar-track">
                  <div className="fee-quarter-bar fee-quarter-bar--collected" style={{ height: `${(b.collected / quarterMax) * 100}%` }} title={`Collected: ${fmt(b.collected)}`} />
                  <div className="fee-quarter-bar fee-quarter-bar--pending" style={{ height: `${(b.pending / quarterMax) * 100}%` }} title={`Pending: ${fmt(b.pending)}`} />
                </div>
                <div className="fee-quarter-total">{fmt(total)}</div>
                <div className="fee-quarter-label">{b.label}</div>
              </div>
            )
          })}
          {quarterData.unscheduled > 0 && (
            <div className="fee-quarter-bar-col">
              <div className="fee-quarter-bar-track">
                <div className="fee-quarter-bar fee-quarter-bar--unscheduled" style={{ height: `${(quarterData.unscheduled / quarterMax) * 100}%` }} title={`No due date on file: ${fmt(quarterData.unscheduled)}`} />
              </div>
              <div className="fee-quarter-total">{fmt(quarterData.unscheduled)}</div>
              <div className="fee-quarter-label">Not Scheduled</div>
            </div>
          )}
          <div className="fee-quarter-legend">
            <span><i className="fee-quarter-dot fee-quarter-dot--collected" /> Collected</span>
            <span><i className="fee-quarter-dot fee-quarter-dot--pending" /> Pending</span>
            {quarterData.unscheduled > 0 && (
              <span><i className="fee-quarter-dot fee-quarter-dot--unscheduled" /> No installment due-dates set for {currentYear.name} — set them up under fee structures to see this split by quarter</span>
            )}
          </div>
        </div>
      )}

      {/* Installment sheet */}
      <div className="fee-report-sheet-header">
        <h2 className="fee-report-section-title">Installment Status Sheet</h2>
        <select value={gradeFilter} onChange={e => setGradeFilter(e.target.value)} className="fee-report-grade-select">
          <option value="">All Classes</option>
          {grades.map(g => <option key={g.id} value={g.id}>{g.name}</option>)}
        </select>
      </div>

      {sheet && (
        <div className="fee-report-summary">
          <div className="fee-report-stat">
            <span>{sheet.installment1_paid_count} / {sheet.total_students}</span>
            <label>Students Paid Installment 1</label>
          </div>
        </div>
      )}

      <div className="table-card fee-report-sheet-scroll">
        <table className="data-table fee-report-sheet-table">
          <thead>
            <tr>
              <th>Student</th><th>Class</th>
              {Array.from({ length: sheet?.max_installments || 0 }, (_, i) => (
                <th key={i}>Inst. {i + 1}</th>
              ))}
              <th>Total Due</th><th>Total Paid</th><th>Balance</th>
            </tr>
          </thead>
          <tbody>
            {loadingSheet ? (
              <tr><td colSpan={99} className="data-table__empty">Loading...</td></tr>
            ) : !sheet || sheet.items.length === 0 ? (
              <tr><td colSpan={99} className="data-table__empty">No fee accounts found</td></tr>
            ) : sheet.items.map(row => (
              <tr key={row.student_id}>
                <td>
                  <span className="fee-report-name">{row.student_name}</span>
                  <span className="fee-report-code">{row.student_code}</span>
                </td>
                <td className="data-table__muted">{row.grade_level_name}</td>
                {Array.from({ length: sheet.max_installments }, (_, i) => {
                  const cell = row.installments[i]
                  if (!cell) return <td key={i}>—</td>
                  return (
                    <td key={i}>
                      <span className={`fee-report-badge fee-report-badge--${cell.status}`}>
                        {cell.status === 'paid' ? 'Paid' : cell.status === 'partial' ? 'Partial' : 'Pending'}
                      </span>
                      <div className="fee-report-cell-amount">{fmt(cell.paid_amount)} / {fmt(cell.planned_amount)}</div>
                    </td>
                  )
                })}
                <td>{fmt(row.total_due)}</td>
                <td>{fmt(row.total_paid)}</td>
                <td className={row.balance > 0 ? 'fee-report-negative' : ''}>{fmt(row.balance)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}

export default FeeReport
