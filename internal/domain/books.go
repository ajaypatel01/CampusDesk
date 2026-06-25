package domain

import (
	"time"

	"github.com/google/uuid"
)

type Book struct {
	ID        uuid.UUID `json:"id"`
	SchoolID  uuid.UUID `json:"school_id"`
	Title     string    `json:"title"`
	Author    string    `json:"author,omitempty"`
	Publisher string    `json:"publisher,omitempty"`
	ISBN      string    `json:"isbn,omitempty"`
	Price     int       `json:"price"`
	Subject   string    `json:"subject,omitempty"`
	Timestamps
}

type BookList struct {
	ID             uuid.UUID `json:"id"`
	SchoolID       uuid.UUID `json:"school_id"`
	AcademicYearID uuid.UUID `json:"academic_year_id"`
	GradeLevelID   uuid.UUID `json:"grade_level_id"`
	Name           string    `json:"name"`
	Timestamps
}

type BookListItem struct {
	ID          uuid.UUID `json:"id"`
	BookListID  uuid.UUID `json:"book_list_id"`
	BookID      uuid.UUID `json:"book_id"`
	Quantity    int       `json:"quantity"`
	IsMandatory bool      `json:"is_mandatory"`
	Timestamps
}

type StudentBookReceipt struct {
	ID          uuid.UUID `json:"id"`
	StudentID   uuid.UUID `json:"student_id"`
	BookListID  uuid.UUID `json:"book_list_id"`
	ReceivedDate time.Time `json:"received_date"`
	ReceivedBy  string    `json:"received_by,omitempty"`
	Notes       string    `json:"notes,omitempty"`
	Timestamps
}
