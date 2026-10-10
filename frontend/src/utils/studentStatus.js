// Student statuses, in the order the pickers show them. "inactive" is a
// student who left with a Transfer Certificate (it needs a TC date and year).
export const STUDENT_STATUSES = [
  { value: 'active', label: 'Active' },
  { value: 'inactive', label: 'Left (TC issued)' },
  { value: 'left_without_tc', label: 'Left without TC' },
  { value: 'defaulted', label: 'Defaulted' },
  { value: 'graduated', label: 'Graduated' },
  { value: 'transferred', label: 'Transferred' },
  // Entered twice by mistake: hidden from every list and fee total; only
  // shown when the Students list is filtered to this status.
  { value: 'duplicate', label: 'Duplicate entry' },
]

export function studentStatusLabel(status) {
  return STUDENT_STATUSES.find(s => s.value === status)?.label || status || '-'
}

// Badge modifier for a status: green while studying, red for a defaulter,
// grey for everyone who has left.
export function studentStatusBadge(status) {
  if (status === 'active') return 'success'
  if (status === 'defaulted' || status === 'duplicate') return 'danger'
  return 'muted'
}
