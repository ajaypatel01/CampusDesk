import { useEffect, useState } from 'react'

// useState that survives leaving the page and coming back in the same tab:
// list filters, sort and page are restored when you open a student and
// press Back. Stored in sessionStorage, so a new tab starts fresh.
export default function useSessionState(key, initial) {
  const [value, setValue] = useState(() => {
    try {
      const saved = sessionStorage.getItem(key)
      return saved !== null ? JSON.parse(saved) : initial
    } catch {
      return initial
    }
  })
  useEffect(() => {
    try {
      sessionStorage.setItem(key, JSON.stringify(value))
    } catch {
      // Storage full or blocked: the filter still works, it just won't persist.
    }
  }, [key, value])
  return [value, setValue]
}
