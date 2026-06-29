import { NavLink } from 'react-router-dom'
import {
  LayoutDashboard,
  Users,
  ClipboardList,
  IndianRupee,
  UserCog,
  Briefcase,
  Settings,
  GraduationCap,
  FileText,
  MessageCircle,
  BarChart2,
  BookOpen,
  CreditCard,
  Bus,
  ShieldCheck,
  Library,
  LogOut,
} from 'lucide-react'
import './Sidebar.css'

const navItems = [
  { to: '/', icon: LayoutDashboard, label: 'Dashboard' },
  { to: '/admissions', icon: ClipboardList, label: 'Admissions' },
  { to: '/students', icon: Users, label: 'Students' },
  { to: '/fees', icon: IndianRupee, label: 'Fees' },
  { to: '/teachers', icon: UserCog, label: 'Teachers' },
  { to: '/staff', icon: Briefcase, label: 'Staff' },
  { to: '/documents', icon: FileText, label: 'Documents' },
  { to: '/broadcasts', icon: MessageCircle, label: 'Broadcasts' },
  { to: '/results', icon: BarChart2, label: 'Results' },
  { to: '/homework', icon: BookOpen, label: 'Homework' },
  { to: '/transport', icon: Bus, label: 'Transport' },
  { to: '/rte', icon: ShieldCheck, label: 'RTE' },
  { to: '/books', icon: Library, label: 'Books' },
  { to: '/id-cards', icon: CreditCard, label: 'ID Cards' },
  { to: '/settings', icon: Settings, label: 'Settings' },
]

function Sidebar({ open, user, onLogout }) {
  const initials = user?.role ? user.role[0].toUpperCase() : 'A'

  return (
    <aside className={`sidebar ${open ? '' : 'sidebar--collapsed'}`}>
      <div className="sidebar__logo">
        <div className="sidebar__logo-icon">
          <GraduationCap size={24} />
        </div>
        {open && <span className="sidebar__logo-text">CampusDesk</span>}
      </div>

      <nav className="sidebar__nav">
        {navItems.map(({ to, icon: Icon, label }) => (
          <NavLink
            key={to}
            to={to}
            end={to === '/'}
            className={({ isActive }) =>
              `sidebar__link ${isActive ? 'sidebar__link--active' : ''}`
            }
          >
            <Icon size={20} />
            {open && <span>{label}</span>}
          </NavLink>
        ))}
      </nav>

      <div className="sidebar__footer">
        <div className="sidebar__user">
          <div className="sidebar__avatar">{initials}</div>
          {open && (
            <div className="sidebar__user-info">
              <span className="sidebar__user-name">{user?.role || 'User'}</span>
              <button className="sidebar__logout" onClick={onLogout} title="Sign out">
                <LogOut size={14} />
                <span>Sign out</span>
              </button>
            </div>
          )}
        </div>
        {!open && (
          <button className="sidebar__logout sidebar__logout--icon" onClick={onLogout} title="Sign out">
            <LogOut size={16} />
          </button>
        )}
      </div>
    </aside>
  )
}

export default Sidebar
