import { useState, useEffect } from 'react'
import { useParams, Link } from 'react-router-dom'
import { ArrowLeft, Plus, XCircle, IndianRupee, Download, MessageCircle } from 'lucide-react'
import { feesApi } from '../services/api'
import { useConfig } from '../services/ConfigContext'
import './FeeAccountDetail.css'

function FeeAccountDetail() {
  const { id } = useParams()
  const { whatsapp_enabled } = useConfig()
  const [account, setAccount] = useState(null)
  const [loading, setLoading] = useState(true)
  const [showPaymentModal, setShowPaymentModal] = useState(false)
  const [paymentForm, setPaymentForm] = useState({
    fee_type: 'tuition', amount: '', payment_mode: 'cash',
    installment_number: '', reference_number: '', notes: '', payment_date: new Date().toISOString().split('T')[0],
  })
  const [saving, setSaving] = useState(false)

  async function loadAccount() {
    setLoading(true)
    try {
      const data = await feesApi.getAccount(id)
      setAccount(data)
    } catch {}
    setLoading(false)
  }

  useEffect(() => { loadAccount() }, [id])

  async function handlePayment(e) {
    e.preventDefault()
    setSaving(true)
    try {
      await feesApi.recordPayment({
        student_fee_account_id: id,
        fee_type: paymentForm.fee_type,
        amount: parseInt(paymentForm.amount, 10),
        payment_mode: paymentForm.payment_mode,
        payment_date: paymentForm.payment_date || undefined,
        installment_number: paymentForm.installment_number ? parseInt(paymentForm.installment_number, 10) : undefined,
        reference_number: paymentForm.reference_number || undefined,
        notes: paymentForm.notes || undefined,
      })
      setShowPaymentModal(false)
      setPaymentForm({ fee_type: 'tuition', amount: '', payment_mode: 'cash', installment_number: '', reference_number: '', notes: '', payment_date: new Date().toISOString().split('T')[0] })
      loadAccount()
    } catch (err) {
      alert(err.message)
    } finally {
      setSaving(false)
    }
  }

  async function handleDownloadReceipt(paymentId) {
    try {
      const blob = await feesApi.downloadReceipt(paymentId)
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `receipt_${paymentId.substring(0, 8)}.pdf`
      a.click()
      URL.revokeObjectURL(url)
    } catch (err) {
      alert(err.message)
    }
  }

  async function handleWhatsAppReceipt(paymentId) {
    const phone = prompt('Enter WhatsApp number (with country code, e.g. 919876543210):')
    if (!phone) return
    try {
      await feesApi.sendReceiptWhatsApp(paymentId, phone)
      alert('Receipt sent via WhatsApp.')
    } catch (err) {
      alert(err.message)
    }
  }

  async function handleVoid(paymentId) {
    if (!confirm('Void this payment? This cannot be undone.')) return
    try {
      await feesApi.voidPayment(paymentId)
      loadAccount()
    } catch (err) {
      alert(err.message)
    }
  }

  function fmt(amt) {
    return new Intl.NumberFormat('en-IN', { style: 'currency', currency: 'INR', maximumFractionDigits: 0 }).format(amt)
  }

  if (loading) return <p className="loading-text">Loading...</p>
  if (!account) return <p className="empty-text">Fee account not found</p>

  return (
    <div className="fee-detail">
      <Link to="/fees" className="back-link"><ArrowLeft size={18} /> Back to Fees</Link>

      <div className="fee-detail__top">
        <div>
          <h1>{account.student_name}</h1>
          <span className="fee-detail__code">{account.student_code}</span>
          {account.grade_level_name && <span className="fee-detail__grade">{account.grade_level_name}</span>}
          {account.is_rte && <span className="badge badge--info">RTE</span>}
        </div>
        <button className="btn btn--primary" onClick={() => setShowPaymentModal(true)}>
          <Plus size={16} /> Record Payment
        </button>
      </div>

      <div className="fee-detail__summary">
        <div className="fee-detail__summary-item">
          <span>Tuition Fee</span>
          <strong>{fmt(account.tuition_fee)}</strong>
        </div>
        <div className="fee-detail__summary-item">
          <span>Discount</span>
          <strong>{fmt(account.discount_amount)}</strong>
          {account.discount_reason && <small>{account.discount_reason}</small>}
        </div>
        <div className="fee-detail__summary-item">
          <span>Van Fee</span>
          <strong>{fmt(account.van_fee)}</strong>
        </div>
        <div className="fee-detail__summary-item">
          <span>Previous Dues</span>
          <strong>{fmt(account.previous_year_dues)}</strong>
        </div>
        <div className="fee-detail__summary-item fee-detail__summary-item--highlight">
          <span>Total Due</span>
          <strong>{fmt(account.total_due)}</strong>
        </div>
        <div className="fee-detail__summary-item fee-detail__summary-item--green">
          <span>Total Paid</span>
          <strong>{fmt(account.total_paid)}</strong>
        </div>
        <div className={`fee-detail__summary-item ${account.balance_remaining > 0 ? 'fee-detail__summary-item--red' : 'fee-detail__summary-item--green'}`}>
          <span>Balance</span>
          <strong>{fmt(account.balance_remaining)}</strong>
        </div>
      </div>

      <div className="fee-detail__breakdown">
        <div className="fee-detail__breakdown-item">
          <span>Tuition Paid</span>
          <span className="fee-detail__breakdown-value">{fmt(account.tuition_paid)}</span>
        </div>
        <div className="fee-detail__breakdown-item">
          <span>Van Paid</span>
          <span className="fee-detail__breakdown-value">{fmt(account.van_paid)}</span>
        </div>
        <div className="fee-detail__breakdown-item">
          <span>Previous Dues Paid</span>
          <span className="fee-detail__breakdown-value">{fmt(account.previous_dues_paid)}</span>
        </div>
      </div>

      <div className="fee-detail__payments">
        <h2>Payment History</h2>
        {(!account.payments || account.payments.length === 0) ? (
          <p className="empty-text">No payments recorded yet</p>
        ) : (
          <table className="data-table">
            <thead>
              <tr>
                <th>Date</th>
                <th>Type</th>
                <th>Installment</th>
                <th>Amount</th>
                <th>Mode</th>
                <th>Reference</th>
                <th>Notes</th>
                <th>Status</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {account.payments.map(p => (
                <tr key={p.id} className={p.voided ? 'fee-detail__voided-row' : ''}>
                  <td className="data-table__muted">{new Date(p.payment_date).toLocaleDateString('en-IN')}</td>
                  <td><span className="badge badge--muted">{p.fee_type}</span></td>
                  <td className="data-table__muted">{p.installment_number || '-'}</td>
                  <td className="fees-page__paid">{fmt(p.amount)}</td>
                  <td className="data-table__muted">{p.payment_mode}</td>
                  <td className="data-table__muted">{p.reference_number || '-'}</td>
                  <td className="data-table__muted">{p.notes || '-'}</td>
                  <td>
                    {p.voided ? <span className="badge badge--danger">Voided</span> : <span className="badge badge--success">Active</span>}
                  </td>
                  <td style={{ display: 'flex', gap: '4px' }}>
                    {!p.voided && (
                      <>
                        <button className="btn btn--outline btn--sm" onClick={() => handleDownloadReceipt(p.id)} title="Download receipt">
                          <Download size={14} />
                        </button>
                        {whatsapp_enabled && (
                          <button className="btn btn--outline btn--sm" onClick={() => handleWhatsAppReceipt(p.id)} title="Send via WhatsApp">
                            <MessageCircle size={14} />
                          </button>
                        )}
                        <button className="btn btn--outline btn--sm" onClick={() => handleVoid(p.id)} title="Void payment">
                          <XCircle size={14} />
                        </button>
                      </>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      {showPaymentModal && (
        <div className="modal-overlay" onClick={() => setShowPaymentModal(false)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <h2>Record Payment</h2>
            <form className="modal__form" onSubmit={handlePayment}>
              <div className="form-row">
                <label className="form-field">
                  <span>Fee Type *</span>
                  <select value={paymentForm.fee_type} onChange={e => setPaymentForm({ ...paymentForm, fee_type: e.target.value })}>
                    <option value="tuition">Tuition</option>
                    <option value="van">Van</option>
                    <option value="previous_dues">Previous Dues</option>
                  </select>
                </label>
                <label className="form-field">
                  <span>Amount (INR) *</span>
                  <input type="number" required min="1" value={paymentForm.amount} onChange={e => setPaymentForm({ ...paymentForm, amount: e.target.value })} />
                </label>
              </div>
              <div className="form-row">
                <label className="form-field">
                  <span>Payment Mode</span>
                  <select value={paymentForm.payment_mode} onChange={e => setPaymentForm({ ...paymentForm, payment_mode: e.target.value })}>
                    <option value="cash">Cash</option>
                    <option value="online">Online</option>
                    <option value="cheque">Cheque</option>
                    <option value="upi">UPI</option>
                  </select>
                </label>
                <label className="form-field">
                  <span>Payment Date</span>
                  <input type="date" value={paymentForm.payment_date} onChange={e => setPaymentForm({ ...paymentForm, payment_date: e.target.value })} />
                </label>
              </div>
              <div className="form-row">
                <label className="form-field">
                  <span>Installment #</span>
                  <input type="number" min="1" value={paymentForm.installment_number} onChange={e => setPaymentForm({ ...paymentForm, installment_number: e.target.value })} placeholder="Optional" />
                </label>
                <label className="form-field">
                  <span>Reference Number</span>
                  <input value={paymentForm.reference_number} onChange={e => setPaymentForm({ ...paymentForm, reference_number: e.target.value })} placeholder="Transaction ID, cheque #" />
                </label>
              </div>
              <label className="form-field">
                <span>Notes</span>
                <textarea rows={2} value={paymentForm.notes} onChange={e => setPaymentForm({ ...paymentForm, notes: e.target.value })} />
              </label>
              <div className="modal__actions">
                <button type="button" className="btn btn--outline" onClick={() => setShowPaymentModal(false)}>Cancel</button>
                <button type="submit" className="btn btn--primary" disabled={saving}>
                  {saving ? 'Saving...' : 'Record Payment'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}

export default FeeAccountDetail
