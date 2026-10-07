package academic

import (
	"context"
	"fmt"
	"strings"
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

type YearInput struct {
	SchoolID  uuid.UUID `json:"school_id"`
	Name      string    `json:"name"`
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
	IsCurrent bool      `json:"is_current"`
}

type GradeInput struct {
	SchoolID  uuid.UUID `json:"school_id"`
	Name      string    `json:"name"`
	SortOrder int       `json:"sort_order"`
}

type GradeUpdateInput struct {
	Name               string  `json:"name"`
	SortOrder          int     `json:"sort_order"`
	ReportCardTemplate *string `json:"report_card_template"`
}

var validReportCardTemplates = map[string]bool{"kg": true, "primary": true, "middle": true}

type SectionInput struct {
	SchoolID          uuid.UUID  `json:"school_id"`
	AcademicYearID    uuid.UUID  `json:"academic_year_id"`
	GradeLevelID      uuid.UUID  `json:"grade_level_id"`
	Name              string     `json:"name"`
	Capacity          int        `json:"capacity"`
	HomeroomTeacherID *uuid.UUID `json:"homeroom_teacher_id"`
	// ViceTeacherIDs: vice class teachers (any number), same access as the
	// class teacher.
	ViceTeacherIDs []uuid.UUID `json:"vice_teacher_ids"`
}

// SectionUpdateInput mirrors SectionInput minus the identifiers that never
// change after creation (school/year/grade) -- a section can only be
// renamed, resized, or reassigned a class teacher.
type SectionUpdateInput struct {
	Name              string     `json:"name"`
	Capacity          int        `json:"capacity"`
	HomeroomTeacherID *uuid.UUID `json:"homeroom_teacher_id"`
	// ViceTeacherIDs replaces the vice class teachers when sent; when the
	// field is left out the current ones are kept, so an older client that
	// doesn't know about vice teachers can't clear them by saving.
	ViceTeacherIDs *[]uuid.UUID `json:"vice_teacher_ids"`
}

func (s *Service) CreateYear(ctx context.Context, in YearInput) (*domain.AcademicYear, error) {
	if in.SchoolID == uuid.Nil || strings.TrimSpace(in.Name) == "" || !in.EndDate.After(in.StartDate) {
		return nil, apperr.ErrInvalidInput
	}
	y := &domain.AcademicYear{
		SchoolID: in.SchoolID, Name: strings.TrimSpace(in.Name),
		StartDate: in.StartDate, EndDate: in.EndDate, IsCurrent: in.IsCurrent,
	}
	if err := s.repo.CreateYear(ctx, y); err != nil {
		return nil, err
	}
	return y, nil
}

func (s *Service) ListYears(ctx context.Context, schoolID uuid.UUID) ([]domain.AcademicYear, error) {
	if schoolID == uuid.Nil {
		return nil, apperr.ErrInvalidInput
	}
	return s.repo.ListYears(ctx, schoolID)
}

func (s *Service) CreateGrade(ctx context.Context, in GradeInput) (*domain.GradeLevel, error) {
	if in.SchoolID == uuid.Nil || strings.TrimSpace(in.Name) == "" {
		return nil, apperr.ErrInvalidInput
	}
	g := &domain.GradeLevel{SchoolID: in.SchoolID, Name: strings.TrimSpace(in.Name), SortOrder: in.SortOrder}
	if err := s.repo.CreateGrade(ctx, g); err != nil {
		return nil, err
	}
	return g, nil
}

func (s *Service) ListGrades(ctx context.Context, schoolID uuid.UUID) ([]domain.GradeLevel, error) {
	if schoolID == uuid.Nil {
		return nil, apperr.ErrInvalidInput
	}
	return s.repo.ListGrades(ctx, schoolID)
}

func (s *Service) UpdateGrade(ctx context.Context, id uuid.UUID, in GradeUpdateInput) (*domain.GradeLevel, error) {
	if id == uuid.Nil || strings.TrimSpace(in.Name) == "" {
		return nil, apperr.ErrInvalidInput
	}
	if in.ReportCardTemplate != nil && *in.ReportCardTemplate != "" && !validReportCardTemplates[*in.ReportCardTemplate] {
		return nil, apperr.ErrInvalidInput
	}
	// An empty string from a UI "None" option means "clear it", same as nil.
	template := in.ReportCardTemplate
	if template != nil && *template == "" {
		template = nil
	}
	g := &domain.GradeLevel{ID: id, Name: strings.TrimSpace(in.Name), SortOrder: in.SortOrder, ReportCardTemplate: template}
	if err := s.repo.UpdateGrade(ctx, g); err != nil {
		return nil, err
	}
	return s.repo.GetGrade(ctx, id)
}

func (s *Service) CreateSection(ctx context.Context, in SectionInput) (*domain.ClassSection, error) {
	if in.SchoolID == uuid.Nil || in.AcademicYearID == uuid.Nil || in.GradeLevelID == uuid.Nil || strings.TrimSpace(in.Name) == "" {
		return nil, apperr.ErrInvalidInput
	}
	cap := in.Capacity
	if cap <= 0 {
		cap = 30
	}
	c := &domain.ClassSection{
		SchoolID: in.SchoolID, AcademicYearID: in.AcademicYearID, GradeLevelID: in.GradeLevelID,
		Name: strings.TrimSpace(in.Name), Capacity: cap, HomeroomTeacherID: in.HomeroomTeacherID,
	}
	vice, err := s.cleanViceTeachers(ctx, in.SchoolID, in.HomeroomTeacherID, in.ViceTeacherIDs)
	if err != nil {
		return nil, err
	}
	if err := s.repo.CreateSection(ctx, c); err != nil {
		return nil, err
	}
	if len(vice) > 0 {
		if err := s.repo.SetSectionViceTeachers(ctx, c.ID, vice); err != nil {
			return nil, err
		}
	}
	return s.repo.GetSection(ctx, c.ID)
}

// cleanViceTeachers drops duplicates and the class teacher himself/herself
// from a vice list, and checks every one is an active teacher at the school.
func (s *Service) cleanViceTeachers(ctx context.Context, schoolID uuid.UUID, classTeacher *uuid.UUID, ids []uuid.UUID) ([]uuid.UUID, error) {
	seen := map[uuid.UUID]bool{}
	out := []uuid.UUID{}
	for _, id := range ids {
		if id == uuid.Nil || seen[id] || (classTeacher != nil && *classTeacher == id) {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	if len(out) == 0 {
		return out, nil
	}
	ok, err := s.repo.TeachersInSchool(ctx, schoolID, out)
	if err != nil {
		return nil, err
	}
	for _, id := range out {
		if !ok[id] {
			return nil, fmt.Errorf("%w: every vice class teacher must be an active teacher at this school", apperr.ErrInvalidInput)
		}
	}
	return out, nil
}

func (s *Service) ListSections(ctx context.Context, schoolID, yearID uuid.UUID) ([]domain.ClassSection, error) {
	if schoolID == uuid.Nil || yearID == uuid.Nil {
		return nil, apperr.ErrInvalidInput
	}
	return s.repo.ListSections(ctx, schoolID, yearID)
}

// UpdateSection renames/resizes a section or (re)assigns its class teacher.
// A nil HomeroomTeacherID clears the assignment -- same "explicit null
// means clear" convention as UpdateGrade's report_card_template above.
func (s *Service) UpdateSection(ctx context.Context, id uuid.UUID, in SectionUpdateInput) (*domain.ClassSection, error) {
	if id == uuid.Nil || strings.TrimSpace(in.Name) == "" {
		return nil, apperr.ErrInvalidInput
	}
	cap := in.Capacity
	if cap <= 0 {
		cap = 30
	}
	current, err := s.repo.GetSection(ctx, id)
	if err != nil {
		return nil, err
	}
	viceIn := current.ViceTeacherIDs
	if in.ViceTeacherIDs != nil {
		viceIn = *in.ViceTeacherIDs
	}
	// Re-cleaned even when unchanged, so a vice teacher who has just been
	// made the class teacher isn't listed twice.
	vice, err := s.cleanViceTeachers(ctx, current.SchoolID, in.HomeroomTeacherID, viceIn)
	if err != nil {
		return nil, err
	}
	c := &domain.ClassSection{ID: id, Name: strings.TrimSpace(in.Name), Capacity: cap, HomeroomTeacherID: in.HomeroomTeacherID}
	if err := s.repo.UpdateSection(ctx, c); err != nil {
		return nil, err
	}
	if err := s.repo.SetSectionViceTeachers(ctx, id, vice); err != nil {
		return nil, err
	}
	return s.repo.GetSection(ctx, id)
}
