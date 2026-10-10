import { useEffect, useMemo, useRef, useState } from 'react'
import { useOutletContext } from 'react-router-dom'
import { Images, FileText, Upload, Trash2, X, ChevronLeft, ChevronRight, Download } from 'lucide-react'
import { classMediaApi } from '../services/api'
import { useSchool } from '../services/SchoolContext'
import { formatDate } from '../utils/date'
import './ClassGallery.css'

const DOC_ACCEPT = '.pdf,.doc,.docx,.xls,.xlsx,.ppt,.pptx,.txt,.jpg,.jpeg,.png'
const PHOTO_ACCEPT = 'image/*,.heic,.heif'
const today = () => new Date().toISOString().slice(0, 10)
const sizeLabel = (b) => (b >= 1 << 20 ? `${(b / (1 << 20)).toFixed(1)} MB` : `${Math.max(1, Math.round(b / 1024))} KB`)
const sectionLabel = (s) => `${s.grade_name} ${s.section_name}${s.is_current_year ? '' : ` (${s.academic_year})`}`

// Class gallery (event photos) and class documents. Teachers upload for
// their own class; admins for any class; parents see their child's class.
function ClassGallery() {
  const { user } = useOutletContext() || {}
  const { currentSchool } = useSchool()
  const isParent = user?.role === 'parent'
  const [sections, setSections] = useState([])
  const [storageReady, setStorageReady] = useState(true)
  const [sectionId, setSectionId] = useState('')
  const [tab, setTab] = useState('photo')
  const [items, setItems] = useState([])
  const [canUpload, setCanUpload] = useState(false)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [viewer, setViewer] = useState(-1) // index into photos, -1 = closed

  useEffect(() => {
    if (user?.role === 'super_admin' && !currentSchool) return
    classMediaApi.sections(user?.role === 'super_admin' ? currentSchool?.id : undefined)
      .then(res => {
        setSections(res.items || [])
        setStorageReady(res.storage_ready !== false)
        setSectionId(prev => prev && res.items.some(s => s.id === prev) ? prev : (res.items[0]?.id || ''))
      })
      .catch(e => setError(e.message))
  }, [user?.role, currentSchool])

  function load() {
    if (!sectionId) { setItems([]); return }
    setLoading(true); setError('')
    classMediaApi.list(sectionId, tab)
      .then(res => { setItems(res.items || []); setCanUpload(!!res.can_upload) })
      .catch(e => { setItems([]); setError(e.message) })
      .finally(() => setLoading(false))
  }
  useEffect(load, [sectionId, tab])

  // Group by event (newest first, as the server orders them).
  const groups = useMemo(() => {
    const out = []
    for (const it of items) {
      const key = `${it.event}|${it.event_date || ''}`
      let g = out.find(x => x.key === key)
      if (!g) { g = { key, event: it.event, date: it.event_date, items: [] }; out.push(g) }
      g.items.push(it)
    }
    return out
  }, [items])
  const photos = tab === 'photo' ? items : []

  async function remove(it) {
    if (!window.confirm(`Delete ${it.file_name}?`)) return
    try { await classMediaApi.remove(it.id); load() } catch (e) { setError(e.message) }
  }

  return (
    <div className="cg">
      <div className="page-header">
        <div>
          <h1>Class Gallery</h1>
          <p className="cg__subtitle">{isParent ? "Photos and documents from your child's class" : 'Event photos and documents for each class'}</p>
        </div>
      </div>

      {!storageReady && <p className="doc-msg doc-msg--error">File storage is not set up on the server yet.</p>}

      {sections.length === 0 ? (
        <p className="empty-text">
          {isParent ? 'No class found for your child yet.' : user?.role === 'teacher' ? 'You are not the class teacher of any class yet.' : 'No classes yet.'}
        </p>
      ) : (
        <>
          <div className="cg__bar">
            <label className="form-field cg__class">
              <span>Class</span>
              <select value={sectionId} onChange={e => setSectionId(e.target.value)}>
                {sections.map(s => (
                  <option key={s.id} value={s.id}>
                    {sectionLabel(s)}{s.children ? ` · ${s.children}` : ''} ({s.photo_count} photos, {s.document_count} docs)
                  </option>
                ))}
              </select>
            </label>
            <div className="docs-tabs cg__tabs">
              <button className={`docs-tab ${tab === 'photo' ? 'docs-tab--active' : ''}`} onClick={() => setTab('photo')}><Images size={16} /> Photos</button>
              <button className={`docs-tab ${tab === 'document' ? 'docs-tab--active' : ''}`} onClick={() => setTab('document')}><FileText size={16} /> Documents</button>
            </div>
          </div>

          {canUpload && storageReady && <Uploader sectionId={sectionId} kind={tab} onDone={load} />}

          {error && <p className="doc-msg doc-msg--error">{error}</p>}
          {loading && items.length === 0 && <p className="loading-text">Loading...</p>}
          {!loading && items.length === 0 && !error && (
            <p className="empty-text">{tab === 'photo' ? 'No photos yet.' : 'No documents yet.'}</p>
          )}

          {groups.map(g => (
            <section key={g.key} className="cg__group">
              <h3 className="cg__event">
                {g.event || (tab === 'photo' ? 'Photos' : 'Documents')}
                {g.date && <span className="cg__date">{formatDate(g.date)}</span>}
                <span className="cg__count">{g.items.length}</span>
              </h3>
              {tab === 'photo' ? (
                <div className="cg__grid">
                  {g.items.map(it => (
                    <figure key={it.id} className="cg__tile">
                      {it.thumb_url || it.content_type !== 'image/heic' ? (
                        <button className="cg__thumb" onClick={() => setViewer(photos.indexOf(it))} title={it.file_name}>
                          <img src={it.thumb_url || it.url} alt={it.file_name} loading="lazy" />
                        </button>
                      ) : (
                        // iPhone HEIC photos have no preview in most browsers: offer the file.
                        <a className="cg__thumb cg__thumb-ph" href={it.url} target="_blank" rel="noreferrer" title={it.file_name}>
                          <Images size={22} /> HEIC photo<br />Tap to open
                        </a>
                      )}
                      {it.can_delete && (
                        <button className="cg__del" onClick={() => remove(it)} title="Delete" aria-label={`Delete ${it.file_name}`}><Trash2 size={14} /></button>
                      )}
                    </figure>
                  ))}
                </div>
              ) : (
                <ul className="cg__docs">
                  {g.items.map(it => (
                    <li key={it.id} className="cg__doc">
                      <FileText size={18} className="cg__doc-icon" />
                      <a href={it.url} target="_blank" rel="noreferrer" className="cg__doc-name">{it.file_name}</a>
                      <span className="cg__doc-meta">{sizeLabel(it.size_bytes)}{it.uploaded_by_name ? ` · ${it.uploaded_by_name}` : ''}</span>
                      <a href={it.url} target="_blank" rel="noreferrer" className="btn btn--outline btn--sm" aria-label={`Open ${it.file_name}`}><Download size={14} /></a>
                      {it.can_delete && <button className="btn btn--outline btn--sm" onClick={() => remove(it)} aria-label={`Delete ${it.file_name}`}><Trash2 size={14} /></button>}
                    </li>
                  ))}
                </ul>
              )}
            </section>
          ))}
        </>
      )}

      {viewer >= 0 && photos[viewer] && (
        <Viewer photos={photos} index={viewer} onIndex={setViewer} onClose={() => setViewer(-1)} />
      )}
    </div>
  )
}

function Uploader({ sectionId, kind, onDone }) {
  const [event, setEvent] = useState('')
  const [date, setDate] = useState(today())
  const [queue, setQueue] = useState([]) // { name, status: 'waiting'|'uploading'|'done'|error text }
  const [busy, setBusy] = useState(false)
  const input = useRef(null)

  async function upload(files) {
    if (!files.length) return
    const list = Array.from(files)
    setQueue(list.map(f => ({ name: f.name, status: 'waiting' })))
    setBusy(true)
    for (let i = 0; i < list.length; i++) {
      setQueue(q => q.map((x, j) => (j === i ? { ...x, status: 'uploading' } : x)))
      try {
        await classMediaApi.upload(sectionId, kind, list[i], event.trim(), date)
        setQueue(q => q.map((x, j) => (j === i ? { ...x, status: 'done' } : x)))
      } catch (e) {
        setQueue(q => q.map((x, j) => (j === i ? { ...x, status: e.message } : x)))
      }
    }
    setBusy(false)
    if (input.current) input.current.value = ''
    onDone()
  }

  return (
    <div className="cg__upload">
      <div className="cg__upload-row">
        <label className="form-field"><span>{kind === 'photo' ? 'Event' : 'Title / topic'}</span>
          <input value={event} onChange={e => setEvent(e.target.value)} maxLength={120} placeholder={kind === 'photo' ? 'e.g. Annual Day' : 'e.g. Timetable, Holiday homework'} /></label>
        <label className="form-field"><span>Date</span>
          <input type="date" value={date} onChange={e => setDate(e.target.value)} /></label>
        <label className={`btn btn--primary cg__pick ${busy ? 'cg__pick--busy' : ''}`}>
          <Upload size={16} /> {busy ? 'Uploading...' : kind === 'photo' ? 'Add photos' : 'Add documents'}
          <input ref={input} type="file" multiple hidden disabled={busy} accept={kind === 'photo' ? PHOTO_ACCEPT : DOC_ACCEPT} onChange={e => upload(e.target.files)} />
        </label>
      </div>
      <p className="cg__hint">{kind === 'photo' ? 'JPG, PNG, WEBP or HEIC, up to 15 MB each.' : 'PDF, Word, Excel, PowerPoint or images, up to 20 MB each.'}</p>
      {queue.length > 0 && (
        <ul className="cg__queue">
          {queue.map((q, i) => (
            <li key={i} className={`cg__q cg__q--${['waiting', 'uploading', 'done'].includes(q.status) ? q.status : 'error'}`}>
              <span className="cg__q-name">{q.name}</span>
              <span>{q.status === 'waiting' ? 'Waiting' : q.status === 'uploading' ? 'Uploading...' : q.status === 'done' ? 'Uploaded' : q.status}</span>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}

function Viewer({ photos, index, onIndex, onClose }) {
  const p = photos[index]
  useEffect(() => {
    function onKey(e) {
      if (e.key === 'Escape') onClose()
      if (e.key === 'ArrowRight' && index < photos.length - 1) onIndex(index + 1)
      if (e.key === 'ArrowLeft' && index > 0) onIndex(index - 1)
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [index, photos.length, onIndex, onClose])
  return (
    <div className="cg__viewer" role="dialog" aria-label={p.file_name} onClick={onClose}>
      <button className="cg__viewer-close" onClick={onClose} aria-label="Close"><X size={22} /></button>
      {index > 0 && <button className="cg__viewer-nav cg__viewer-nav--prev" onClick={e => { e.stopPropagation(); onIndex(index - 1) }} aria-label="Previous"><ChevronLeft size={28} /></button>}
      <img src={p.url} alt={p.file_name} onClick={e => e.stopPropagation()} />
      {index < photos.length - 1 && <button className="cg__viewer-nav cg__viewer-nav--next" onClick={e => { e.stopPropagation(); onIndex(index + 1) }} aria-label="Next"><ChevronRight size={28} /></button>}
      <div className="cg__viewer-cap" onClick={e => e.stopPropagation()}>
        {p.event || p.file_name}{p.uploaded_by_name ? ` · ${p.uploaded_by_name}` : ''} · {index + 1}/{photos.length}
        <a href={p.url} target="_blank" rel="noreferrer"><Download size={14} /> Original</a>
      </div>
    </div>
  )
}

export default ClassGallery
