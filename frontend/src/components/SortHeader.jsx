import { ArrowUp, ArrowDown, ArrowUpDown } from 'lucide-react'
import './SortHeader.css'

function SortHeader({ label, field, sortField, sortDir, onSort }) {
  const active = sortField === field
  return (
    <th className="sort-header" onClick={() => onSort(field)}>
      <span className="sort-header__inner">
        {label}
        <span className={`sort-header__icon ${active ? 'sort-header__icon--active' : ''}`}>
          {active
            ? (sortDir === 'asc' ? <ArrowUp size={13} /> : <ArrowDown size={13} />)
            : <ArrowUpDown size={13} />
          }
        </span>
      </span>
    </th>
  )
}

export default SortHeader
