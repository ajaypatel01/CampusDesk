package domain

// Feature is one permissionable section of the app -- roughly one module.
// Key is what modules pass to httpx.RequireFeature and what
// permission_overrides.feature_key stores; Label is what the access-control
// matrix shows an admin.
type Feature struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

// Features is the fixed catalog of every permissionable section, in the
// order the access-control matrix displays them. Adding a module's routes
// under a new key here (and wiring httpx.RequireFeature at the route) is
// what makes a section appear in the matrix and become assignable.
var Features = []Feature{
	{Key: "students", Label: "Student Records"},
	{Key: "staff", Label: "Staff & HR"},
	{Key: "fees", Label: "Fee Management"},
	{Key: "results", Label: "Exams & Results"},
	{Key: "homework", Label: "Homework"},
	{Key: "documents", Label: "Documents (Bonafide / TC / Salary Slip)"},
	{Key: "communications", Label: "Broadcasts & Communications"},
	{Key: "books", Label: "Library / Books"},
	{Key: "transport", Label: "Transport (Vans)"},
	{Key: "rte", Label: "RTE Quotas"},
	{Key: "id_cards", Label: "ID Cards"},
	{Key: "payroll", Label: "Payroll & Staff Leaves"},
	{Key: "academic", Label: "Academic Years, Grades & Sections"},
	{Key: "enrollment", Label: "Enrollment & Attendance"},
	{Key: "guardians", Label: "Guardians"},
	{Key: "tc_vouchers", Label: "TC & Vouchers"},
	{Key: "custom_fields", Label: "Custom Fields"},
	{Key: "user_management", Label: "User Accounts & Approvals"},
	{Key: "school_settings", Label: "School Settings"},
}

var validFeatureKeys = func() map[string]bool {
	m := make(map[string]bool, len(Features))
	for _, f := range Features {
		m[f.Key] = true
	}
	return m
}()

// IsValidFeatureKey reports whether key is a known permission-matrix feature.
func IsValidFeatureKey(key string) bool { return validFeatureKeys[key] }
