import { useState, useEffect } from 'react'
import { Routes, Route, Navigate } from 'react-router-dom'
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
import Login from './pages/Login'
import { getToken, clearToken } from './services/api'

function App() {
  const [user, setUser] = useState(() => {
    const t = getToken()
    if (!t) return null
    try {
      const payload = JSON.parse(atob(t.split('.')[1]))
      return { id: payload.sub, role: payload.role }
    } catch (_) { return null }
  })

  function handleLogin(u) { setUser(u) }

  function handleLogout() {
    clearToken()
    setUser(null)
  }

  if (!user) {
    return (
      <Routes>
        <Route path="/login" element={<Login onLogin={handleLogin} />} />
        <Route path="*" element={<Navigate to="/login" replace />} />
      </Routes>
    )
  }

  return (
    <Routes>
      <Route element={<Layout onLogout={handleLogout} user={user} />}>
        <Route index element={<Dashboard />} />
        <Route path="students" element={<Students />} />
        <Route path="students/:id" element={<StudentDetail />} />
        <Route path="fees" element={<Fees />} />
        <Route path="fees/:id" element={<FeeAccountDetail />} />
        <Route path="teachers" element={<Teachers />} />
        <Route path="teachers/:id" element={<TeacherDetail />} />
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
      </Route>
    </Routes>
  )
}

export default App
