import { useEffect, useMemo, useState } from 'react'
import { Download, RefreshCw, AlertTriangle, Trophy, ChevronDown } from 'lucide-react'
import { resultsApi } from '../services/api'
import './ResultDashboard.css'

// Grade scale from the backend's gradeFromPercent, best first.
const GRADES = ['A1', 'A2', 'B1', 'B2', 'C1', 'C2', 'D', 'F']
// Letters a grading-only subject is given instead of marks.
const GRADE_LETTERS = ['A', 'B', 'C', 'D']
const fmt = (n) => (Number.isInteger(n) ? String(n) : n.toFixed(1))
const pct = (n) => `${n.toFixed(1)}%`

// Results dashboard for one class and exam: headline numbers, how each subject
// went, the grade spread, who did best and who needs attention, then the full
// class sheet. View only -- for the owner, admins, registrars and class
// teachers (the backend shows a class teacher only their own section).
function ResultDashboard({ exams }) {
  const [examId, setExamId] = useState('')
  const [sheet, setSheet] = useState(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [sortBy, setSortBy] = useState('rank')

  // Default to the class's latest exam; reset when the class changes.
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

  const stats = useMemo(() => buildStats(sheet), [sheet])

  const sorted = useMemo(() => {
    const list = [...(sheet?.students || [])]
    if (sortBy === 'name') list.sort((a, b) => a.student_name.localeCompare(b.student_name))
    else if (sortBy === 'code') list.sort((a, b) => a.student_code.localeCompare(b.student_code, undefined, { numeric: true }))
    else list.sort((a, b) => (a.rank || 1e9) - (b.rank || 1e9) || a.student_name.localeCompare(b.student_name))
    return list
  }, [sheet, sortBy])

  function exportCSV() {
    if (!sheet) return
    const esc = v => `"${String(v ?? '').replace(/"/g, '""')}"`
    const header = ['Rank', 'Scholar No.', 'Student', ...sheet.subjects.map(s => `${s.name} (/${s.max_marks})`), 'Total', 'Max', '%', 'Grade', 'Result']
    const lines = [header.map(esc).join(',')]
    for (const st of sorted) {
      lines.push([
        st.rank || '', st.student_code, st.student_name,
        ...sheet.subjects.map(sub => { const c = st.marks[sub.id]; return !c ? '' : c.is_absent ? 'AB' : c.grade_letter || fmt(c.obtained) }),
        st.has_marks ? fmt(st.total_obtained) : '', st.has_marks ? st.total_max : '',
        st.has_marks ? st.percentage.toFixed(1) : '', st.grade || '', st.result || '',
      ].map(esc).join(','))
    }
    const url = URL.createObjectURL(new Blob(['﻿' + lines.join('\n')], { type: 'text/csv;charset=utf-8' }))
    const a = document.createElement('a')
    a.href = url
    a.download = `results_${sheet.grade_level_name}_${sheet.exam_name}.csv`.replace(/\s+/g, '-')
    a.click()
    URL.revokeObjectURL(url)
  }

  if (exams.length === 0) return <p className="empty-text">No exams yet for this class.</p>

  return (
    <div className={`rd ${loading && sheet ? 'rd--refreshing' : ''}`}>
      {/* One filter row scopes everything below. */}
      <div className="rd-filters">
        <label className="form-field rd-filters__exam">
          <span>Exam</span>
          <select value={examId} onChange={e => setExamId(e.target.value)}>
            {exams.map(e => <option key={e.id} value={e.id}>{e.name}</option>)}
          </select>
        </label>
        {sheet && !sheet.is_published && <span className="badge badge--muted">Draft: not visible to parents</span>}
        <div className="rd-filters__actions">
          <button className="btn btn--outline" onClick={() => load()} disabled={loading || !examId}>
            <RefreshCw size={16} /> {loading ? 'Loading...' : 'Refresh'}
          </button>
          <button className="btn btn--outline" onClick={exportCSV} disabled={!sheet}>
            <Download size={16} /> Export CSV
          </button>
        </div>
      </div>

      {error && <p className="doc-msg doc-msg--error">{error}</p>}
      {loading && !sheet && <p className="loading-text">Loading results...</p>}

      {sheet && stats && (stats.appeared === 0 ? (
        <p className="empty-text">No marks entered yet for {sheet.exam_name}. {sheet.students.length} students in {sheet.grade_level_name}.</p>
      ) : (
        <>
          {/* Headline numbers -- pass rate leads. */}
          <div className="rd-kpis">
            <div className="rd-tile rd-tile--hero">
              <div className="rd-tile__label">Pass rate</div>
              <div className="rd-tile__hero">{pct(stats.passRate)}</div>
              <div className="rd-tile__sub">{stats.passed} of {stats.appeared} students passed</div>
            </div>
            <div className="rd-tile">
              <div className="rd-tile__label">Appeared</div>
              <div className="rd-tile__value">{stats.appeared}<span className="rd-tile__of"> / {sheet.students.length}</span></div>
              <div className="rd-tile__sub">{stats.noMarks ? `${stats.noMarks} without marks` : 'Everyone has marks'}</div>
            </div>
            <div className="rd-tile">
              <div className="rd-tile__label">Class average</div>
              <div className="rd-tile__value">{pct(stats.avgPct)}</div>
              <div className="rd-tile__sub">Median {pct(stats.medianPct)}</div>
            </div>
            <div className="rd-tile">
              <div className="rd-tile__label">Highest</div>
              <div className="rd-tile__value">{pct(stats.top[0].percentage)}</div>
              <div className="rd-tile__sub" title={stats.top[0].student_name}>{stats.top[0].student_name}</div>
            </div>
            <div className="rd-tile">
              <div className="rd-tile__label">Failed</div>
              <div className="rd-tile__value">{stats.failed}</div>
              <div className={`rd-tile__sub ${stats.failed ? 'rd-status--critical' : ''}`}>
                {stats.failed ? <><AlertTriangle size={14} /> Needs attention</> : 'No failures'}
              </div>
            </div>
          </div>

          <div className="rd-grid">
            {/* Subject performance: average % per subject, pass count as text. */}
            <section className="rd-card">
              <h3 className="rd-card__title">Subject performance</h3>
              <p className="rd-card__subtitle">Average score per subject, among students who sat it</p>
              <div className="rd-hbars" role="table" aria-label="Average score by subject">
                {stats.subjects.map(s => (
                  <div key={s.id} className="rd-hbar" role="row" tabIndex={0}
                    title={`${s.name}: average ${s.avgPct === null ? '–' : pct(s.avgPct)} · passed ${s.passed}/${s.sat}${s.absent ? ` · ${s.absent} absent` : ''}`}>
                    <div className="rd-hbar__name" role="cell">
                      {s.name}{s.coScholastic && <span className="rd-muted"> · CS</span>}
                    </div>
                    {s.letters && s.avgPct === null ? (
                      <div className="rd-hbar__letters" role="cell">
                        {GRADE_LETTERS.map(g => (
                          <span key={g} className="rd-letter"><strong>{g}</strong> {s.letters[g] || 0}</span>
                        ))}
                      </div>
                    ) : (
                      <>
                        <div className="rd-hbar__track" role="cell">
                          {s.avgPct !== null && <div className="rd-hbar__fill" style={{ width: `${Math.min(Math.max(s.avgPct, 1), 100)}%` }} />}
                        </div>
                        <span className="rd-hbar__value" role="cell">{s.avgPct === null ? '–' : pct(s.avgPct)}</span>
                      </>
                    )}
                    <div className="rd-hbar__meta" role="cell">
                      {s.letters && s.avgPct === null ? 'Graded' : s.sat ? `${s.passed}/${s.sat} passed` : 'No marks'}
                      {s.absent > 0 && <span className="rd-muted"> · {s.absent} AB</span>}
                    </div>
                  </div>
                ))}
              </div>
            </section>

            {/* Grade distribution: one column per grade, best to worst. */}
            <section className="rd-card">
              <h3 className="rd-card__title">Grade distribution</h3>
              <p className="rd-card__subtitle">Students by overall grade</p>
              <div className="rd-cols" role="table" aria-label="Students by overall grade">
                {GRADES.map(g => {
                  const n = stats.gradeCounts[g] || 0
                  return (
                    <div key={g} className="rd-col" role="row" tabIndex={0} title={`${g}: ${n} student${n === 1 ? '' : 's'}`}>
                      <div className="rd-col__plot">
                        <span className="rd-col__value" role="cell">{n || ''}</span>
                        <div className="rd-col__bar" style={{ height: `${stats.maxGradeCount ? (n / stats.maxGradeCount) * 100 : 0}%` }} />
                      </div>
                      <div className="rd-col__label" role="cell">{g}</div>
                    </div>
                  )
                })}
              </div>
            </section>

            {/* Top performers */}
            <section className="rd-card">
              <h3 className="rd-card__title"><Trophy size={16} /> Top performers</h3>
              <ol className="rd-list">
                {stats.top.map(st => (
                  <li key={st.student_id} className="rd-list__item">
                    <span className="rd-list__rank">{st.rank}</span>
                    <span className="rd-list__name">{st.student_name}<span className="rd-muted rd-list__code"><span className="rd-list__sep"> · </span>{st.student_code}</span></span>
                    <span className="rd-list__value">{pct(st.percentage)}</span>
                  </li>
                ))}
              </ol>
            </section>

            {/* Needs attention: failed, then absent from any subject. */}
            <section className="rd-card">
              <h3 className="rd-card__title"><AlertTriangle size={16} /> Needs attention</h3>
              {stats.attention.length === 0 ? (
                <p className="rd-muted">Nobody failed or missed a subject.</p>
              ) : (
                <ul className="rd-list rd-list--attention">
                  {stats.attention.map(a => (
                    <li key={a.student.student_id} className="rd-list__item">
                      <span className={`rd-list__tag ${a.failed.length ? 'rd-status--critical' : ''}`}>{a.failed.length ? 'Fail' : 'Absent'}</span>
                      <span className="rd-list__name">
                        {a.student.student_name}
                        <span className="rd-muted">
                          {a.failed.length ? ` · failed ${a.failed.join(', ')}` : ''}
                          {a.absent.length ? ` · absent ${a.absent.join(', ')}` : ''}
                        </span>
                      </span>
                      <span className="rd-list__value">{pct(a.student.percentage)}</span>
                    </li>
                  ))}
                </ul>
              )}
            </section>
          </div>

          {/* Full class sheet -- the table view of everything above. */}
          <section className="rd-card rd-card--wide">
            <div className="rd-card__head">
              <div>
                <h3 className="rd-card__title">Full class sheet</h3>
                <p className="rd-card__subtitle">{sheet.grade_level_name} · {sheet.exam_name}</p>
              </div>
              <label className="form-field rd-sort">
                <span>Sort by</span>
                <select value={sortBy} onChange={e => setSortBy(e.target.value)}>
                  <option value="rank">Rank</option>
                  <option value="name">Name</option>
                  <option value="code">Scholar No.</option>
                </select>
              </label>
            </div>
            {/* Phones: one card per student, subject marks folded away. */}
            <ul className="rd-cards">
              {sorted.map(st => (
                <li key={st.student_id} className={`rd-scard ${st.has_marks ? '' : 'rd-row--empty'}`}>
                  <details>
                    <summary className="rd-scard__head">
                      <span className="rd-scard__rank">{st.rank || '–'}</span>
                      <span className="rd-scard__who">
                        <span className="rd-scard__name">{st.student_name}</span>
                        <span className="rd-muted rd-scard__code">{st.student_code}</span>
                      </span>
                      <span className="rd-scard__pct">{st.has_marks ? pct(st.percentage) : '–'}</span>
                      <ChevronDown size={16} className="rd-scard__chev" aria-hidden="true" />
                    </summary>
                    {st.has_marks && (
                      <div className="rd-scard__body">
                        {sheet.subjects.map(sub => {
                          const c = st.marks[sub.id]
                          const fail = c && !c.is_absent && !c.grade_letter && c.status === 'Fail' && !sub.is_co_scholastic
                          return (
                            <div key={sub.id} className="rd-scard__mark">
                              <span className="rd-scard__sub">{sub.name}{sub.is_co_scholastic && <span className="rd-muted"> · CS</span>}</span>
                              <span className={`rd-num ${fail ? 'rd-fail' : ''} ${c?.is_absent ? 'rd-absent' : ''}`}>
                                {!c ? '–' : c.is_absent ? 'AB' : c.grade_letter ? <strong>{c.grade_letter}</strong> : <>{fmt(c.obtained)}<span className="rd-muted">/{c.max_marks}</span></>}
                              </span>
                            </div>
                          )
                        })}
                      </div>
                    )}
                  </details>
                  <div className="rd-scard__foot">
                    {st.has_marks ? (
                      <>
                        <span className="rd-num"><strong>{fmt(st.total_obtained)}</strong><span className="rd-muted">/{st.total_max}</span>{st.is_total_overridden && <span className="rd-muted"> · edited</span>}</span>
                        {st.grade && <span className="badge badge--muted">{st.grade}</span>}
                        {st.result && <span className={`badge badge--${st.result === 'Pass' ? 'success' : 'danger'}`}>{st.result}</span>}
                      </>
                    ) : <span className="rd-muted">No marks</span>}
                  </div>
                </li>
              ))}
            </ul>
            <div className="table-card rd-sheet">
              <table className="data-table rd-sheet__table">
                <thead>
                  <tr>
                    <th className="rd-sticky rd-sticky--rank">Rank</th>
                    <th className="rd-sticky rd-sticky--name">Student</th>
                    {sheet.subjects.map(sub => (
                      <th key={sub.id} className="rd-num" title={sub.is_co_scholastic ? 'Co-scholastic: not counted in the total' : `Pass mark ${sub.passing_marks}`}>
                        {sub.name}<div className="rd-th-sub">/{sub.max_marks}{sub.is_co_scholastic ? ' · CS' : ''}</div>
                      </th>
                    ))}
                    <th className="rd-num">Total</th>
                    <th className="rd-num">%</th>
                    <th>Grade</th>
                    <th>Result</th>
                  </tr>
                </thead>
                <tbody>
                  {sorted.map(st => (
                    <tr key={st.student_id} className={st.has_marks ? '' : 'rd-row--empty'}>
                      <td className="rd-sticky rd-sticky--rank">{st.rank || '–'}</td>
                      <td className="rd-sticky rd-sticky--name">{st.student_name}<div className="data-table__muted">{st.student_code}</div></td>
                      {sheet.subjects.map(sub => {
                        const c = st.marks[sub.id]
                        if (!c) return <td key={sub.id} className="rd-num data-table__muted">–</td>
                        if (c.is_absent) return <td key={sub.id} className="rd-num rd-absent">AB</td>
                        if (c.grade_letter) return <td key={sub.id} className="rd-num" title="Graded"><strong>{c.grade_letter}</strong></td>
                        const fail = c.status === 'Fail' && !sub.is_co_scholastic
                        return <td key={sub.id} className={`rd-num ${fail ? 'rd-fail' : ''}`} title={`${c.grade} · ${c.status}`}>{fmt(c.obtained)}</td>
                      })}
                      <td className="rd-num">
                        {st.has_marks ? <><strong>{fmt(st.total_obtained)}</strong><span className="data-table__muted">/{st.total_max}</span></> : '–'}
                        {st.is_total_overridden && <span className="badge badge--muted" style={{ marginLeft: '4px' }} title="Total edited by the owner">edited</span>}
                      </td>
                      <td className="rd-num">{st.has_marks ? pct(st.percentage) : '–'}</td>
                      <td>{st.grade ? <span className="badge badge--muted">{st.grade}</span> : '–'}</td>
                      <td>{st.result ? <span className={`badge badge--${st.result === 'Pass' ? 'success' : 'danger'}`}>{st.result}</span> : <span className="data-table__muted">No marks</span>}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </section>
        </>
      ))}
    </div>
  )
}

// buildStats derives every dashboard number from the result sheet, so the
// tiles, charts, lists and table always agree.
function buildStats(sheet) {
  if (!sheet) return null
  const withMarks = sheet.students.filter(s => s.has_marks)
  const appeared = withMarks.length
  const passed = withMarks.filter(s => s.result === 'Pass').length
  const failed = withMarks.filter(s => s.result === 'Fail').length
  const pcts = withMarks.map(s => s.percentage).sort((a, b) => a - b)
  const median = pcts.length ? (pcts.length % 2 ? pcts[(pcts.length - 1) / 2] : (pcts[pcts.length / 2 - 1] + pcts[pcts.length / 2]) / 2) : 0

  const subjects = sheet.subjects.map(sub => {
    const cells = withMarks.map(s => s.marks[sub.id]).filter(Boolean)
    const sat = cells.filter(c => !c.is_absent)
    // Letter-graded cells (A-D) have no marks: count letters instead of averaging.
    const lettered = sat.filter(c => c.grade_letter)
    const marked = sat.filter(c => !c.grade_letter)
    const avg = marked.length ? marked.reduce((t, c) => t + (c.max_marks ? (c.obtained / c.max_marks) * 100 : 0), 0) / marked.length : null
    const letters = {}
    for (const c of lettered) letters[c.grade_letter] = (letters[c.grade_letter] || 0) + 1
    return {
      id: sub.id, name: sub.name, coScholastic: sub.is_co_scholastic,
      avgPct: avg, sat: marked.length, absent: cells.length - sat.length,
      passed: marked.filter(c => c.status === 'Pass').length,
      letters: lettered.length ? letters : null,
    }
  })

  const gradeCounts = {}
  for (const s of withMarks) gradeCounts[s.grade] = (gradeCounts[s.grade] || 0) + 1

  const top = [...withMarks].sort((a, b) => a.rank - b.rank || a.student_name.localeCompare(b.student_name)).slice(0, 5)

  const attention = withMarks.map(student => {
    const failedSubs = [], absentSubs = []
    for (const sub of sheet.subjects) {
      const c = student.marks[sub.id]
      if (!c) continue
      if (c.is_absent) absentSubs.push(sub.name)
      else if (c.status === 'Fail' && !sub.is_co_scholastic) failedSubs.push(sub.name)
    }
    return { student, failed: failedSubs, absent: absentSubs }
  }).filter(a => a.failed.length || a.absent.length)
    .sort((a, b) => (b.failed.length - a.failed.length) || (a.student.percentage - b.student.percentage))

  return {
    appeared, passed, failed, noMarks: sheet.students.length - appeared,
    passRate: appeared ? (passed / appeared) * 100 : 0,
    avgPct: appeared ? pcts.reduce((a, b) => a + b, 0) / appeared : 0,
    medianPct: median,
    subjects, gradeCounts, maxGradeCount: Math.max(0, ...Object.values(gradeCounts)),
    top, attention,
  }
}

export default ResultDashboard
