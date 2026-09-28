import { useState } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import { GraduationCap, Eye, EyeOff } from 'lucide-react'
import { usersApi } from '../services/api'
import './Login.css'

function ResetPassword() {
  const [searchParams] = useSearchParams()
  const token = searchParams.get('token') || ''
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [loading, setLoading] = useState(false)
  const [done, setDone] = useState(false)
  const [error, setError] = useState('')
  const navigate = useNavigate()

  async function handleSubmit(e) {
    e.preventDefault()
    setError('')
    if (password.length < 6) {
      setError('Password must be at least 6 characters.')
      return
    }
    if (password !== confirmPassword) {
      setError('Passwords do not match.')
      return
    }
    setLoading(true)
    try {
      await usersApi.confirmPasswordReset(token, password)
      setDone(true)
      setTimeout(() => navigate('/login', { replace: true }), 2500)
    } catch (err) {
      setError(err.message || 'This reset link is invalid or has expired.')
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
        <h1 className="login-title">Set a new password</h1>

        {!token ? (
          <div className="login-error">This reset link is missing its token. Please use the link from your email.</div>
        ) : done ? (
          <p className="login-subtitle">Your password has been updated. Redirecting to sign in...</p>
        ) : (
          <>
            {error && <div className="login-error">{error}</div>}
            <form className="login-form" onSubmit={handleSubmit}>
              <label className="login-field">
                <span>New Password</span>
                <div className="login-field__password">
                  <input
                    type={showPassword ? 'text' : 'password'}
                    required autoFocus minLength={6}
                    value={password} onChange={e => setPassword(e.target.value)}
                    placeholder="••••••••"
                  />
                  <button type="button" className="login-field__eye" onClick={() => setShowPassword(p => !p)}>
                    {showPassword ? <EyeOff size={16} /> : <Eye size={16} />}
                  </button>
                </div>
              </label>
              <label className="login-field">
                <span>Confirm New Password</span>
                <input
                  type={showPassword ? 'text' : 'password'}
                  required minLength={6}
                  value={confirmPassword} onChange={e => setConfirmPassword(e.target.value)}
                  placeholder="••••••••"
                />
              </label>
              <button type="submit" className="login-btn" disabled={loading}>
                {loading ? 'Saving...' : 'Set New Password'}
              </button>
            </form>
          </>
        )}
        <p className="login-switch"><Link to="/login">Back to sign in</Link></p>
      </div>
    </div>
  )
}

export default ResetPassword
