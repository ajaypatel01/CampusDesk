import { useState, useEffect, Fragment } from 'react'
import { useOutletContext } from 'react-router-dom'
import { Plus, Trash2, Download, BookOpen, ClipboardList, BarChart2, GraduationCap, Pencil, RotateCcw, Table } from 'lucide-react'
import { useSchool } from '../services/SchoolContext'
import { resultsApi, academicApi, studentsApi } from '../services/api'
import CustomFieldsSection from '../components/CustomFieldsSection'
import ExamMarkFormats from '../components/ExamMarkFormats'
import ResultDashboard from '../components/ResultDashboard'
import './Results.css'
import { formatDate } from '../utils/date'

function downloadBlob(blob, filename) {
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url; a.download = filename; a.click()
  URL.revokeObjectURL(url)
}

const ALL_TABS = [['subjects','Subjects', BookOpen], ['exams','Exams', ClipboardList], ['marks','Enter Marks', Plus], ['marksheet','Marksheet', BarChart2], ['report-card','Report Card', GraduationCap], ['result-sheet','Result Dashboard', Table]]
// Teachers enter marks, view marksheets and see their own class's results
// dashboard; subjects, exams and report cards are managed by admins.
const TEACHER_TABS = ['marks', 'marksheet', 'result-sheet']
// The results dashboard (one class, one exam) is for the owner, the school's
// admins, registrars, and class teachers (their own section only).
const RESULT_SHEET_ROLES = ['super_admin', 'school_admin', 'registrar', 'teacher']
// Grading-only (co-scholastic) subjects get one of these instead of marks.
const GRADE_LETTERS = ['A', 'B', 'C', 'D']

function Results() {
  const { user } = useOutletContext() || {}
  const isTeacher = user?.role === 'teacher'
  const isSuperAdmin = user?.role === 'super_admin'
  const canSeeResultSheet = RESULT_SHEET_ROLES.includes(user?.role)
  const tabs = isTeacher
    ? ALL_TABS.filter(([key]) => TEACHER_TABS.includes(key))
    : ALL_TABS.filter(([key]) => key !== 'result-sheet' || canSeeResultSheet)
  const { currentSchool, currentYear } = useSchool()
  const [tab, setTab] = useState(isTeacher ? 'marks' : 'subjects')
  const [grades, setGrades] = useState([])
  const [selectedGrade, setSelectedGrade] = useState('')
  // Grade level ids of the class section(s) this teacher is homeroom teacher of —
  // a class teacher only sees results for their own class, not the whole school.
  const [myGradeIds, setMyGradeIds] = useState(null)
  // The teacher's own homeroom sections, so student pickers only list students
  // the backend will let them save marks for (same section, not just same grade).
  const [mySections, setMySections] = useState([])

  // Subjects
  const [subjects, setSubjects] = useState([])
  const [showSubjectForm, setShowSubjectForm] = useState(false)
  const [subjectForm, setSubjectForm] = useState({ name: '', code: '', max_marks: 100, passing_marks: 33, sort_order: 0, is_co_scholastic: false })
  // The subject being edited: its id and a copy of its fields.
  const [editingSubject, setEditingSubject] = useState(null)
  const [subjectSaving, setSubjectSaving] = useState(false)

  // Exams
  const [exams, setExams] = useState([])
  const [showExamForm, setShowExamForm] = useState(false)
  const [examForm, setExamForm] = useState({ name: '', exam_date: '', weight_percent: 100 })
  const [publishingId, setPublishingId] = useState(null)
  const [formatExamId, setFormatExamId] = useState(null) // exam whose marks format is open
  const [editingExam, setEditingExam] = useState(null) // { id, name, exam_date, weight_percent, fee_lock_enabled, fee_lock_min_due } while a row is edited
  // Publishing asks whether to hide results from parents with fees due.
  const [publishDialog, setPublishDialog] = useState(null) // { exam, fee_lock_enabled, fee_lock_min_due }
  const [examSaving, setExamSaving] = useState(false)

  // Marks
  const [selectedExamId, setSelectedExamId] = useState('')
  const [selectedStudentId, setSelectedStudentId] = useState('')
  const [students, setStudents] = useState([])
  const [markSubjects, setMarkSubjects] = useState([])
  const [marks, setMarks] = useState({}) // subjectId → { marks_obtained, is_absent, grade_letter, components }
  // Subjects changed since the saved marks were loaded -- only these are saved,
  // so saving one subject can never overwrite the others.
  const [touched, setTouched] = useState(() => new Set())
  const [marksLoading, setMarksLoading] = useState(false)
  const [markSaving, setMarkSaving] = useState(false)
  const [markMsg, setMarkMsg] = useState('')
  // The selected exam's fields per subject (its own format, or the subject's fields).
  const [examFormats, setExamFormats] = useState({})
  const [addingComponentFor, setAddingComponentFor] = useState(null) // subject id or null
  const [newComponentForm, setNewComponentForm] = useState({ label: '', max_marks: '' })
  const [addingComponentSaving, setAddingComponentSaving] = useState(false)
  // { subjectId, key, label, max_marks } while a field is being edited
  const [editingField, setEditingField] = useState(null)
  const [editingFieldSaving, setEditingFieldSaving] = useState(false)

  // Marksheet
  const [msExamId, setMsExamId] = useState('')
  const [msStudentId, setMsStudentId] = useState('')
  const [marksheet, setMarksheet] = useState(null)
  const [msLoading, setMsLoading] = useState(false)
  const [editingTotal, setEditingTotal] = useState(false)
  const [totalInput, setTotalInput] = useState('')
  const [totalSaving, setTotalSaving] = useState(false)
  const [totalMsg, setTotalMsg] = useState('')

  // Report card (combined multi-exam, class-wise template)
  const [rcStudentId, setRcStudentId] = useState('')
  const [reportCard, setReportCard] = useState(null)
  const [rcLoading, setRcLoading] = useState(false)
  const [rcError, setRcError] = useState('')
  const [rcDetailsForm, setRcDetailsForm] = useState({ roll_no: '', attendance: '', remark: '', promoted_to: '', moral_remark: '', gk_remark: '' })
  const [rcSaving, setRcSaving] = useState(false)
  const [rcMsg, setRcMsg] = useState('')
  const [disciplineCriteria, setDisciplineCriteria] = useState([])
  const [disciplineGrades, setDisciplineGrades] = useState({})
  const [rcDesign, setRcDesign] = useState('classic')

  useEffect(() => {
    resultsApi.listDisciplineCriteria().then(r => setDisciplineCriteria(r.items || [])).catch(() => {})
  }, [])

  useEffect(() => {
    if (!currentSchool) return
    if (isTeacher) {
      // A class teacher only sees results for the grade(s) of their own homeroom section(s).
      if (!currentYear) return
      Promise.all([
        academicApi.listGrades(currentSchool.id),
        academicApi.listSections({ school_id: currentSchool.id, academic_year_id: currentYear.id }),
      ]).then(([gradeRes, sectionRes]) => {
        const allGrades = gradeRes.items || []
        // Class teacher or a vice class teacher of the section -- same access.
        const ownSections = (sectionRes.items || []).filter(s => s.homeroom_teacher_id === user.id || (s.vice_teacher_ids || []).includes(user.id))
        const myGrades = new Set(ownSections.map(s => s.grade_level_id))
        // Strictly their own class section(s) only, same as the backend now
        // enforces -- a teacher not yet assigned as any section's class
        // teacher sees no grades at all (an empty grade list below prompts
        // them to ask an admin to assign one), not every class in the school.
        const g = allGrades.filter(gr => myGrades.has(gr.id))
        setMyGradeIds(myGrades)
        setMySections(ownSections)
        setGrades(g)
        setSelectedGrade(prev => (prev && g.some(gr => gr.id === prev)) ? prev : (g[0]?.id || ''))
      }).catch(() => { setGrades([]); setMyGradeIds(new Set()); setMySections([]) })
      return
    }
    academicApi.listGrades(currentSchool.id)
      .then(r => { const g = r.items || []; setGrades(g); if (g.length) setSelectedGrade(g[0].id) })
      .catch(() => {})
  }, [currentSchool, currentYear, isTeacher, user?.id])

  useEffect(() => {
    if (!currentSchool || !currentYear || !selectedGrade) return
    resultsApi.listSubjects({ school_id: currentSchool.id, grade_level_id: selectedGrade })
      .then(r => setSubjects(r.items || [])).catch(() => {})
    resultsApi.listExams({ school_id: currentSchool.id, academic_year_id: currentYear.id, grade_level_id: selectedGrade })
      .then(r => setExams(r.items || [])).catch(() => {})
    const gradeName = grades.find(g => g.id === selectedGrade)?.name
    const studentParams = { school_id: currentSchool.id, limit: 500 }
    // Scope the student picker to the selected grade -- for every role, not
    // just teachers. This was teacher-only before, so an admin/registrar
    // picking e.g. Nursery still saw all 352 students in the school in the
    // Enter Marks/Marksheet/Report Card dropdowns instead of just Nursery's.
    if (gradeName) {
      studentParams.academic_year_id = currentYear.id
      studentParams.grade_level = gradeName
    }
    if (isTeacher) {
      const sectionIds = mySections.filter(s => s.grade_level_id === selectedGrade).map(s => s.id)
      if (!sectionIds.length) { setStudents([]); return }
      studentParams.academic_year_id = currentYear.id
      studentParams.class_section_ids = sectionIds.join(',')
    }
    studentsApi.list(studentParams)
      .then(r => setStudents(r.items || [])).catch(() => {})
  }, [currentSchool, currentYear, selectedGrade, grades, isTeacher, mySections])

  useEffect(() => {
    if (!selectedGrade) return
    resultsApi.listSubjects({ school_id: currentSchool?.id, grade_level_id: selectedGrade })
      .then(r => setMarkSubjects(r.items || [])).catch(() => {})
  }, [selectedGrade, currentSchool])

  async function handleAddSubject(e) {
    e.preventDefault()
    try {
      await resultsApi.createSubject({ school_id: currentSchool.id, grade_level_id: selectedGrade, ...subjectForm })
      setShowSubjectForm(false)
      setSubjectForm({ name: '', code: '', max_marks: 100, passing_marks: 33, sort_order: 0, is_co_scholastic: false })
      reloadSubjects()
    } catch (err) { alert(err.message) }
  }

  async function handleSaveSubject(e) {
    e.preventDefault()
    const sub = editingSubject
    setSubjectSaving(true)
    try {
      await resultsApi.updateSubject(sub.id, {
        name: sub.name.trim(), code: sub.code || '',
        max_marks: parseInt(sub.max_marks, 10) || 0, passing_marks: parseInt(sub.passing_marks, 10) || 0,
        sort_order: parseInt(sub.sort_order, 10) || 0, is_co_scholastic: !!sub.is_co_scholastic,
      })
      setEditingSubject(null)
      reloadSubjects()
    } catch (err) {
      alert(err.message)
    } finally {
      setSubjectSaving(false)
    }
  }

  async function handleDeleteSubject(id) {
    if (!confirm('Delete this subject?')) return
    try {
      await resultsApi.deleteSubject(id)
      setSubjects(prev => prev.filter(s => s.id !== id))
    } catch (err) { alert(err.message) }
  }

  async function handleAddExam(e) {
    e.preventDefault()
    try {
      await resultsApi.createExam({
        school_id: currentSchool.id, academic_year_id: currentYear.id, grade_level_id: selectedGrade,
        name: examForm.name, exam_date: examForm.exam_date || undefined,
        weight_percent: parseInt(examForm.weight_percent, 10) || 100,
      })
      setShowExamForm(false)
      setExamForm({ name: '', exam_date: '', weight_percent: 100 })
      resultsApi.listExams({ school_id: currentSchool.id, academic_year_id: currentYear.id, grade_level_id: selectedGrade })
        .then(r => setExams(r.items || []))
    } catch (err) { alert(err.message) }
  }

  function reloadExams() {
    return resultsApi.listExams({ school_id: currentSchool.id, academic_year_id: currentYear.id, grade_level_id: selectedGrade })
      .then(r => setExams(r.items || []))
  }

  function startEditExam(e) {
    setEditingExam({
      id: e.id,
      name: e.name,
      exam_date: e.exam_date ? e.exam_date.slice(0, 10) : '',
      weight_percent: String(e.weight_percent),
      fee_lock_enabled: !!e.fee_lock_enabled,
      fee_lock_min_due: String(e.fee_lock_min_due || 0),
    })
  }

  async function handleSaveExam() {
    if (!editingExam.name.trim()) return
    setExamSaving(true)
    try {
      await resultsApi.updateExam(editingExam.id, {
        name: editingExam.name.trim(),
        exam_date: editingExam.exam_date || '',
        weight_percent: parseInt(editingExam.weight_percent, 10) || 0,
        fee_lock_enabled: editingExam.fee_lock_enabled,
        fee_lock_min_due: parseInt(editingExam.fee_lock_min_due, 10) || 0,
      })
      setEditingExam(null)
      await reloadExams()
    } catch (err) {
      alert(err.message)
    } finally {
      setExamSaving(false)
    }
  }

  // Deleting an exam deletes its marks too, so the backend refuses it while
  // the exam is published or has marks entered -- that reason is shown as is.
  async function handleDeleteExam(e) {
    if (!confirm(`Delete "${e.name}"? This can't be undone.`)) return
    try {
      await resultsApi.deleteExam(e.id)
      if (formatExamId === e.id) setFormatExamId(null)
      if (selectedExamId === e.id) pickMarksExam('')
      if (msExamId === e.id) { setMsExamId(''); setMarksheet(null) }
      await reloadExams()
    } catch (err) {
      alert(err.message)
    }
  }

  // A published exam's marks become visible to parents/on report cards; an
  // admin un-publishing one hides them again without deleting anything, same
  // toggle either direction.
  async function handleTogglePublish(exam) {
    if (!exam.is_published) {
      setPublishDialog({ exam, fee_lock_enabled: !!exam.fee_lock_enabled, fee_lock_min_due: String(exam.fee_lock_min_due || 0) })
      return
    }
    await setPublished(exam, false)
  }

  async function setPublished(exam, publish, extra = {}) {
    setPublishingId(exam.id)
    try {
      await resultsApi.publishExam(exam.id, publish, extra)
      setPublishDialog(null)
      const res = await resultsApi.listExams({ school_id: currentSchool.id, academic_year_id: currentYear.id, grade_level_id: selectedGrade })
      setExams(res.items || [])
    } catch (err) {
      alert(err.message)
    } finally {
      setPublishingId(null)
    }
  }

  // The selected exam's component scheme (Written/Note Book/.../Theory), if
  // Each subject carries its own effective mark-component scheme ("sections"
  // -- Oral/Unit Test/Activity/Practical/Written/...): its own custom one if
  // it has one, else its grade-template default -- straight off the subject
  // payload so this page never hardcodes a second copy of it. Different
  // subjects can have different sections, unlike the exam-wide scheme this
  // used to share.
  function componentTotal(sub) {
    const vals = marks[sub.id]?.components || {}
    return fieldsFor(sub).reduce((sum, c) => sum + (parseFloat(vals[c.key]) || 0), 0)
  }

  // Sum of a subject's own field max_marks -- compared against the subject's
  // overall max_marks so a mismatch (fields adding up to less/more than the
  // subject is actually graded out of) is visible instead of silently wrong.
  function componentsMaxSum(sub) {
    return (sub.mark_components || []).reduce((sum, c) => sum + c.max_marks, 0)
  }

  // Subjects and Enter Marks both list the same subjects (with the same
  // mark_components), just rendered differently -- one reload keeps both in
  // sync after a field is added/removed from either tab.
  function reloadSubjects() {
    if (!selectedGrade) return
    resultsApi.listSubjects({ school_id: currentSchool?.id, grade_level_id: selectedGrade })
      .then(r => { const items = r.items || []; setSubjects(items); setMarkSubjects(items) }).catch(() => {})
  }

  async function handleAddComponent(sub) {
    if (!newComponentForm.label.trim() || !newComponentForm.max_marks) return
    setAddingComponentSaving(true)
    try {
      await resultsApi.addSubjectComponent(sub.id, {
        label: newComponentForm.label.trim(),
        max_marks: parseInt(newComponentForm.max_marks, 10),
      })
      setAddingComponentFor(null)
      setNewComponentForm({ label: '', max_marks: '' })
      reloadSubjects()
    } catch (err) {
      alert(err.message)
    } finally {
      setAddingComponentSaving(false)
    }
  }

  async function handleSaveField() {
    const maxMarks = parseInt(editingField.max_marks, 10)
    if (!editingField.label.trim() || !maxMarks) return
    setEditingFieldSaving(true)
    try {
      await resultsApi.updateSubjectComponent(editingField.subjectId, editingField.key, {
        label: editingField.label.trim(),
        max_marks: maxMarks,
      })
      setEditingField(null)
      reloadSubjects()
    } catch (err) {
      alert(err.message)
    } finally {
      setEditingFieldSaving(false)
    }
  }

  // Component add/delete is subject setup: the backend blocks teachers and
  // parents, and the UI doesn't offer it to teachers -- they only enter marks.
  async function handleDeleteComponent(subjectId, key) {
    if (!confirm('Remove this field?')) return
    try {
      await resultsApi.deleteSubjectComponent(subjectId, key)
      reloadSubjects()
    } catch (err) { alert(err.message) }
  }

  // Exams and students belong to a grade, so switching grade clears every
  // selection and result made for the old one (same as the mobile app).
  function pickGrade(id) {
    if (id === selectedGrade) return
    setSelectedGrade(id)
    setSelectedExamId(''); setSelectedStudentId(''); setMarks({}); setMarkMsg('')
    setMsExamId(''); setMsStudentId(''); setMarksheet(null); setEditingTotal(false); setTotalMsg('')
    setRcStudentId(''); setReportCard(null); setRcError(''); setRcMsg('')
    setAddingComponentFor(null); setEditingField(null); setFormatExamId(null); setEditingExam(null)
  }

  // Clear the form whenever the exam or student changes, so one student's
  // numbers can never be saved against the next; the effect below then
  // fills in what is already saved for the new pair.
  function pickMarksExam(id) {
    setSelectedExamId(id); setMarks({}); setTouched(new Set()); setMarkMsg('')
  }
  function pickMarksStudent(id) {
    setSelectedStudentId(id); setMarks({}); setTouched(new Set()); setMarkMsg('')
  }

  useEffect(() => {
    if (!selectedExamId || !selectedStudentId) return
    let cancelled = false
    setMarksLoading(true)
    resultsApi.studentExamMarks(selectedExamId, selectedStudentId)
      .then(res => {
        if (cancelled) return
        const loaded = {}
        for (const m of res.items || []) {
          loaded[m.subject_id] = {
            is_absent: m.is_absent,
            grade_letter: m.grade_letter || '',
            marks_obtained: m.is_absent || m.grade_letter ? '' : String(m.marks_obtained),
            components: Object.fromEntries(Object.entries(m.components || {}).map(([k, v]) => [k, String(v)])),
          }
        }
        setMarks(loaded)
        setTouched(new Set())
      })
      .catch(err => { if (!cancelled) setMarkMsg('Error: could not load saved marks (' + err.message + '). Reload before saving.') })
      .finally(() => { if (!cancelled) setMarksLoading(false) })
    return () => { cancelled = true }
  }, [selectedExamId, selectedStudentId])

  // Changes one subject's entry and remembers that it changed.
  function editMark(subjectId, change) {
    setMarks(prev => ({ ...prev, [subjectId]: change(prev[subjectId] || {}) }))
    setTouched(prev => new Set(prev).add(subjectId))
  }

  function loadExamFormats(examId) {
    if (!examId) { setExamFormats({}); return }
    resultsApi.listExamFormats(examId)
      .then(r => setExamFormats(Object.fromEntries((r.items || []).map(f => [f.subject_id, f.components]))))
      .catch(() => setExamFormats({}))
  }

  useEffect(() => { loadExamFormats(selectedExamId) }, [selectedExamId])

  // Fields a subject is marked on in the selected exam.
  function fieldsFor(sub) {
    return examFormats[sub.id] || sub.mark_components || []
  }

  async function handleSaveMarks(e) {
    e.preventDefault()
    if (!selectedExamId || !selectedStudentId) return
    setMarkSaving(true); setMarkMsg('')
    try {
      // Only subjects changed in this form are saved; the rest keep what is
      // already stored. A grading-only subject with no grade picked is left
      // out, so it is never saved as 0 marks.
      const marksArr = markSubjects.filter(sub => {
        if (!touched.has(sub.id)) return false
        const entry = marks[sub.id] || {}
        return !sub.is_co_scholastic || entry.is_absent || entry.grade_letter
      }).map(sub => {
        const entry = marks[sub.id] || {}
        const base = {
          exam_id: selectedExamId,
          student_id: selectedStudentId,
          subject_id: sub.id,
          max_marks: sub.max_marks,
          is_absent: entry.is_absent || false,
          remarks: '',
        }
        if (sub.is_co_scholastic) {
          return { ...base, grade_letter: entry.is_absent ? '' : entry.grade_letter }
        }
        const subComponents = fieldsFor(sub)
        if (subComponents.length > 0) {
          const components = {}
          subComponents.forEach(c => { components[c.key] = parseFloat(entry.components?.[c.key] || 0) })
          return { ...base, components }
        }
        return { ...base, marks_obtained: parseFloat(entry.marks_obtained || 0) }
      })
      if (marksArr.length === 0) {
        setMarkMsg('Nothing changed -- no marks to save.')
        setMarkSaving(false)
        return
      }
      await resultsApi.bulkUpsertMarks(marksArr)
      setTouched(new Set())
      setMarkMsg(`Saved ${marksArr.length} subject${marksArr.length === 1 ? '' : 's'}.`)
    } catch (err) { setMarkMsg('Error: ' + err.message) }
    setMarkSaving(false)
  }

  async function loadMarksheet() {
    if (!msExamId || !msStudentId) return
    setMsLoading(true); setMarksheet(null)
    setEditingTotal(false); setTotalMsg('')
    try {
      const ms = await resultsApi.getMarksheet(msExamId, msStudentId)
      setMarksheet(ms)
    } catch (err) { alert(err.message) }
    setMsLoading(false)
  }

  function startEditTotal() {
    setTotalInput(String(marksheet.total_obtained))
    setTotalMsg('')
    setEditingTotal(true)
  }

  async function saveTotalOverride() {
    const value = parseFloat(totalInput)
    if (Number.isNaN(value)) { setTotalMsg('Enter a valid number'); return }
    setTotalSaving(true); setTotalMsg('')
    try {
      const ms = await resultsApi.setTotalOverride(msExamId, msStudentId, value)
      setMarksheet(ms)
      setEditingTotal(false)
    } catch (err) { setTotalMsg(err.message) }
    setTotalSaving(false)
  }

  async function resetTotalOverride() {
    if (!confirm('Reset this total back to the auto-calculated value?')) return
    setTotalSaving(true); setTotalMsg('')
    try {
      const ms = await resultsApi.clearTotalOverride(msExamId, msStudentId)
      setMarksheet(ms)
    } catch (err) { setTotalMsg(err.message) }
    setTotalSaving(false)
  }

  async function downloadMarksheet() {
    if (!msExamId || !msStudentId) return
    try {
      const blob = await resultsApi.downloadMarksheet(msExamId, msStudentId)
      downloadBlob(blob, `marksheet.pdf`)
    } catch (err) { alert(err.message) }
  }

  async function loadReportCard() {
    if (!rcStudentId || !currentYear) return
    setRcLoading(true); setReportCard(null); setRcError(''); setRcMsg('')
    try {
      const rc = await resultsApi.getReportCard(rcStudentId, currentYear.id)
      // The backend returns null (not []) for exams/subjects when there's no
      // published exam data yet for this student -- normalize here so the
      // table below can always safely .map() over them instead of crashing
      // the whole page with an uncaught TypeError.
      setReportCard({
        ...rc,
        exams: rc.exams || [],
        subjects: (rc.subjects || []).map(sub => ({ ...sub, by_exam: sub.by_exam || [] })),
      })
      setRcDetailsForm({
        roll_no: rc.details?.roll_no || '',
        attendance: rc.details?.attendance || '',
        remark: rc.details?.remark || '',
        promoted_to: rc.details?.promoted_to || '',
        moral_remark: rc.details?.moral_remark || '',
        gk_remark: rc.details?.gk_remark || '',
      })
      const g = {}
      ;(rc.discipline_grades || []).forEach(dg => { g[dg.criterion_key] = dg.grade })
      setDisciplineGrades(g)
    } catch (err) { setRcError(err.message) }
    setRcLoading(false)
  }

  async function downloadReportCardPDF() {
    if (!rcStudentId || !currentYear) return
    try {
      const blob = await resultsApi.downloadReportCard(rcStudentId, currentYear.id, rcDesign)
      downloadBlob(blob, `report_card.pdf`)
    } catch (err) { alert(err.message) }
  }

  async function handleSaveReportCardDetails(e) {
    e.preventDefault()
    if (!reportCard) return
    setRcSaving(true); setRcMsg('')
    try {
      await resultsApi.upsertReportCardDetails({
        school_id: currentSchool.id,
        academic_year_id: currentYear.id,
        grade_level_id: reportCard.grade_level_id,
        student_id: rcStudentId,
        ...rcDetailsForm,
      })
      setRcMsg('Saved.')
    } catch (err) { setRcMsg('Error: ' + err.message) }
    setRcSaving(false)
  }

  async function handleSaveDisciplineGrades() {
    if (!reportCard) return
    setRcMsg('')
    try {
      await resultsApi.upsertDisciplineGrades({
        school_id: currentSchool.id,
        academic_year_id: currentYear.id,
        student_id: rcStudentId,
        grades: disciplineGrades,
      })
      setRcMsg('Discipline grades saved.')
    } catch (err) { setRcMsg('Error: ' + err.message) }
  }

  if (!currentSchool || !currentYear) return <p className="empty-text">Select a school and academic year first.</p>

  if (isTeacher && myGradeIds !== null && myGradeIds.size === 0) {
    return <p className="empty-text">You haven&apos;t been assigned as a class teacher yet. Ask your school admin to assign you to a class section under Settings → Grades &amp; Sections.</p>
  }

  return (
    <div className="results-page">
      <div className="page-header">
        <div>
          <h1>Results & Marksheets</h1>
          <p className="page-subtitle">Manage subjects, exams, marks, and download marksheets</p>
        </div>
      </div>

      <div className="results-grade-bar">
        <span className="results-grade-label">Grade:</span>
        {grades.map(g => (
          <button key={g.id} className={`grade-chip ${selectedGrade === g.id ? 'grade-chip--active' : ''}`} onClick={() => pickGrade(g.id)}>
            {g.name}
          </button>
        ))}
      </div>

      <div className="docs-tabs">
        {tabs.map(([key, label, Icon]) => (
          <button key={key} className={`docs-tab ${tab === key ? 'docs-tab--active' : ''}`} onClick={() => setTab(key)}>
            <Icon size={16} /> {label}
          </button>
        ))}
      </div>

      {/* Subjects Tab */}
      {tab === 'subjects' && !isTeacher && (
        <div className="results-section">
          <div className="results-section__header">
            <h2>Subjects for {grades.find(g => g.id === selectedGrade)?.name || '—'}</h2>
            {!isTeacher && (
              <button className="btn btn--primary" onClick={() => setShowSubjectForm(!showSubjectForm)}>
                <Plus size={16} /> Add Subject
              </button>
            )}
          </div>
          {!isTeacher && showSubjectForm && (
            <form className="results-inline-form" onSubmit={handleAddSubject}>
              <div className="form-row">
                <label className="form-field"><span>Name *</span><input required value={subjectForm.name} onChange={e => setSubjectForm({ ...subjectForm, name: e.target.value })} placeholder="e.g. Mathematics" /></label>
                <label className="form-field"><span>Code</span><input value={subjectForm.code} onChange={e => setSubjectForm({ ...subjectForm, code: e.target.value })} placeholder="MATH" /></label>
              </div>
              <div className="form-row">
                <label className="form-field"><span>Max Marks</span><input type="number" min="1" value={subjectForm.max_marks} onChange={e => setSubjectForm({ ...subjectForm, max_marks: parseInt(e.target.value) })} /></label>
                <label className="form-field"><span>Passing Marks</span><input type="number" min="1" value={subjectForm.passing_marks} onChange={e => setSubjectForm({ ...subjectForm, passing_marks: parseInt(e.target.value) })} /></label>
                <label className="form-field"><span>Sort Order</span><input type="number" min="0" value={subjectForm.sort_order} onChange={e => setSubjectForm({ ...subjectForm, sort_order: parseInt(e.target.value) })} /></label>
              </div>
              <div className="form-row">
                <label className="form-field--checkbox">
                  <input type="checkbox" checked={subjectForm.is_co_scholastic} onChange={e => setSubjectForm({ ...subjectForm, is_co_scholastic: e.target.checked })} />
                  <span>Co-Scholastic / Grading subject (graded, but not counted in the overall total)</span>
                </label>
              </div>
              <div style={{ display: 'flex', gap: '8px', flexWrap: 'wrap' }}>
                <button type="submit" className="btn btn--primary">Save</button>
                <button type="button" className="btn btn--outline" onClick={() => setShowSubjectForm(false)}>Cancel</button>
              </div>
            </form>
          )}
          {subjects.length === 0 ? (
            <p className="empty-text">No subjects yet</p>
          ) : (
            <div className="subject-cards">
              {subjects.map(s => {
                const components = s.mark_components || []
                const maxSum = componentsMaxSum(s)
                const mismatch = components.length > 0 && maxSum !== s.max_marks
                return (
                  <div key={s.id} className="subject-card">
                    <div className="subject-card__header">
                      <div className="subject-card__title">
                        <strong>{s.name}</strong>
                        {s.code && <span className="data-table__muted"> ({s.code})</span>}
                        {s.is_co_scholastic && <span className="badge badge--muted" style={{ marginLeft: '8px' }}>Grading only</span>}
                      </div>
                      <div className="subject-card__meta">
                        <span>Max <strong>{s.max_marks}</strong></span>
                        <span>Pass <strong>{s.passing_marks}</strong></span>
                        {!isTeacher && (
                          <button
                            className="btn btn--outline btn--sm" title="Edit subject" aria-label={`Edit ${s.name}`}
                            onClick={() => setEditingSubject({ id: s.id, name: s.name, code: s.code || '', max_marks: String(s.max_marks), passing_marks: String(s.passing_marks), sort_order: String(s.sort_order || 0), is_co_scholastic: !!s.is_co_scholastic })}
                          >
                            <Pencil size={14} />
                          </button>
                        )}
                        {!isTeacher && (
                          <button className="btn btn--outline btn--sm" onClick={() => handleDeleteSubject(s.id)} title="Delete subject">
                            <Trash2 size={14} />
                          </button>
                        )}
                      </div>
                    </div>

                    {editingSubject?.id === s.id && (
                      <form className="subject-form" onSubmit={handleSaveSubject} style={{ marginBottom: '12px' }}>
                        <div className="form-row">
                          <label className="form-field"><span>Name *</span><input required value={editingSubject.name} onChange={e => setEditingSubject({ ...editingSubject, name: e.target.value })} /></label>
                          <label className="form-field"><span>Code</span><input value={editingSubject.code} onChange={e => setEditingSubject({ ...editingSubject, code: e.target.value })} /></label>
                        </div>
                        <div className="form-row">
                          <label className="form-field"><span>Total (max) marks *</span><input type="number" min="1" required={!editingSubject.is_co_scholastic} disabled={editingSubject.is_co_scholastic} value={editingSubject.max_marks} onChange={e => setEditingSubject({ ...editingSubject, max_marks: e.target.value })} /></label>
                          <label className="form-field"><span>Passing marks</span><input type="number" min="0" max={editingSubject.max_marks || undefined} disabled={editingSubject.is_co_scholastic} value={editingSubject.passing_marks} onChange={e => setEditingSubject({ ...editingSubject, passing_marks: e.target.value })} /></label>
                          <label className="form-field"><span>Sort order</span><input type="number" min="0" value={editingSubject.sort_order} onChange={e => setEditingSubject({ ...editingSubject, sort_order: e.target.value })} /></label>
                        </div>
                        <label className="form-field--checkbox">
                          <input type="checkbox" checked={editingSubject.is_co_scholastic} onChange={e => setEditingSubject({ ...editingSubject, is_co_scholastic: e.target.checked })} />
                          <span>Grading only (A–D instead of marks)</span>
                        </label>
                        {!editingSubject.is_co_scholastic && String(editingSubject.max_marks) !== String(s.max_marks) && (
                          <p className="doc-msg" style={{ marginTop: '6px' }}>
                            Marks already entered for {s.name} in exams that aren&apos;t published will be counted out of {editingSubject.max_marks || '?'} instead of {s.max_marks}. Published exams don&apos;t change.
                            {components.length > 0 && ' This subject has its own fields, so its total comes from the fields.'}
                          </p>
                        )}
                        {editingSubject.is_co_scholastic !== !!s.is_co_scholastic && (
                          <p className="doc-msg doc-msg--error" style={{ marginTop: '6px' }}>
                            {editingSubject.is_co_scholastic
                              ? 'Marks already entered for this subject will no longer count -- teachers will pick a grade (A–D) instead.'
                              : 'Grades already picked for this subject will no longer show -- teachers will enter marks instead.'}
                          </p>
                        )}
                        <div style={{ display: 'flex', gap: '8px', marginTop: '8px' }}>
                          <button type="submit" className="btn btn--primary btn--sm" disabled={subjectSaving}>{subjectSaving ? 'Saving...' : 'Save'}</button>
                          <button type="button" className="btn btn--outline btn--sm" onClick={() => setEditingSubject(null)}>Cancel</button>
                        </div>
                      </form>
                    )}

                    <div className="subject-card__fields">
                      <div className="subject-card__fields-label">Custom Fields</div>
                      {components.length === 0 ? (
                        <p className="empty-text" style={{ padding: 0, textAlign: 'left' }}>
                          No custom fields -- marks are entered as one number out of {s.max_marks}.
                        </p>
                      ) : (
                        <>
                          <div className="field-chips">
                            {components.map(c => (
                              <span key={c.key} className="field-chip">
                                {c.label} <span className="data-table__muted">/{c.max_marks}</span>
                                <button
                                  type="button" className="field-chip__remove" title="Edit field"
                                  onClick={() => { setAddingComponentFor(null); setEditingField({ subjectId: s.id, key: c.key, label: c.label, max_marks: String(c.max_marks) }) }}
                                >
                                  <Pencil size={11} />
                                </button>
                                <button type="button" className="field-chip__remove" onClick={() => handleDeleteComponent(s.id, c.key)} title="Remove field">×</button>
                              </span>
                            ))}
                          </div>
                          <p className={`subject-card__fields-total ${mismatch ? 'subject-card__fields-total--mismatch' : ''}`}>
                            Fields total {maxSum} / subject max {s.max_marks}
                            {mismatch && " -- these don't match, double check the field max marks"}
                          </p>
                        </>
                      )}
                      {editingField?.subjectId === s.id && (
                        <div className="marks-add-component" style={{ marginTop: '8px' }}>
                          <input placeholder="Field name" value={editingField.label} onChange={e => setEditingField({ ...editingField, label: e.target.value })} />
                          <input type="number" min="1" placeholder="Max" style={{ width: '70px' }} value={editingField.max_marks} onChange={e => setEditingField({ ...editingField, max_marks: e.target.value })} />
                          <button type="button" className="btn btn--primary btn--sm" onClick={handleSaveField} disabled={editingFieldSaving}>
                            {editingFieldSaving ? 'Saving...' : 'Save'}
                          </button>
                          <button type="button" className="btn btn--outline btn--sm" onClick={() => setEditingField(null)}>Cancel</button>
                        </div>
                      )}
                      {addingComponentFor === s.id ? (
                        <div className="marks-add-component" style={{ marginTop: '8px' }}>
                          <input placeholder="Field name (e.g. Oral)" value={newComponentForm.label} onChange={e => setNewComponentForm({ ...newComponentForm, label: e.target.value })} />
                          <input type="number" min="1" placeholder="Max" style={{ width: '70px' }} value={newComponentForm.max_marks} onChange={e => setNewComponentForm({ ...newComponentForm, max_marks: e.target.value })} />
                          <button type="button" className="btn btn--primary btn--sm" onClick={() => handleAddComponent(s)} disabled={addingComponentSaving}>
                            {addingComponentSaving ? 'Adding...' : 'Add'}
                          </button>
                          <button type="button" className="btn btn--outline btn--sm" onClick={() => setAddingComponentFor(null)}>Cancel</button>
                        </div>
                      ) : (
                        <button
                          type="button" className="btn btn--outline btn--sm" style={{ marginTop: '8px' }}
                          onClick={() => { setEditingField(null); setAddingComponentFor(s.id); setNewComponentForm({ label: '', max_marks: '' }) }}
                        >
                          <Plus size={13} /> Add Field
                        </button>
                      )}
                    </div>
                  </div>
                )
              })}
            </div>
          )}
        </div>
      )}

      {/* Exams Tab */}
      {tab === 'exams' && !isTeacher && (
        <div className="results-section">
          <div className="results-section__header">
            <h2>Exams</h2>
            {!isTeacher && (
              <button className="btn btn--primary" onClick={() => setShowExamForm(!showExamForm)}>
                <Plus size={16} /> Add Exam
              </button>
            )}
          </div>
          {!isTeacher && showExamForm && (
            <form className="results-inline-form" onSubmit={handleAddExam}>
              <div className="form-row">
                <label className="form-field"><span>Exam Name *</span><input required value={examForm.name} onChange={e => setExamForm({ ...examForm, name: e.target.value })} placeholder="e.g. Unit Test 1" /></label>
                <label className="form-field"><span>Exam Date</span><input type="date" value={examForm.exam_date} onChange={e => setExamForm({ ...examForm, exam_date: e.target.value })} /></label>
                <label className="form-field"><span>Weight %</span><input type="number" min="1" max="100" value={examForm.weight_percent} onChange={e => setExamForm({ ...examForm, weight_percent: e.target.value })} /></label>
              </div>
              <div style={{ display: 'flex', gap: '8px', flexWrap: 'wrap' }}>
                <button type="submit" className="btn btn--primary">Save</button>
                <button type="button" className="btn btn--outline" onClick={() => setShowExamForm(false)}>Cancel</button>
              </div>
            </form>
          )}
          <div className="table-card">
            <table className="data-table">
              <thead><tr><th>Exam Name</th><th>Date</th><th>Weight</th><th>Published</th><th>Fee lock</th>{!isTeacher && <th></th>}</tr></thead>
              <tbody>
                {exams.length === 0 ? (
                  <tr><td colSpan={isTeacher ? 5 : 6} className="data-table__empty">No exams yet</td></tr>
                ) : exams.map(e => editingExam?.id === e.id ? (
                  <tr key={e.id}>
                    <td><input value={editingExam.name} onChange={ev => setEditingExam({ ...editingExam, name: ev.target.value })} placeholder="Exam name" /></td>
                    <td><input type="date" value={editingExam.exam_date} onChange={ev => setEditingExam({ ...editingExam, exam_date: ev.target.value })} /></td>
                    <td><input type="number" min="1" max="100" style={{ width: '70px' }} value={editingExam.weight_percent} onChange={ev => setEditingExam({ ...editingExam, weight_percent: ev.target.value })} /></td>
                    <td><span className={`badge badge--${e.is_published ? 'success' : 'muted'}`}>{e.is_published ? 'Published' : 'Draft'}</span></td>
                    <td style={{ whiteSpace: 'nowrap' }}>
                      <label className="form-field--checkbox" style={{ margin: 0 }}>
                        <input type="checkbox" checked={editingExam.fee_lock_enabled} onChange={ev => setEditingExam({ ...editingExam, fee_lock_enabled: ev.target.checked })} />
                        <span>Lock if due &gt; ₹</span>
                        <input type="number" min="0" step="100" style={{ width: '90px' }} disabled={!editingExam.fee_lock_enabled} value={editingExam.fee_lock_min_due} onChange={ev => setEditingExam({ ...editingExam, fee_lock_min_due: ev.target.value })} aria-label="Lock when fee due is more than" />
                      </label>
                    </td>
                    <td style={{ whiteSpace: 'nowrap' }}>
                      <button className="btn btn--primary btn--sm" onClick={handleSaveExam} disabled={examSaving || !editingExam.name.trim()}>
                        {examSaving ? 'Saving...' : 'Save'}
                      </button>
                      <button className="btn btn--outline btn--sm" style={{ marginLeft: '6px' }} onClick={() => setEditingExam(null)} disabled={examSaving}>Cancel</button>
                    </td>
                  </tr>
                ) : (
                  <tr key={e.id}>
                    <td>{e.name}</td>
                    <td className="data-table__muted">{formatDate(e.exam_date)}</td>
                    <td>{e.weight_percent}%</td>
                    <td><span className={`badge badge--${e.is_published ? 'success' : 'muted'}`}>{e.is_published ? 'Published' : 'Draft'}</span></td>
                    <td className="data-table__muted">
                      {e.fee_lock_enabled
                        ? (e.fee_lock_min_due > 0 ? `Due > ₹${Number(e.fee_lock_min_due).toLocaleString('en-IN')}` : 'Any due')
                        : '—'}
                    </td>
                    {!isTeacher && (
                      <td style={{ whiteSpace: 'nowrap' }}>
                        <button className="btn btn--outline btn--sm" title="Edit exam" onClick={() => startEditExam(e)} style={{ marginRight: '6px' }}>
                          <Pencil size={13} />
                        </button>
                        <button className="btn btn--outline btn--sm" title="Delete exam" onClick={() => handleDeleteExam(e)} style={{ marginRight: '6px' }}>
                          <Trash2 size={13} />
                        </button>
                        <button
                          className="btn btn--outline btn--sm"
                          onClick={() => handleTogglePublish(e)}
                          disabled={publishingId === e.id}
                        >
                          {publishingId === e.id ? 'Saving...' : (e.is_published ? 'Unpublish' : 'Publish')}
                        </button>
                        <button
                          className={`btn btn--sm ${formatExamId === e.id ? 'btn--primary' : 'btn--outline'}`}
                          style={{ marginLeft: '6px' }}
                          onClick={() => setFormatExamId(formatExamId === e.id ? null : e.id)}
                        >
                          Marks format
                        </button>
                      </td>
                    )}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {publishDialog && (
            <div className="modal-overlay" onClick={() => setPublishDialog(null)}>
              <div className="modal" onClick={ev => ev.stopPropagation()}>
                <h2>Publish {publishDialog.exam.name}?</h2>
                <p className="empty-text">Parents will be able to see the marks recorded for this exam.</p>
                <label className="form-field--checkbox">
                  <input type="checkbox" checked={publishDialog.fee_lock_enabled} onChange={ev => setPublishDialog({ ...publishDialog, fee_lock_enabled: ev.target.checked })} />
                  <span>Hide results from parents who have fees due</span>
                </label>
                {publishDialog.fee_lock_enabled && (
                  <label className="form-field" style={{ marginTop: '8px' }}>
                    <span>Only when the due amount is more than (₹)</span>
                    <input type="number" min="0" step="100" value={publishDialog.fee_lock_min_due} onChange={ev => setPublishDialog({ ...publishDialog, fee_lock_min_due: ev.target.value })} />
                    <span className="data-table__muted" style={{ fontSize: '0.8rem' }}>
                      0 = any amount due hides the results. Parents see &ldquo;Results are locked: ₹… fee is due. Please pay at the school office&rdquo;. Results unlock as soon as the payment is recorded.
                    </span>
                  </label>
                )}
                <div className="modal__actions">
                  <button type="button" className="btn btn--outline" onClick={() => setPublishDialog(null)}>Cancel</button>
                  <button
                    type="button" className="btn btn--primary" disabled={publishingId === publishDialog.exam.id}
                    onClick={() => setPublished(publishDialog.exam, true, {
                      fee_lock_enabled: publishDialog.fee_lock_enabled,
                      fee_lock_min_due: parseInt(publishDialog.fee_lock_min_due, 10) || 0,
                    })}
                  >
                    {publishingId === publishDialog.exam.id ? 'Publishing...' : 'Publish'}
                  </button>
                </div>
              </div>
            </div>
          )}
          {formatExamId && exams.some(e => e.id === formatExamId) && (
            <div style={{ marginTop: '16px' }}>
              <div className="results-section__header">
                <h2>Marks format: {exams.find(e => e.id === formatExamId)?.name}</h2>
                <button className="btn btn--outline btn--sm" onClick={() => setFormatExamId(null)}>Close</button>
              </div>
              <ExamMarkFormats
                exam={exams.find(e => e.id === formatExamId)}
                onChanged={() => { if (formatExamId === selectedExamId) loadExamFormats(selectedExamId) }}
              />
            </div>
          )}
        </div>
      )}

      {/* Enter Marks Tab */}
      {tab === 'marks' && (
        <div className="results-section">
          <h2>Enter Marks</h2>
          <div className="form-row" style={{ marginBottom: '16px' }}>
            <label className="form-field">
              <span>Exam *</span>
              <select value={selectedExamId} onChange={e => pickMarksExam(e.target.value)}>
                <option value="">Select exam...</option>
                {exams.map(ex => <option key={ex.id} value={ex.id}>{ex.name}</option>)}
              </select>
            </label>
            <label className="form-field">
              <span>Student *</span>
              <select value={selectedStudentId} onChange={e => pickMarksStudent(e.target.value)}>
                <option value="">Select student...</option>
                {students.map(s => <option key={s.id} value={s.id}>{s.first_name} {s.last_name} ({s.student_code})</option>)}
              </select>
            </label>
          </div>
          {exams.length === 0 && (
            <p className="empty-text">
              No exams yet for this grade --{' '}
              {isTeacher
                ? 'ask your admin or registrar to add one under the Exams tab before you can enter marks.'
                : 'add one under the Exams tab above first.'}
            </p>
          )}
          {selectedExamId && selectedStudentId && markSubjects.length > 0 && marksLoading && (
            <p className="loading-text">Loading saved marks...</p>
          )}
          {selectedExamId && selectedStudentId && markSubjects.length > 0 && !marksLoading && (
            <form onSubmit={handleSaveMarks}>
              <div className="mark-subject-cards">
                {markSubjects.map(sub => {
                  const components = fieldsFor(sub)
                  const fieldsMax = components.reduce((sum, c) => sum + c.max_marks, 0)
                  const isAbsent = marks[sub.id]?.is_absent || false
                  return (
                    <div key={sub.id} className="mark-subject-card">
                      <div className="mark-subject-card__header">
                        <span className="mark-subject-card__title">{sub.name}</span>
                        <label className="mark-subject-card__absent">
                          <input
                            type="checkbox"
                            checked={isAbsent}
                            onChange={e => editMark(sub.id, entry => ({ ...entry, is_absent: e.target.checked }))}
                          />
                          Absent
                        </label>
                      </div>

                      {sub.is_co_scholastic ? (
                        <label className="marks-component-field">
                          <span>Grade <span className="data-table__muted">(grading only)</span></span>
                          <select
                            className="marks-input"
                            disabled={isAbsent}
                            value={marks[sub.id]?.grade_letter || ''}
                            onChange={e => editMark(sub.id, entry => ({ ...entry, grade_letter: e.target.value }))}
                          >
                            <option value="">Select grade...</option>
                            {GRADE_LETTERS.map(g => <option key={g} value={g}>{g}</option>)}
                          </select>
                        </label>
                      ) : components.length > 0 ? (
                        <div className="mark-fields-grid">
                          {components.map(c => (
                            <label key={c.key} className="marks-component-field">
                              <span>{c.label} <span className="data-table__muted">/{c.max_marks}</span></span>
                              <input
                                type="number" min="0" max={c.max_marks} step="0.5"
                                className="marks-input"
                                disabled={isAbsent}
                                value={marks[sub.id]?.components?.[c.key] || ''}
                                onChange={e => editMark(sub.id, entry => ({ ...entry, components: { ...entry.components, [c.key]: e.target.value } }))}
                              />
                            </label>
                          ))}
                        </div>
                      ) : (
                        <label className="marks-component-field">
                          <span>Marks <span className="data-table__muted">/{sub.max_marks}</span></span>
                          <input
                            type="number" min="0" max={sub.max_marks} step="0.5"
                            className="marks-input"
                            disabled={isAbsent}
                            value={marks[sub.id]?.marks_obtained || ''}
                            placeholder={`out of ${sub.max_marks}`}
                            onChange={e => editMark(sub.id, entry => ({ ...entry, marks_obtained: e.target.value }))}
                          />
                        </label>
                      )}

                      <div className="mark-subject-card__total">
                        {sub.is_co_scholastic
                          ? 'Graded A–D, not counted in the total'
                          : components.length > 0 ? `Total: ${componentTotal(sub)} / ${fieldsMax}` : `Out of ${sub.max_marks}`}
                      </div>

                      {/* Teachers only enter marks; admins change fields per exam under Exams > Marks format */}
                      {!isTeacher && (
                        <button
                          type="button" className="btn btn--outline btn--sm" style={{ marginTop: '10px' }}
                          onClick={() => { setTab('exams'); setFormatExamId(selectedExamId) }}
                        >
                          <Pencil size={13} /> Change fields for this exam
                        </button>
                      )}
                    </div>
                  )
                })}
              </div>
              {markMsg && <p className={`doc-msg ${markMsg.startsWith('Error') ? 'doc-msg--error' : 'doc-msg--ok'}`} style={{ marginTop: '12px' }}>{markMsg}</p>}
              <button type="submit" className="btn btn--primary" style={{ marginTop: '12px' }} disabled={markSaving || touched.size === 0}>
                {markSaving ? 'Saving...' : touched.size > 0 ? `Save Marks (${touched.size} changed)` : 'Save Marks'}
              </button>
            </form>
          )}
          {markSubjects.length === 0 && selectedGrade && <p className="empty-text">No subjects found for this grade. Add subjects first.</p>}
        </div>
      )}

      {/* Marksheet Tab */}
      {tab === 'marksheet' && (
        <div className="results-section">
          <h2>Student Marksheet</h2>
          <div className="form-row" style={{ marginBottom: '16px' }}>
            <label className="form-field">
              <span>Exam</span>
              <select value={msExamId} onChange={e => setMsExamId(e.target.value)}>
                <option value="">Select exam...</option>
                {exams.map(ex => <option key={ex.id} value={ex.id}>{ex.name}</option>)}
              </select>
            </label>
            <label className="form-field">
              <span>Student</span>
              <select value={msStudentId} onChange={e => setMsStudentId(e.target.value)}>
                <option value="">Select student...</option>
                {students.map(s => <option key={s.id} value={s.id}>{s.first_name} {s.last_name} ({s.student_code})</option>)}
              </select>
            </label>
          </div>
          <div style={{ display: 'flex', gap: '10px', marginBottom: '20px', flexWrap: 'wrap' }}>
            <button className="btn btn--primary" onClick={loadMarksheet} disabled={!msExamId || !msStudentId || msLoading}>
              {msLoading ? 'Loading...' : 'View Marksheet'}
            </button>
            {marksheet && (
              <button className="btn btn--outline" onClick={downloadMarksheet}>
                <Download size={16} /> Download PDF
              </button>
            )}
          </div>

          {marksheet && (
            <div className="marksheet-preview">
              <div className="marksheet-header">
                <h3>{marksheet.school_name}</h3>
                <p>{marksheet.exam_name} · {marksheet.academic_year}</p>
                <p><strong>{marksheet.student_name}</strong> · {marksheet.student_code} · {marksheet.grade_level_name}</p>
              </div>
              <div className="table-card" style={{ overflowX: 'auto' }}>
                <table className="data-table">
                <thead><tr><th>Subject</th><th>Max</th><th>Pass</th><th>Obtained</th><th>%</th><th>Grade</th><th>Status</th></tr></thead>
                <tbody>
                  {(marksheet.rows || []).map((row, i) => (
                    <tr key={i}>
                      <td>
                        {row.subject_name}
                        {row.is_co_scholastic && <div className="data-table__muted">Co-scholastic — not in total</div>}
                      </td>
                      <td className="data-table__muted">{row.grade_letter ? '-' : row.max_marks}</td>
                      <td className="data-table__muted">{row.grade_letter ? '-' : row.passing_marks}</td>
                      <td>{row.is_absent ? 'Absent' : row.grade_letter ? '-' : row.marks_obtained}</td>
                      <td className="data-table__muted">{row.is_absent || row.grade_letter ? '-' : row.percentage?.toFixed(1) + '%'}</td>
                      <td><span className="badge badge--muted">{row.grade}</span></td>
                      <td>
                        <span className={`badge badge--${row.status === 'Pass' ? 'success' : row.status === 'Fail' ? 'danger' : 'muted'}`}>
                          {row.status}
                        </span>
                      </td>
                    </tr>
                  ))}
                </tbody>
                <tfoot>
                  <tr className="marksheet-total">
                    <td colSpan={3}><strong>Total / Result</strong></td>
                    <td>
                      {editingTotal ? (
                        <div className="marksheet-total-edit">
                          <input
                            type="number" min="0" max={marksheet.total_max} step="0.01"
                            value={totalInput} onChange={e => setTotalInput(e.target.value)}
                            style={{ width: '80px' }}
                          />
                          <span> / {marksheet.total_max}</span>
                          <button className="btn btn--primary btn--sm" onClick={saveTotalOverride} disabled={totalSaving}>
                            {totalSaving ? 'Saving...' : 'Save'}
                          </button>
                          <button className="btn btn--outline btn--sm" onClick={() => setEditingTotal(false)} disabled={totalSaving}>Cancel</button>
                        </div>
                      ) : (
                        <>
                          <strong>{marksheet.total_obtained?.toFixed(1)} / {marksheet.total_max}</strong>
                          {marksheet.is_total_overridden && (
                            <span className="badge badge--muted" style={{ marginLeft: '6px' }} title={`Auto-calculated: ${marksheet.computed_total_obtained?.toFixed(1)}`}>edited</span>
                          )}
                          {isSuperAdmin && (
                            <button className="btn-icon" title="Edit total" onClick={startEditTotal} style={{ marginLeft: '6px' }}>
                              <Pencil size={13} />
                            </button>
                          )}
                          {isSuperAdmin && marksheet.is_total_overridden && (
                            <button className="btn-icon" title="Reset to auto-calculated" onClick={resetTotalOverride} disabled={totalSaving}>
                              <RotateCcw size={13} />
                            </button>
                          )}
                        </>
                      )}
                    </td>
                    <td><strong>{marksheet.percentage?.toFixed(1)}%</strong></td>
                    <td><span className="badge badge--muted">{marksheet.overall_grade}</span></td>
                    <td><span className={`badge badge--${marksheet.result === 'Pass' ? 'success' : 'danger'}`}>{marksheet.result}</span></td>
                  </tr>
                </tfoot>
                </table>
              </div>
              {totalMsg && <p className="marksheet-total-error">{totalMsg}</p>}
              <p className="marksheet-cgpa">CGPA: <strong>{marksheet.cgpa?.toFixed(2)}</strong></p>
            </div>
          )}
        </div>
      )}

      {/* Report Card Tab */}
      {tab === 'result-sheet' && canSeeResultSheet && (
        <div className="results-section">
          <h2>Results: {grades.find(g => g.id === selectedGrade)?.name || '—'}</h2>
          <ResultDashboard exams={exams} />
        </div>
      )}

      {tab === 'report-card' && !isTeacher && (
        <div className="results-section">
          <h2>Report Card</h2>
          <div className="form-row" style={{ marginBottom: '16px' }}>
            <label className="form-field">
              <span>Student</span>
              <select value={rcStudentId} onChange={e => setRcStudentId(e.target.value)}>
                <option value="">Select student...</option>
                {students.map(s => <option key={s.id} value={s.id}>{s.first_name} {s.last_name} ({s.student_code})</option>)}
              </select>
            </label>
          </div>
          <div style={{ display: 'flex', gap: '10px', marginBottom: '20px', flexWrap: 'wrap' }}>
            <button className="btn btn--primary" onClick={loadReportCard} disabled={!rcStudentId || rcLoading}>
              {rcLoading ? 'Loading...' : 'View Report Card'}
            </button>
            {reportCard && (
              <>
                <select value={rcDesign} onChange={e => setRcDesign(e.target.value)} title="PDF design (doesn't change the marks/grades, just the look)">
                  <option value="classic">Design: Classic</option>
                  <option value="modern">Design: Modern Color</option>
                  <option value="minimal">Design: Minimal</option>
                </select>
                <button className="btn btn--outline" onClick={downloadReportCardPDF}>
                  <Download size={16} /> Download PDF
                </button>
              </>
            )}
          </div>

          {rcError && <p className="doc-msg doc-msg--error">{rcError}</p>}

          {reportCard && (
            <div className="marksheet-preview">
              <div className="marksheet-header">
                <h3>{reportCard.school_name}</h3>
                <p>{reportCard.academic_year} · {reportCard.grade_level_name} ({reportCard.template})</p>
                <p><strong>{reportCard.student_name}</strong> · {reportCard.student_code}</p>
              </div>

              {reportCard.exams.length === 0 ? (
                <p className="empty-text">
                  No published exam results yet for this student -- publish an exam under the Exams tab once marks are entered.
                </p>
              ) : (
              <div className="table-card" style={{ overflowX: 'auto' }}>
                <table className="data-table">
                  <thead>
                    <tr>
                      <th rowSpan={2}>Subject</th>
                      {reportCard.exams.map(ex => (
                        <th key={ex.exam_id} colSpan={2}>{ex.exam_name} ({['I','II','III'][ex.position - 1] || ex.position})</th>
                      ))}
                      <th colSpan={3}>Overall</th>
                    </tr>
                    <tr>
                      {reportCard.exams.map(ex => (
                        <Fragment key={ex.exam_id}>
                          <th>Obtained</th>
                          <th className="data-table__muted">Max</th>
                        </Fragment>
                      ))}
                      <th>Total</th><th>%</th><th>Grade</th>
                    </tr>
                  </thead>
                  <tbody>
                    {reportCard.subjects.map(sub => (
                      <tr key={sub.subject_id}>
                        <td>
                          {sub.subject_name}
                          {sub.is_co_scholastic && <div className="data-table__muted">Co-scholastic — not in total</div>}
                        </td>
                        {sub.by_exam.map((cell, i) => (
                          <Fragment key={i}>
                            <td>{cell.is_absent ? 'Absent' : cell.grade_letter ? <strong>{cell.grade_letter}</strong> : cell.obtained}</td>
                            <td className="data-table__muted">{cell.max_marks}</td>
                          </Fragment>
                        ))}
                        <td>{sub.is_graded ? '-' : `${sub.overall_obtained} / ${sub.overall_max}`}</td>
                        <td className="data-table__muted">{sub.is_graded ? '-' : `${sub.overall_percent?.toFixed(1)}%`}</td>
                        <td>{sub.grade ? <span className="badge badge--muted">{sub.grade}</span> : '-'}</td>
                      </tr>
                    ))}
                  </tbody>
                  <tfoot>
                    <tr className="marksheet-total">
                      <td colSpan={1 + reportCard.exams.length * 2}><strong>G.Total / %</strong></td>
                      <td><strong>{reportCard.overall_obtained} / {reportCard.overall_max}</strong></td>
                      <td><strong>{reportCard.overall_percent?.toFixed(1)}%</strong></td>
                      <td>{reportCard.overall_grade && <span className="badge badge--muted">{reportCard.overall_grade}</span>}</td>
                    </tr>
                  </tfoot>
                </table>
              </div>
              )}

              <h3 style={{ marginTop: '24px' }}>Report Card Details</h3>
              <form onSubmit={handleSaveReportCardDetails} className="results-inline-form">
                <div className="form-row">
                  <label className="form-field"><span>Roll No.</span><input value={rcDetailsForm.roll_no} onChange={e => setRcDetailsForm({ ...rcDetailsForm, roll_no: e.target.value })} /></label>
                  <label className="form-field"><span>Attendance</span><input value={rcDetailsForm.attendance} onChange={e => setRcDetailsForm({ ...rcDetailsForm, attendance: e.target.value })} placeholder="e.g. 210/220" /></label>
                  <label className="form-field"><span>Promoted To</span><input value={rcDetailsForm.promoted_to} onChange={e => setRcDetailsForm({ ...rcDetailsForm, promoted_to: e.target.value })} /></label>
                </div>
                <div className="form-row">
                  <label className="form-field"><span>Remark</span><input value={rcDetailsForm.remark} onChange={e => setRcDetailsForm({ ...rcDetailsForm, remark: e.target.value })} /></label>
                  {reportCard.template === 'primary' && (
                    <>
                      <label className="form-field"><span>Moral</span><input value={rcDetailsForm.moral_remark} onChange={e => setRcDetailsForm({ ...rcDetailsForm, moral_remark: e.target.value })} /></label>
                      <label className="form-field"><span>G.K.</span><input value={rcDetailsForm.gk_remark} onChange={e => setRcDetailsForm({ ...rcDetailsForm, gk_remark: e.target.value })} /></label>
                    </>
                  )}
                </div>
                <button type="submit" className="btn btn--primary" disabled={rcSaving}>{rcSaving ? 'Saving...' : 'Save Details'}</button>
              </form>

              {reportCard.template === 'middle' && (
                <>
                  <h3 style={{ marginTop: '24px' }}>Co-Scholastic / Discipline Grades</h3>
                  <div className="table-card">
                    <table className="data-table">
                      <thead><tr><th>Criterion</th><th>Grade</th></tr></thead>
                      <tbody>
                        {disciplineCriteria.map(key => (
                          <tr key={key}>
                            <td>{key}</td>
                            <td>
                              <select
                                value={disciplineGrades[key] || ''}
                                onChange={e => setDisciplineGrades(prev => ({ ...prev, [key]: e.target.value }))}
                              >
                                <option value="">-</option>
                                {['A+', 'A', 'B+', 'B', 'C+', 'C'].map(g => <option key={g} value={g}>{g}</option>)}
                              </select>
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                  <button className="btn btn--primary" style={{ marginTop: '12px' }} onClick={handleSaveDisciplineGrades}>Save Discipline Grades</button>
                </>
              )}

              {rcMsg && <p className={`doc-msg ${rcMsg.startsWith('Error') ? 'doc-msg--error' : 'doc-msg--ok'}`} style={{ marginTop: '12px' }}>{rcMsg}</p>}

              <CustomFieldsSection entityType="student_result" entityId={rcStudentId} scopeId={currentYear?.id} schoolId={currentSchool?.id} user={user} title="Extra Result Fields" />
            </div>
          )}
        </div>
      )}
    </div>
  )
}

export default Results
