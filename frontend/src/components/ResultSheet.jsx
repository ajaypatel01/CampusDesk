import { useEffect, useMemo, useState } from 'react'
import { Download, RefreshCw } from 'lucide-react'
import { resultsApi } from '../services/api'
import './ResultSheet.css'

const fmtNum = (n) => (Number.isInteger(n) ? String(n) : n.toFixed(1))

// Whole-class result sheet for one exam: a row per student, a column per
// subject, then total, percentage, grade, result and rank. Admins and the
// owner only -- the backend refuses everyone else.
function ResultSheet({ exams }) {
  const [examId, setExamId] = useState('')
  const [sheet, setSheet] = useState(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [sortBy, setSortBy] = useState('rank') // rank | name | code

  // Default to the latest exam of the class; reset when the class changes.
  useEffect(() => {
    setSheet(null); setError('')
    setExamId(exams.length ? exams[exams.length - 1].id : '')
  }, [exams])

  function load(id = examId) {
    if (!id) return
    setLoading(true); setError('')
    resultsApi.getResultSheet(id)
      .then(setSheet)
      .catch(err => { setSheet(null); setError(err.message) })
      .finally(() => setLoading(false))
  }

  useEffect(() => { if (examId) load(examId) }, [examId]) // eslint-disable-line react-hooks/exhaustive-deps

  const students = useMemo(() => {
    const list = [...(sheet?.students || [])]
    if (sortBy === 'name') list.sort((a, b) => a.student_name.localeCompare(b.student_name))
    else if (sortBy === 'code') list.sort((a, b) => a.student_code.localeCompare(b.student_code, undefined, { numeric: true }))
    else list.sort((a, b) => (a.rank || 1e9) - (b.rank || 1e9) || a.student_name.localeCompare(b.student_name))
    return list
  }, [sheet, sortBy])

  // Class summary: per-subject average of students who sat it, and pass counts.
  const summary = useMemo(() => {
    if (!sheet) return null
    const withMarks = sheet.students.filter(s => s.has_marks)
    const bySubject = {}
    for (const sub of sheet.subjects) {
      const cells = withMarks.map(s => s.marks[sub.id]).filter(c => c && !c.is_absent)
      bySubject[sub.id] = cells.length ? cells.reduce((t, c) => t + c.obtained, 0) / cells.length : null
    }
    const pcts = withMarks.map(s => s.percentage)
    return {
      bySubject,
      avgPct: pcts.length ? pcts.reduce((a, b) => a + b, 0) / pcts.length : null,
      passed: withMarks.filter(s => s.result === 'Pass').length,
      failed: withMarks.filter(s => s.result === 'Fail').length,
      noMarks: sheet.students.length - withMarks.length,
    }
  }, [sheet])

  function exportCSV() {
    if (!sheet) return
    const esc = v => `"${String(v ?? '').replace(/"/g, '""')}"`
    const header = ['Rank', 'Scholar No.', 'Student', ...sheet.subjects.map(s => `${s.name} (/${s.max_marks})`), 'Total', 'Max', '%', 'Grade', 'Result']
    const lines = [header.map(esc).join(',')]
    for (const st of students) {
      lines.push([
        st.rank || '', st.student_code, st.student_name,
        ...sheet.subjects.map(sub => {
          const c = st.marks[sub.id]
          return !c ? '' : c.is_absent ? 'AB' : fmtNum(c.obtained)
        }),
        st.has_marks ? fmtNum(st.total_obtained) : '', st.has_marks ? st.total_max : '',
        st.has_marks ? st.percentage.toFixed(1) : '', st.grade || '', st.result || '',
      ].map(esc).join(','))
    }
    const blob = new Blob(['﻿' + lines.join('\n')], { type: 'text/csv;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `result-sheet_${sheet.grade_level_name}_${sheet.exam_name}.csv`.replace(/\s+/g, '-')
    a.click()
    URL.revokeObjectURL(url)
  }

  if (exams.length === 0) return <p className="empty-text">No exams yet for this class.</p>

  return (
    <div className="result-sheet">
      <div className="result-sheet__toolbar">
        <label className="form-field" style={{ margin: 0, minWidth: '200px' }}>
          <span>Exam</span>
          <select value={examId} onChange={e => setExamId(e.target.value)}>
            {exams.map(e => <option key={e.id} value={e.id}>{e.name}</option>)}
          </select>
        </label>
        <label className="form-field" style={{ margin: 0, minWidth: '150px' }}>
          <span>Sort by</span>
          <select value={sortBy} onChange={e => setSortBy(e.target.value)}>
            <option value="rank">Rank</option>
            <option value="name">Name</option>
            <option value="code">Scholar No.</option>
          </select>
        </label>
        <div className="result-sheet__actions">
          <button className="btn btn--outline" onClick={() => load()} disabled={loading || !examId}>
            <RefreshCw size={16} /> {loading ? 'Loading...' : 'Refresh'}
          </button>
          <button className="btn btn--outline" onClick={exportCSV} disabled={!sheet || loading}>
            <Download size={16} /> Export CSV
          </button>
        </div>
      </div>

      {error && <p className="doc-msg doc-msg--error">{error}</p>}

      {sheet && summary && (
        <div className="result-sheet__stats">
          <span><strong>{sheet.students.length}</strong> students</span>
          <span><strong>{summary.passed}</strong> passed</span>
          <span className={summary.failed ? 'result-sheet__stat--fail' : ''}><strong>{summary.failed}</strong> failed</span>
          {summary.noMarks > 0 && <span><strong>{summary.noMarks}</strong> without marks</span>}
          {summary.avgPct !== null && <span>Class average <strong>{summary.avgPct.toFixed(1)}%</strong></span>}
          {!sheet.is_published && <span className="badge badge--muted">Draft: not visible to parents</span>}
        </div>
      )}

      {sheet && (
        sheet.subjects.length === 0 ? (
          <p className="empty-text">No subjects set up for {sheet.grade_level_name}.</p>
        ) : (
          <div className="table-card result-sheet__scroll">
            <table className="data-table result-sheet__table">
              <thead>
                <tr>
                  <th className="result-sheet__sticky result-sheet__rank">Rank</th>
                  <th className="result-sheet__sticky result-sheet__name">Student</th>
                  {sheet.subjects.map(sub => (
                    <th key={sub.id} className="result-sheet__num" title={sub.is_co_scholastic ? 'Co-scholastic: not counted in the total' : `Pass mark ${sub.passing_marks}`}>
                      {sub.name}
                      <div className="result-sheet__sub">/{sub.max_marks}{sub.is_co_scholastic ? ' · CS' : ''}</div>
                    </th>
                  ))}
                  <th className="result-sheet__num">Total</th>
                  <th className="result-sheet__num">%</th>
                  <th>Grade</th>
                  <th>Result</th>
                </tr>
              </thead>
              <tbody>
                {students.map(st => (
                  <tr key={st.student_id} className={st.has_marks ? '' : 'result-sheet__row--empty'}>
                    <td className="result-sheet__sticky result-sheet__rank">{st.rank || '–'}</td>
                    <td className="result-sheet__sticky result-sheet__name">
                      {st.student_name}
                      <div className="data-table__muted">{st.student_code}</div>
                    </td>
                    {sheet.subjects.map(sub => {
                      const c = st.marks[sub.id]
                      if (!c) return <td key={sub.id} className="result-sheet__num data-table__muted">–</td>
                      if (c.is_absent) return <td key={sub.id} className="result-sheet__num result-sheet__absent">AB</td>
                      return (
                        <td key={sub.id} className={`result-sheet__num ${c.status === 'Fail' && !sub.is_co_scholastic ? 'result-sheet__fail' : ''}`} title={`${c.grade} · ${c.status}`}>
                          {fmtNum(c.obtained)}
                        </td>
                      )
                    })}
                    <td className="result-sheet__num">
                      {st.has_marks ? <><strong>{fmtNum(st.total_obtained)}</strong><span className="data-table__muted">/{st.total_max}</span></> : '–'}
                      {st.is_total_overridden && <span className="badge badge--muted" style={{ marginLeft: '4px' }} title="Total edited by the owner">edited</span>}
                    </td>
                    <td className="result-sheet__num">{st.has_marks ? `${st.percentage.toFixed(1)}%` : '–'}</td>
                    <td>{st.grade ? <span className="badge badge--muted">{st.grade}</span> : '–'}</td>
                    <td>
                      {st.result
                        ? <span className={`badge badge--${st.result === 'Pass' ? 'success' : 'danger'}`}>{st.result}</span>
                        : <span className="data-table__muted">No marks</span>}
                    </td>
                  </tr>
                ))}
              </tbody>
              <tfoot>
                <tr className="result-sheet__summary">
                  <td className="result-sheet__sticky result-sheet__rank"></td>
                  <td className="result-sheet__sticky result-sheet__name">Class average</td>
                  {sheet.subjects.map(sub => (
                    <td key={sub.id} className="result-sheet__num">
                      {summary.bySubject[sub.id] === null ? '–' : fmtNum(Math.round(summary.bySubject[sub.id] * 10) / 10)}
                    </td>
                  ))}
                  <td></td>
                  <td className="result-sheet__num">{summary.avgPct === null ? '–' : `${summary.avgPct.toFixed(1)}%`}</td>
                  <td></td>
                  <td>{summary.passed}/{summary.passed + summary.failed} passed</td>
                </tr>
              </tfoot>
            </table>
          </div>
        )
      )}
    </div>
  )
}

export default ResultSheet
