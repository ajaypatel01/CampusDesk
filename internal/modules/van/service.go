package van

import (
	"context"
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

type CreateVanInput struct {
	SchoolID    uuid.UUID `json:"school_id"`
	VanNumber   string    `json:"van_number"`
	DriverName  string    `json:"driver_name"`
	DriverPhone string    `json:"driver_phone"`
	Capacity    int       `json:"capacity"`
	RouteName   string    `json:"route_name"`
	Notes       string    `json:"notes"`
}

type UpdateVanInput struct {
	VanNumber   string `json:"van_number"`
	DriverName  string `json:"driver_name"`
	DriverPhone string `json:"driver_phone"`
	Capacity    int    `json:"capacity"`
	RouteName   string `json:"route_name"`
	Notes       string `json:"notes"`
	IsActive    *bool  `json:"is_active"`
}

type AddRouteInput struct {
	StopName   string `json:"stop_name"`
	StopOrder  int    `json:"stop_order"`
	MonthlyFee int    `json:"monthly_fee"`
}

type AssignStudentInput struct {
	StudentID      uuid.UUID `json:"student_id"`
	VanID          uuid.UUID `json:"van_id"`
	AcademicYearID uuid.UUID `json:"academic_year_id"`
	PickupStop     string    `json:"pickup_stop"`
}

type VanDetail struct {
	domain.Van
	Routes      []domain.VanRoute  `json:"routes"`
	Assignments []AssignmentDetail `json:"assignments,omitempty"`
}

// ---- Service methods ----

func (s *Service) CreateVan(ctx context.Context, in CreateVanInput) (*domain.Van, error) {
	if in.SchoolID == uuid.Nil || in.VanNumber == "" || in.DriverName == "" {
		return nil, apperr.ErrInvalidInput
	}
	if in.Capacity <= 0 {
		in.Capacity = 20
	}
	v := &domain.Van{
		SchoolID: in.SchoolID, VanNumber: in.VanNumber, DriverName: in.DriverName,
		DriverPhone: in.DriverPhone, Capacity: in.Capacity, RouteName: in.RouteName,
		Notes: in.Notes, IsActive: true,
	}
	if err := s.repo.CreateVan(ctx, v); err != nil {
		return nil, err
	}
	return v, nil
}

func (s *Service) GetVan(ctx context.Context, id, yearID uuid.UUID) (*VanDetail, error) {
	v, err := s.repo.GetVanByID(ctx, id)
	if err != nil {
		return nil, err
	}
	routes, _ := s.repo.ListRoutes(ctx, id)
	detail := &VanDetail{Van: *v, Routes: routes}
	if yearID != uuid.Nil {
		detail.Assignments, _ = s.repo.ListAssignments(ctx, id, yearID)
	}
	return detail, nil
}

func (s *Service) ListVans(ctx context.Context, schoolID uuid.UUID) ([]domain.Van, error) {
	if schoolID == uuid.Nil {
		return nil, apperr.ErrInvalidInput
	}
	return s.repo.ListVans(ctx, schoolID)
}

func (s *Service) UpdateVan(ctx context.Context, id uuid.UUID, in UpdateVanInput) (*domain.Van, error) {
	v, err := s.repo.GetVanByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if in.VanNumber != "" {
		v.VanNumber = in.VanNumber
	}
	if in.DriverName != "" {
		v.DriverName = in.DriverName
	}
	v.DriverPhone = in.DriverPhone
	if in.Capacity > 0 {
		v.Capacity = in.Capacity
	}
	v.RouteName = in.RouteName
	v.Notes = in.Notes
	if in.IsActive != nil {
		v.IsActive = *in.IsActive
	}
	if err := s.repo.UpdateVan(ctx, v); err != nil {
		return nil, err
	}
	return v, nil
}

func (s *Service) DeleteVan(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteVan(ctx, id)
}

func (s *Service) AddRoute(ctx context.Context, vanID uuid.UUID, in AddRouteInput) (*domain.VanRoute, error) {
	if in.StopName == "" {
		return nil, apperr.ErrInvalidInput
	}
	rt := &domain.VanRoute{
		VanID: vanID, StopName: in.StopName, StopOrder: in.StopOrder, MonthlyFee: in.MonthlyFee,
	}
	if err := s.repo.AddRoute(ctx, rt); err != nil {
		return nil, err
	}
	return rt, nil
}

func (s *Service) DeleteRoute(ctx context.Context, routeID uuid.UUID) error {
	return s.repo.DeleteRoute(ctx, routeID)
}

func (s *Service) AssignStudent(ctx context.Context, in AssignStudentInput) (*domain.StudentVanAssignment, error) {
	if in.StudentID == uuid.Nil || in.VanID == uuid.Nil || in.AcademicYearID == uuid.Nil {
		return nil, apperr.ErrInvalidInput
	}
	a := &domain.StudentVanAssignment{
		StudentID: in.StudentID, VanID: in.VanID, AcademicYearID: in.AcademicYearID,
		PickupStop: in.PickupStop, AssignedDate: time.Now(), IsActive: true,
	}
	if err := s.repo.AssignStudent(ctx, a); err != nil {
		return nil, err
	}
	return a, nil
}

func (s *Service) RemoveAssignment(ctx context.Context, id uuid.UUID) error {
	return s.repo.RemoveAssignment(ctx, id)
}

func (s *Service) ListAssignments(ctx context.Context, vanID, yearID uuid.UUID) ([]AssignmentDetail, error) {
	if vanID == uuid.Nil || yearID == uuid.Nil {
		return nil, apperr.ErrInvalidInput
	}
	return s.repo.ListAssignments(ctx, vanID, yearID)
}
