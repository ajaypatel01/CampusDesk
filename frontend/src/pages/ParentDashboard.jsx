import { useState, useEffect } from 'react'
import { GraduationCap, BookOpen, BarChart2, Download, IndianRupee } from 'lucide-react'
import { useSchool } from '../services/SchoolContext'
import { studentsApi, homeworkApi, resultsApi, feesApi } from '../services/api'
import './ParentDashboard.css'

function fmtAmount(n) {
  return `₹${(n || 0).toLocaleString('en-IN')}`
}

function installmentLabel(p) {
  const type = (p.fee_type || 'tuition').replace(/_/g, ' ')
  const label = type.charAt(0).toUpperCase() + type.slice(1)
  return p.installment_number ? `${label} - Installment ${p.installment_number}` : label
}

function downloadBlob(blob, filename) {
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url; a.download = filename; a.click()
  URL.revokeObjectURL(url)
}

function fmtDate(d) {
  return d ? new Date(d).toLocaleDateString('en-IN') : '-'
}

const submissionBadge = {
  submitted: 'success',
  late: 'warning',
  missing: 'danger',
  pending: 'muted',
}

function ParentDashboard() {
  const { currentYear } = useSchool()
  const [wards, setWards] = useState([])
  const [selectedWardId, setSelectedWardId] = useState(null)
  const [loading, setLoading] = useState(true)

  const [homework, setHomework] = useState([])
  const [homeworkLoading, setHomeworkLoading] = useState(false)

  const [exams, setExams] = useState([])
  const [examId, setExamId] = useState('')
  const [marksheet, setMarksheet] = useState(null)
  const [marksheetLoading, setMarksheetLoading] = useState(false)

  const [feeSummary, setFeeSummary] = useState(null)
  const [feeLoading, setFeeLoading] = useState(false)
  const [feeError, setFeeError] = useState('')

  const ward = wards.find(w => w.id === selectedWardId) || null

  useEffect(() => {
    studentsApi.myWards()
      .then(res => {
        const items = res.items || []
        setWards(items)
        if (items.length > 0) setSelectedWardId(items[0].id)
      })
      .catch(() => setWards([]))
      .finally(() => setLoading(false))
  }, [])

  useEffect(() => {
    if (!selectedWardId || !currentYear) return
    setHomeworkLoading(true)
    homeworkApi.wardHomework({ student_id: selectedWardId, academic_year_id: currentYear.id })
      .then(res => setHomework(res.items || []))
      .catch(() => setHomework([]))
      .finally(() => setHomeworkLoading(false))

    resultsApi.wardExams({ student_id: selectedWardId, academic_year_id: currentYear.id })
      .then(res => { setExams(res.items || []); setExamId('') })
      .catch(() => setExams([]))
    setMarksheet(null)

    setFeeLoading(true)
    setFeeError('')
    feesApi.studentSummary(selectedWardId, currentYear.id)
      .then(setFeeSummary)
      .catch(err => { setFeeSummary(null); setFeeError(err.message || 'Failed to load fee details') })
      .finally(() => setFeeLoading(false))
  }, [selectedWardId, currentYear])

  function loadMarksheet() {
    if (!examId || !selectedWardId) return
    setMarksheetLoading(true)
    resultsApi.getMarksheet(examId, selectedWardId)
      .then(setMarksheet)
      .catch(() => setMarksheet(null))
      .finally(() => setMarksheetLoading(false))
  }

  async function downloadMarksheet() {
    try {
      const blob = await resultsApi.downloadMarksheet(examId, selectedWardId)
      downloadBlob(blob, `marksheet_${ward?.student_code || 'ward'}.pdf`)
    } catch (err) {
      alert(err.message)
    }
  }

  if (loading) return <p className="loading-text">Loading...</p>

  if (wards.length === 0) {
    return (
      <div className="parent-dashboard">
        <div className="page-header"><h1>My Ward</h1></div>
        <p className="empty-text">No ward is linked to your account yet. Contact your school&apos;s registrar or admin.</p>
      </div>
    )
  }

  return (
    <div className="parent-dashboard">
      <div className="page-header">
        <div>
          <h1>My Ward</h1>
          <p className="page-subtitle">Your ward&apos;s profile, class homework, and results</p>
        </div>
        {wards.length > 1 && (
          <select className="parent-dashboard__ward-select" value={selectedWardId || ''} onChange={e => setSelectedWardId(e.target.value)}>
            {wards.map(w => <option key={w.id} value={w.id}>{w.first_name} {w.last_name} ({w.student_code})</option>)}
          </select>
        )}
      </div>

      {ward && (
        <div className="detail-card parent-dashboard__profile">
          <div className="parent-dashboard__avatar"><GraduationCap size={28} /></div>
          <div>
            <h2>{ward.first_name} {ward.last_name}</h2>
            <div className="parent-dashboard__meta">
              <span>Scholar No: <strong>{ward.student_code}</strong></span>
              <span className={`badge badge--${ward.status === 'active' ? 'success' : 'muted'}`}>{ward.status}</span>
            </div>
          </div>
        </div>
      )}

      <div className="parent-dashboard__grid">
        <div className="detail-card">
          <h3><BookOpen size={16} /> Class Homework</h3>
          {homeworkLoading ? <p className="loading-text">Loading...</p> : homework.length === 0 ? (
            <p className="empty-text">No homework assigned yet</p>
          ) : (
            <div className="table-card">
              <table className="data-table">
                <thead><tr><th>Title</th><th>Due Date</th><th>Status</th></tr></thead>
                <tbody>
                  {homework.map(hw => (
                    <tr key={hw.id}>
                      <td>
                        {hw.title}
                        {hw.description && <div className="data-table__muted">{hw.description}</div>}
                      </td>
                      <td className="data-table__muted">{fmtDate(hw.due_date)}</td>
                      <td>
                        <span className={`badge badge--${submissionBadge[hw.submission_status] || 'muted'}`}>
                          {hw.submission_status || 'not tracked'}
                        </span>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>

        <div className="detail-card">
          <h3><BarChart2 size={16} /> Results</h3>
          {exams.length === 0 ? (
            <p className="empty-text">No published exams yet</p>
          ) : (
            <>
              <div className="parent-dashboard__exam-row">
                <select value={examId} onChange={e => setExamId(e.target.value)}>
                  <option value="">Select exam</option>
                  {exams.map(ex => <option key={ex.id} value={ex.id}>{ex.name}</option>)}
                </select>
                <button className="btn btn--primary btn--sm" onClick={loadMarksheet} disabled={!examId || marksheetLoading}>
                  {marksheetLoading ? 'Loading...' : 'View'}
                </button>
                {marksheet && (
                  <button className="btn btn--outline btn--sm" onClick={downloadMarksheet}>
                    <Download size={14} /> PDF
                  </button>
                )}
              </div>

              {marksheet && (
                <div className="marksheet-preview">
                  <p><strong>{marksheet.exam_name}</strong> · {marksheet.academic_year}</p>
                  <table className="data-table">
                    <thead><tr><th>Subject</th><th>Max</th><th>Obtained</th><th>%</th><th>Grade</th><th>Status</th></tr></thead>
                    <tbody>
                      {(marksheet.rows || []).map((row, i) => (
                        <tr key={i}>
                          <td>{row.subject_name}</td>
                          <td className="data-table__muted">{row.max_marks}</td>
                          <td>{row.is_absent ? 'Absent' : row.marks_obtained}</td>
                          <td className="data-table__muted">{row.is_absent ? '-' : row.percentage?.toFixed(1) + '%'}</td>
                          <td><span className="badge badge--muted">{row.grade}</span></td>
                          <td>
                            <span className={`badge badge--${row.status === 'Pass' ? 'success' : row.status === 'Fail' ? 'danger' : 'muted'}`}>
                              {row.status}
                            </span>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                  <p className="parent-dashboard__total">
                    Total: {marksheet.total_obtained} / {marksheet.total_max} ({marksheet.percentage?.toFixed(1)}%) · {marksheet.result}
                  </p>
                </div>
              )}
            </>
          )}
        </div>

        <div className="detail-card">
          <h3><IndianRupee size={16} /> Fees</h3>
          {feeLoading ? (
            <p className="loading-text">Loading...</p>
          ) : feeError || !feeSummary ? (
            <p className="empty-text">No fee details available for this year yet.</p>
          ) : (
            <>
              <div className="parent-dashboard__fee-totals">
                <div>
                  <span className="parent-dashboard__fee-label">Total Due</span>
                  <strong>{fmtAmount(feeSummary.total_due)}</strong>
                </div>
                <div>
                  <span className="parent-dashboard__fee-label">Total Paid</span>
                  <strong>{fmtAmount(feeSummary.total_paid)}</strong>
                </div>
                <div>
                  <span className="parent-dashboard__fee-label">Balance Remaining</span>
                  <strong className={feeSummary.balance_remaining > 0 ? 'parent-dashboard__fee-due' : 'parent-dashboard__fee-clear'}>
                    {fmtAmount(feeSummary.balance_remaining)}
                  </strong>
                </div>
              </div>

              {feeSummary.payments?.length === 0 ? (
                <p className="empty-text">No payments recorded yet</p>
              ) : (
                <div className="table-card">
                  <table className="data-table">
                    <thead><tr><th>Date</th><th>Details</th><th>Mode</th><th>Amount</th></tr></thead>
                    <tbody>
                      {feeSummary.payments.map(p => (
                        <tr key={p.id}>
                          <td className="data-table__muted">{fmtDate(p.payment_date)}</td>
                          <td>
                            {installmentLabel(p)}
                            {p.reference_number && <div className="data-table__muted">Ref: {p.reference_number}</div>}
                          </td>
                          <td className="data-table__muted" style={{ textTransform: 'capitalize' }}>{p.payment_mode}</td>
                          <td>{fmtAmount(p.amount)}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </>
          )}
        </div>
      </div>
    </div>
  )
}

export default ParentDashboard
