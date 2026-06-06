package result

import (
	"context"
	"errors"
	"fmt"

	"github.com/ajaypatel01/CampusDesk/internal/domain"
	"github.com/ajaypatel01/CampusDesk/internal/platform/database"
	apperr "github.com/ajaypatel01/CampusDesk/internal/platform/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type ListFilter struct {
	SchoolID       uuid.UUID
	AcademicYearID uuid.UUID
	StudentID      *uuid.UUID
	ClassSectionID *uuid.UUID
	ExamName       string
}

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, res *domain.Result) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	row := tx.QueryRow(ctx, `
		INSERT INTO student_results (
			student_id, school_id, academic_year_id, class_section_id, exam_name,
			total_marks, max_total_marks, percentage, final_grade, remarks, result_date, status
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		RETURNING id, created_at, updated_at`,
		res.StudentID, res.SchoolID, res.AcademicYearID, res.ClassSectionID, res.ExamName,
		res.TotalMarks, res.MaxTotalMarks, res.Percentage, res.FinalGrade, res.Remarks, res.ResultDate, res.Status,
	)
	if err := row.Scan(&res.ID, &res.CreatedAt, &res.UpdatedAt); err != nil {
		return database.MapError(err)
	}
	if err := r.insertSubjects(ctx, tx, res.ID, res.Subjects); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Result, error) {
	res, err := scanResultRow(r.pool.QueryRow(ctx, resultSelectSQL()+" WHERE id=$1", id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get result: %w", err)
	}

	subjects, err := r.listSubjects(ctx, id)
	if err != nil {
		return nil, err
	}
	res.Subjects = subjects
	return res, nil
}

func (r *Repository) List(ctx context.Context, filter ListFilter, limit, offset int) ([]domain.Result, int, error) {
	args := []interface{}{filter.SchoolID, filter.AcademicYearID}
	where := "WHERE school_id=$1 AND academic_year_id=$2"
	argN := 3

	if filter.StudentID != nil {
		where += fmt.Sprintf(" AND student_id=$%d", argN)
		args = append(args, *filter.StudentID)
		argN++
	}
	if filter.ClassSectionID != nil {
		where += fmt.Sprintf(" AND class_section_id=$%d", argN)
		args = append(args, *filter.ClassSectionID)
		argN++
	}
	if filter.ExamName != "" {
		where += fmt.Sprintf(" AND exam_name ILIKE $%d", argN)
		args = append(args, "%"+filter.ExamName+"%")
		argN++
	}

	var total int
	countQ := "SELECT COUNT(*) FROM student_results " + where
	if err := r.pool.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	q := resultSelectSQL() + " " + where + fmt.Sprintf(" ORDER BY result_date DESC, exam_name LIMIT $%d OFFSET $%d", argN, argN+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}

	var items []domain.Result
	for rows.Next() {
		res, err := scanResultRow(rows)
		if err != nil {
			rows.Close()
			return nil, 0, err
		}
		items = append(items, *res)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, 0, err
	}
	rows.Close()

	for i := range items {
		subjects, err := r.listSubjects(ctx, items[i].ID)
		if err != nil {
			return nil, 0, err
		}
		items[i].Subjects = subjects
	}
	return items, total, nil
}

func (r *Repository) Update(ctx context.Context, res *domain.Result) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `
		UPDATE student_results SET
			class_section_id=$2, exam_name=$3, total_marks=$4, max_total_marks=$5,
			percentage=$6, final_grade=$7, remarks=$8, result_date=$9, status=$10, updated_at=NOW()
		WHERE id=$1`,
		res.ID, res.ClassSectionID, res.ExamName, res.TotalMarks, res.MaxTotalMarks,
		res.Percentage, res.FinalGrade, res.Remarks, res.ResultDate, res.Status,
	)
	if err != nil {
		return database.MapError(err)
	}
	if tag.RowsAffected() == 0 {
		return apperr.ErrNotFound
	}

	if _, err := tx.Exec(ctx, `DELETE FROM student_result_subjects WHERE result_id=$1`, res.ID); err != nil {
		return err
	}
	if err := r.insertSubjects(ctx, tx, res.ID, res.Subjects); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM student_results WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return apperr.ErrNotFound
	}
	return nil
}

func (r *Repository) insertSubjects(ctx context.Context, tx pgx.Tx, resultID uuid.UUID, subjects []domain.ResultSubject) error {
	for i := range subjects {
		subjects[i].ResultID = resultID
		row := tx.QueryRow(ctx, `
			INSERT INTO student_result_subjects (
				result_id, subject_name, marks_obtained, max_marks, grade, remarks, sort_order
			)
			VALUES ($1,$2,$3,$4,$5,$6,$7)
			RETURNING id, created_at, updated_at`,
			resultID, subjects[i].SubjectName, subjects[i].MarksObtained, subjects[i].MaxMarks,
			subjects[i].Grade, subjects[i].Remarks, subjects[i].SortOrder,
		)
		if err := row.Scan(&subjects[i].ID, &subjects[i].CreatedAt, &subjects[i].UpdatedAt); err != nil {
			return database.MapError(err)
		}
	}
	return nil
}

func (r *Repository) listSubjects(ctx context.Context, resultID uuid.UUID) ([]domain.ResultSubject, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, result_id, subject_name, marks_obtained::float8, max_marks::float8, grade, remarks,
			sort_order, created_at, updated_at
		FROM student_result_subjects WHERE result_id=$1 ORDER BY sort_order, subject_name`, resultID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subjects []domain.ResultSubject
	for rows.Next() {
		var subject domain.ResultSubject
		if err := rows.Scan(
			&subject.ID, &subject.ResultID, &subject.SubjectName, &subject.MarksObtained, &subject.MaxMarks,
			&subject.Grade, &subject.Remarks, &subject.SortOrder, &subject.CreatedAt, &subject.UpdatedAt,
		); err != nil {
			return nil, err
		}
		subjects = append(subjects, subject)
	}
	return subjects, rows.Err()
}

func resultSelectSQL() string {
	return `
		SELECT id, student_id, school_id, academic_year_id, class_section_id, exam_name,
			total_marks::float8, max_total_marks::float8, percentage::float8,
			final_grade, remarks, result_date, status, created_at, updated_at
		FROM student_results`
}

type scannable interface {
	Scan(dest ...interface{}) error
}

func scanResultRow(row scannable) (*domain.Result, error) {
	var res domain.Result
	err := row.Scan(
		&res.ID, &res.StudentID, &res.SchoolID, &res.AcademicYearID, &res.ClassSectionID,
		&res.ExamName, &res.TotalMarks, &res.MaxTotalMarks, &res.Percentage, &res.FinalGrade,
		&res.Remarks, &res.ResultDate, &res.Status, &res.CreatedAt, &res.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &res, nil
}
