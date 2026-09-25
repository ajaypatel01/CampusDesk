import { NavLink } from 'react-router-dom'
import {
  LayoutDashboard,
  Users,
  ClipboardList,
  IndianRupee,
  UserCog,
  Briefcase,
  FileMinus,
  Receipt,
  BookOpen as LedgerIcon,
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
  Wallet,
  PieChart,
  ClipboardCheck,
} from 'lucide-react'
import './Sidebar.css'

const navItems = [
  { to: '/', icon: LayoutDashboard, label: 'Dashboard', deny: ['parent'] },
  { to: '/my-ward', icon: Users, label: 'My Ward', only: ['parent'] },
  { to: '/admissions', icon: ClipboardList, label: 'Admissions', deny: ['parent'] },
  { to: '/students', icon: Users, label: 'Students', deny: ['parent'] },
  { to: '/fees', icon: IndianRupee, label: 'Fees', deny: ['parent'] },
  { to: '/teachers', icon: UserCog, label: 'Teachers', deny: ['registrar', 'parent'] },
  { to: '/staff', icon: Briefcase, label: 'Staff', deny: ['registrar', 'parent'] },
  { to: '/ledger', icon: LedgerIcon, label: 'Fee Ledger', deny: ['parent'] },
  { to: '/fee-report', icon: PieChart, label: 'Fee Report', deny: ['registrar', 'parent'] },
  { to: '/tc-records', icon: FileMinus, label: 'TC Records', deny: ['parent'] },
  { to: '/udise-checklist', icon: ClipboardCheck, label: 'UDISE+ Checklist', deny: ['parent'] },
  { to: '/vouchers', icon: Receipt, label: 'Vouchers', deny: ['parent'] },
  { to: '/documents', icon: FileText, label: 'Documents', deny: ['registrar', 'parent'] },
  { to: '/broadcasts', icon: MessageCircle, label: 'Broadcasts', deny: ['registrar', 'parent'] },
  { to: '/results', icon: BarChart2, label: 'Results', deny: ['parent'] },
  { to: '/homework', icon: BookOpen, label: 'Homework', deny: ['parent'] },
  { to: '/transport', icon: Bus, label: 'Transport', deny: ['parent'] },
  { to: '/rte', icon: ShieldCheck, label: 'RTE', deny: ['parent'] },
  { to: '/books', icon: Library, label: 'Books', deny: ['registrar', 'parent'] },
  { to: '/id-cards', icon: CreditCard, label: 'ID Cards', deny: ['registrar', 'parent'] },
  { to: '/payroll', icon: Wallet, label: 'Payroll', only: ['super_admin'] },
  { to: '/settings', icon: Settings, label: 'Settings' },
]

const roleLabel = {
  super_admin: 'Owner',
  school_admin: 'School Admin',
  teacher: 'Teacher',
  registrar: 'Registrar',
  parent: 'Parent',
}

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
        {navItems
          .filter(item => !item.deny?.includes(user?.role) && (!item.only || item.only.includes(user?.role)))
          .map(({ to, icon: Icon, label }) => (
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
              <span className="sidebar__user-name">{roleLabel[user?.role] || user?.role || 'User'}</span>
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
