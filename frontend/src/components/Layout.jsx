import { useEffect, useRef, useState } from 'react'
import { Outlet } from 'react-router-dom'
import Sidebar from './Sidebar'
import Header from './Header'
import './Layout.css'

const MOBILE_BREAKPOINT = 768

function Layout({ onLogout, user }) {
  // On mobile the sidebar starts closed (it's a full-screen drawer there);
  // on desktop it starts open (full-width sidebar), matching prior behavior.
  const [sidebarOpen, setSidebarOpen] = useState(() =>
    typeof window === 'undefined' ? true : window.innerWidth > MOBILE_BREAKPOINT
  )
  const wasMobile = useRef(typeof window !== 'undefined' && window.innerWidth <= MOBILE_BREAKPOINT)

  useEffect(() => {
    function handleResize() {
      const isMobile = window.innerWidth <= MOBILE_BREAKPOINT
      // Only react to actually crossing the breakpoint, so a manual desktop
      // collapse isn't fought on every resize event.
      if (isMobile !== wasMobile.current) {
        setSidebarOpen(!isMobile)
        wasMobile.current = isMobile
      }
    }
    window.addEventListener('resize', handleResize)
    return () => window.removeEventListener('resize', handleResize)
  }, [])

  function handleNavigate() {
    if (window.innerWidth <= MOBILE_BREAKPOINT) setSidebarOpen(false)
  }

  return (
    <div className={`layout ${sidebarOpen ? '' : 'layout--collapsed'}`}>
      <Sidebar open={sidebarOpen} user={user} onLogout={onLogout} onNavigate={handleNavigate} />
      <div className="layout__backdrop" onClick={() => setSidebarOpen(false)} />
      <div className="layout__main">
        <Header onToggleSidebar={() => setSidebarOpen(!sidebarOpen)} sidebarOpen={sidebarOpen} />
        <main className="layout__content">
          <Outlet context={{ user }} />
        </main>
      </div>
    </div>
  )
}

export default Layout
