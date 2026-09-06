import { useState } from 'react'
import { Routes, Route, Navigate, useLocation } from 'react-router-dom'
import Layout from './components/Layout'
import Dashboard from './pages/Dashboard'
import Students from './pages/Students'
import StudentDetail from './pages/StudentDetail'
import Fees from './pages/Fees'
import FeeAccountDetail from './pages/FeeAccountDetail'
import Teachers from './pages/Teachers'
import TeacherDetail from './pages/TeacherDetail'
import Settings from './pages/Settings'
import Documents from './pages/Documents'
import Broadcasts from './pages/Broadcasts'
import Results from './pages/Results'
import Homework from './pages/Homework'
import IdCards from './pages/IdCards'
import Transport from './pages/Transport'
import Rte from './pages/Rte'
import Books from './pages/Books'
import Admissions from './pages/Admissions'
import Staff from './pages/Staff'
import StaffDetail from './pages/StaffDetail'
import TCRecords from './pages/TCRecords'
import Vouchers from './pages/Vouchers'
import Ledger from './pages/Ledger'
import Payroll from './pages/Payroll'
import Login from './pages/Login'
import Register from './pages/Register'
import { getToken, clearToken } from './services/api'
import { SchoolProvider } from './services/SchoolContext'
import { ConfigProvider } from './services/ConfigContext'

// Pages a registrar isn't allowed to open, even by typing the URL directly —
// mirrors the backend's BlockRoles("registrar") checks on the same modules.
const REGISTRAR_BLOCKED_PATHS = ['/teachers', '/staff', '/documents', '/broadcasts', '/id-cards', '/books', '/settings']
// Pages only super_admin may open — mirrors the backend's RequireRole("super_admin") check.
const SUPER_ADMIN_ONLY_PATHS = ['/payroll']

function RegistrarGuard({ user, children }) {
  const location = useLocation()
  const isRegistrarBlocked = user?.role === 'registrar' &&
    REGISTRAR_BLOCKED_PATHS.some(p => location.pathname === p || location.pathname.startsWith(p + '/'))
  const isSuperAdminOnly = user?.role !== 'super_admin' &&
    SUPER_ADMIN_ONLY_PATHS.some(p => location.pathname === p || location.pathname.startsWith(p + '/'))
  if (isRegistrarBlocked || isSuperAdminOnly) return <Navigate to="/" replace />
  return children
}

function decodeUser(token) {
  if (!token) return null
  try {
    const payload = JSON.parse(atob(token.split('.')[1]))
    return { id: payload.sub, role: payload.role, schoolId: payload.school_id || null }
  } catch (_) { return null }
}

function App() {
  const [user, setUser] = useState(() => decodeUser(getToken()))

  function handleLogin() { setUser(decodeUser(getToken())) }

  function handleLogout() {
    clearToken()
    setUser(null)
  }

  if (!user) {
    return (
      <Routes>
        <Route path="/login" element={<Login onLogin={handleLogin} />} />
        <Route path="/register" element={<Register />} />
        <Route path="*" element={<Navigate to="/login" replace />} />
      </Routes>
    )
  }

  return (
    <ConfigProvider>
      <SchoolProvider user={user}>
        <Routes>
          <Route element={<RegistrarGuard user={user}><Layout onLogout={handleLogout} user={user} /></RegistrarGuard>}>
            <Route index element={<Dashboard />} />
            <Route path="admissions" element={<Admissions />} />
            <Route path="students" element={<Students />} />
            <Route path="students/:id" element={<StudentDetail />} />
            <Route path="fees" element={<Fees />} />
            <Route path="fees/:id" element={<FeeAccountDetail />} />
            <Route path="teachers" element={<Teachers />} />
            <Route path="teachers/:id" element={<TeacherDetail />} />
            <Route path="staff" element={<Staff />} />
            <Route path="staff/:id" element={<StaffDetail />} />
            <Route path="tc-records" element={<TCRecords />} />
            <Route path="vouchers" element={<Vouchers />} />
            <Route path="ledger" element={<Ledger />} />
            <Route path="payroll" element={<Payroll />} />
            <Route path="documents" element={<Documents />} />
            <Route path="broadcasts" element={<Broadcasts />} />
            <Route path="results" element={<Results />} />
            <Route path="homework" element={<Homework />} />
            <Route path="id-cards" element={<IdCards />} />
            <Route path="transport" element={<Transport />} />
            <Route path="rte" element={<Rte />} />
            <Route path="books" element={<Books />} />
            <Route path="settings" element={<Settings />} />
            <Route path="login" element={<Navigate to="/" replace />} />
            <Route path="register" element={<Navigate to="/" replace />} />
          </Route>
        </Routes>
      </SchoolProvider>
    </ConfigProvider>
  )
}

export default App
