import { useState } from 'react'
import { Outlet } from 'react-router-dom'
import Sidebar from './Sidebar'
import Header from './Header'
import './Layout.css'

function Layout({ onLogout, user }) {
  const [sidebarOpen, setSidebarOpen] = useState(true)

  return (
    <div className={`layout ${sidebarOpen ? '' : 'layout--collapsed'}`}>
      <Sidebar open={sidebarOpen} user={user} onLogout={onLogout} />
      <div className="layout__main">
        <Header onToggleSidebar={() => setSidebarOpen(!sidebarOpen)} />
        <main className="layout__content">
          <Outlet />
        </main>
      </div>
    </div>
  )
}

export default Layout
