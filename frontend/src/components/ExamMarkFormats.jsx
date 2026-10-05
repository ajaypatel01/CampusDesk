import { useEffect, useState } from 'react'
import { Plus, Trash2, Pencil, RotateCcw } from 'lucide-react'
import { resultsApi } from '../services/api'

// Per-exam marks distribution: every subject in an exam can have its own
// fields (Unit Test: Written /20; Half Yearly: Written /60 + Oral /10). A
// subject without one uses its own fields from the Subjects tab. Admin-only --
// the backend blocks teachers and parents from changing formats.
function ExamMarkFormats({ exam, onChanged }) {
  const [formats, setFormats] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [editing, setEditing] = useState(null) // { subjectId, rows: [{ key, label, max_marks }] }
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    setLoading(true); setError(''); setEditing(null)
    resultsApi.listExamFormats(exam.id)
      .then(r => setFormats(r.items || []))
      .catch(err => setError(err.message))
      .finally(() => setLoading(false))
  }, [exam.id])

  function startEdit(f) {
    setError('')
    setEditing({
      subjectId: f.subject_id,
      rows: f.components.map(c => ({ key: c.key, label: c.label, max_marks: String(c.max_marks) })),
    })
  }

  function updateRow(i, patch) {
    setEditing(prev => ({ ...prev, rows: prev.rows.map((r, j) => (j === i ? { ...r, ...patch } : r)) }))
  }

  async function save() {
    setSaving(true); setError('')
    try {
      const res = await resultsApi.setExamSubjectFormat(exam.id, editing.subjectId, {
        components: editing.rows.map(r => ({ key: r.key || '', label: r.label.trim(), max_marks: parseInt(r.max_marks, 10) || 0 })),
      })
      setFormats(res.items || [])
      setEditing(null)
      onChanged?.()
    } catch (err) {
      setError(err.message)
    } finally {
      setSaving(false)
    }
  }

  async function reset(f) {
    if (!confirm(`Use ${f.subject_name}'s own fields for ${exam.name} again?`)) return
    setError('')
    try {
      const res = await resultsApi.resetExamSubjectFormat(exam.id, f.subject_id)
      setFormats(res.items || [])
      onChanged?.()
    } catch (err) {
      setError(err.message)
    }
  }

  if (loading) return <p className="empty-text">Loading marks format...</p>

  return (
    <div className="exam-formats">
      <p className="data-table__muted" style={{ marginBottom: '10px' }}>
        Set how each subject is marked in <strong>{exam.name}</strong>. Subjects you don&apos;t change here use their own
        fields from the Subjects tab. Remove every field to enter one plain mark.
      </p>
      {error && <p className="doc-msg doc-msg--error">{error}</p>}
      {formats.length === 0 ? (
        <p className="empty-text">No subjects for this grade yet.</p>
      ) : (
        <div className="subject-cards">
          {formats.map(f => {
            const isEditing = editing?.subjectId === f.subject_id
            const total = f.components.reduce((sum, c) => sum + c.max_marks, 0)
            return (
              <div key={f.subject_id} className="subject-card">
                <div className="subject-card__header">
                  <div className="subject-card__title">
                    <strong>{f.subject_name}</strong>
                    <span className={`badge badge--${f.custom ? 'success' : 'muted'}`} style={{ marginLeft: '8px' }}>
                      {f.custom ? 'For this exam' : "Subject's own fields"}
                    </span>
                  </div>
                  {!isEditing && (
                    <div className="subject-card__meta">
                      <button className="btn btn--outline btn--sm" onClick={() => startEdit(f)}><Pencil size={13} /> Edit</button>
                      {f.custom && (
                        <button className="btn btn--outline btn--sm" onClick={() => reset(f)} title="Use the subject's own fields again">
                          <RotateCcw size={13} />
                        </button>
                      )}
                    </div>
                  )}
                </div>

                {isEditing ? (
                  <div className="subject-card__fields">
                    {editing.rows.length === 0 && <p className="empty-text" style={{ padding: 0, textAlign: 'left' }}>No fields -- one plain mark out of {f.max_marks}.</p>}
                    {editing.rows.map((row, i) => (
                      <div key={i} className="marks-add-component" style={{ marginBottom: '6px' }}>
                        <input placeholder="Field name (e.g. Oral)" value={row.label} onChange={e => updateRow(i, { label: e.target.value })} />
                        <input type="number" min="1" placeholder="Max" style={{ width: '70px' }} value={row.max_marks} onChange={e => updateRow(i, { max_marks: e.target.value })} />
                        <button type="button" className="btn btn--outline btn--sm" title="Remove field"
                          onClick={() => setEditing(prev => ({ ...prev, rows: prev.rows.filter((_, j) => j !== i) }))}>
                          <Trash2 size={13} />
                        </button>
                      </div>
                    ))}
                    <p className="subject-card__fields-total">
                      Fields total {editing.rows.reduce((s, r) => s + (parseInt(r.max_marks, 10) || 0), 0)}
                    </p>
                    <div style={{ display: 'flex', gap: '8px', flexWrap: 'wrap', marginTop: '8px' }}>
                      <button type="button" className="btn btn--outline btn--sm"
                        onClick={() => setEditing(prev => ({ ...prev, rows: [...prev.rows, { key: '', label: '', max_marks: '' }] }))}>
                        <Plus size={13} /> Add Field
                      </button>
                      <button type="button" className="btn btn--primary btn--sm" onClick={save} disabled={saving}>{saving ? 'Saving...' : 'Save'}</button>
                      <button type="button" className="btn btn--outline btn--sm" onClick={() => setEditing(null)} disabled={saving}>Cancel</button>
                    </div>
                  </div>
                ) : (
                  <div className="subject-card__fields">
                    {f.components.length === 0 ? (
                      <p className="empty-text" style={{ padding: 0, textAlign: 'left' }}>One plain mark out of {f.max_marks}.</p>
                    ) : (
                      <>
                        <div className="field-chips">
                          {f.components.map(c => (
                            <span key={c.key} className="field-chip" style={{ paddingRight: '12px' }}>
                              {c.label} <span className="data-table__muted">/{c.max_marks}</span>
                            </span>
                          ))}
                        </div>
                        <p className="subject-card__fields-total">Total {total}</p>
                      </>
                    )}
                  </div>
                )}
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}

export default ExamMarkFormats
