import { Menu, X, Sun, Moon } from 'lucide-react'
import { useSchool } from '../services/SchoolContext'
import { useTheme } from '../services/ThemeContext'
import './Header.css'

function Header({ onToggleSidebar, sidebarOpen }) {
  const { schools, currentSchool, setCurrentSchool, academicYears, currentYear, setCurrentYear, isSuperAdmin } = useSchool()
  const { theme, toggleTheme } = useTheme()

  return (
    <header className="header">
      <div className="header__left">
        <button
          className={`header__menu-btn ${sidebarOpen ? 'header__menu-btn--open' : ''}`}
          onClick={onToggleSidebar}
          aria-label={sidebarOpen ? 'Close menu' : 'Open menu'}
        >
          {/* Desktop always shows the hamburger (it expands/collapses the rail);
              on mobile CSS swaps it to an X while the drawer is open. */}
          <Menu size={20} className="header__menu-icon" />
          <X size={20} className="header__close-icon" />
        </button>
        <div className="header__selectors">
          {isSuperAdmin && schools.length > 1 ? (
            <select
              className="header__select"
              value={currentSchool?.id || ''}
              onChange={e => {
                const s = schools.find(s => s.id === e.target.value)
                if (s) setCurrentSchool(s)
              }}
            >
              {schools.map(s => (
                <option key={s.id} value={s.id}>{s.name}</option>
              ))}
            </select>
          ) : currentSchool ? (
            <span className="header__school-name">{currentSchool.name}</span>
          ) : null}
          {academicYears.length > 0 && (
            <select
              className="header__select"
              value={currentYear?.id || ''}
              onChange={e => {
                const y = academicYears.find(y => y.id === e.target.value)
                if (y) setCurrentYear(y)
              }}
            >
              {academicYears.map(y => (
                <option key={y.id} value={y.id}>
                  {y.name}{y.is_current ? ' (Current)' : ''}
                </option>
              ))}
            </select>
          )}
        </div>
      </div>
      <div className="header__right">
        <span className="header__school-code">
          {currentSchool?.code || ''}
        </span>
        <button className="header__theme-btn" onClick={toggleTheme} title={theme === 'dark' ? 'Switch to light mode' : 'Switch to dark mode'}>
          {theme === 'dark' ? <Sun size={18} /> : <Moon size={18} />}
        </button>
      </div>
    </header>
  )
}

export default Header
