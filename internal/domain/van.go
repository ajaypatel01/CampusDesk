package domain

import (
	"time"

	"github.com/google/uuid"
)

type Van struct {
	ID          uuid.UUID `json:"id"`
	SchoolID    uuid.UUID `json:"school_id"`
	VanNumber   string    `json:"van_number"`
	DriverName  string    `json:"driver_name"`
	DriverPhone string    `json:"driver_phone,omitempty"`
	Capacity    int       `json:"capacity"`
	RouteName   string    `json:"route_name,omitempty"`
	Notes       string    `json:"notes,omitempty"`
	IsActive    bool      `json:"is_active"`
	Timestamps
}

type VanRoute struct {
	ID         uuid.UUID `json:"id"`
	VanID      uuid.UUID `json:"van_id"`
	StopName   string    `json:"stop_name"`
	StopOrder  int       `json:"stop_order"`
	MonthlyFee int       `json:"monthly_fee"`
	Timestamps
}

type StudentVanAssignment struct {
	ID             uuid.UUID `json:"id"`
	StudentID      uuid.UUID `json:"student_id"`
	VanID          uuid.UUID `json:"van_id"`
	AcademicYearID uuid.UUID `json:"academic_year_id"`
	PickupStop     string    `json:"pickup_stop,omitempty"`
	AssignedDate   time.Time `json:"assigned_date"`
	IsActive       bool      `json:"is_active"`
	Timestamps
}
