import { createContext, useContext, useState, useEffect } from 'react'
import { schoolsApi, academicApi, getToken } from './api'

const SchoolContext = createContext(null)

export function SchoolProvider({ user, children }) {
  const [schools, setSchools] = useState([])
  const [currentSchool, setCurrentSchool] = useState(null)
  const [academicYears, setAcademicYears] = useState([])
  const [currentYear, setCurrentYear] = useState(null)
  const [loading, setLoading] = useState(true)

  const isSuperAdmin = user?.role === 'super_admin'

  useEffect(() => {
    if (!getToken()) { setLoading(false); return }
    schoolsApi.list({ limit: 100 })
      .then(res => {
        const items = res.items || []
        setSchools(items)
        if (items.length > 0) setCurrentSchool(items[0])
      })
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [])

  useEffect(() => {
    if (!currentSchool) return
    academicApi.listYears(currentSchool.id)
      .then(res => {
        const items = res.items || []
        setAcademicYears(items)
        const current = items.find(y => y.is_current) || items[0] || null
        setCurrentYear(current)
      })
      .catch(() => {})
  }, [currentSchool])

  return (
    <SchoolContext.Provider value={{
      schools,
      currentSchool,
      // school_admin cannot switch schools — their school is locked from the API response
      setCurrentSchool: isSuperAdmin ? setCurrentSchool : () => {},
      academicYears,
      currentYear,
      setCurrentYear,
      loading,
      isSuperAdmin,
    }}>
      {children}
    </SchoolContext.Provider>
  )
}

export function useSchool() {
  const ctx = useContext(SchoolContext)
  if (!ctx) throw new Error('useSchool must be used within SchoolProvider')
  return ctx
}
