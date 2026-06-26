import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { GraduationCap, Eye, EyeOff } from 'lucide-react'
import { usersApi } from '../services/api'
import { setToken } from '../services/api'
import './Login.css'

function Login({ onLogin }) {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const navigate = useNavigate()

  async function handleSubmit(e) {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      const res = await usersApi.login({ email, password })
      setToken(res.token)
      onLogin(res.user)
      navigate('/', { replace: true })
    } catch (err) {
      setError(err.message || 'Invalid email or password')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="login-page">
      <div className="login-card">
        <div className="login-logo">
          <div className="login-logo__icon"><GraduationCap size={32} /></div>
          <div className="login-logo__text">CampusDesk</div>
        </div>
        <h1 className="login-title">Welcome back</h1>
        <p className="login-subtitle">Sign in to your school account</p>

        {error && <div className="login-error">{error}</div>}

        <form className="login-form" onSubmit={handleSubmit}>
          <label className="login-field">
            <span>Email</span>
            <input
              type="email"
              required
              autoFocus
              value={email}
              onChange={e => setEmail(e.target.value)}
              placeholder="admin@school.com"
            />
          </label>
          <label className="login-field">
            <span>Password</span>
            <div className="login-field__password">
              <input
                type={showPassword ? 'text' : 'password'}
                required
                value={password}
                onChange={e => setPassword(e.target.value)}
                placeholder="••••••••"
              />
              <button type="button" className="login-field__eye" onClick={() => setShowPassword(p => !p)}>
                {showPassword ? <EyeOff size={16} /> : <Eye size={16} />}
              </button>
            </div>
          </label>
          <button type="submit" className="login-btn" disabled={loading}>
            {loading ? 'Signing in...' : 'Sign In'}
          </button>
        </form>
      </div>
    </div>
  )
}

export default Login
