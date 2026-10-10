import { useEffect, useState } from 'react'
import { FileCheck2, Download } from 'lucide-react'
import { classMediaApi } from '../services/api'
import { formatDate } from '../utils/date'

const KIND = { tc: 'TC', marksheet: 'Marksheet', report_card: 'Report card' }

// Copies of this student's TCs, marksheets and report cards exactly as they
// were issued (kept in S3 whenever one is downloaded, emailed or sent).
function IssuedDocuments({ studentId }) {
  const [items, setItems] = useState(null)
  const [error, setError] = useState('')
  useEffect(() => {
    classMediaApi.issued(studentId).then(r => setItems(r.items || [])).catch(e => setError(e.message))
  }, [studentId])

  return (
    <div className="detail-card">
      <h3 style={{ display: 'flex', alignItems: 'center', gap: '6px' }}><FileCheck2 size={16} /> Issued documents</h3>
      {error && <p className="empty-text">{error}</p>}
      {items && items.length === 0 && <p className="empty-text">No TC, marksheet or report card issued yet.</p>}
      {items && items.length > 0 && (
        <ul style={{ listStyle: 'none', margin: 0, padding: 0, display: 'flex', flexDirection: 'column', gap: '8px' }}>
          {items.map(d => (
            <li key={d.id} style={{ display: 'flex', alignItems: 'center', gap: '10px', fontSize: '0.875rem' }}>
              <span className="badge badge--muted">{KIND[d.kind] || d.kind}</span>
              <span style={{ flex: 1, minWidth: 0 }}>
                {d.title}
                <span className="data-table__muted" style={{ display: 'block', fontSize: '0.78rem' }}>
                  {formatDate(d.issued_at)}{d.issued_by_name ? ` · ${d.issued_by_name}` : ''}
                </span>
              </span>
              {d.url && <a className="btn btn--outline btn--sm" href={d.url} target="_blank" rel="noreferrer" aria-label={`Open ${d.title}`}><Download size={14} /></a>}
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}

export default IssuedDocuments
