import { useState, useEffect, useMemo } from 'react'
import { Search, Download, Plus, X, Receipt } from 'lucide-react'
import { useSchool } from '../services/SchoolContext'
import { vouchersApi } from '../services/api'
import './Vouchers.css'

function fmt(d) {
  if (!d) return '—'
  return new Date(d).toLocaleDateString('en-IN')
}

function Vouchers() {
  const { currentSchool } = useSchool()
  const [vouchers, setVouchers] = useState([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [search, setSearch] = useState('')
  const [fromDate, setFromDate] = useState('')
  const [toDate, setToDate] = useState('')
  const [showModal, setShowModal] = useState(false)
  const [form, setForm] = useState({ date: '', account_name: '', payee: '', amount: '', description: '', mode_of_payment: 'Cash' })
  const [saving, setSaving] = useState(false)

  function load() {
    if (!currentSchool) return
    setLoading(true)
    vouchersApi.list({ school_id: currentSchool.id, from: fromDate || undefined, to: toDate || undefined, limit: 1000 })
      .then(res => { setVouchers(res.items || []); setTotal(res.total || 0) })
      .catch(() => setVouchers([]))
      .finally(() => setLoading(false))
  }

  useEffect(() => { load() }, [currentSchool, fromDate, toDate])

  const filtered = useMemo(() => {
    if (!search) return vouchers
    const q = search.toLowerCase()
    return vouchers.filter(v =>
      v.account_name.toLowerCase().includes(q) ||
      (v.payee || '').toLowerCase().includes(q) ||
      (v.description || '').toLowerCase().includes(q)
    )
  }, [vouchers, search])

  const totalAmount = useMemo(() => filtered.reduce((s, v) => s + (v.amount || 0), 0), [filtered])

  function exportCSV() {
    const headers = ['Date', 'Account Name', 'To (Payee)', 'Amount', 'Description', 'Mode']
    const rows = filtered.map(v => [
      new Date(v.date).toLocaleDateString('en-IN'),
      `"${v.account_name}"`,
      `"${v.payee || ''}"`,
      v.amount,
      `"${v.description || ''}"`,
      v.mode_of_payment || '',
    ].join(','))
    const csv = [headers.join(','), ...rows, `,,Total,${totalAmount},,`].join('\n')
    const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a'); a.href = url; a.download = 'vouchers.csv'; a.click()
    URL.revokeObjectURL(url)
  }

  async function handleCreate(e) {
    e.preventDefault()
    setSaving(true)
    try {
      const payload = {
        school_id: currentSchool.id,
        date: form.date,
        account_name: form.account_name,
        payee: form.payee || null,
        amount: parseFloat(form.amount) || 0,
        description: form.description || null,
        mode_of_payment: form.mode_of_payment || null,
      }
      await vouchersApi.create(payload)
      setShowModal(false)
      setForm({ date: '', account_name: '', payee: '', amount: '', description: '', mode_of_payment: 'Cash' })
      load()
    } catch (err) { alert(err.message) }
    finally { setSaving(false) }
  }

  if (!currentSchool) return <p className="empty-text">Select a school first.</p>

  return (
    <div className="vouchers-page">
      <div className="page-header">
        <div>
          <h1>Expense Vouchers</h1>
          <p className="page-subtitle">School expense register — {total} entries</p>
        </div>
        <div className="vouchers-header-actions">
          <button className="btn btn--outline" onClick={exportCSV}><Download size={16} /> Export CSV</button>
          <button className="btn btn--primary" onClick={() => setShowModal(true)}><Plus size={16} /> Add Voucher</button>
        </div>
      </div>

      <div className="vouchers-summary">
        <div className="vsumm-card">
          <span className="vsumm-label">Total Entries</span>
          <span className="vsumm-value">{filtered.length}</span>
        </div>
        <div className="vsumm-card vsumm-card--red">
          <span className="vsumm-label">Total Expense</span>
          <span className="vsumm-value">₹{totalAmount.toLocaleString('en-IN', { maximumFractionDigits: 0 })}</span>
        </div>
      </div>

      <div className="page-filters">
        <div className="filter-search">
          <Search size={18} />
          <input placeholder="Search account, payee, description..." value={search} onChange={e => setSearch(e.target.value)} />
        </div>
        <div className="voucher-date-filter">
          <label>From</label>
          <input type="date" value={fromDate} onChange={e => setFromDate(e.target.value)} />
        </div>
        <div className="voucher-date-filter">
          <label>To</label>
          <input type="date" value={toDate} onChange={e => setToDate(e.target.value)} />
        </div>
        {(fromDate || toDate || search) && (
          <button className="filter-clear" onClick={() => { setSearch(''); setFromDate(''); setToDate('') }}>
            <X size={14} /> Clear
          </button>
        )}
      </div>

      <div className="page-count">Showing {filtered.length} of {vouchers.length} vouchers</div>

      {loading ? <p className="loading-text">Loading...</p> : filtered.length === 0 ? (
        <p className="empty-text">No vouchers found</p>
      ) : (
        <div className="table-card">
          <table className="data-table">
            <thead>
              <tr>
                <th>#</th>
                <th>Date</th>
                <th>Account Name</th>
                <th>To (Payee)</th>
                <th>Description</th>
                <th>Mode</th>
                <th style={{textAlign:'right'}}>Amount</th>
              </tr>
            </thead>
            <tbody>
              {filtered.map((v, i) => (
                <tr key={v.id}>
                  <td className="data-table__muted">{i + 1}</td>
                  <td className="data-table__muted">{fmt(v.date)}</td>
                  <td>
                    <span className="voucher-account"><Receipt size={13} /> {v.account_name}</span>
                  </td>
                  <td className="data-table__muted">{v.payee || '—'}</td>
                  <td className="data-table__muted voucher-desc">{v.description || '—'}</td>
                  <td>
                    <span className={`voucher-mode-badge voucher-mode--${(v.mode_of_payment || 'cash').toLowerCase()}`}>
                      {v.mode_of_payment || '—'}
                    </span>
                  </td>
                  <td style={{textAlign:'right'}} className="voucher-amount">
                    ₹{Number(v.amount).toLocaleString('en-IN', { maximumFractionDigits: 0 })}
                  </td>
                </tr>
              ))}
            </tbody>
            <tfoot>
              <tr className="vouchers-total-row">
                <td colSpan={6} style={{textAlign:'right', fontWeight:600}}>Total</td>
                <td style={{textAlign:'right'}} className="voucher-amount voucher-amount--total">
                  ₹{totalAmount.toLocaleString('en-IN', { maximumFractionDigits: 0 })}
                </td>
              </tr>
            </tfoot>
          </table>
        </div>
      )}

      {showModal && (
        <div className="modal-overlay" onClick={() => setShowModal(false)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <h2>Add Voucher</h2>
            <form className="modal__form" onSubmit={handleCreate}>
              <div className="form-row">
                <label className="form-field"><span>Date *</span><input type="date" required value={form.date} onChange={e => setForm({...form, date: e.target.value})} /></label>
                <label className="form-field"><span>Amount *</span><input type="number" required min="0" step="0.01" value={form.amount} onChange={e => setForm({...form, amount: e.target.value})} /></label>
              </div>
              <label className="form-field"><span>Account Name *</span><input required value={form.account_name} onChange={e => setForm({...form, account_name: e.target.value})} /></label>
              <label className="form-field"><span>Payee (To)</span><input value={form.payee} onChange={e => setForm({...form, payee: e.target.value})} /></label>
              <label className="form-field"><span>Description</span><input value={form.description} onChange={e => setForm({...form, description: e.target.value})} /></label>
              <label className="form-field">
                <span>Mode of Payment</span>
                <select value={form.mode_of_payment} onChange={e => setForm({...form, mode_of_payment: e.target.value})}>
                  <option value="Cash">Cash</option>
                  <option value="Online">Online</option>
                  <option value="Cheque">Cheque</option>
                  <option value="UPI">UPI</option>
                </select>
              </label>
              <div className="modal__actions">
                <button type="button" className="btn btn--outline" onClick={() => setShowModal(false)}>Cancel</button>
                <button type="submit" className="btn btn--primary" disabled={saving}>{saving ? 'Saving...' : 'Save Voucher'}</button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}

export default Vouchers
