package books

import (
	"context"
	"errors"

	"github.com/ajaypatel01/CampusDesk/internal/domain"
	"github.com/ajaypatel01/CampusDesk/internal/platform/database"
	apperr "github.com/ajaypatel01/CampusDesk/internal/platform/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// ---- Books catalog ----

func (r *Repository) CreateBook(ctx context.Context, b *domain.Book) error {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO books (school_id, title, author, publisher, isbn, price, subject)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING id, created_at, updated_at`,
		b.SchoolID, b.Title, b.Author, b.Publisher, b.ISBN, b.Price, b.Subject,
	)
	if err := row.Scan(&b.ID, &b.CreatedAt, &b.UpdatedAt); err != nil {
		return database.MapError(err)
	}
	return nil
}

func (r *Repository) GetBookByID(ctx context.Context, id uuid.UUID) (*domain.Book, error) {
	var b domain.Book
	err := r.pool.QueryRow(ctx, `
		SELECT id, school_id, title, COALESCE(author,''), COALESCE(publisher,''),
			COALESCE(isbn,''), price, COALESCE(subject,''), created_at, updated_at
		FROM books WHERE id=$1`, id,
	).Scan(&b.ID, &b.SchoolID, &b.Title, &b.Author, &b.Publisher,
		&b.ISBN, &b.Price, &b.Subject, &b.CreatedAt, &b.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.ErrNotFound
	}
	return &b, err
}

func (r *Repository) ListBooks(ctx context.Context, schoolID uuid.UUID, search string) ([]domain.Book, error) {
	q := `SELECT id, school_id, title, COALESCE(author,''), COALESCE(publisher,''),
		COALESCE(isbn,''), price, COALESCE(subject,''), created_at, updated_at
		FROM books WHERE school_id=$1`
	args := []interface{}{schoolID}
	if search != "" {
		q += ` AND (title ILIKE $2 OR author ILIKE $2 OR subject ILIKE $2)`
		args = append(args, "%"+search+"%")
	}
	q += ` ORDER BY title`

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []domain.Book
	for rows.Next() {
		var b domain.Book
		if err := rows.Scan(&b.ID, &b.SchoolID, &b.Title, &b.Author, &b.Publisher,
			&b.ISBN, &b.Price, &b.Subject, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, b)
	}
	return items, rows.Err()
}

func (r *Repository) UpdateBook(ctx context.Context, b *domain.Book) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE books SET title=$2, author=$3, publisher=$4, isbn=$5, price=$6, subject=$7, updated_at=NOW()
		WHERE id=$1`,
		b.ID, b.Title, b.Author, b.Publisher, b.ISBN, b.Price, b.Subject,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return apperr.ErrNotFound
	}
	return nil
}

func (r *Repository) DeleteBook(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM books WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return apperr.ErrNotFound
	}
	return nil
}

// ---- Book Lists ----

func (r *Repository) CreateBookList(ctx context.Context, bl *domain.BookList) error {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO book_lists (school_id, academic_year_id, grade_level_id, name)
		VALUES ($1,$2,$3,$4)
		RETURNING id, created_at, updated_at`,
		bl.SchoolID, bl.AcademicYearID, bl.GradeLevelID, bl.Name,
	)
	if err := row.Scan(&bl.ID, &bl.CreatedAt, &bl.UpdatedAt); err != nil {
		return database.MapError(err)
	}
	return nil
}

func (r *Repository) GetBookListByID(ctx context.Context, id uuid.UUID) (*domain.BookList, error) {
	var bl domain.BookList
	err := r.pool.QueryRow(ctx, `
		SELECT id, school_id, academic_year_id, grade_level_id, name, created_at, updated_at
		FROM book_lists WHERE id=$1`, id,
	).Scan(&bl.ID, &bl.SchoolID, &bl.AcademicYearID, &bl.GradeLevelID, &bl.Name, &bl.CreatedAt, &bl.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.ErrNotFound
	}
	return &bl, err
}

func (r *Repository) ListBookLists(ctx context.Context, schoolID, yearID uuid.UUID) ([]BookListSummary, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT bl.id, bl.school_id, bl.academic_year_id, bl.grade_level_id, bl.name, bl.created_at, bl.updated_at,
			gl.name, ay.name, COUNT(bli.id) AS item_count,
			COALESCE(SUM(b.price * bli.quantity), 0) AS total_price
		FROM book_lists bl
		JOIN grade_levels gl ON gl.id = bl.grade_level_id
		JOIN academic_years ay ON ay.id = bl.academic_year_id
		LEFT JOIN book_list_items bli ON bli.book_list_id = bl.id
		LEFT JOIN books b ON b.id = bli.book_id
		WHERE bl.school_id=$1 AND bl.academic_year_id=$2
		GROUP BY bl.id, gl.name, gl.sort_order, ay.name
		ORDER BY gl.sort_order`, schoolID, yearID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []BookListSummary
	for rows.Next() {
		var s BookListSummary
		if err := rows.Scan(&s.ID, &s.SchoolID, &s.AcademicYearID, &s.GradeLevelID, &s.Name,
			&s.CreatedAt, &s.UpdatedAt, &s.GradeLevelName, &s.AcademicYearName,
			&s.ItemCount, &s.TotalPrice); err != nil {
			return nil, err
		}
		items = append(items, s)
	}
	return items, rows.Err()
}

// ---- Book List Items ----

func (r *Repository) AddItemToList(ctx context.Context, item *domain.BookListItem) error {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO book_list_items (book_list_id, book_id, quantity, is_mandatory)
		VALUES ($1,$2,$3,$4)
		ON CONFLICT (book_list_id, book_id)
		DO UPDATE SET quantity=$3, is_mandatory=$4, updated_at=NOW()
		RETURNING id, created_at, updated_at`,
		item.BookListID, item.BookID, item.Quantity, item.IsMandatory,
	)
	if err := row.Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return database.MapError(err)
	}
	return nil
}

func (r *Repository) RemoveItemFromList(ctx context.Context, itemID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM book_list_items WHERE id=$1`, itemID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return apperr.ErrNotFound
	}
	return nil
}

type BookListItemDetail struct {
	domain.BookListItem
	BookTitle   string `json:"book_title"`
	Author      string `json:"author"`
	Publisher   string `json:"publisher"`
	ISBN        string `json:"isbn"`
	Subject     string `json:"subject"`
	UnitPrice   int    `json:"unit_price"`
	TotalPrice  int    `json:"total_price"`
}

func (r *Repository) GetBookListDetail(ctx context.Context, bookListID uuid.UUID) ([]BookListItemDetail, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT bli.id, bli.book_list_id, bli.book_id, bli.quantity, bli.is_mandatory, bli.created_at, bli.updated_at,
			b.title, COALESCE(b.author,''), COALESCE(b.publisher,''), COALESCE(b.isbn,''), COALESCE(b.subject,''),
			b.price, b.price * bli.quantity AS total_price
		FROM book_list_items bli
		JOIN books b ON b.id = bli.book_id
		WHERE bli.book_list_id=$1
		ORDER BY b.subject, b.title`, bookListID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []BookListItemDetail
	for rows.Next() {
		var d BookListItemDetail
		if err := rows.Scan(&d.ID, &d.BookListID, &d.BookID, &d.Quantity, &d.IsMandatory, &d.CreatedAt, &d.UpdatedAt,
			&d.BookTitle, &d.Author, &d.Publisher, &d.ISBN, &d.Subject, &d.UnitPrice, &d.TotalPrice); err != nil {
			return nil, err
		}
		items = append(items, d)
	}
	return items, rows.Err()
}

// ---- Student Book Receipts ----

func (r *Repository) RecordReceipt(ctx context.Context, rec *domain.StudentBookReceipt) error {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO student_book_receipts (student_id, book_list_id, received_date, received_by, notes)
		VALUES ($1,$2,$3,$4,$5)
		ON CONFLICT (student_id, book_list_id)
		DO UPDATE SET received_date=$3, received_by=$4, notes=$5, updated_at=NOW()
		RETURNING id, created_at, updated_at`,
		rec.StudentID, rec.BookListID, rec.ReceivedDate, rec.ReceivedBy, rec.Notes,
	)
	if err := row.Scan(&rec.ID, &rec.CreatedAt, &rec.UpdatedAt); err != nil {
		return database.MapError(err)
	}
	return nil
}

type ReceiptDetail struct {
	domain.StudentBookReceipt
	StudentName string `json:"student_name"`
	StudentCode string `json:"student_code"`
}

func (r *Repository) ListReceipts(ctx context.Context, bookListID uuid.UUID) ([]ReceiptDetail, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT r.id, r.student_id, r.book_list_id, r.received_date, COALESCE(r.received_by,''), COALESCE(r.notes,''),
			r.created_at, r.updated_at,
			s.first_name || ' ' || s.last_name, s.student_code
		FROM student_book_receipts r
		JOIN students s ON s.id = r.student_id
		WHERE r.book_list_id=$1
		ORDER BY s.last_name`, bookListID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ReceiptDetail
	for rows.Next() {
		var d ReceiptDetail
		if err := rows.Scan(&d.ID, &d.StudentID, &d.BookListID, &d.ReceivedDate, &d.ReceivedBy, &d.Notes,
			&d.CreatedAt, &d.UpdatedAt, &d.StudentName, &d.StudentCode); err != nil {
			return nil, err
		}
		items = append(items, d)
	}
	return items, rows.Err()
}

type BookListPDFData struct {
	SchoolName       string
	GradeLevelName   string
	AcademicYearName string
	ListName         string
	Items            []BookListItemDetail
	TotalPrice       int
}

func (r *Repository) GetBookListPDFData(ctx context.Context, bookListID uuid.UUID) (*BookListPDFData, error) {
	var d BookListPDFData
	err := r.pool.QueryRow(ctx, `
		SELECT sch.name, gl.name, ay.name, bl.name
		FROM book_lists bl
		JOIN schools sch ON sch.id = bl.school_id
		JOIN grade_levels gl ON gl.id = bl.grade_level_id
		JOIN academic_years ay ON ay.id = bl.academic_year_id
		WHERE bl.id=$1`, bookListID,
	).Scan(&d.SchoolName, &d.GradeLevelName, &d.AcademicYearName, &d.ListName)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	items, err := r.GetBookListDetail(ctx, bookListID)
	if err != nil {
		return nil, err
	}
	d.Items = items
	for _, it := range items {
		d.TotalPrice += it.TotalPrice
	}
	return &d, nil
}

