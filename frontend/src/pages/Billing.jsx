import { useState, useEffect } from 'react'
import { useOutletContext } from 'react-router-dom'
import { CircleDollarSign, Check, X } from 'lucide-react'
import { useSchool } from '../services/SchoolContext'
import { billingApi } from '../services/api'
import './Billing.css'

const STATUS_LABELS = {
  created: 'Awaiting payment', authenticated: 'Awaiting first charge', active: 'Active',
  pending: 'Payment pending', halted: 'Payment failed -- action needed', cancelled: 'Cancelled',
  completed: 'Completed', expired: 'Expired',
}

function formatPaise(paise) {
  return `₹${(paise / 100).toLocaleString('en-IN')}`
}

function loadRazorpayScript() {
  if (window.Razorpay) return Promise.resolve()
  return new Promise((resolve, reject) => {
    const script = document.createElement('script')
    script.src = 'https://checkout.razorpay.com/v1/checkout.js'
    script.onload = resolve
    script.onerror = () => reject(new Error('Could not load Razorpay checkout -- check your connection and try again.'))
    document.body.appendChild(script)
  })
}

export default function Billing() {
  const { user } = useOutletContext() || {}
  const { currentSchool } = useSchool()
  const isSuperAdmin = user?.role === 'super_admin'

  const [plans, setPlans] = useState([])
  const [subscription, setSubscription] = useState(null)
  const [payments, setPayments] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [subscribing, setSubscribing] = useState('') // `${planId}-${cycle}` while in flight
  const [razorpayKeyId, setRazorpayKeyId] = useState('')

  useEffect(() => {
    if (!currentSchool) return
    setLoading(true)
    Promise.all([
      billingApi.listPlans(),
      billingApi.getSubscription(currentSchool.id).catch(() => ({ subscription: null, payments: [] })),
    ]).then(([plansRes, subRes]) => {
      setPlans(plansRes.items || [])
      setRazorpayKeyId(plansRes.razorpay_key_id || '')
      setSubscription(subRes.subscription || null)
      setPayments(subRes.payments || [])
    }).catch(err => setError(err.message)).finally(() => setLoading(false))
  }, [currentSchool])

  async function handleSubscribe(plan, cycle) {
    setError('')
    setSubscribing(`${plan.id}-${cycle}`)
    try {
      const result = await billingApi.subscribe({ school_id: currentSchool.id, plan_id: plan.id, billing_cycle: cycle })
      await loadRazorpayScript()
      const rzp = new window.Razorpay({
        key: result.razorpay_key_id || razorpayKeyId,
        subscription_id: result.razorpay_subscription_id,
        name: 'CampusDesk',
        description: `${plan.name} -- ${cycle === 'annual' ? 'Annual' : 'Monthly'} subscription`,
        theme: { color: '#4f46e5' },
        handler: async (response) => {
          try {
            await billingApi.confirmCheckout({
              razorpay_subscription_id: response.razorpay_subscription_id,
              razorpay_payment_id: response.razorpay_payment_id,
              razorpay_signature: response.razorpay_signature,
            })
            const subRes = await billingApi.getSubscription(currentSchool.id)
            setSubscription(subRes.subscription || null)
            setPayments(subRes.payments || [])
          } catch (err) {
            setError(`Payment went through, but confirming it failed: ${err.message}. It'll still update shortly once Razorpay's own webhook arrives.`)
          }
        },
        modal: { ondismiss: () => setSubscribing('') },
      })
      rzp.on('payment.failed', (resp) => setError(resp.error?.description || 'Payment failed'))
      rzp.open()
    } catch (err) {
      setError(err.message)
    } finally {
      setSubscribing('')
    }
  }

  async function handleCancel(atCycleEnd) {
    if (!confirm(atCycleEnd
      ? 'Cancel at the end of the current billing period? You’ll keep access until then.'
      : 'Cancel immediately? This stops the subscription right away.')) return
    setError('')
    try {
      await billingApi.cancel({ school_id: currentSchool.id, cancel_at_cycle_end: atCycleEnd })
      const subRes = await billingApi.getSubscription(currentSchool.id)
      setSubscription(subRes.subscription || null)
    } catch (err) { setError(err.message) }
  }

  if (loading) return <p className="loading-text">Loading...</p>

  const currentPlan = subscription ? plans.find(p => p.id === subscription.plan_id) : null

  return (
    <div className="billing-page">
      <div className="page-header">
        <div>
          <h1>Billing</h1>
          <p className="page-subtitle">Your CampusDesk subscription -- separate from the Fees module, which is for collecting fees from your own students.</p>
        </div>
      </div>

      {error && <p className="doc-msg doc-msg--error">{error}</p>}

      {subscription && subscription.status !== 'cancelled' ? (
        <div className="billing-card billing-card--current">
          <div className="billing-card__header">
            <div>
              <span className="billing-card__plan-name">{currentPlan?.name || 'Current plan'}</span>
              <span className={`billing-status billing-status--${subscription.status}`}>
                {STATUS_LABELS[subscription.status] || subscription.status}
              </span>
            </div>
            <span className="settings-list__meta">{subscription.billing_cycle === 'annual' ? 'Billed annually' : 'Billed monthly'}</span>
          </div>
          {subscription.current_period_end && (
            <p className="settings-list__meta">
              {subscription.cancel_at_cycle_end ? 'Access ends' : 'Next renewal'}: {new Date(subscription.current_period_end).toLocaleDateString()}
            </p>
          )}
          {!subscription.cancel_at_cycle_end && (
            <div className="billing-card__actions">
              <button className="btn btn--outline btn--sm" onClick={() => handleCancel(true)}>Cancel at period end</button>
              <button className="btn btn--outline btn--sm" onClick={() => handleCancel(false)}>Cancel immediately</button>
            </div>
          )}
        </div>
      ) : (
        <p className="empty-text">No active subscription -- choose a plan below.</p>
      )}

      <h2 style={{ marginTop: '28px', marginBottom: '14px' }}>
        <CircleDollarSign size={18} style={{ verticalAlign: '-3px', marginRight: '6px' }} />
        Plans
      </h2>
      {plans.length === 0 ? (
        <p className="empty-text">No plans configured yet{isSuperAdmin ? ' -- add one below' : ''}.</p>
      ) : (
        <div className="billing-plans">
          {plans.map(plan => (
            <div key={plan.id} className="billing-plan-card">
              <h3>{plan.name}</h3>
              {plan.description && <p className="settings-list__meta">{plan.description}</p>}
              <ul className="billing-plan-card__features">
                {(plan.features || []).map((f, i) => (
                  <li key={i}><Check size={14} /> {f}</li>
                ))}
                {plan.max_students && <li><Check size={14} /> Up to {plan.max_students} students</li>}
              </ul>
              <div className="billing-plan-card__prices">
                <div>
                  <strong>{formatPaise(plan.price_monthly_paise)}</strong>/mo
                  <button
                    className="btn btn--primary btn--sm"
                    disabled={!!subscribing || !plan.razorpay_plan_id_monthly}
                    onClick={() => handleSubscribe(plan, 'monthly')}
                    title={!plan.razorpay_plan_id_monthly ? 'Not available yet -- monthly billing isn’t set up for this plan' : ''}
                  >
                    {subscribing === `${plan.id}-monthly` ? 'Opening checkout...' : 'Subscribe monthly'}
                  </button>
                </div>
                <div>
                  <strong>{formatPaise(plan.price_annual_paise)}</strong>/yr
                  <button
                    className="btn btn--outline btn--sm"
                    disabled={!!subscribing || !plan.razorpay_plan_id_annual}
                    onClick={() => handleSubscribe(plan, 'annual')}
                    title={!plan.razorpay_plan_id_annual ? 'Not available yet -- annual billing isn’t set up for this plan' : ''}
                  >
                    {subscribing === `${plan.id}-annual` ? 'Opening checkout...' : 'Subscribe annually'}
                  </button>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      {payments.length > 0 && (
        <>
          <h2 style={{ marginTop: '28px', marginBottom: '14px' }}>Payment History</h2>
          <div className="table-card">
            <table className="data-table">
              <thead><tr><th>Date</th><th>Amount</th><th>Status</th></tr></thead>
              <tbody>
                {payments.map(p => (
                  <tr key={p.id}>
                    <td>{p.paid_at ? new Date(p.paid_at).toLocaleDateString() : new Date(p.created_at).toLocaleDateString()}</td>
                    <td>{formatPaise(p.amount_paise)}</td>
                    <td>
                      {p.status === 'captured'
                        ? <span className="badge badge--success"><Check size={12} /> Paid</span>
                        : <span className="badge badge--danger"><X size={12} /> {p.status}</span>}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </>
      )}

      {isSuperAdmin && (
        <p className="settings-list__meta" style={{ marginTop: '24px' }}>
          Managing what plans exist (pricing, features, linking each one to its Razorpay plan ID) is done directly
          via the API for now -- ask your developer if you need a plan added or changed.
        </p>
      )}
    </div>
  )
}
