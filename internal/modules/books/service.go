package books

import (
	"context"
	"fmt"
	"time"

	"github.com/ajaypatel01/CampusDesk/internal/domain"
	apperr "github.com/ajaypatel01/CampusDesk/internal/platform/errors"
	"github.com/google/uuid"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// ---- Input types ----

type CreateBookInput struct {
	SchoolID  uuid.UUID `json:"school_id"`
	Title     string    `json:"title"`
	Author    string    `json:"author"`
	Publisher string    `json:"publisher"`
	ISBN      string    `json:"isbn"`
	Price     int       `json:"price"`
	Subject   string    `json:"subject"`
}

type UpdateBookInput struct {
	Title     string `json:"title"`
	Author    string `json:"author"`
	Publisher string `json:"publisher"`
	ISBN      string `json:"isbn"`
	Price     int    `json:"price"`
	Subject   string `json:"subject"`
}

type CreateBookListInput struct {
	SchoolID       uuid.UUID `json:"school_id"`
	AcademicYearID uuid.UUID `json:"academic_year_id"`
	GradeLevelID   uuid.UUID `json:"grade_level_id"`
	Name           string    `json:"name"`
}

type AddItemInput struct {
	BookID      uuid.UUID `json:"book_id"`
	Quantity    int       `json:"quantity"`
	IsMandatory *bool     `json:"is_mandatory"`
}

type RecordReceiptInput struct {
	StudentID   uuid.UUID `json:"student_id"`
	BookListID  uuid.UUID `json:"book_list_id"`
	ReceivedDate string   `json:"received_date"`
	ReceivedBy  string    `json:"received_by"`
	Notes       string    `json:"notes"`
}

// ---- Response types ----

type BookListSummary struct {
	domain.BookList
	GradeLevelName   string `json:"grade_level_name"`
	AcademicYearName string `json:"academic_year_name"`
	ItemCount        int    `json:"item_count"`
	TotalPrice       int    `json:"total_price"`
}

type BookListDetail struct {
	domain.BookList
	GradeLevelName   string               `json:"grade_level_name"`
	AcademicYearName string               `json:"academic_year_name"`
	Items            []BookListItemDetail `json:"items"`
	TotalPrice       int                  `json:"total_price"`
}

// ---- Service methods ----

func (s *Service) CreateBook(ctx context.Context, in CreateBookInput) (*domain.Book, error) {
	if in.SchoolID == uuid.Nil || in.Title == "" {
		return nil, apperr.ErrInvalidInput
	}
	b := &domain.Book{
		SchoolID: in.SchoolID, Title: in.Title, Author: in.Author,
		Publisher: in.Publisher, ISBN: in.ISBN, Price: in.Price, Subject: in.Subject,
	}
	if err := s.repo.CreateBook(ctx, b); err != nil {
		return nil, err
	}
	return b, nil
}

func (s *Service) GetBook(ctx context.Context, id uuid.UUID) (*domain.Book, error) {
	return s.repo.GetBookByID(ctx, id)
}

func (s *Service) ListBooks(ctx context.Context, schoolID uuid.UUID, search string) ([]domain.Book, error) {
	if schoolID == uuid.Nil {
		return nil, apperr.ErrInvalidInput
	}
	return s.repo.ListBooks(ctx, schoolID, search)
}

func (s *Service) UpdateBook(ctx context.Context, id uuid.UUID, in UpdateBookInput) (*domain.Book, error) {
	b, err := s.repo.GetBookByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if in.Title != "" {
		b.Title = in.Title
	}
	b.Author = in.Author
	b.Publisher = in.Publisher
	b.ISBN = in.ISBN
	b.Price = in.Price
	b.Subject = in.Subject
	if err := s.repo.UpdateBook(ctx, b); err != nil {
		return nil, err
	}
	return b, nil
}

func (s *Service) DeleteBook(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteBook(ctx, id)
}

func (s *Service) CreateBookList(ctx context.Context, in CreateBookListInput) (*domain.BookList, error) {
	if in.SchoolID == uuid.Nil || in.AcademicYearID == uuid.Nil || in.GradeLevelID == uuid.Nil {
		return nil, apperr.ErrInvalidInput
	}
	if in.Name == "" {
		in.Name = "Book List"
	}
	bl := &domain.BookList{
		SchoolID: in.SchoolID, AcademicYearID: in.AcademicYearID,
		GradeLevelID: in.GradeLevelID, Name: in.Name,
	}
	if err := s.repo.CreateBookList(ctx, bl); err != nil {
		return nil, err
	}
	return bl, nil
}

func (s *Service) ListBookLists(ctx context.Context, schoolID, yearID uuid.UUID) ([]BookListSummary, error) {
	if schoolID == uuid.Nil || yearID == uuid.Nil {
		return nil, apperr.ErrInvalidInput
	}
	return s.repo.ListBookLists(ctx, schoolID, yearID)
}

func (s *Service) GetBookListDetail(ctx context.Context, id uuid.UUID) (*BookListDetail, error) {
	bl, err := s.repo.GetBookListByID(ctx, id)
	if err != nil {
		return nil, err
	}
	items, err := s.repo.GetBookListDetail(ctx, id)
	if err != nil {
		return nil, err
	}
	var total int
	for _, it := range items {
		total += it.TotalPrice
	}
	return &BookListDetail{BookList: *bl, Items: items, TotalPrice: total}, nil
}

func (s *Service) AddItemToList(ctx context.Context, bookListID uuid.UUID, in AddItemInput) (*domain.BookListItem, error) {
	if bookListID == uuid.Nil || in.BookID == uuid.Nil {
		return nil, apperr.ErrInvalidInput
	}
	if in.Quantity <= 0 {
		in.Quantity = 1
	}
	mandatory := true
	if in.IsMandatory != nil {
		mandatory = *in.IsMandatory
	}
	item := &domain.BookListItem{
		BookListID: bookListID, BookID: in.BookID,
		Quantity: in.Quantity, IsMandatory: mandatory,
	}
	if err := s.repo.AddItemToList(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *Service) RemoveItemFromList(ctx context.Context, itemID uuid.UUID) error {
	return s.repo.RemoveItemFromList(ctx, itemID)
}

func (s *Service) RecordReceipt(ctx context.Context, in RecordReceiptInput) (*domain.StudentBookReceipt, error) {
	if in.StudentID == uuid.Nil || in.BookListID == uuid.Nil {
		return nil, apperr.ErrInvalidInput
	}
	receivedDate := time.Now()
	if in.ReceivedDate != "" {
		for _, layout := range []string{"2006-01-02", "02/01/2006"} {
			if t, err := time.Parse(layout, in.ReceivedDate); err == nil {
				receivedDate = t
				break
			}
		}
	}
	rec := &domain.StudentBookReceipt{
		StudentID: in.StudentID, BookListID: in.BookListID,
		ReceivedDate: receivedDate, ReceivedBy: in.ReceivedBy, Notes: in.Notes,
	}
	if err := s.repo.RecordReceipt(ctx, rec); err != nil {
		return nil, err
	}
	return rec, nil
}

func (s *Service) ListReceipts(ctx context.Context, bookListID uuid.UUID) ([]ReceiptDetail, error) {
	if bookListID == uuid.Nil {
		return nil, apperr.ErrInvalidInput
	}
	return s.repo.ListReceipts(ctx, bookListID)
}

func (s *Service) GenerateBookListPDF(ctx context.Context, bookListID uuid.UUID) ([]byte, string, error) {
	data, err := s.repo.GetBookListPDFData(ctx, bookListID)
	if err != nil {
		return nil, "", err
	}
	pdf, err := generateBookListPDF(*data)
	if err != nil {
		return nil, "", fmt.Errorf("generate book list pdf: %w", err)
	}
	filename := fmt.Sprintf("booklist_%s.pdf", bookListID.String()[:8])
	return pdf, filename, nil
}
