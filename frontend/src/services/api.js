const BASE = '/api/v1'

export function getToken() { return localStorage.getItem('cd_token') }
export function setToken(t) { localStorage.setItem('cd_token', t) }
export function clearToken() { localStorage.removeItem('cd_token') }

function authHeader() {
  const t = getToken()
  return t ? { Authorization: `Bearer ${t}` } : {}
}

// unauthorizedMessage turns a 401 response body into a user-facing message.
// The backend returns the same status for two very different situations:
// an already-logged-in session whose token expired/was revoked, and a plain
// login attempt with the wrong credentials (or a pending/rejected/disabled
// account). Only the first is really a "session expired" -- callers pass
// hadToken so we can tell them apart instead of showing "session expired"
// on someone's very first (failed) login attempt.
async function unauthorizedMessage(res, hadToken) {
  if (hadToken) return 'Session expired. Please log in again.'
  let msg = 'Invalid email or password.'
  try {
    const d = await res.json()
    if (d.error) {
      // Known account-status cases come back as "unauthorized: <reason>";
      // a bare "unauthorized" just means the credentials didn't match.
      msg = d.error === 'unauthorized' ? msg : d.error.replace(/^unauthorized:\s*/, '')
    }
  } catch (_) { /* no JSON body */ }
  return msg
}

async function request(path, options = {}) {
  const url = `${BASE}${path}`
  const hadToken = !!getToken()
  const res = await fetch(url, {
    ...options,
    headers: { 'Content-Type': 'application/json', ...authHeader(), ...options.headers },
  })
  if (res.status === 401) {
    const msg = await unauthorizedMessage(res, hadToken)
    clearToken()
    if (hadToken && window.location.pathname !== '/login') {
      window.location.replace('/login')
    }
    throw new Error(msg)
  }
  if (res.status === 204) return null
  const data = await res.json()
  if (!res.ok) throw new Error(data.error || `Request failed: ${res.status}`)
  return data
}

async function requestBlob(path, options = {}) {
  const url = `${BASE}${path}`
  const hadToken = !!getToken()
  const res = await fetch(url, {
    ...options,
    headers: { ...authHeader(), ...options.headers },
  })
  if (res.status === 401) {
    const msg = await unauthorizedMessage(res, hadToken)
    clearToken()
    if (hadToken) window.location.href = '/login'
    throw new Error(msg)
  }
  if (!res.ok) {
    let msg = `Request failed: ${res.status}`
    try { const d = await res.json(); if (d.error) msg = d.error } catch (_) {}
    throw new Error(msg)
  }
  return res.blob()
}

function qs(params) {
  const p = new URLSearchParams()
  Object.entries(params).forEach(([k, v]) => {
    if (v !== undefined && v !== null && v !== '') p.set(k, v)
  })
  return p.toString() ? `?${p}` : ''
}

export const schoolsApi = {
  list: (params = {}) => request(`/schools${qs(params)}`),
  get: (id) => request(`/schools/${id}`),
  create: (body) => request('/schools', { method: 'POST', body: JSON.stringify(body) }),
  update: (id, body) => request(`/schools/${id}`, { method: 'PUT', body: JSON.stringify(body) }),
  delete: (id) => request(`/schools/${id}`, { method: 'DELETE' }),
  // Unauthenticated directory (id/name/code only) for the registration form's school picker.
  publicList: () => request(`/schools/public`),
}

export const studentsApi = {
  list: (params) => request(`/students${qs(params)}`),
  get: (id) => request(`/students/${id}`),
  create: (body) => request('/students', { method: 'POST', body: JSON.stringify(body) }),
  update: (id, body) => request(`/students/${id}`, { method: 'PUT', body: JSON.stringify(body) }),
  delete: (id) => request(`/students/${id}`, { method: 'DELETE' }),
  // Parent portal: the logged-in parent's own ward(s).
  myWards: () => request('/my-wards'),
  // Bulk import: a downloadable/fillable .xlsx template, and the matching
  // upload that bulk-creates students from it (row-by-row result, a bad
  // row doesn't block the rest of the file).
  downloadImportTemplate: () => requestBlob('/students/import-template'),
  importStudents: (schoolId, file) => {
    const fd = new FormData(); fd.append('file', file)
    return fetch(`${BASE}/students/import${qs({ school_id: schoolId })}`, { method: 'POST', headers: authHeader(), body: fd }).then(async res => {
      const data = await res.json(); if (!res.ok) throw new Error(data.error || 'Import failed'); return data
    })
  },
  // Promotion/demotion (or a mid-year class change): points a student's fee
  // account at a different grade's fee structure and keeps their enrollment
  // row in sync. moveGrade is one student; bulkPromote is the normal
  // end-of-year shape (a whole grade/section moving up together).
  moveGrade: (id, body) => request(`/students/${id}/move-grade`, { method: 'POST', body: JSON.stringify(body) }),
  bulkPromote: (body) => request('/students/promote-bulk', { method: 'POST', body: JSON.stringify(body) }),
  getSection: (id, academicYearId) => request(`/students/${id}/section${qs({ academic_year_id: academicYearId })}`),
  updateSection: (id, body) => request(`/students/${id}/section`, { method: 'PUT', body: JSON.stringify(body) }),
}

export const guardiansApi = {
  list: (studentId) => request(`/guardians${qs({ student_id: studentId })}`),
  get: (id) => request(`/guardians/${id}`),
  create: (body) => request('/guardians', { method: 'POST', body: JSON.stringify(body) }),
  link: (body) => request('/guardians/link', { method: 'POST', body: JSON.stringify(body) }),
}

export const usersApi = {
  list: (params = {}) => request(`/users${qs(params)}`),
  get: (id) => request(`/users/${id}`),
  create: (body) => request('/users', { method: 'POST', body: JSON.stringify(body) }),
  update: (id, body) => request(`/users/${id}`, { method: 'PUT', body: JSON.stringify(body) }),
  remove: (id) => request(`/users/${id}`, { method: 'DELETE' }),
  login: (body) => request('/auth/login', { method: 'POST', body: JSON.stringify(body) }),
  // Public self-registration; account is created pending until an admin approves it.
  register: (body) => request('/auth/register', { method: 'POST', body: JSON.stringify(body) }),
  // Self-service password reset. requestPasswordReset always "succeeds" --
  // the backend never reveals whether the email is actually registered.
  requestPasswordReset: (email) => request('/auth/password-reset/request', { method: 'POST', body: JSON.stringify({ email }) }),
  confirmPasswordReset: (token, newPassword) => request('/auth/password-reset/confirm', { method: 'POST', body: JSON.stringify({ token, new_password: newPassword }) }),
  listPending: (params = {}) => request(`/users${qs({ ...params, status: 'pending' })}`),
  approve: (id) => request(`/users/${id}/approve`, { method: 'POST' }),
  reject: (id) => request(`/users/${id}/reject`, { method: 'POST' }),
  // Invalidates every session for the current account, including this one --
  // the caller must still clear its own stored token and redirect, same as
  // a normal logout, right after this resolves (or fails/times out).
  logoutEverywhere: () => request('/auth/logout-everywhere', { method: 'POST' }),
  // The caller's own profile -- works for every role, unlike get(id) above
  // which registrars can't use on themselves.
  me: () => request('/auth/me'),
  // Self-service phone verification: request sends an OTP to a number not
  // yet saved anywhere; confirm attaches it to the caller's own account.
  requestPhoneVerification: (phone) => request('/auth/phone/verify/request', { method: 'POST', body: JSON.stringify({ phone }) }),
  confirmPhoneVerification: (phone, otp) => request('/auth/phone/verify/confirm', { method: 'POST', body: JSON.stringify({ phone, otp }) }),
  // OTP login -- only works once a number has been verified via the above.
  requestOTPLogin: (phone) => request('/auth/otp/send', { method: 'POST', body: JSON.stringify({ phone }) }),
  verifyOTPLogin: (phone, otp) => request('/auth/otp/verify', { method: 'POST', body: JSON.stringify({ phone, otp }) }),
  // Phone-based password reset, an alternative to the emailed-link flow for
  // an account with a verified number.
  requestPasswordResetOTP: (phone) => request('/auth/password-reset/otp-request', { method: 'POST', body: JSON.stringify({ phone }) }),
  confirmPasswordResetOTP: (phone, otp, newPassword) => request('/auth/password-reset/otp-confirm', { method: 'POST', body: JSON.stringify({ phone, otp, new_password: newPassword }) }),
}

export const academicApi = {
  listYears: (schoolId) => request(`/academic-years${qs({ school_id: schoolId })}`),
  createYear: (body) => request('/academic-years', { method: 'POST', body: JSON.stringify(body) }),
  listGrades: (schoolId) => request(`/grade-levels${qs({ school_id: schoolId })}`),
  createGrade: (body) => request('/grade-levels', { method: 'POST', body: JSON.stringify(body) }),
  updateGrade: (id, body) => request(`/grade-levels/${id}`, { method: 'PUT', body: JSON.stringify(body) }),
  listSections: (params) => request(`/class-sections${qs(params)}`),
  createSection: (body) => request('/class-sections', { method: 'POST', body: JSON.stringify(body) }),
  updateSection: (id, body) => request(`/class-sections/${id}`, { method: 'PUT', body: JSON.stringify(body) }),
}

export const enrollmentsApi = {
  list: (params) => request(`/enrollments${qs(params)}`),
  get: (id) => request(`/enrollments/${id}`),
  create: (body) => request('/enrollments', { method: 'POST', body: JSON.stringify(body) }),
  update: (id, body) => request(`/enrollments/${id}`, { method: 'PUT', body: JSON.stringify(body) }),
}

export const attendanceApi = {
  list: (params) => request(`/attendance${qs(params)}`),
  record: (body) => request('/attendance', { method: 'POST', body: JSON.stringify(body) }),
}

export const feesApi = {
  listStructures: (params) => request(`/fee-structures${qs(params)}`),
  getStructure: (id) => request(`/fee-structures/${id}`),
  createStructure: (body) => request('/fee-structures', { method: 'POST', body: JSON.stringify(body) }),
  updateStructure: (id, body) => request(`/fee-structures/${id}`, { method: 'PUT', body: JSON.stringify(body) }),

  listAccounts: (params) => request(`/fee-accounts${qs(params)}`),
  getAccount: (id) => request(`/fee-accounts/${id}`),
  createAccount: (body) => request('/fee-accounts', { method: 'POST', body: JSON.stringify(body) }),
  updateAccount: (id, body) => request(`/fee-accounts/${id}`, { method: 'PUT', body: JSON.stringify(body) }),

  listPayments: (accountId) => request(`/fee-payments${qs({ student_fee_account_id: accountId })}`),
  // All of a school's payments for one year, with student name/code/class (office staff only).
  listLedgerPayments: (params) => request(`/fee-payments/ledger${qs(params)}`),
  recordPayment: (body) => request('/fee-payments', { method: 'POST', body: JSON.stringify(body) }),
  voidPayment: (id) => request(`/fee-payments/${id}`, { method: 'DELETE' }),
  // Reassigns a payment to a different academic year's fee account for the
  // same student -- for correcting a payment entered under the wrong year.
  // Registrar/super_admin only; the target year's fee account must exist.
  movePayment: (id, academicYearId) => request(`/fee-payments/${id}/move`, { method: 'PUT', body: JSON.stringify({ academic_year_id: academicYearId }) }),

  schoolSummary: (params) => request(`/fee-summary${qs(params)}`),
  studentSummary: (studentId, yearId) => request(`/fee-summary/student/${studentId}${qs({ academic_year_id: yearId })}`),
  installmentSheet: (params) => request(`/fee-installment-sheet${qs(params)}`),
  downloadReceipt: (paymentId) => requestBlob(`/fee-receipts/${paymentId}`),
  sendReceiptWhatsApp: (paymentId, phone) => request(`/fee-receipts/${paymentId}/whatsapp`, { method: 'POST', body: JSON.stringify({ phone }) }),
}

export const documentsApi = {
  downloadBonafide: (studentId, yearId) =>
    requestBlob(`/documents/bonafide?student_id=${studentId}&academic_year_id=${yearId}`),
  emailBonafide: (body) => request('/documents/bonafide/email', { method: 'POST', body: JSON.stringify(body) }),
  whatsappBonafide: (body) => request('/documents/bonafide/whatsapp', { method: 'POST', body: JSON.stringify(body) }),

  downloadTC: (params) =>
    requestBlob(`/documents/transfer-certificate?${new URLSearchParams(params)}`),
  emailTC: (body) => request('/documents/transfer-certificate/email', { method: 'POST', body: JSON.stringify(body) }),
  whatsappTC: (body) => request('/documents/transfer-certificate/whatsapp', { method: 'POST', body: JSON.stringify(body) }),

  downloadSalarySlip: (body) =>
    requestBlob(`/documents/salary-slip`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) }),
  emailSalarySlip: (body) => request('/documents/salary-slip/email', { method: 'POST', body: JSON.stringify(body) }),
  whatsappSalarySlip: (body) => request('/documents/salary-slip/whatsapp', { method: 'POST', body: JSON.stringify(body) }),
}

export const resultsApi = {
  listSubjects: (params) => request(`/subjects${qs(params)}`),
  createSubject: (body) => request('/subjects', { method: 'POST', body: JSON.stringify(body) }),
  updateSubject: (id, body) => request(`/subjects/${id}`, { method: 'PUT', body: JSON.stringify(body) }),
  deleteSubject: (id) => request(`/subjects/${id}`, { method: 'DELETE' }),

  // Per-subject graded "sections" (Oral/Unit Test/Activity/Practical/
  // Written/...). Admin subject setup -- the backend blocks teachers and
  // parents from adding/editing/removing them.
  listSubjectComponents: (subjectId) => request(`/subjects/${subjectId}/mark-components`),
  addSubjectComponent: (subjectId, body) => request(`/subjects/${subjectId}/mark-components`, { method: 'POST', body: JSON.stringify(body) }),
  // Rename a field and/or change its max marks (refused once marks are recorded under it).
  updateSubjectComponent: (subjectId, key, body) => request(`/subjects/${subjectId}/mark-components/${key}`, { method: 'PUT', body: JSON.stringify(body) }),
  deleteSubjectComponent: (subjectId, key) => request(`/subjects/${subjectId}/mark-components/${key}`, { method: 'DELETE' }),

  listExams: (params) => request(`/exams${qs(params)}`),
  createExam: (body) => request('/exams', { method: 'POST', body: JSON.stringify(body) }),
  // Edit name/date/weight; delete is refused while published or once marks are entered.
  updateExam: (id, body) => request(`/exams/${id}`, { method: 'PUT', body: JSON.stringify(body) }),
  deleteExam: (id) => request(`/exams/${id}`, { method: 'DELETE' }),
  publishExam: (id, publish) => request(`/exams/${id}/publish`, { method: 'POST', body: JSON.stringify({ publish }) }),
  // Per-exam marks distribution for each subject (admins change it; teachers read it).
  listExamFormats: (examId) => request(`/exams/${examId}/mark-formats`),
  setExamSubjectFormat: (examId, subjectId, body) => request(`/exams/${examId}/mark-formats/${subjectId}`, { method: 'PUT', body: JSON.stringify(body) }),
  resetExamSubjectFormat: (examId, subjectId) => request(`/exams/${examId}/mark-formats/${subjectId}`, { method: 'DELETE' }),

  upsertMark: (body) => request('/exam-marks', { method: 'POST', body: JSON.stringify(body) }),
  bulkUpsertMarks: (marks) => request('/exam-marks/bulk', { method: 'POST', body: JSON.stringify({ marks }) }),

  // Whole-class marks for one exam (admins and the owner only).
  getResultSheet: (examId) => request(`/exams/${examId}/result-sheet`),
  getMarksheet: (examId, studentId) => request(`/marksheets${qs({ exam_id: examId, student_id: studentId })}`),
  downloadMarksheet: (examId, studentId) =>
    requestBlob(`/marksheets/pdf?exam_id=${examId}&student_id=${studentId}`),
  // super_admin only: manually correct a marksheet's total.
  setTotalOverride: (examId, studentId, totalObtained) =>
    request('/marksheets/total-override', { method: 'PUT', body: JSON.stringify({ exam_id: examId, student_id: studentId, total_obtained: totalObtained }) }),
  clearTotalOverride: (examId, studentId) =>
    request(`/marksheets/total-override${qs({ exam_id: examId, student_id: studentId })}`, { method: 'DELETE' }),
  // Parent portal: published exams for a ward's current class.
  wardExams: (params) => request(`/ward-exams${qs(params)}`),

  // Combined, multi-exam, class-wise-templated report card (kg/primary/middle).
  getReportCard: (studentId, academicYearId) =>
    request(`/report-cards${qs({ student_id: studentId, academic_year_id: academicYearId })}`),
  // design: 'classic' | 'modern' | 'minimal' -- a visual skin only, never
  // changes the school's actual grading rubric/content on the PDF.
  downloadReportCard: (studentId, academicYearId, design) =>
    requestBlob(`/report-cards/pdf${qs({ student_id: studentId, academic_year_id: academicYearId, design })}`),
  upsertReportCardDetails: (body) => request('/report-cards/details', { method: 'PUT', body: JSON.stringify(body) }),
  upsertDisciplineGrades: (body) => request('/report-cards/discipline-grades', { method: 'PUT', body: JSON.stringify(body) }),
  listDisciplineCriteria: () => request('/discipline-criteria'),
}

// Super-admin-only custom fields (Student Detail, Results, ...). Every call
// here 403s for anyone else -- CustomFieldsSection never even calls it.
export const customFieldsApi = {
  listDefinitions: (schoolId, entityType) => request(`/custom-fields/definitions${qs({ school_id: schoolId, entity_type: entityType })}`),
  createDefinition: (body) => request('/custom-fields/definitions', { method: 'POST', body: JSON.stringify(body) }),
  updateDefinition: (id, body) => request(`/custom-fields/definitions/${id}`, { method: 'PUT', body: JSON.stringify(body) }),
  deleteDefinition: (id) => request(`/custom-fields/definitions/${id}`, { method: 'DELETE' }),
  listValues: (schoolId, entityType, entityId, scopeId) =>
    request(`/custom-fields/values${qs({ school_id: schoolId, entity_type: entityType, entity_id: entityId, scope_id: scopeId })}`),
  upsertValue: (body) => request('/custom-fields/values', { method: 'PUT', body: JSON.stringify(body) }),
}

export const homeworkApi = {
  list: (params) => request(`/homework${qs(params)}`),
  get: (id) => request(`/homework/${id}`),
  create: (body) => request('/homework', { method: 'POST', body: JSON.stringify(body) }),
  delete: (id) => request(`/homework/${id}`, { method: 'DELETE' }),
  listSubmissions: (id) => request(`/homework/${id}/submissions`),
  upsertSubmission: (id, body) => request(`/homework/${id}/submissions`, { method: 'POST', body: JSON.stringify(body) }),
  studentTracker: (params) => request(`/homework-tracker${qs(params)}`),
  // Parent portal: a ward's class homework, with that ward's own submission status.
  wardHomework: (params) => request(`/ward-homework${qs(params)}`),
}

export const idCardsApi = {
  generateStudents: (body) =>
    requestBlob(`/id-cards/students`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) }),
  generateTeachers: (body) =>
    requestBlob(`/id-cards/teachers`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) }),
}

export const payrollApi = {
  computeMonth: (params) => request(`/payroll${qs(params)}`),
  downloadSlip: (params) => requestBlob(`/payroll/slip${qs(params)}`),
  listLeaves: (params) => request(`/staff-leaves${qs(params)}`),
  createLeave: (body) => request('/staff-leaves', { method: 'POST', body: JSON.stringify(body) }),
  updateLeave: (id, body) => request(`/staff-leaves/${id}`, { method: 'PUT', body: JSON.stringify(body) }),
  deleteLeave: (id) => request(`/staff-leaves/${id}`, { method: 'DELETE' }),
}

export const broadcastsApi = {
  list: (schoolId) => request(`/broadcasts${qs({ school_id: schoolId })}`),
  send: (body) => request('/broadcasts', { method: 'POST', body: JSON.stringify(body) }),
  listRecipients: (id) => request(`/broadcasts/${id}/recipients`),
}

export const vansApi = {
  list: (schoolId) => request(`/vans${qs({ school_id: schoolId })}`),
  get: (id, params = {}) => request(`/vans/${id}${qs(params)}`),
  create: (body) => request('/vans', { method: 'POST', body: JSON.stringify(body) }),
  update: (id, body) => request(`/vans/${id}`, { method: 'PUT', body: JSON.stringify(body) }),
  delete: (id) => request(`/vans/${id}`, { method: 'DELETE' }),
  addRoute: (vanId, body) => request(`/vans/${vanId}/routes`, { method: 'POST', body: JSON.stringify(body) }),
  deleteRoute: (vanId, routeId) => request(`/vans/${vanId}/routes/${routeId}`, { method: 'DELETE' }),
  listAssignments: (params) => request(`/van-assignments${qs(params)}`),
  assignStudent: (body) => request('/van-assignments', { method: 'POST', body: JSON.stringify(body) }),
  removeAssignment: (id) => request(`/van-assignments/${id}`, { method: 'DELETE' }),
}

export const rteApi = {
  getSummary: (params) => request(`/rte/summary${qs(params)}`),
  listStudents: (params) => request(`/rte/students${qs(params)}`),
  listQuotas: (params) => request(`/rte/quotas${qs(params)}`),
  upsertQuota: (body) => request('/rte/quotas', { method: 'POST', body: JSON.stringify(body) }),
  deleteQuota: (id) => request(`/rte/quotas/${id}`, { method: 'DELETE' }),
}

export const booksApi = {
  listBooks: (params) => request(`/books${qs(params)}`),
  getBook: (id) => request(`/books/${id}`),
  createBook: (body) => request('/books', { method: 'POST', body: JSON.stringify(body) }),
  updateBook: (id, body) => request(`/books/${id}`, { method: 'PUT', body: JSON.stringify(body) }),
  deleteBook: (id) => request(`/books/${id}`, { method: 'DELETE' }),
  listBookLists: (params) => request(`/book-lists${qs(params)}`),
  getBookList: (id) => request(`/book-lists/${id}`),
  createBookList: (body) => request('/book-lists', { method: 'POST', body: JSON.stringify(body) }),
  addItem: (listId, body) => request(`/book-lists/${listId}/items`, { method: 'POST', body: JSON.stringify(body) }),
  removeItem: (listId, itemId) => request(`/book-lists/${listId}/items/${itemId}`, { method: 'DELETE' }),
  downloadPDF: (listId) => requestBlob(`/book-lists/${listId}/pdf`),
  listReceipts: (bookListId) => request(`/book-receipts${qs({ book_list_id: bookListId })}`),
  recordReceipt: (body) => request('/book-receipts', { method: 'POST', body: JSON.stringify(body) }),
}

export const mediaApi = {
  uploadStudentPhoto: (id, file) => {
    const fd = new FormData(); fd.append('photo', file)
    return fetch(`${BASE}/media/students/${id}/photo`, { method: 'POST', headers: authHeader(), body: fd }).then(async res => {
      const data = await res.json(); if (!res.ok) throw new Error(data.error || 'Upload failed'); return data
    })
  },
  uploadUserPhoto: (id, file) => {
    const fd = new FormData(); fd.append('photo', file)
    return fetch(`${BASE}/media/users/${id}/photo`, { method: 'POST', headers: authHeader(), body: fd }).then(async res => {
      const data = await res.json(); if (!res.ok) throw new Error(data.error || 'Upload failed'); return data
    })
  },
}

export const configApi = {
  get: () => request('/config'),
}

export const tcRecordsApi = {
  list: (params = {}) => request(`/tc-records${qs(params)}`),
  get: (id) => request(`/tc-records/${id}`),
  create: (body) => request('/tc-records', { method: 'POST', body: JSON.stringify(body) }),
  update: (id, body) => request(`/tc-records/${id}`, { method: 'PUT', body: JSON.stringify(body) }),
  remove: (id) => request(`/tc-records/${id}`, { method: 'DELETE' }),
}

export const vouchersApi = {
  list: (params = {}) => request(`/vouchers${qs(params)}`),
  create: (body) => request('/vouchers', { method: 'POST', body: JSON.stringify(body) }),
}

export const staffApi = {
  list: (params = {}) => request(`/staff${qs(params)}`),
  get: (id) => request(`/staff/${id}`),
  upsertProfile: (id, body) => request(`/staff/${id}/profile`, { method: 'PUT', body: JSON.stringify(body) }),
}

// Access-control matrix: an admin narrowing (or restoring) a specific
// user's view/write access to a section of the app, on top of their role's
// default (everything allowed).
export const permissionsApi = {
  listFeatures: () => request('/permissions/features'),
  getMatrix: (userId) => request(`/permissions/users/${userId}`),
  setMatrix: (userId, overrides) => request(`/permissions/users/${userId}`, { method: 'PUT', body: JSON.stringify({ overrides }) }),
}
