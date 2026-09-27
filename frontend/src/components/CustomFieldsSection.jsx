import { useState, useEffect } from 'react'
import { Plus, Trash2 } from 'lucide-react'
import { customFieldsApi } from '../services/api'
import './CustomFieldsSection.css'

const FIELD_TYPES = [
  { value: 'string', label: 'Text' },
  { value: 'int', label: 'Whole number' },
  { value: 'float', label: 'Decimal number' },
  { value: 'boolean', label: 'Yes / No' },
  { value: 'enum', label: 'Choice list' },
]

// Renders a super_admin-only "extra fields" block for one entity instance
// (e.g. one student). entityType must be one already known to the backend
// (see allowedEntityTypes in internal/modules/customfields/service.go).
// scopeId is only needed when the same field means something different per
// context (e.g. a "student_result" field is scoped by academic year).
function CustomFieldsSection({ entityType, entityId, schoolId, user, scopeId, title = 'Extra Fields' }) {
  const isSuperAdmin = user?.role === 'super_admin'
  const [fields, setFields] = useState([])
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState({}) // definitionId -> bool
  const [showAddForm, setShowAddForm] = useState(false)
  const [addForm, setAddForm] = useState({ label: '', field_type: 'string', enum_options: '' })
  const [addSaving, setAddSaving] = useState(false)
  const [error, setError] = useState('')

  function load() {
    if (!isSuperAdmin || !schoolId || !entityId) return
    setLoading(true)
    customFieldsApi.listValues(schoolId, entityType, entityId, scopeId)
      .then(r => setFields(r.items || []))
      .catch(() => setFields([]))
      .finally(() => setLoading(false))
  }

  useEffect(() => { load() }, [isSuperAdmin, schoolId, entityType, entityId, scopeId])

  if (!isSuperAdmin) return null

  async function handleValueChange(field, rawValue) {
    setFields(prev => prev.map(f => f.id === field.id ? { ...f, value: rawValue } : f))
  }

  async function handleValueSave(field, rawValue) {
    setSaving(prev => ({ ...prev, [field.id]: true }))
    setError('')
    try {
      await customFieldsApi.upsertValue({
        definition_id: field.id, entity_id: entityId, scope_id: scopeId, value: rawValue,
      })
    } catch (err) {
      setError(err.message)
      load()
    } finally {
      setSaving(prev => ({ ...prev, [field.id]: false }))
    }
  }

  async function handleAddField(e) {
    e.preventDefault()
    setAddSaving(true)
    setError('')
    try {
      const enumOptions = addForm.field_type === 'enum'
        ? addForm.enum_options.split(',').map(s => s.trim()).filter(Boolean)
        : undefined
      await customFieldsApi.createDefinition({
        school_id: schoolId, entity_type: entityType, label: addForm.label,
        field_type: addForm.field_type, enum_options: enumOptions,
      })
      setShowAddForm(false)
      setAddForm({ label: '', field_type: 'string', enum_options: '' })
      load()
    } catch (err) {
      setError(err.message)
    } finally {
      setAddSaving(false)
    }
  }

  async function handleDeleteField(field) {
    if (!confirm(`Remove the "${field.label}" field? This deletes its value for every ${entityType.replace('_', ' ')}, not just this one.`)) return
    try {
      await customFieldsApi.deleteDefinition(field.id)
      load()
    } catch (err) {
      alert(err.message)
    }
  }

  function renderInput(field) {
    const disabled = saving[field.id]
    if (field.field_type === 'boolean') {
      return (
        <select
          disabled={disabled}
          value={field.value || ''}
          onChange={e => { handleValueChange(field, e.target.value); handleValueSave(field, e.target.value) }}
        >
          <option value="">-</option>
          <option value="true">Yes</option>
          <option value="false">No</option>
        </select>
      )
    }
    if (field.field_type === 'enum') {
      return (
        <select
          disabled={disabled}
          value={field.value || ''}
          onChange={e => { handleValueChange(field, e.target.value); handleValueSave(field, e.target.value) }}
        >
          <option value="">-</option>
          {(field.enum_options || []).map(opt => <option key={opt} value={opt}>{opt}</option>)}
        </select>
      )
    }
    return (
      <input
        type={field.field_type === 'int' || field.field_type === 'float' ? 'number' : 'text'}
        step={field.field_type === 'float' ? 'any' : undefined}
        disabled={disabled}
        value={field.value || ''}
        onChange={e => handleValueChange(field, e.target.value)}
        onBlur={e => handleValueSave(field, e.target.value)}
      />
    )
  }

  return (
    <div className="custom-fields">
      <div className="custom-fields__header">
        <h3>{title}</h3>
        <button type="button" className="btn btn--outline btn--sm" onClick={() => setShowAddForm(v => !v)}>
          <Plus size={14} /> Add Field
        </button>
      </div>

      {error && <p className="doc-msg doc-msg--error">{error}</p>}

      {showAddForm && (
        <form className="custom-fields__add-form" onSubmit={handleAddField}>
          <div className="form-row">
            <label className="form-field">
              <span>Field Name *</span>
              <input required value={addForm.label} onChange={e => setAddForm({ ...addForm, label: e.target.value })} placeholder="e.g. Blood Group" />
            </label>
            <label className="form-field">
              <span>Type *</span>
              <select value={addForm.field_type} onChange={e => setAddForm({ ...addForm, field_type: e.target.value })}>
                {FIELD_TYPES.map(t => <option key={t.value} value={t.value}>{t.label}</option>)}
              </select>
            </label>
          </div>
          {addForm.field_type === 'enum' && (
            <label className="form-field">
              <span>Choices (comma-separated) *</span>
              <input required value={addForm.enum_options} onChange={e => setAddForm({ ...addForm, enum_options: e.target.value })} placeholder="e.g. A+, A-, B+, B-, O+, O-" />
            </label>
          )}
          <div style={{ display: 'flex', gap: '8px' }}>
            <button type="submit" className="btn btn--primary btn--sm" disabled={addSaving}>{addSaving ? 'Saving...' : 'Save Field'}</button>
            <button type="button" className="btn btn--outline btn--sm" onClick={() => setShowAddForm(false)}>Cancel</button>
          </div>
        </form>
      )}

      {loading ? (
        <p className="empty-text">Loading...</p>
      ) : fields.length === 0 ? (
        <p className="empty-text">No extra fields defined yet.</p>
      ) : (
        <div className="custom-fields__grid">
          {fields.map(field => (
            <div key={field.id} className="custom-fields__item">
              <label className="form-field">
                <span>{field.label}</span>
                {renderInput(field)}
              </label>
              <button type="button" className="btn-icon custom-fields__remove" onClick={() => handleDeleteField(field)} title="Remove this field">
                <Trash2 size={13} />
              </button>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

export default CustomFieldsSection
