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
  History,
} from 'lucide-react'
import './Sidebar.css'

// A teacher's access is intentionally narrow: entering/viewing results for
// their own homeroom class (enforced server-side, not just hidden here),
// homework, their own salary, and basic account settings -- nothing else.
// Every other item below explicitly denies 'teacher' for that reason.
const navItems = [
  { to: '/', icon: LayoutDashboard, label: 'Dashboard', deny: ['parent', 'teacher'] },
  { to: '/my-ward', icon: Users, label: 'My Ward', only: ['parent'] },
  { to: '/admissions', icon: ClipboardList, label: 'Admissions', deny: ['parent', 'teacher'] },
  { to: '/students', icon: Users, label: 'Students', deny: ['parent', 'teacher'] },
  { to: '/fees', icon: IndianRupee, label: 'Fees', deny: ['parent', 'teacher'] },
  { to: '/teachers', icon: UserCog, label: 'Teachers', deny: ['registrar', 'parent', 'teacher'] },
  { to: '/staff', icon: Briefcase, label: 'Staff', deny: ['registrar', 'parent', 'teacher'] },
  { to: '/ledger', icon: LedgerIcon, label: 'Fee Ledger', deny: ['parent', 'teacher'] },
  { to: '/fee-report', icon: PieChart, label: 'Fee Report', deny: ['registrar', 'parent', 'teacher'] },
  { to: '/tc-records', icon: FileMinus, label: 'TC Records', deny: ['parent', 'teacher'] },
  { to: '/udise-checklist', icon: ClipboardCheck, label: 'UDISE+ Checklist', deny: ['parent', 'teacher'] },
  { to: '/vouchers', icon: Receipt, label: 'Vouchers', deny: ['parent', 'teacher'] },
  { to: '/documents', icon: FileText, label: 'Documents', deny: ['registrar', 'parent', 'teacher'] },
  { to: '/broadcasts', icon: MessageCircle, label: 'Broadcasts', deny: ['registrar', 'parent', 'teacher'] },
  { to: '/results', icon: BarChart2, label: 'Results', deny: ['parent'] },
  { to: '/homework', icon: BookOpen, label: 'Homework', deny: ['parent'] },
  { to: '/transport', icon: Bus, label: 'Transport', deny: ['parent', 'teacher'] },
  { to: '/rte', icon: ShieldCheck, label: 'RTE', deny: ['parent', 'teacher'] },
  { to: '/books', icon: Library, label: 'Books', deny: ['registrar', 'parent', 'teacher'] },
  { to: '/id-cards', icon: CreditCard, label: 'ID Cards', deny: ['registrar', 'parent', 'teacher'] },
  { to: '/payroll', icon: Wallet, label: 'Payroll', only: ['super_admin'] },
  { to: '/my-salary', icon: Wallet, label: 'My Salary', only: ['teacher'] },
  { to: '/activity', icon: History, label: 'Activity Log', only: ['super_admin', 'school_admin'] },
  { to: '/settings', icon: Settings, label: 'Settings' },
]

const roleLabel = {
  super_admin: 'Owner',
  school_admin: 'School Admin',
  teacher: 'Teacher',
  registrar: 'Registrar',
  parent: 'Parent',
}

function Sidebar({ open, user, onLogout, onNavigate }) {
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
            onClick={onNavigate}
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
