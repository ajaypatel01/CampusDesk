import { useState, useEffect } from 'react'
import { Download, CreditCard } from 'lucide-react'
import { useSchool } from '../services/SchoolContext'
import { idCardsApi, studentsApi, usersApi } from '../services/api'
import './IdCards.css'

function downloadBlob(blob, filename) {
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url; a.download = filename; a.click()
  URL.revokeObjectURL(url)
}

function IdCards() {
  const { currentSchool, currentYear } = useSchool()
  const [tab, setTab] = useState('students')
  const [students, setStudents] = useState([])
  const [staff, setStaff] = useState([])
  const [selectedStudents, setSelectedStudents] = useState(new Set())
  const [selectedStaff, setSelectedStaff] = useState(new Set())
  const [busy, setBusy] = useState(false)
  const [msg, setMsg] = useState('')
  const [search, setSearch] = useState('')

  useEffect(() => {
    if (!currentSchool) return
    studentsApi.list({ school_id: currentSchool.id, limit: 500 })
      .then(r => setStudents(r.items || [])).catch(() => {})
    usersApi.list({ school_id: currentSchool.id })
      .then(r => setStaff(r.items || [])).catch(() => {})
  }, [currentSchool])

  function toggleStudent(id) {
    setSelectedStudents(prev => {
      const next = new Set(prev)
      next.has(id) ? next.delete(id) : next.add(id)
      return next
    })
  }

  function toggleStaff(id) {
    setSelectedStaff(prev => {
      const next = new Set(prev)
      next.has(id) ? next.delete(id) : next.add(id)
      return next
    })
  }

  function selectAll() {
    if (tab === 'students') {
      const filtered = filteredStudents()
      setSelectedStudents(prev => {
        const allSelected = filtered.every(s => prev.has(s.id))
        const next = new Set(prev)
        filtered.forEach(s => allSelected ? next.delete(s.id) : next.add(s.id))
        return next
      })
    } else {
      const filtered = filteredStaff()
      setSelectedStaff(prev => {
        const allSelected = filtered.every(s => prev.has(s.id))
        const next = new Set(prev)
        filtered.forEach(s => allSelected ? next.delete(s.id) : next.add(s.id))
        return next
      })
    }
  }

  function filteredStudents() {
    const q = search.toLowerCase()
    return students.filter(s =>
      !q || `${s.first_name} ${s.last_name} ${s.student_code}`.toLowerCase().includes(q)
    )
  }

  function filteredStaff() {
    const q = search.toLowerCase()
    return staff.filter(u =>
      !q || `${u.first_name} ${u.last_name} ${u.role}`.toLowerCase().includes(q)
    )
  }

  async function handleGenerate() {
    setMsg('')
    if (tab === 'students') {
      if (selectedStudents.size === 0) { setMsg('Select at least one student.'); return }
      if (!currentYear) { setMsg('Select an academic year first.'); return }
      setBusy(true)
      try {
        const blob = await idCardsApi.generateStudents({ student_ids: [...selectedStudents], academic_year_id: currentYear.id })
        downloadBlob(blob, 'student_id_cards.pdf')
        setMsg(`Generated ${selectedStudents.size} student ID cards.`)
      } catch (err) { setMsg('Error: ' + err.message) }
      setBusy(false)
    } else {
      if (selectedStaff.size === 0) { setMsg('Select at least one staff member.'); return }
      setBusy(true)
      try {
        const blob = await idCardsApi.generateTeachers({ user_ids: [...selectedStaff] })
        downloadBlob(blob, 'teacher_id_cards.pdf')
        setMsg(`Generated ${selectedStaff.size} teacher ID cards.`)
      } catch (err) { setMsg('Error: ' + err.message) }
      setBusy(false)
    }
  }

  if (!currentSchool) return <p className="empty-text">Select a school first.</p>

  const fStudents = filteredStudents()
  const fStaff = filteredStaff()

  return (
    <div className="idcards-page">
      <div className="page-header">
        <div>
          <h1>ID Card Generator</h1>
          <p className="page-subtitle">Generate credit-card size ID cards as PDF</p>
        </div>
        <button className="btn btn--primary" onClick={handleGenerate} disabled={busy}>
          <Download size={16} /> {busy ? 'Generating...' : 'Download PDF'}
        </button>
      </div>

      <div className="docs-tabs">
        {[['students','Students'], ['teachers','Teachers / Staff']].map(([key, label]) => (
          <button key={key} className={`docs-tab ${tab === key ? 'docs-tab--active' : ''}`} onClick={() => { setTab(key); setSearch(''); setMsg('') }}>
            <CreditCard size={16} /> {label}
          </button>
        ))}
      </div>

      <div className="idcards-toolbar">
        <input
          className="idcards-search"
          placeholder={tab === 'students' ? 'Search students...' : 'Search staff...'}
          value={search}
          onChange={e => setSearch(e.target.value)}
        />
        <button className="btn btn--outline btn--sm" onClick={selectAll}>
          {tab === 'students'
            ? (fStudents.every(s => selectedStudents.has(s.id)) ? 'Deselect All' : 'Select All')
            : (fStaff.every(s => selectedStaff.has(s.id)) ? 'Deselect All' : 'Select All')}
        </button>
        <span className="idcards-count">
          {tab === 'students' ? selectedStudents.size : selectedStaff.size} selected
        </span>
      </div>

      {msg && <p className={`doc-msg ${msg.startsWith('Error') ? 'doc-msg--error' : 'doc-msg--ok'}`} style={{ marginBottom: '12px' }}>{msg}</p>}

      {tab === 'students' && (
        <div className="idcards-grid">
          {fStudents.map(s => (
            <label key={s.id} className={`idcard-chip ${selectedStudents.has(s.id) ? 'idcard-chip--selected' : ''}`}>
              <input type="checkbox" checked={selectedStudents.has(s.id)} onChange={() => toggleStudent(s.id)} />
              <div className="idcard-chip__avatar">{s.first_name[0]}{s.last_name[0]}</div>
              <div className="idcard-chip__info">
                <span className="idcard-chip__name">{s.first_name} {s.last_name}</span>
                <span className="idcard-chip__sub">{s.student_code}</span>
              </div>
            </label>
          ))}
          {fStudents.length === 0 && <p className="empty-text">No students found.</p>}
        </div>
      )}

      {tab === 'teachers' && (
        <div className="idcards-grid">
          {fStaff.map(u => (
            <label key={u.id} className={`idcard-chip ${selectedStaff.has(u.id) ? 'idcard-chip--selected' : ''}`}>
              <input type="checkbox" checked={selectedStaff.has(u.id)} onChange={() => toggleStaff(u.id)} />
              <div className="idcard-chip__avatar idcard-chip__avatar--green">{u.first_name[0]}{u.last_name[0]}</div>
              <div className="idcard-chip__info">
                <span className="idcard-chip__name">{u.first_name} {u.last_name}</span>
                <span className="idcard-chip__sub">{u.role}</span>
              </div>
            </label>
          ))}
          {fStaff.length === 0 && <p className="empty-text">No staff found.</p>}
        </div>
      )}
    </div>
  )
}

export default IdCards
