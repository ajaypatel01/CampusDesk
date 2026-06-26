import { useState, useEffect } from 'react'
import { Plus, Trash2, Edit2, Download, BookOpen, List, Receipt, Search } from 'lucide-react'
import { useSchool } from '../services/SchoolContext'
import { booksApi, academicApi, studentsApi } from '../services/api'
import './Books.css'

function downloadBlob(blob, filename) {
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url; a.download = filename; a.click()
  URL.revokeObjectURL(url)
}

function Books() {
  const { currentSchool, currentYear } = useSchool()
  const [tab, setTab] = useState('catalog')
  const [grades, setGrades] = useState([])
  const [students, setStudents] = useState([])

  // Catalog
  const [books, setBooks] = useState([])
  const [bookSearch, setBookSearch] = useState('')
  const [editingBook, setEditingBook] = useState(null)
  const [showBookModal, setShowBookModal] = useState(false)
  const [bookForm, setBookForm] = useState({ title: '', author: '', publisher: '', isbn: '', price: '', subject: '' })

  // Book Lists
  const [bookLists, setBookLists] = useState([])
  const [selectedList, setSelectedList] = useState(null)
  const [listDetail, setListDetail] = useState(null)
  const [showListModal, setShowListModal] = useState(false)
  const [listForm, setListForm] = useState({ grade_level_id: '', name: '' })
  const [showItemModal, setShowItemModal] = useState(false)
  const [itemForm, setItemForm] = useState({ book_id: '', quantity: 1, is_mandatory: true })

  // Receipts
  const [receipts, setReceipts] = useState([])
  const [showReceiptModal, setShowReceiptModal] = useState(false)
  const [receiptForm, setReceiptForm] = useState({ student_id: '', book_list_id: '', received_date: '', received_by: '', notes: '' })

  const [saving, setSaving] = useState(false)
  const [loading, setLoading] = useState(false)

  function fmt(n) {
    return `₹${(n || 0).toLocaleString('en-IN')}`
  }

  useEffect(() => {
    if (!currentSchool) return
    academicApi.listGrades(currentSchool.id).then(r => setGrades(r.items || [])).catch(() => {})
    studentsApi.list({ school_id: currentSchool.id, limit: 500 }).then(r => setStudents(r.items || [])).catch(() => {})
    loadBooks()
  }, [currentSchool])

  useEffect(() => {
    if (!currentSchool || !currentYear) return
    loadBookLists()
  }, [currentSchool, currentYear])

  function loadBooks() {
    if (!currentSchool) return
    booksApi.listBooks({ school_id: currentSchool.id, search: bookSearch }).then(r => setBooks(r.items || [])).catch(() => {})
  }

  function loadBookLists() {
    if (!currentSchool || !currentYear) return
    booksApi.listBookLists({ school_id: currentSchool.id, academic_year_id: currentYear.id }).then(r => setBookLists(r.items || [])).catch(() => {})
  }

  async function loadListDetail(list) {
    setSelectedList(list)
    setListDetail(null)
    setLoading(true)
    try {
      const [detail, recs] = await Promise.all([
        booksApi.getBookList(list.id),
        booksApi.listReceipts(list.id).catch(() => ({ items: [] })),
      ])
      setListDetail(detail)
      setReceipts(recs.items || [])
    } catch (err) { alert(err.message) }
    finally { setLoading(false) }
  }

  function openCreateBook() {
    setEditingBook(null)
    setBookForm({ title: '', author: '', publisher: '', isbn: '', price: '', subject: '' })
    setShowBookModal(true)
  }

  function openEditBook(book) {
    setEditingBook(book)
    setBookForm({ title: book.title, author: book.author || '', publisher: book.publisher || '', isbn: book.isbn || '', price: book.price || '', subject: book.subject || '' })
    setShowBookModal(true)
  }

  async function handleSaveBook(e) {
    e.preventDefault()
    setSaving(true)
    try {
      const payload = { ...bookForm, price: parseInt(bookForm.price, 10) || 0 }
      if (editingBook) {
        await booksApi.updateBook(editingBook.id, payload)
      } else {
        await booksApi.createBook({ school_id: currentSchool.id, ...payload })
      }
      setShowBookModal(false)
      loadBooks()
    } catch (err) { alert(err.message) }
    finally { setSaving(false) }
  }

  async function handleDeleteBook(id) {
    if (!confirm('Delete this book?')) return
    try {
      await booksApi.deleteBook(id)
      setBooks(prev => prev.filter(b => b.id !== id))
    } catch (err) { alert(err.message) }
  }

  async function handleCreateList(e) {
    e.preventDefault()
    setSaving(true)
    try {
      await booksApi.createBookList({ school_id: currentSchool.id, academic_year_id: currentYear.id, ...listForm })
      setShowListModal(false)
      setListForm({ grade_level_id: '', name: '' })
      loadBookLists()
    } catch (err) { alert(err.message) }
    finally { setSaving(false) }
  }

  async function handleAddItem(e) {
    e.preventDefault()
    if (!selectedList) return
    setSaving(true)
    try {
      await booksApi.addItem(selectedList.id, { ...itemForm, quantity: parseInt(itemForm.quantity, 10) || 1, is_mandatory: itemForm.is_mandatory })
      setShowItemModal(false)
      setItemForm({ book_id: '', quantity: 1, is_mandatory: true })
      loadListDetail(selectedList)
    } catch (err) { alert(err.message) }
    finally { setSaving(false) }
  }

  async function handleRemoveItem(itemId) {
    if (!confirm('Remove this book from the list?')) return
    try {
      await booksApi.removeItem(selectedList.id, itemId)
      loadListDetail(selectedList)
    } catch (err) { alert(err.message) }
  }

  async function handleDownloadPDF(listId) {
    try {
      const blob = await booksApi.downloadPDF(listId)
      downloadBlob(blob, `book-list.pdf`)
    } catch (err) { alert(err.message) }
  }

  async function handleRecordReceipt(e) {
    e.preventDefault()
    setSaving(true)
    try {
      await booksApi.recordReceipt({ ...receiptForm, book_list_id: selectedList.id })
      setShowReceiptModal(false)
      setReceiptForm({ student_id: '', book_list_id: '', received_date: '', received_by: '', notes: '' })
      if (selectedList) loadListDetail(selectedList)
    } catch (err) { alert(err.message) }
    finally { setSaving(false) }
  }

  const filteredBooks = bookSearch
    ? books.filter(b => b.title.toLowerCase().includes(bookSearch.toLowerCase()) || (b.author || '').toLowerCase().includes(bookSearch.toLowerCase()))
    : books

  const items = listDetail?.items || []
  const listItems = Array.isArray(items) ? items : []

  if (!currentSchool) return <div className="books-page"><p className="empty-text">Select a school first.</p></div>

  return (
    <div className="books-page">
      <div className="page-header">
        <div>
          <h1>Books</h1>
          <p className="page-subtitle">Manage book catalog, grade-wise book lists, and student receipts</p>
        </div>
      </div>

      <div className="docs-tabs">
        {[['catalog', 'Book Catalog', BookOpen], ['lists', 'Book Lists', List], ['receipts', 'Receipts', Receipt]].map(([key, label, Icon]) => (
          <button key={key} className={`docs-tab ${tab === key ? 'docs-tab--active' : ''}`} onClick={() => setTab(key)}>
            <Icon size={15} /> {label}
          </button>
        ))}
      </div>

      {/* Catalog Tab */}
      {tab === 'catalog' && (
        <div>
          <div className="books-catalog-header">
            <div className="filter-search">
              <Search size={16} />
              <input
                placeholder="Search by title or author..."
                value={bookSearch}
                onChange={e => setBookSearch(e.target.value)}
                onKeyDown={e => e.key === 'Enter' && loadBooks()}
              />
            </div>
            <button className="btn btn--outline btn--sm" onClick={loadBooks}>Search</button>
            <button className="btn btn--primary" onClick={openCreateBook}>
              <Plus size={16} /> Add Book
            </button>
          </div>
          <div className="table-card">
            <table className="data-table">
              <thead>
                <tr><th>Title</th><th>Author</th><th>Publisher</th><th>Subject</th><th>Price</th><th></th></tr>
              </thead>
              <tbody>
                {filteredBooks.length === 0 ? (
                  <tr><td colSpan={6} className="data-table__empty">No books found</td></tr>
                ) : filteredBooks.map(b => (
                  <tr key={b.id}>
                    <td><strong>{b.title}</strong>{b.isbn && <div className="data-table__muted" style={{ fontSize: '11px' }}>ISBN: {b.isbn}</div>}</td>
                    <td>{b.author || '-'}</td>
                    <td>{b.publisher || '-'}</td>
                    <td>{b.subject || '-'}</td>
                    <td>{fmt(b.price)}</td>
                    <td>
                      <div style={{ display: 'flex', gap: '6px' }}>
                        <button className="btn btn--outline btn--sm" onClick={() => openEditBook(b)}><Edit2 size={13} /></button>
                        <button className="btn btn--outline btn--sm" onClick={() => handleDeleteBook(b.id)}><Trash2 size={13} /></button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Book Lists Tab */}
      {tab === 'lists' && (
        <div>
          {!currentYear ? (
            <p className="empty-text">Select an academic year to manage book lists.</p>
          ) : (
            <div className="lists-layout">
              <div className="lists-sidebar">
                <div className="lists-sidebar__header">
                  <strong>Book Lists</strong>
                  <button className="btn btn--primary btn--sm" onClick={() => setShowListModal(true)}>
                    <Plus size={14} />
                  </button>
                </div>
                {bookLists.length === 0 && <p className="empty-text" style={{ padding: '12px' }}>No lists yet</p>}
                {bookLists.map(bl => (
                  <div
                    key={bl.id}
                    className={`list-item ${selectedList?.id === bl.id ? 'list-item--active' : ''}`}
                    onClick={() => loadListDetail(bl)}
                  >
                    <div className="list-item__name">{bl.name}</div>
                    <div className="list-item__meta">{bl.grade_level_name} · {bl.item_count || 0} books · {fmt(bl.total_price)}</div>
                  </div>
                ))}
              </div>

              <div className="list-detail">
                {!selectedList && (
                  <div className="list-detail--empty">
                    <BookOpen size={40} />
                    <p>Select a book list to view details</p>
                  </div>
                )}
                {selectedList && loading && <p className="empty-text">Loading...</p>}
                {selectedList && !loading && listDetail && (
                  <>
                    <div className="list-detail__header">
                      <div>
                        <h2>{listDetail.name}</h2>
                        <p className="page-subtitle">{listDetail.grade_level_name} · {listDetail.academic_year_name}</p>
                      </div>
                      <div style={{ display: 'flex', gap: '8px' }}>
                        <button className="btn btn--outline btn--sm" onClick={() => handleDownloadPDF(selectedList.id)}>
                          <Download size={14} /> PDF
                        </button>
                        <button className="btn btn--primary btn--sm" onClick={() => setShowItemModal(true)}>
                          <Plus size={14} /> Add Book
                        </button>
                      </div>
                    </div>
                    <div className="table-card">
                      <table className="data-table">
                        <thead><tr><th>Book</th><th>Subject</th><th>Qty</th><th>Price</th><th>Mandatory</th><th></th></tr></thead>
                        <tbody>
                          {listItems.length === 0 ? (
                            <tr><td colSpan={6} className="data-table__empty">No books in list</td></tr>
                          ) : listItems.map(item => (
                            <tr key={item.id}>
                              <td>
                                <strong>{item.book_title || item.book_id}</strong>
                                {item.book_author && <div className="data-table__muted" style={{ fontSize: '11px' }}>{item.book_author}</div>}
                              </td>
                              <td>{item.book_subject || '-'}</td>
                              <td>{item.quantity}</td>
                              <td>{fmt(item.book_price)}</td>
                              <td><span className={`badge badge--${item.is_mandatory ? 'success' : 'muted'}`}>{item.is_mandatory ? 'Yes' : 'No'}</span></td>
                              <td>
                                <button className="btn btn--outline btn--sm" onClick={() => handleRemoveItem(item.id)}><Trash2 size={13} /></button>
                              </td>
                            </tr>
                          ))}
                        </tbody>
                        {listItems.length > 0 && (
                          <tfoot>
                            <tr>
                              <td colSpan={3}><strong>Total</strong></td>
                              <td><strong>{fmt(listItems.reduce((sum, i) => sum + (i.book_price || 0) * (i.quantity || 1), 0))}</strong></td>
                              <td colSpan={2}></td>
                            </tr>
                          </tfoot>
                        )}
                      </table>
                    </div>
                  </>
                )}
              </div>
            </div>
          )}
        </div>
      )}

      {/* Receipts Tab */}
      {tab === 'receipts' && (
        <div>
          {!currentYear ? (
            <p className="empty-text">Select an academic year to manage receipts.</p>
          ) : !selectedList ? (
            <div>
              <p className="empty-text" style={{ marginBottom: 16 }}>Select a book list first. Go to the Book Lists tab and select a list.</p>
              <div className="table-card">
                <table className="data-table">
                  <thead><tr><th>Book List</th><th>Grade</th><th></th></tr></thead>
                  <tbody>
                    {bookLists.length === 0 ? (
                      <tr><td colSpan={3} className="data-table__empty">No book lists</td></tr>
                    ) : bookLists.map(bl => (
                      <tr key={bl.id} style={{ cursor: 'pointer' }} onClick={() => { loadListDetail(bl); }}>
                        <td><strong>{bl.name}</strong></td>
                        <td>{bl.grade_level_name}</td>
                        <td><button className="btn btn--outline btn--sm">View Receipts</button></td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          ) : (
            <div>
              <div className="books-catalog-header">
                <div>
                  <strong>{selectedList.name}</strong>
                  <span className="data-table__muted" style={{ marginLeft: 8 }}>
                    <button className="btn btn--outline btn--sm" style={{ marginLeft: 8 }} onClick={() => { setSelectedList(null); setReceipts([]) }}>← Back</button>
                  </span>
                </div>
                <button className="btn btn--primary" onClick={() => setShowReceiptModal(true)}>
                  <Plus size={16} /> Record Receipt
                </button>
              </div>
              <div className="table-card">
                <table className="data-table">
                  <thead><tr><th>Student</th><th>Received Date</th><th>Received By</th><th>Notes</th></tr></thead>
                  <tbody>
                    {loading && <tr><td colSpan={4} className="data-table__empty">Loading...</td></tr>}
                    {!loading && receipts.length === 0 ? (
                      <tr><td colSpan={4} className="data-table__empty">No receipts recorded</td></tr>
                    ) : receipts.map(r => {
                      const stu = students.find(s => s.id === r.student_id)
                      return (
                        <tr key={r.id}>
                          <td>{stu ? `${stu.first_name} ${stu.last_name} (${stu.student_code})` : r.student_id}</td>
                          <td>{r.received_date ? new Date(r.received_date).toLocaleDateString('en-IN') : '-'}</td>
                          <td>{r.received_by || '-'}</td>
                          <td className="data-table__muted">{r.notes || '-'}</td>
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

      {/* Book modal */}
      {showBookModal && (
        <div className="modal-overlay" onClick={() => setShowBookModal(false)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <h2>{editingBook ? 'Edit Book' : 'Add Book'}</h2>
            <form className="modal__form" onSubmit={handleSaveBook}>
              <label className="form-field"><span>Title *</span><input required value={bookForm.title} onChange={e => setBookForm({ ...bookForm, title: e.target.value })} /></label>
              <div className="form-row">
                <label className="form-field"><span>Author</span><input value={bookForm.author} onChange={e => setBookForm({ ...bookForm, author: e.target.value })} /></label>
                <label className="form-field"><span>Subject</span><input value={bookForm.subject} onChange={e => setBookForm({ ...bookForm, subject: e.target.value })} /></label>
              </div>
              <div className="form-row">
                <label className="form-field"><span>Publisher</span><input value={bookForm.publisher} onChange={e => setBookForm({ ...bookForm, publisher: e.target.value })} /></label>
                <label className="form-field"><span>ISBN</span><input value={bookForm.isbn} onChange={e => setBookForm({ ...bookForm, isbn: e.target.value })} /></label>
              </div>
              <label className="form-field"><span>Price (₹)</span><input type="number" min="0" value={bookForm.price} onChange={e => setBookForm({ ...bookForm, price: e.target.value })} /></label>
              <div className="modal__actions">
                <button type="button" className="btn btn--outline" onClick={() => setShowBookModal(false)}>Cancel</button>
                <button type="submit" className="btn btn--primary" disabled={saving}>{saving ? 'Saving...' : editingBook ? 'Update' : 'Add Book'}</button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Book list modal */}
      {showListModal && (
        <div className="modal-overlay" onClick={() => setShowListModal(false)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <h2>Create Book List</h2>
            <form className="modal__form" onSubmit={handleCreateList}>
              <label className="form-field"><span>List Name *</span><input required value={listForm.name} onChange={e => setListForm({ ...listForm, name: e.target.value })} placeholder="e.g. Class 1 Book Set" /></label>
              <label className="form-field">
                <span>Grade Level *</span>
                <select required value={listForm.grade_level_id} onChange={e => setListForm({ ...listForm, grade_level_id: e.target.value })}>
                  <option value="">Select grade...</option>
                  {grades.map(g => <option key={g.id} value={g.id}>{g.name}</option>)}
                </select>
              </label>
              <div className="modal__actions">
                <button type="button" className="btn btn--outline" onClick={() => setShowListModal(false)}>Cancel</button>
                <button type="submit" className="btn btn--primary" disabled={saving}>{saving ? 'Creating...' : 'Create List'}</button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Add item modal */}
      {showItemModal && (
        <div className="modal-overlay" onClick={() => setShowItemModal(false)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <h2>Add Book to List</h2>
            <form className="modal__form" onSubmit={handleAddItem}>
              <label className="form-field">
                <span>Book *</span>
                <select required value={itemForm.book_id} onChange={e => setItemForm({ ...itemForm, book_id: e.target.value })}>
                  <option value="">Select book...</option>
                  {books.map(b => <option key={b.id} value={b.id}>{b.title}{b.author ? ` — ${b.author}` : ''}</option>)}
                </select>
              </label>
              <div className="form-row">
                <label className="form-field"><span>Quantity</span><input type="number" min="1" value={itemForm.quantity} onChange={e => setItemForm({ ...itemForm, quantity: parseInt(e.target.value) || 1 })} /></label>
                <label className="form-field" style={{ justifyContent: 'flex-end' }}>
                  <span>Mandatory?</span>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8, paddingTop: 6 }}>
                    <input type="checkbox" checked={itemForm.is_mandatory} onChange={e => setItemForm({ ...itemForm, is_mandatory: e.target.checked })} style={{ width: 'auto' }} />
                    <span style={{ fontSize: '13px', color: 'var(--gray-700)' }}>Required book</span>
                  </div>
                </label>
              </div>
              <div className="modal__actions">
                <button type="button" className="btn btn--outline" onClick={() => setShowItemModal(false)}>Cancel</button>
                <button type="submit" className="btn btn--primary" disabled={saving}>{saving ? 'Adding...' : 'Add'}</button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Record receipt modal */}
      {showReceiptModal && (
        <div className="modal-overlay" onClick={() => setShowReceiptModal(false)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <h2>Record Book Receipt</h2>
            <form className="modal__form" onSubmit={handleRecordReceipt}>
              <label className="form-field">
                <span>Student *</span>
                <select required value={receiptForm.student_id} onChange={e => setReceiptForm({ ...receiptForm, student_id: e.target.value })}>
                  <option value="">Select student...</option>
                  {students.map(s => <option key={s.id} value={s.id}>{s.first_name} {s.last_name} ({s.student_code})</option>)}
                </select>
              </label>
              <div className="form-row">
                <label className="form-field"><span>Date Received</span><input type="date" value={receiptForm.received_date} onChange={e => setReceiptForm({ ...receiptForm, received_date: e.target.value })} /></label>
                <label className="form-field"><span>Received By</span><input value={receiptForm.received_by} onChange={e => setReceiptForm({ ...receiptForm, received_by: e.target.value })} placeholder="Staff name" /></label>
              </div>
              <label className="form-field"><span>Notes</span><textarea rows={2} value={receiptForm.notes} onChange={e => setReceiptForm({ ...receiptForm, notes: e.target.value })} /></label>
              <div className="modal__actions">
                <button type="button" className="btn btn--outline" onClick={() => setShowReceiptModal(false)}>Cancel</button>
                <button type="submit" className="btn btn--primary" disabled={saving}>{saving ? 'Saving...' : 'Record'}</button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}

export default Books
