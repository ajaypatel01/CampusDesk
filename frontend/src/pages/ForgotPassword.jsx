import { useState } from 'react'
import { Link } from 'react-router-dom'
import { GraduationCap } from 'lucide-react'
import { usersApi } from '../services/api'
import './Login.css'

function ForgotPassword() {
  const [email, setEmail] = useState('')
  const [loading, setLoading] = useState(false)
  const [sent, setSent] = useState(false)
  const [error, setError] = useState('')

  async function handleSubmit(e) {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      await usersApi.requestPasswordReset(email)
      // Always shows the same message whether or not the email is
      // registered -- the backend never reveals which, on purpose.
      setSent(true)
    } catch (err) {
      setError(err.message || 'Something went wrong. Please try again.')
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
        <h1 className="login-title">Reset your password</h1>

        {sent ? (
          <>
            <p className="login-subtitle">
              If that email is registered, a reset link has been sent to it. The link
              expires in 30 minutes.
            </p>
            <p className="login-switch"><Link to="/login">Back to sign in</Link></p>
          </>
        ) : (
          <>
            <p className="login-subtitle">Enter your account email and we'll send you a reset link.</p>
            {error && <div className="login-error">{error}</div>}
            <form className="login-form" onSubmit={handleSubmit}>
              <label className="login-field">
                <span>Email</span>
                <input
                  type="email" required autoFocus
                  value={email} onChange={e => setEmail(e.target.value)}
                  placeholder="admin@school.com"
                />
              </label>
              <button type="submit" className="login-btn" disabled={loading}>
                {loading ? 'Sending...' : 'Send Reset Link'}
              </button>
            </form>
            <p className="login-switch"><Link to="/login">Back to sign in</Link></p>
          </>
        )}
      </div>
    </div>
  )
}

export default ForgotPassword
