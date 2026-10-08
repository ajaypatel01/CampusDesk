package database

import (
	"errors"
	"fmt"
	"strings"

	apperr "github.com/ajaypatel01/CampusDesk/internal/platform/errors"
	"github.com/jackc/pgconn"
)

// conflictReasons says, per unique constraint, what a duplicate means to the
// person saving -- so a 409 explains itself instead of a bare "conflict".
// Constraint names are Postgres's defaults (<table>_<columns>_key) or the
// index names from the migrations.
var conflictReasons = map[string]string{
	"academic_years_school_id_name_key":                               "an academic year with this name already exists in this school",
	"attendance_records_student_id_record_date_key":                   "attendance is already recorded for this student on this date",
	"book_list_items_book_list_id_book_id_key":                        "this book is already on the book list",
	"book_lists_school_id_academic_year_id_grade_level_id_key":        "this grade already has a book list for this academic year",
	"class_sections_academic_year_id_grade_level_id_name_key":         "this grade already has a section with this name in this academic year",
	"custom_field_definitions_school_id_entity_type_field_key_key":    "a field with this name already exists here",
	"custom_field_values_definition_id_entity_id_scope_id_key":        "this field already has a value saved",
	"discipline_grades_student_id_academic_year_id_criterion_key_key": "this discipline grade is already saved for this student and year",
	"enrollments_student_id_academic_year_id_key":                     "this student is already enrolled for this academic year",
	"exam_mark_components_exam_mark_id_component_key_key":             "this mark field is already saved for this student and exam",
	"exam_marks_exam_id_student_id_subject_id_key":                    "marks for this subject are already saved for this student in this exam",
	"fee_installment_plans_fee_structure_id_installment_number_key":   "this fee structure already has an installment with this number",
	"fee_structures_school_id_academic_year_id_grade_level_id_key":    "this grade already has a fee structure for this academic year",
	"grade_levels_school_id_name_key":                                 "a grade with this name already exists in this school",
	"homework_submissions_assignment_id_student_id_key":               "this student already has a submission for this homework",
	"marksheet_total_overrides_exam_id_student_id_key":                "this marksheet total has already been edited",
	"password_reset_tokens_token_hash_key":                            "this password reset link was already issued -- request a new one",
	"permission_overrides_user_id_feature_key_key":                    "this user already has a permission setting for this feature",
	"report_card_details_student_id_academic_year_id_key":             "report card details are already saved for this student and year",
	"rte_quotas_school_id_academic_year_id_grade_level_id_key":        "this grade already has an RTE quota for this academic year",
	"schools_code_key":                                        "a school with this code already exists",
	"staff_profiles_user_id_key":                              "this staff member already has a profile",
	"student_book_receipts_student_id_book_list_id_key":       "this student has already received this book list",
	"student_fee_accounts_student_id_academic_year_id_key":    "this student already has a fee account for this academic year",
	"student_van_assignments_student_id_academic_year_id_key": "this student is already assigned a van for this academic year",
	"students_school_id_student_code_key":                     "another student in this school already has this scholar no.",
	"subject_mark_components_subject_id_key_key":              "this subject already has a field with this name",
	"subjects_school_id_grade_level_id_name_key":              "this grade already has a subject with this name",
	"idx_users_phone_number":                                  "this phone number is already used by another account",
	"users_email_key":                                         "this email is already used by another account",
	"vans_school_id_van_number_key":                           "a van with this number already exists in this school",
}

// MapError turns a unique violation into apperr.ErrConflict carrying the
// reason (errors.Is(err, apperr.ErrConflict) still holds).
func MapError(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return fmt.Errorf("%w: %s", apperr.ErrConflict, conflictReason(pgErr))
	}
	return err
}

// conflictReason is the reason for one unique violation: the known sentence
// for its constraint, or one built from the table and its non-id columns.
func conflictReason(pgErr *pgconn.PgError) string {
	if reason, ok := conflictReasons[pgErr.ConstraintName]; ok {
		return reason
	}
	thing := strings.TrimSuffix(strings.ReplaceAll(pgErr.TableName, "_", " "), "s")
	if thing == "" {
		thing = "record"
	}
	var cols []string
	// Detail looks like: Key (school_id, van_number)=(..., 7) already exists.
	if open := strings.Index(pgErr.Detail, "("); open >= 0 {
		if end := strings.Index(pgErr.Detail[open:], ")"); end > 0 {
			for _, c := range strings.Split(pgErr.Detail[open+1:open+end], ",") {
				c = strings.TrimSpace(c)
				if c != "" && c != "id" && !strings.HasSuffix(c, "_id") {
					cols = append(cols, strings.ReplaceAll(c, "_", " "))
				}
			}
		}
	}
	if len(cols) == 0 {
		return fmt.Sprintf("this %s already exists", thing)
	}
	return fmt.Sprintf("a %s with this %s already exists", thing, strings.Join(cols, " and "))
}
