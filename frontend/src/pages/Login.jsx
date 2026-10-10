import { useState } from 'react'
import { useNavigate, Link } from 'react-router-dom'
import { GraduationCap, Eye, EyeOff, Smartphone, KeyRound } from 'lucide-react'
import { usersApi } from '../services/api'
import { setToken } from '../services/api'
import './Login.css'

function Login({ onLogin }) {
  const [mode, setMode] = useState('password') // 'password' | 'otp'
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const navigate = useNavigate()

  // OTP login state
  const [phone, setPhone] = useState('')
  const [otp, setOtp] = useState('')
  const [otpSent, setOtpSent] = useState(false)
  const [otpChannel, setOtpChannel] = useState('') // 'sms' | 'whatsapp'
  const [otpSending, setOtpSending] = useState(false)

  async function handleSubmit(e) {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      const res = await usersApi.login({ email, password })
      setToken(res.token)
      onLogin()
      navigate('/', { replace: true })
    } catch (err) {
      setError(err.message || 'Invalid email or password')
    } finally {
      setLoading(false)
    }
  }

  function switchMode(next) {
    setMode(next)
    setError('')
    setOtpSent(false)
    setOtp('')
  }

  async function handleSendOtp(e) {
    e.preventDefault()
    setError('')
    setOtpSending(true)
    try {
      const res = await usersApi.requestOTPLogin(phone)
      setOtpSent(true)
      setOtpChannel(res?.channel || '')
    } catch (err) {
      setError(err.message || 'Could not send OTP')
    } finally {
      setOtpSending(false)
    }
  }

  async function handleVerifyOtp(e) {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      const res = await usersApi.verifyOTPLogin(phone, otp)
      setToken(res.token)
      onLogin()
      navigate('/', { replace: true })
    } catch (err) {
      setError(err.message || 'Invalid or expired OTP')
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

        {mode === 'password' ? (
          <form className="login-form" onSubmit={handleSubmit}>
            <label className="login-field">
              <span>Email or mobile number</span>
              <input
                type="text"
                inputMode="email"
                autoComplete="username"
                required
                autoFocus
                value={email}
                onChange={e => setEmail(e.target.value)}
                placeholder="you@school.com or 98765 43210"
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
            <p style={{ textAlign: 'right', margin: '-8px 0 4px' }}>
              <Link to="/forgot-password" style={{ fontSize: '13px' }}>Forgot password?</Link>
            </p>
            <button type="submit" className="login-btn" disabled={loading}>
              {loading ? 'Signing in...' : 'Sign In'}
            </button>
          </form>
        ) : (
          <form className="login-form" onSubmit={otpSent ? handleVerifyOtp : handleSendOtp}>
            <label className="login-field">
              <span>Mobile Number</span>
              <input
                type="tel"
                required
                autoFocus
                disabled={otpSent}
                value={phone}
                onChange={e => setPhone(e.target.value)}
                placeholder="98765 43210"
              />
            </label>
            {otpSent && otpChannel && (
              <p className="empty-text" style={{ margin: 0 }}>
                We sent a 6-digit code {otpChannel === 'whatsapp' ? 'on WhatsApp' : 'by SMS'} to this number.
              </p>
            )}
            {otpSent && (
              <label className="login-field">
                <span>OTP</span>
                <input
                  type="text"
                  inputMode="numeric"
                  required
                  autoFocus
                  value={otp}
                  onChange={e => setOtp(e.target.value)}
                  placeholder="6-digit code"
                />
              </label>
            )}
            {otpSent && (
              <p style={{ textAlign: 'right', margin: '-8px 0 4px' }}>
                <button type="button" className="login-field__eye" style={{ position: 'static', fontSize: '13px' }} onClick={() => { setOtpSent(false); setOtp('') }}>
                  Use a different number
                </button>
              </p>
            )}
            <button type="submit" className="login-btn" disabled={loading || otpSending}>
              {otpSent ? (loading ? 'Verifying...' : 'Verify & Sign In') : (otpSending ? 'Sending...' : 'Send OTP')}
            </button>
          </form>
        )}

        <p className="login-switch" style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '6px' }}>
          {mode === 'password' ? (
            <button type="button" className="login-field__eye" style={{ position: 'static', display: 'inline-flex', alignItems: 'center', gap: '6px' }} onClick={() => switchMode('otp')}>
              <Smartphone size={14} /> Login with OTP instead
            </button>
          ) : (
            <button type="button" className="login-field__eye" style={{ position: 'static', display: 'inline-flex', alignItems: 'center', gap: '6px' }} onClick={() => switchMode('password')}>
              <KeyRound size={14} /> Login with password instead
            </button>
          )}
        </p>

        <p className="login-switch">
          New here? <Link to="/register">Create an account</Link>
        </p>
      </div>
    </div>
  )
}

export default Login
