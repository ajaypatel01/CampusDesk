package domain

import "time"

type StudentStatus string

const (
	StudentStatusActive      StudentStatus = "active"
	StudentStatusInactive    StudentStatus = "inactive"
	StudentStatusGraduated   StudentStatus = "graduated"
	StudentStatusTransferred StudentStatus = "transferred"
	// StudentStatusLeftWithoutTC is a student who stopped coming without
	// taking a Transfer Certificate ("inactive" is left with a TC).
	StudentStatusLeftWithoutTC StudentStatus = "left_without_tc"
	// StudentStatusDefaulted marks a fee defaulter.
	StudentStatusDefaulted StudentStatus = "defaulted"
	// StudentStatusDuplicate marks a record entered twice by mistake. It is
	// left out of every list and total (fees, results, broadcasts, ...) and
	// only shows in the Students list when filtering by this status.
	StudentStatusDuplicate StudentStatus = "duplicate"
)

type UserRole string

const (
	RoleSuperAdmin  UserRole = "super_admin"
	RoleSchoolAdmin UserRole = "school_admin"
	RoleTeacher     UserRole = "teacher"
	RoleRegistrar   UserRole = "registrar"
	RoleParent      UserRole = "parent"
)

// UserStatus tracks the approval state of a user account. Self-registered
// accounts start as pending and cannot log in until an admin approves them.
type UserStatus string

const (
	UserStatusPending  UserStatus = "pending"
	UserStatusApproved UserStatus = "approved"
	UserStatusRejected UserStatus = "rejected"
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

type FeeType string

const (
	FeeTypeTuition      FeeType = "tuition"
	FeeTypeVan          FeeType = "van"
	FeeTypePreviousDues FeeType = "previous_dues"
)

type PaymentMode string

const (
	PaymentModeCash   PaymentMode = "cash"
	PaymentModeOnline PaymentMode = "online"
	PaymentModeCheque PaymentMode = "cheque"
	PaymentModeUPI    PaymentMode = "upi"
)

type Timestamps struct {
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
