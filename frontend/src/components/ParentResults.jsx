import { useEffect, useState } from 'react'
import { BarChart2, Download, Lock, Award } from 'lucide-react'
import { resultsApi } from '../services/api'
import './ParentResults.css'

const fmt = (n) => (Number.isInteger(n) ? String(n) : Number(n).toFixed(1))
const rupees = (n) => `₹${Number(n || 0).toLocaleString('en-IN')}`

// A parent's view of their child's results: exams as buttons (latest open
// by default), the overall result first, then one easy-to-read row per
// subject. Exams the school locked for unpaid fees show how much is due.
function ParentResults({ wardId, wardCode, exams }) {
  const [examId, setExamId] = useState('')
  const [marksheet, setMarksheet] = useState(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  // Open the most recent exam whenever the child or the exam list changes.
  useEffect(() => {
    setExamId(exams.length ? exams[exams.length - 1].id : '')
  }, [exams])

  const exam = exams.find(e => e.id === examId)
  useEffect(() => {
    setMarksheet(null); setError('')
    if (!exam || exam.fee_locked || !wardId) return
    setLoading(true)
    resultsApi.getMarksheet(exam.id, wardId)
      .then(setMarksheet)
      .catch(err => setError(err.message || 'Could not load the results'))
      .finally(() => setLoading(false))
  }, [exam, wardId])

  async function download() {
    try {
      const blob = await resultsApi.downloadMarksheet(examId, wardId)
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url; a.download = `marksheet_${wardCode || 'ward'}_${exam?.name || ''}.pdf`.replace(/\s+/g, '_')
      a.click(); URL.revokeObjectURL(url)
    } catch (err) {
      setError(err.message)
    }
  }

  const passed = marksheet?.result === 'Pass'
  const rows = marksheet?.rows || []

  return (
    <section className="pr detail-card">
      <div className="pr__head">
        <h3><BarChart2 size={16} /> Results</h3>
        {marksheet && (
          <button className="btn btn--outline btn--sm" onClick={download}><Download size={14} /> Download PDF</button>
        )}
      </div>

      {exams.length === 0 ? (
        <p className="empty-text">No results have been published yet.</p>
      ) : (
        <>
          {exams.length > 1 && (
            <div className="pr__exams" role="tablist" aria-label="Exams">
              {exams.map(e => (
                <button key={e.id} role="tab" aria-selected={e.id === examId}
                  className={`pr__exam ${e.id === examId ? 'pr__exam--on' : ''}`} onClick={() => setExamId(e.id)}>
                  {e.fee_locked && <Lock size={12} aria-label="locked" />} {e.name}
                </button>
              ))}
            </div>
          )}

          {exam?.fee_locked ? (
            <div className="pr__locked" role="status">
              <Lock size={22} aria-hidden="true" />
              <div>
                <strong>{exam.name} results are ready, but locked</strong>
                <p>{rupees(exam.fee_due)} fee is due. Please pay at the school office; the results open here as soon as the payment is recorded.</p>
              </div>
            </div>
          ) : loading ? (
            <p className="loading-text">Loading results...</p>
          ) : error ? (
            <p className="doc-msg doc-msg--error">{error}</p>
          ) : marksheet && (
            <>
              <div className={`pr__summary ${passed ? 'pr__summary--pass' : 'pr__summary--fail'}`}>
                <div className="pr__pct">
                  <span className="pr__pct-num">{marksheet.percentage?.toFixed(1)}%</span>
                  <span className="pr__pct-label">{marksheet.exam_name}</span>
                </div>
                <div className="pr__facts">
                  <div><span>Result</span><strong className={passed ? 'pr__ok' : 'pr__bad'}>{marksheet.result || '-'}</strong></div>
                  <div><span>Grade</span><strong>{marksheet.overall_grade || '-'}</strong></div>
                  <div><span>Total marks</span><strong>{fmt(marksheet.total_obtained)} / {marksheet.total_max}</strong></div>
                </div>
              </div>

              <ul className="pr__subjects">
                {rows.map((r, i) => {
                  const graded = !!r.grade_letter
                  const failed = r.status === 'Fail' && !r.is_co_scholastic
                  return (
                    <li key={i} className={`pr__subject ${failed ? 'pr__subject--fail' : ''}`}>
                      <div className="pr__subject-top">
                        <span className="pr__subject-name">
                          {r.subject_name}
                          {r.is_co_scholastic && <span className="pr__note">Not counted in total</span>}
                        </span>
                        <span className="pr__subject-marks">
                          {r.is_absent ? 'Absent' : graded ? `Grade ${r.grade_letter}` : <>{fmt(r.marks_obtained)}<span className="pr__muted"> / {r.max_marks}</span></>}
                        </span>
                      </div>
                      {!r.is_absent && !graded && (
                        <div className="pr__bar-row">
                          <div className="pr__bar" aria-hidden="true"><div className="pr__bar-fill" style={{ width: `${Math.min(100, Math.max(0, r.percentage || 0))}%` }} /></div>
                          <span className="pr__grade">{r.grade}</span>
                          {failed && <span className="pr__fail-tag">Needs improvement</span>}
                        </div>
                      )}
                    </li>
                  )
                })}
              </ul>
              {passed && marksheet.percentage >= 90 && (
                <p className="pr__cheer"><Award size={16} /> Excellent result!</p>
              )}
            </>
          )}
        </>
      )}
    </section>
  )
}

export default ParentResults
