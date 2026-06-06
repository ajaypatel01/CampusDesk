package domain

import "time"

type StudentStatus string

const (
	StudentStatusActive      StudentStatus = "active"
	StudentStatusInactive    StudentStatus = "inactive"
	StudentStatusGraduated   StudentStatus = "graduated"
	StudentStatusTransferred StudentStatus = "transferred"
)

type UserRole string

const (
	RoleSuperAdmin  UserRole = "super_admin"
	RoleSchoolAdmin UserRole = "school_admin"
	RoleTeacher     UserRole = "teacher"
	RoleRegistrar   UserRole = "registrar"
	RoleParent      UserRole = "parent"
)

type EnrollmentStatus string

const (
	EnrollmentStatusActive    EnrollmentStatus = "active"
	EnrollmentStatusWithdrawn EnrollmentStatus = "withdrawn"
	EnrollmentStatusCompleted EnrollmentStatus = "completed"
)

type AttendanceStatus string

const (
	AttendancePresent AttendanceStatus = "present"
	AttendanceAbsent  AttendanceStatus = "absent"
	AttendanceLate    AttendanceStatus = "late"
	AttendanceExcused AttendanceStatus = "excused"
)

type ResultStatus string

const (
	ResultStatusDraft     ResultStatus = "draft"
	ResultStatusPublished ResultStatus = "published"
)

type Timestamps struct {
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
