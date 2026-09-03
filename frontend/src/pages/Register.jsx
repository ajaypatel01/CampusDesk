import { useState, useEffect } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { GraduationCap, Eye, EyeOff, CheckCircle2 } from 'lucide-react'
import { usersApi, schoolsApi } from '../services/api'
import './Login.css'

const ROLES = [
  { value: 'school_admin', label: 'School Admin' },
  { value: 'teacher', label: 'Teacher' },
  { value: 'registrar', label: 'Registrar' },
  { value: 'parent', label: 'Parent' },
]

function Register() {
  const [schools, setSchools] = useState([])
  const [form, setForm] = useState({
    first_name: '', last_name: '', email: '', password: '', confirm: '',
    role: 'teacher', school_id: '',
  })
  const [showPassword, setShowPassword] = useState(false)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [submitted, setSubmitted] = useState(false)
  const navigate = useNavigate()

  useEffect(() => {
    schoolsApi.publicList()
      .then(res => setSchools(res.items || []))
      .catch(() => setSchools([]))
  }, [])

  function update(field) {
    return e => setForm(f => ({ ...f, [field]: e.target.value }))
  }

  async function handleSubmit(e) {
    e.preventDefault()
    setError('')
    if (form.password !== form.confirm) {
      setError('Passwords do not match')
      return
    }
    if (form.password.length < 8) {
      setError('Password must be at least 8 characters')
      return
    }
    if (!form.school_id) {
      setError('Please select a school')
      return
    }
    setLoading(true)
    try {
      await usersApi.register({
        first_name: form.first_name,
        last_name: form.last_name,
        email: form.email,
        password: form.password,
        role: form.role,
        school_id: form.school_id,
      })
      setSubmitted(true)
    } catch (err) {
      setError(err.message || 'Registration failed')
    } finally {
      setLoading(false)
    }
  }

  if (submitted) {
    return (
      <div className="login-page">
        <div className="login-card">
          <div className="login-logo">
            <div className="login-logo__icon"><CheckCircle2 size={32} /></div>
            <div className="login-logo__text">CampusDesk</div>
          </div>
          <h1 className="login-title">Request submitted</h1>
          <p className="login-subtitle">
            Your account is pending approval. A school or super admin needs to review and
            approve your registration before you can sign in.
          </p>
          <button className="login-btn" onClick={() => navigate('/login', { replace: true })}>
            Back to sign in
          </button>
        </div>
      </div>
    )
  }

  return (
    <div className="login-page">
      <div className="login-card">
        <div className="login-logo">
          <div className="login-logo__icon"><GraduationCap size={32} /></div>
          <div className="login-logo__text">CampusDesk</div>
        </div>
        <h1 className="login-title">Create an account</h1>
        <p className="login-subtitle">Registration requires admin approval before you can sign in</p>

        {error && <div className="login-error">{error}</div>}

        <form className="login-form" onSubmit={handleSubmit}>
          <div className="login-field-row">
            <label className="login-field">
              <span>First name</span>
              <input required value={form.first_name} onChange={update('first_name')} placeholder="Jane" />
            </label>
            <label className="login-field">
              <span>Last name</span>
              <input required value={form.last_name} onChange={update('last_name')} placeholder="Doe" />
            </label>
          </div>

          <label className="login-field">
            <span>Email</span>
            <input type="email" required value={form.email} onChange={update('email')} placeholder="you@school.com" />
          </label>

          <label className="login-field">
            <span>School</span>
            <select required value={form.school_id} onChange={update('school_id')}>
              <option value="">Select your school</option>
              {schools.map(s => <option key={s.id} value={s.id}>{s.name} ({s.code})</option>)}
            </select>
          </label>

          <label className="login-field">
            <span>Role</span>
            <select value={form.role} onChange={update('role')}>
              {ROLES.map(r => <option key={r.value} value={r.value}>{r.label}</option>)}
            </select>
          </label>

          <label className="login-field">
            <span>Password</span>
            <div className="login-field__password">
              <input
                type={showPassword ? 'text' : 'password'}
                required
                minLength={8}
                value={form.password}
                onChange={update('password')}
                placeholder="At least 8 characters"
              />
              <button type="button" className="login-field__eye" onClick={() => setShowPassword(p => !p)}>
                {showPassword ? <EyeOff size={16} /> : <Eye size={16} />}
              </button>
            </div>
          </label>

          <label className="login-field">
            <span>Confirm password</span>
            <input
              type={showPassword ? 'text' : 'password'}
              required
              value={form.confirm}
              onChange={update('confirm')}
              placeholder="••••••••"
            />
          </label>

          <button type="submit" className="login-btn" disabled={loading}>
            {loading ? 'Submitting...' : 'Request account'}
          </button>
        </form>

        <p className="login-switch">
          Already have an account? <Link to="/login">Sign in</Link>
        </p>
      </div>
    </div>
  )
}

export default Register
