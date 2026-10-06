// One date format for the whole app: "06 Oct 2026" (day first, month as a
// word, so 6 Oct and 10 Jun can't be confused the way 6/10/2026 can).

const DATE_ONLY = /^(\d{4})-(\d{2})-(\d{2})(?:T00:00:00(?:\.0+)?Z)?$/

const dateFmt = (timeZone) =>
  new Intl.DateTimeFormat('en-IN', { day: '2-digit', month: 'short', year: 'numeric', timeZone })

// formatDate shows a date as "06 Oct 2026". Calendar dates from the API
// ("2026-10-06" or "2026-10-06T00:00:00Z") are shown as that day; real
// timestamps (created_at) are shown in India time. Empty values show `empty`.
export function formatDate(value, empty = '-') {
  if (!value) return empty
  if (typeof value === 'string') {
    const m = value.match(DATE_ONLY)
    if (m) return dateFmt('UTC').format(new Date(Date.UTC(+m[1], +m[2] - 1, +m[3])))
  }
  const d = value instanceof Date ? value : new Date(value)
  if (Number.isNaN(d.getTime())) return empty
  return dateFmt('Asia/Kolkata').format(d)
}

// todayIST is today's date in India as "YYYY-MM-DD", for date inputs and filters.
export function todayIST() {
  return new Intl.DateTimeFormat('en-CA', { timeZone: 'Asia/Kolkata' }).format(new Date())
}

// dateKey reduces an API date ("2026-10-06T00:00:00Z") to "2026-10-06" so it
// compares correctly with a date input's value.
export function dateKey(value) {
  return value ? String(value).slice(0, 10) : ''
}
