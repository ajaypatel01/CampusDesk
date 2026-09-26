package results

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ajaypatel01/CampusDesk/internal/domain"
	"github.com/ajaypatel01/CampusDesk/internal/platform/database"
	apperr "github.com/ajaypatel01/CampusDesk/internal/platform/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4"
)

// ---- Report card: combined, multi-exam, class-wise-templated view ----

type ReportCardExam struct {
	ExamID   uuid.UUID `json:"exam_id"`
	ExamName string    `json:"exam_name"`
	// Position is 1/2/3 for however many exams this grade+year actually has
	// (Quarterly/Half-Yearly/Annual), assigned by exam_date order -- the
	// templates print these positionally ("(I)", "(II)", "(III)"), not by
	// matching the exam's own name text.
	Position int `json:"position"`
}

type ReportCardComponentValue struct {
	Key      string  `json:"key"`
	Label    string  `json:"label"`
	MaxMarks int     `json:"max_marks"`
	Obtained float64 `json:"obtained"`
}

// ReportCardSubjectExamCell is one subject's marks for one exam. Components
// is populated whenever the grade has a report-card template; it's empty for
// a legacy plain mark (no components recorded), in which case Obtained/
// MaxMarks still carry the total.
type ReportCardSubjectExamCell struct {
	Components []ReportCardComponentValue `json:"components,omitempty"`
	Obtained   float64                    `json:"obtained"`
	MaxMarks   int                        `json:"max_marks"`
	IsAbsent   bool                       `json:"is_absent"`
}

type ReportCardSubjectRow struct {
	SubjectID       uuid.UUID                   `json:"subject_id"`
	SubjectName     string                      `json:"subject_name"`
	IsCoScholastic  bool                        `json:"is_co_scholastic"`
	ByExam          []ReportCardSubjectExamCell `json:"by_exam"` // aligned with ReportCard.Exams
	OverallObtained float64                     `json:"overall_obtained"`
	OverallMax      int                         `json:"overall_max"`
	OverallPercent  float64                     `json:"overall_percent"`
	// Grade is this subject's own letter grade (blank for "middle", which has
	// no letter-grade formula), computed the same way as ReportCard.OverallGrade.
	Grade string `json:"grade,omitempty"`
}

type ReportCard struct {
	StudentID      uuid.UUID  `json:"student_id"`
	StudentName    string     `json:"student_name"`
	StudentCode    string     `json:"student_code"`
	DateOfBirth    *time.Time `json:"date_of_birth,omitempty"`
	PenNumber      string     `json:"pen_number,omitempty"`
	AparID         string     `json:"apar_id,omitempty"`
	FatherName     string     `json:"father_name,omitempty"`
	MotherName     string     `json:"mother_name,omitempty"`
	SchoolName     string     `json:"school_name"`
	SchoolCode     string     `json:"school_code"`
	SchoolAddress  string     `json:"school_address,omitempty"`
	DiceCode       string     `json:"dice_code,omitempty"`
	GradeLevelID   uuid.UUID  `json:"grade_level_id"`
	GradeLevelName string     `json:"grade_level_name"`
	Template       string     `json:"template"`
	AcademicYearID uuid.UUID  `json:"academic_year_id"`
	AcademicYear   string     `json:"academic_year"`

	Exams    []ReportCardExam       `json:"exams"`
	Subjects []ReportCardSubjectRow `json:"subjects"`

	OverallObtained float64 `json:"overall_obtained"`
	OverallMax      int     `json:"overall_max"`
	OverallPercent  float64 `json:"overall_percent"`
	// OverallGrade is "" for the "middle" template, which prints percentage
	// only -- the source workbook has no letter-grade formula for it.
	OverallGrade string `json:"overall_grade,omitempty"`

	Details          *domain.ReportCardDetails `json:"details,omitempty"`
	DisciplineGrades []domain.DisciplineGrade  `json:"discipline_grades,omitempty"`
}

// GetExamReportTemplate returns the report-card template of the grade an
// exam belongs to (nil if that grade has none set).
func (r *Repository) GetExamReportTemplate(ctx context.Context, examID uuid.UUID) (*string, error) {
	var template *string
	err := r.pool.QueryRow(ctx, `
		SELECT gl.report_card_template FROM exams e JOIN grade_levels gl ON gl.id = e.grade_level_id WHERE e.id=$1`,
		examID,
	).Scan(&template)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.ErrNotFound
	}
	return template, err
}

// GetGradeReportTemplate returns a grade level's report-card template
// directly (nil if it has none set).
func (r *Repository) GetGradeReportTemplate(ctx context.Context, gradeLevelID uuid.UUID) (*string, error) {
	var template *string
	err := r.pool.QueryRow(ctx, `SELECT report_card_template FROM grade_levels WHERE id=$1`, gradeLevelID).Scan(&template)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.ErrNotFound
	}
	return template, err
}

// UpsertMarkWithComponents saves a mark's component breakdown (Written/Note
// Book/Activity/..., or Test/Project/Theory) and computes the parent
// exam_marks row's total/max from it, in one transaction. mark.Components
// missing a scheme key is treated as 0 for that component, matching how an
// empty input box on the entry form means "not yet scored".
func (r *Repository) UpsertMarkWithComponents(ctx context.Context, mark *domain.ExamMark, components []MarkComponent) error {
	var obtained float64
	for _, c := range components {
		obtained += mark.Components[c.Key]
	}
	mark.MarksObtained = obtained
	mark.MaxMarks = componentsMaxTotal(components)

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	row := tx.QueryRow(ctx, `
		INSERT INTO exam_marks (exam_id, student_id, subject_id, marks_obtained, max_marks, is_absent, remarks)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		ON CONFLICT (exam_id, student_id, subject_id)
		DO UPDATE SET marks_obtained=$4, max_marks=$5, is_absent=$6, remarks=$7, updated_at=NOW()
		RETURNING id, created_at, updated_at`,
		mark.ExamID, mark.StudentID, mark.SubjectID, mark.MarksObtained, mark.MaxMarks, mark.IsAbsent, mark.Remarks,
	)
	if err := row.Scan(&mark.ID, &mark.CreatedAt, &mark.UpdatedAt); err != nil {
		return database.MapError(err)
	}

	for _, c := range components {
		if _, err := tx.Exec(ctx, `
			INSERT INTO exam_mark_components (exam_mark_id, component_key, obtained, max_marks)
			VALUES ($1,$2,$3,$4)
			ON CONFLICT (exam_mark_id, component_key) DO UPDATE SET obtained=$3, max_marks=$4, updated_at=NOW()`,
			mark.ID, c.Key, mark.Components[c.Key], c.MaxMarks,
		); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// GetReportCard resolves the student's current-year grade and its template,
// combines up to 3 exams for that grade+year, and returns the full printable
// report card (minus father/mother name, filled in by the caller from the
// guardian module rather than duplicating that lookup here).
func (r *Repository) GetReportCard(ctx context.Context, studentID, academicYearID uuid.UUID) (*ReportCard, error) {
	var rc ReportCard
	rc.StudentID = studentID
	rc.AcademicYearID = academicYearID

	var schoolID uuid.UUID
	var template *string
	err := r.pool.QueryRow(ctx, `
		SELECT s.first_name||' '||s.last_name, s.student_code, s.date_of_birth, COALESCE(s.pen_number,''), COALESCE(s.apar_id,''),
			sch.id, sch.name, sch.code, COALESCE(sch.address,''), COALESCE(sch.dice_code,''),
			gl.id, gl.name, gl.report_card_template,
			ay.name
		FROM students s
		JOIN schools sch ON sch.id = s.school_id
		JOIN enrollments e ON e.student_id = s.id AND e.academic_year_id = $2
		JOIN class_sections cs ON cs.id = e.class_section_id
		JOIN grade_levels gl ON gl.id = cs.grade_level_id
		JOIN academic_years ay ON ay.id = e.academic_year_id
		WHERE s.id = $1`, studentID, academicYearID,
	).Scan(&rc.StudentName, &rc.StudentCode, &rc.DateOfBirth, &rc.PenNumber, &rc.AparID,
		&schoolID, &rc.SchoolName, &rc.SchoolCode, &rc.SchoolAddress, &rc.DiceCode,
		&rc.GradeLevelID, &rc.GradeLevelName, &template, &rc.AcademicYear)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get report card header: %w", err)
	}
	if template == nil {
		return nil, fmt.Errorf("%s has no report-card template set", rc.GradeLevelName)
	}
	rc.Template = *template
	components := MarkComponentsForTemplate(template)

	examRows, err := r.pool.Query(ctx, `
		SELECT id, name FROM exams WHERE school_id=$1 AND grade_level_id=$2 AND academic_year_id=$3
		ORDER BY exam_date NULLS LAST, name`, schoolID, rc.GradeLevelID, academicYearID)
	if err != nil {
		return nil, err
	}
	var examIDs []uuid.UUID
	pos := 0
	for examRows.Next() {
		pos++
		var e ReportCardExam
		if err := examRows.Scan(&e.ExamID, &e.ExamName); err != nil {
			examRows.Close()
			return nil, err
		}
		e.Position = pos
		rc.Exams = append(rc.Exams, e)
		examIDs = append(examIDs, e.ExamID)
	}
	examRows.Close()
	if err := examRows.Err(); err != nil {
		return nil, err
	}

	subjects, err := r.ListSubjects(ctx, schoolID, rc.GradeLevelID)
	if err != nil {
		return nil, err
	}

	type markKey struct {
		examID, subjectID uuid.UUID
	}
	marksByKey := map[markKey]domain.ExamMark{}
	componentsByMarkID := map[uuid.UUID][]ReportCardComponentValue{}

	if len(examIDs) > 0 {
		rows, err := r.pool.Query(ctx, `
			SELECT id, exam_id, subject_id, marks_obtained, max_marks, is_absent
			FROM exam_marks WHERE student_id=$1 AND exam_id = ANY($2)`, studentID, examIDs)
		if err != nil {
			return nil, err
		}
		var markIDs []uuid.UUID
		for rows.Next() {
			var mk domain.ExamMark
			if err := rows.Scan(&mk.ID, &mk.ExamID, &mk.SubjectID, &mk.MarksObtained, &mk.MaxMarks, &mk.IsAbsent); err != nil {
				rows.Close()
				return nil, err
			}
			marksByKey[markKey{mk.ExamID, mk.SubjectID}] = mk
			markIDs = append(markIDs, mk.ID)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, err
		}

		if len(markIDs) > 0 {
			crows, err := r.pool.Query(ctx, `
				SELECT exam_mark_id, component_key, obtained, max_marks
				FROM exam_mark_components WHERE exam_mark_id = ANY($1)`, markIDs)
			if err != nil {
				return nil, err
			}
			for crows.Next() {
				var markID uuid.UUID
				var cv ReportCardComponentValue
				if err := crows.Scan(&markID, &cv.Key, &cv.Obtained, &cv.MaxMarks); err != nil {
					crows.Close()
					return nil, err
				}
				componentsByMarkID[markID] = append(componentsByMarkID[markID], cv)
			}
			crows.Close()
			if err := crows.Err(); err != nil {
				return nil, err
			}
		}
	}

	var overallObtained float64
	var overallMax int
	for _, sub := range subjects {
		row := ReportCardSubjectRow{SubjectID: sub.ID, SubjectName: sub.Name, IsCoScholastic: sub.IsCoScholastic}
		var subObtained float64
		var subMax int
		for _, e := range rc.Exams {
			cell := ReportCardSubjectExamCell{}
			mk, ok := marksByKey[markKey{e.ExamID, sub.ID}]
			if !ok {
				// Nothing entered yet: show the scheme's empty component
				// slots so the entry UI has something to render inputs for.
				for _, c := range components {
					cell.Components = append(cell.Components, ReportCardComponentValue{Key: c.Key, Label: c.Label, MaxMarks: c.MaxMarks})
				}
				cell.MaxMarks = componentsMaxTotal(components)
			} else {
				cell.IsAbsent = mk.IsAbsent
				cell.Obtained = mk.MarksObtained
				cell.MaxMarks = mk.MaxMarks
				if cvs := componentsByMarkID[mk.ID]; len(cvs) > 0 {
					byKey := make(map[string]ReportCardComponentValue, len(cvs))
					for _, cv := range cvs {
						byKey[cv.Key] = cv
					}
					for _, c := range components {
						if cv, found := byKey[c.Key]; found {
							cv.Label = c.Label
							cell.Components = append(cell.Components, cv)
						} else {
							cell.Components = append(cell.Components, ReportCardComponentValue{Key: c.Key, Label: c.Label, MaxMarks: c.MaxMarks})
						}
					}
				}
				// Else: a legacy plain mark with no components -- leave
				// Components empty; Obtained/MaxMarks still show the total.
			}
			row.ByExam = append(row.ByExam, cell)
			if !cell.IsAbsent {
				subObtained += cell.Obtained
				subMax += cell.MaxMarks
			}
		}
		row.OverallObtained = subObtained
		row.OverallMax = subMax
		if subMax > 0 {
			row.OverallPercent = subObtained / float64(subMax) * 100
		}
		row.Grade = gradeForTemplate(rc.Template, row.OverallPercent)
		rc.Subjects = append(rc.Subjects, row)
		// Co-scholastic subjects are graded individually but excluded from
		// the overall total, same convention as the existing single-exam
		// marksheet (GetStudentMarksheet).
		if !sub.IsCoScholastic {
			overallObtained += subObtained
			overallMax += subMax
		}
	}
	rc.OverallObtained = overallObtained
	rc.OverallMax = overallMax
	if overallMax > 0 {
		rc.OverallPercent = overallObtained / float64(overallMax) * 100
	}
	rc.OverallGrade = gradeForTemplate(rc.Template, rc.OverallPercent)

	details, err := r.GetReportCardDetails(ctx, studentID, academicYearID)
	if err != nil && !errors.Is(err, apperr.ErrNotFound) {
		return nil, err
	}
	rc.Details = details

	if rc.Template == "middle" {
		grades, err := r.ListDisciplineGrades(ctx, studentID, academicYearID)
		if err != nil {
			return nil, err
		}
		rc.DisciplineGrades = grades
	}

	return &rc, nil
}

// ---- Report card details (attendance/remark/promoted-to/roll no/...) ----

func (r *Repository) GetReportCardDetails(ctx context.Context, studentID, academicYearID uuid.UUID) (*domain.ReportCardDetails, error) {
	var d domain.ReportCardDetails
	err := r.pool.QueryRow(ctx, `
		SELECT id, school_id, academic_year_id, grade_level_id, student_id,
			COALESCE(roll_no,''), COALESCE(attendance,''), COALESCE(remark,''), COALESCE(promoted_to,''),
			COALESCE(moral_remark,''), COALESCE(gk_remark,''), created_at, updated_at
		FROM report_card_details WHERE student_id=$1 AND academic_year_id=$2`, studentID, academicYearID,
	).Scan(&d.ID, &d.SchoolID, &d.AcademicYearID, &d.GradeLevelID, &d.StudentID,
		&d.RollNo, &d.Attendance, &d.Remark, &d.PromotedTo, &d.MoralRemark, &d.GKRemark, &d.CreatedAt, &d.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *Repository) UpsertReportCardDetails(ctx context.Context, d *domain.ReportCardDetails) error {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO report_card_details (school_id, academic_year_id, grade_level_id, student_id, roll_no, attendance, remark, promoted_to, moral_remark, gk_remark)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		ON CONFLICT (student_id, academic_year_id)
		DO UPDATE SET grade_level_id=$3, roll_no=$5, attendance=$6, remark=$7, promoted_to=$8, moral_remark=$9, gk_remark=$10, updated_at=NOW()
		RETURNING id, created_at, updated_at`,
		d.SchoolID, d.AcademicYearID, d.GradeLevelID, d.StudentID, d.RollNo, d.Attendance, d.Remark, d.PromotedTo, d.MoralRemark, d.GKRemark,
	)
	if err := row.Scan(&d.ID, &d.CreatedAt, &d.UpdatedAt); err != nil {
		return database.MapError(err)
	}
	return nil
}

// ---- Discipline (Co-Scholastic) grades -- "middle" template only ----

func (r *Repository) ListDisciplineGrades(ctx context.Context, studentID, academicYearID uuid.UUID) ([]domain.DisciplineGrade, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, school_id, academic_year_id, student_id, criterion_key, grade, created_at, updated_at
		FROM discipline_grades WHERE student_id=$1 AND academic_year_id=$2`, studentID, academicYearID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []domain.DisciplineGrade
	for rows.Next() {
		var g domain.DisciplineGrade
		if err := rows.Scan(&g.ID, &g.SchoolID, &g.AcademicYearID, &g.StudentID, &g.CriterionKey, &g.Grade, &g.CreatedAt, &g.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, g)
	}
	return items, rows.Err()
}

func (r *Repository) UpsertDisciplineGrade(ctx context.Context, g *domain.DisciplineGrade) error {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO discipline_grades (school_id, academic_year_id, student_id, criterion_key, grade)
		VALUES ($1,$2,$3,$4,$5)
		ON CONFLICT (student_id, academic_year_id, criterion_key)
		DO UPDATE SET grade=$5, updated_at=NOW()
		RETURNING id, created_at, updated_at`,
		g.SchoolID, g.AcademicYearID, g.StudentID, g.CriterionKey, g.Grade,
	)
	if err := row.Scan(&g.ID, &g.CreatedAt, &g.UpdatedAt); err != nil {
		return database.MapError(err)
	}
	return nil
}
