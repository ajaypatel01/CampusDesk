const BASE = '/api/v1'

export function getToken() { return localStorage.getItem('cd_token') }
export function setToken(t) { localStorage.setItem('cd_token', t) }
export function clearToken() { localStorage.removeItem('cd_token') }

function authHeader() {
  const t = getToken()
  return t ? { Authorization: `Bearer ${t}` } : {}
}

async function request(path, options = {}) {
  const url = `${BASE}${path}`
  const res = await fetch(url, {
    headers: { 'Content-Type': 'application/json', ...authHeader(), ...options.headers },
    ...options,
  })
  if (res.status === 401) {
    clearToken()
    if (window.location.pathname !== '/login') {
      window.location.replace('/login')
    }
    throw new Error('Session expired. Please log in again.')
  }
  if (res.status === 204) return null
  const data = await res.json()
  if (!res.ok) throw new Error(data.error || `Request failed: ${res.status}`)
  return data
}

async function requestBlob(path, options = {}) {
  const url = `${BASE}${path}`
  const res = await fetch(url, {
    headers: { ...authHeader(), ...options.headers },
    ...options,
  })
  if (res.status === 401) {
    clearToken()
    window.location.href = '/login'
    throw new Error('Session expired.')
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
}

export const studentsApi = {
  list: (params) => request(`/students${qs(params)}`),
  get: (id) => request(`/students/${id}`),
  create: (body) => request('/students', { method: 'POST', body: JSON.stringify(body) }),
  update: (id, body) => request(`/students/${id}`, { method: 'PUT', body: JSON.stringify(body) }),
  delete: (id) => request(`/students/${id}`, { method: 'DELETE' }),
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
  login: (body) => request('/auth/login', { method: 'POST', body: JSON.stringify(body) }),
}

export const academicApi = {
  listYears: (schoolId) => request(`/academic-years${qs({ school_id: schoolId })}`),
  createYear: (body) => request('/academic-years', { method: 'POST', body: JSON.stringify(body) }),
  listGrades: (schoolId) => request(`/grade-levels${qs({ school_id: schoolId })}`),
  createGrade: (body) => request('/grade-levels', { method: 'POST', body: JSON.stringify(body) }),
  listSections: (params) => request(`/class-sections${qs(params)}`),
  createSection: (body) => request('/class-sections', { method: 'POST', body: JSON.stringify(body) }),
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
  recordPayment: (body) => request('/fee-payments', { method: 'POST', body: JSON.stringify(body) }),
  voidPayment: (id) => request(`/fee-payments/${id}`, { method: 'DELETE' }),

  schoolSummary: (params) => request(`/fee-summary${qs(params)}`),
  studentSummary: (studentId, yearId) => request(`/fee-summary/student/${studentId}${qs({ academic_year_id: yearId })}`),
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

  listExams: (params) => request(`/exams${qs(params)}`),
  createExam: (body) => request('/exams', { method: 'POST', body: JSON.stringify(body) }),
  publishExam: (id, publish) => request(`/exams/${id}/publish`, { method: 'POST', body: JSON.stringify({ publish }) }),

  upsertMark: (body) => request('/exam-marks', { method: 'POST', body: JSON.stringify(body) }),
  bulkUpsertMarks: (marks) => request('/exam-marks/bulk', { method: 'POST', body: JSON.stringify({ marks }) }),

  getMarksheet: (examId, studentId) => request(`/marksheets${qs({ exam_id: examId, student_id: studentId })}`),
  downloadMarksheet: (examId, studentId) =>
    requestBlob(`/marksheets/pdf?exam_id=${examId}&student_id=${studentId}`),
}

export const homeworkApi = {
  list: (params) => request(`/homework${qs(params)}`),
  get: (id) => request(`/homework/${id}`),
  create: (body) => request('/homework', { method: 'POST', body: JSON.stringify(body) }),
  delete: (id) => request(`/homework/${id}`, { method: 'DELETE' }),
  listSubmissions: (id) => request(`/homework/${id}/submissions`),
  upsertSubmission: (id, body) => request(`/homework/${id}/submissions`, { method: 'POST', body: JSON.stringify(body) }),
  studentTracker: (params) => request(`/homework-tracker${qs(params)}`),
}

export const idCardsApi = {
  generateStudents: (body) =>
    requestBlob(`/id-cards/students`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) }),
  generateTeachers: (body) =>
    requestBlob(`/id-cards/teachers`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) }),
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
