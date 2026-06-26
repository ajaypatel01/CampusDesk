import { useState, useEffect } from 'react'
import { Plus, Trash2, Edit2, Save, X, Bus, MapPin, Users } from 'lucide-react'
import { useSchool } from '../services/SchoolContext'
import { vansApi, studentsApi } from '../services/api'
import './Transport.css'

function Transport() {
  const { currentSchool, currentYear } = useSchool()
  const [tab, setTab] = useState('vans')
  const [vans, setVans] = useState([])
  const [selectedVan, setSelectedVan] = useState(null)
  const [vanDetail, setVanDetail] = useState(null)
  const [assignments, setAssignments] = useState([])
  const [students, setStudents] = useState([])
  const [loading, setLoading] = useState(false)

  const [showVanModal, setShowVanModal] = useState(false)
  const [editingVan, setEditingVan] = useState(null)
  const [vanForm, setVanForm] = useState({ van_number: '', driver_name: '', driver_phone: '', capacity: 20, route_name: '', notes: '' })

  const [showRouteModal, setShowRouteModal] = useState(false)
  const [routeForm, setRouteForm] = useState({ stop_name: '', stop_order: 1, monthly_fee: 0 })

  const [showAssignModal, setShowAssignModal] = useState(false)
  const [assignForm, setAssignForm] = useState({ student_id: '', pickup_stop: '' })

  const [saving, setSaving] = useState(false)

  useEffect(() => {
    if (!currentSchool) return
    loadVans()
    studentsApi.list({ school_id: currentSchool.id, limit: 500 })
      .then(r => setStudents(r.items || [])).catch(() => {})
  }, [currentSchool])

  function loadVans() {
    if (!currentSchool) return
    setLoading(true)
    vansApi.list(currentSchool.id)
      .then(r => setVans(r.items || []))
      .catch(() => {})
      .finally(() => setLoading(false))
  }

  async function loadVanDetail(van) {
    setSelectedVan(van)
    setVanDetail(null)
    setAssignments([])
    try {
      const [detail, assgn] = await Promise.all([
        vansApi.get(van.id, currentYear ? { academic_year_id: currentYear.id } : {}),
        currentYear ? vansApi.listAssignments({ van_id: van.id, academic_year_id: currentYear.id }) : Promise.resolve({ items: [] }),
      ])
      setVanDetail(detail)
      setAssignments(assgn.items || [])
    } catch (err) {
      alert(err.message)
    }
  }

  function openCreateVan() {
    setEditingVan(null)
    setVanForm({ van_number: '', driver_name: '', driver_phone: '', capacity: 20, route_name: '', notes: '' })
    setShowVanModal(true)
  }

  function openEditVan(van) {
    setEditingVan(van)
    setVanForm({ van_number: van.van_number, driver_name: van.driver_name, driver_phone: van.driver_phone || '', capacity: van.capacity, route_name: van.route_name || '', notes: van.notes || '' })
    setShowVanModal(true)
  }

  async function handleSaveVan(e) {
    e.preventDefault()
    setSaving(true)
    try {
      if (editingVan) {
        await vansApi.update(editingVan.id, { ...vanForm, capacity: parseInt(vanForm.capacity, 10) || 20 })
      } else {
        await vansApi.create({ school_id: currentSchool.id, ...vanForm, capacity: parseInt(vanForm.capacity, 10) || 20 })
      }
      setShowVanModal(false)
      loadVans()
      if (selectedVan && editingVan?.id === selectedVan.id) loadVanDetail(selectedVan)
    } catch (err) { alert(err.message) }
    finally { setSaving(false) }
  }

  async function handleDeleteVan(id) {
    if (!confirm('Delete this van?')) return
    try {
      await vansApi.delete(id)
      setVans(prev => prev.filter(v => v.id !== id))
      if (selectedVan?.id === id) { setSelectedVan(null); setVanDetail(null) }
    } catch (err) { alert(err.message) }
  }

  async function handleAddRoute(e) {
    e.preventDefault()
    if (!selectedVan) return
    setSaving(true)
    try {
      await vansApi.addRoute(selectedVan.id, { ...routeForm, stop_order: parseInt(routeForm.stop_order, 10) || 1, monthly_fee: parseInt(routeForm.monthly_fee, 10) || 0 })
      setShowRouteModal(false)
      setRouteForm({ stop_name: '', stop_order: 1, monthly_fee: 0 })
      loadVanDetail(selectedVan)
    } catch (err) { alert(err.message) }
    finally { setSaving(false) }
  }

  async function handleDeleteRoute(vanId, routeId) {
    if (!confirm('Remove this stop?')) return
    try {
      await vansApi.deleteRoute(vanId, routeId)
      loadVanDetail(selectedVan)
    } catch (err) { alert(err.message) }
  }

  async function handleAssignStudent(e) {
    e.preventDefault()
    if (!selectedVan || !currentYear) return
    setSaving(true)
    try {
      await vansApi.assignStudent({ van_id: selectedVan.id, academic_year_id: currentYear.id, ...assignForm })
      setShowAssignModal(false)
      setAssignForm({ student_id: '', pickup_stop: '' })
      loadVanDetail(selectedVan)
    } catch (err) { alert(err.message) }
    finally { setSaving(false) }
  }

  async function handleRemoveAssignment(id) {
    if (!confirm('Remove this student from van?')) return
    try {
      await vansApi.removeAssignment(id)
      setAssignments(prev => prev.filter(a => a.id !== id))
    } catch (err) { alert(err.message) }
  }

  if (!currentSchool) return <p className="empty-text" style={{ padding: 32 }}>Select a school first.</p>

  const routes = vanDetail?.routes || []
  const assignedIds = new Set(assignments.map(a => a.student_id))

  return (
    <div className="transport-page">
      <div className="page-header">
        <div>
          <h1>Transport</h1>
          <p className="page-subtitle">Manage school vans, routes, and student assignments</p>
        </div>
        <button className="btn btn--primary" onClick={openCreateVan}>
          <Plus size={16} /> Add Van
        </button>
      </div>

      <div className="transport-layout">
        {/* Van list */}
        <div className="van-list">
          {loading && <p className="empty-text">Loading...</p>}
          {!loading && vans.length === 0 && <p className="empty-text">No vans yet. Add one to get started.</p>}
          {vans.map(van => (
            <div
              key={van.id}
              className={`van-card ${selectedVan?.id === van.id ? 'van-card--active' : ''}`}
              onClick={() => loadVanDetail(van)}
            >
              <div className="van-card__icon"><Bus size={20} /></div>
              <div className="van-card__info">
                <strong>{van.van_number}</strong>
                <span className="van-card__meta">{van.driver_name}</span>
                {van.route_name && <span className="van-card__route">{van.route_name}</span>}
              </div>
              <div className="van-card__actions">
                <span className={`badge badge--${van.is_active ? 'success' : 'muted'}`} style={{ fontSize: '11px' }}>
                  {van.is_active ? 'Active' : 'Inactive'}
                </span>
                <button className="btn-icon" onClick={e => { e.stopPropagation(); openEditVan(van) }}><Edit2 size={14} /></button>
                <button className="btn-icon btn-icon--danger" onClick={e => { e.stopPropagation(); handleDeleteVan(van.id) }}><Trash2 size={14} /></button>
              </div>
            </div>
          ))}
        </div>

        {/* Van detail */}
        {selectedVan && (
          <div className="van-detail">
            <div className="van-detail__header">
              <div>
                <h2>{selectedVan.van_number}</h2>
                <p className="page-subtitle">Driver: {selectedVan.driver_name} {selectedVan.driver_phone && `· ${selectedVan.driver_phone}`} · Capacity: {selectedVan.capacity}</p>
              </div>
            </div>

            <div className="van-tabs">
              {['routes', 'students'].map(t => (
                <button key={t} className={`van-tab ${tab === t ? 'van-tab--active' : ''}`} onClick={() => setTab(t)}>
                  {t === 'routes' ? <><MapPin size={14} /> Routes</> : <><Users size={14} /> Students</>}
                </button>
              ))}
            </div>

            {tab === 'routes' && (
              <div>
                <div className="van-section-header">
                  <h3>Stops / Routes</h3>
                  <button className="btn btn--primary btn--sm" onClick={() => setShowRouteModal(true)}>
                    <Plus size={14} /> Add Stop
                  </button>
                </div>
                <div className="table-card">
                  <table className="data-table">
                    <thead><tr><th>#</th><th>Stop Name</th><th>Monthly Fee (₹)</th><th></th></tr></thead>
                    <tbody>
                      {routes.length === 0 ? (
                        <tr><td colSpan={4} className="data-table__empty">No stops added</td></tr>
                      ) : [...routes].sort((a, b) => a.stop_order - b.stop_order).map(rt => (
                        <tr key={rt.id}>
                          <td className="data-table__muted">{rt.stop_order}</td>
                          <td>{rt.stop_name}</td>
                          <td>₹{rt.monthly_fee.toLocaleString('en-IN')}</td>
                          <td>
                            <button className="btn btn--outline btn--sm" onClick={() => handleDeleteRoute(selectedVan.id, rt.id)}>
                              <Trash2 size={13} />
                            </button>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            )}

            {tab === 'students' && (
              <div>
                <div className="van-section-header">
                  <h3>Student Assignments {currentYear ? `(${currentYear.name})` : ''}</h3>
                  {currentYear && (
                    <button className="btn btn--primary btn--sm" onClick={() => setShowAssignModal(true)}>
                      <Plus size={14} /> Assign Student
                    </button>
                  )}
                </div>
                {!currentYear && <p className="empty-text">Select an academic year to manage assignments.</p>}
                <div className="table-card">
                  <table className="data-table">
                    <thead><tr><th>Student</th><th>Pickup Stop</th><th></th></tr></thead>
                    <tbody>
                      {assignments.length === 0 ? (
                        <tr><td colSpan={3} className="data-table__empty">No students assigned</td></tr>
                      ) : assignments.map(a => {
                        const stu = students.find(s => s.id === a.student_id)
                        return (
                          <tr key={a.id}>
                            <td>
                              <div>{stu ? `${stu.first_name} ${stu.last_name}` : a.student_id}</div>
                              {stu && <div className="data-table__muted" style={{ fontSize: '12px' }}>{stu.student_code}</div>}
                            </td>
                            <td>{a.pickup_stop || '-'}</td>
                            <td>
                              <button className="btn btn--outline btn--sm" onClick={() => handleRemoveAssignment(a.id)}>
                                <Trash2 size={13} />
                              </button>
                            </td>
                          </tr>
                        )
                      })}
                    </tbody>
                  </table>
                </div>
              </div>
            )}
          </div>
        )}

        {!selectedVan && vans.length > 0 && (
          <div className="van-detail van-detail--empty">
            <Bus size={48} />
            <p>Select a van to manage its routes and students</p>
          </div>
        )}
      </div>

      {/* Van modal */}
      {showVanModal && (
        <div className="modal-overlay" onClick={() => setShowVanModal(false)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <h2>{editingVan ? 'Edit Van' : 'Add Van'}</h2>
            <form className="modal__form" onSubmit={handleSaveVan}>
              <div className="form-row">
                <label className="form-field"><span>Van Number *</span><input required value={vanForm.van_number} onChange={e => setVanForm({ ...vanForm, van_number: e.target.value })} placeholder="e.g. MP09-1234" /></label>
                <label className="form-field"><span>Capacity</span><input type="number" min="1" value={vanForm.capacity} onChange={e => setVanForm({ ...vanForm, capacity: e.target.value })} /></label>
              </div>
              <div className="form-row">
                <label className="form-field"><span>Driver Name *</span><input required value={vanForm.driver_name} onChange={e => setVanForm({ ...vanForm, driver_name: e.target.value })} /></label>
                <label className="form-field"><span>Driver Phone</span><input value={vanForm.driver_phone} onChange={e => setVanForm({ ...vanForm, driver_phone: e.target.value })} /></label>
              </div>
              <label className="form-field"><span>Route Name</span><input value={vanForm.route_name} onChange={e => setVanForm({ ...vanForm, route_name: e.target.value })} placeholder="e.g. North Route" /></label>
              <label className="form-field"><span>Notes</span><textarea rows={2} value={vanForm.notes} onChange={e => setVanForm({ ...vanForm, notes: e.target.value })} /></label>
              <div className="modal__actions">
                <button type="button" className="btn btn--outline" onClick={() => setShowVanModal(false)}>Cancel</button>
                <button type="submit" className="btn btn--primary" disabled={saving}>{saving ? 'Saving...' : editingVan ? 'Update' : 'Add Van'}</button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Route modal */}
      {showRouteModal && (
        <div className="modal-overlay" onClick={() => setShowRouteModal(false)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <h2>Add Stop</h2>
            <form className="modal__form" onSubmit={handleAddRoute}>
              <label className="form-field"><span>Stop Name *</span><input required value={routeForm.stop_name} onChange={e => setRouteForm({ ...routeForm, stop_name: e.target.value })} placeholder="e.g. Main Market" /></label>
              <div className="form-row">
                <label className="form-field"><span>Order</span><input type="number" min="1" value={routeForm.stop_order} onChange={e => setRouteForm({ ...routeForm, stop_order: e.target.value })} /></label>
                <label className="form-field"><span>Monthly Fee (₹)</span><input type="number" min="0" value={routeForm.monthly_fee} onChange={e => setRouteForm({ ...routeForm, monthly_fee: e.target.value })} /></label>
              </div>
              <div className="modal__actions">
                <button type="button" className="btn btn--outline" onClick={() => setShowRouteModal(false)}>Cancel</button>
                <button type="submit" className="btn btn--primary" disabled={saving}>{saving ? 'Saving...' : 'Add Stop'}</button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Assign student modal */}
      {showAssignModal && (
        <div className="modal-overlay" onClick={() => setShowAssignModal(false)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <h2>Assign Student to Van</h2>
            <form className="modal__form" onSubmit={handleAssignStudent}>
              <label className="form-field">
                <span>Student *</span>
                <select required value={assignForm.student_id} onChange={e => setAssignForm({ ...assignForm, student_id: e.target.value })}>
                  <option value="">Select student...</option>
                  {students.filter(s => !assignedIds.has(s.id)).map(s => (
                    <option key={s.id} value={s.id}>{s.first_name} {s.last_name} ({s.student_code})</option>
                  ))}
                </select>
              </label>
              <label className="form-field">
                <span>Pickup Stop</span>
                {routes.length > 0 ? (
                  <select value={assignForm.pickup_stop} onChange={e => setAssignForm({ ...assignForm, pickup_stop: e.target.value })}>
                    <option value="">None / Custom</option>
                    {[...routes].sort((a, b) => a.stop_order - b.stop_order).map(r => (
                      <option key={r.id} value={r.stop_name}>{r.stop_name}</option>
                    ))}
                  </select>
                ) : (
                  <input value={assignForm.pickup_stop} onChange={e => setAssignForm({ ...assignForm, pickup_stop: e.target.value })} placeholder="Stop name" />
                )}
              </label>
              <div className="modal__actions">
                <button type="button" className="btn btn--outline" onClick={() => setShowAssignModal(false)}>Cancel</button>
                <button type="submit" className="btn btn--primary" disabled={saving}>{saving ? 'Assigning...' : 'Assign'}</button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}

export default Transport
