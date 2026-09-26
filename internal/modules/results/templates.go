package results

// This file is the single source of truth for the three fixed report-card
// layouts (grade_levels.report_card_template: "kg", "primary", "middle").
// Component keys/labels/max-marks and the discipline criteria list are pulled
// straight from the school's own report-card spreadsheets, not invented --
// including the exact percentage->grade formulas found in their cell
// formulas, so printed grades match what the school already hands out on
// paper. The frontend never hardcodes a second copy of any of this: it reads
// mark_components off the exam/grade-level API responses.

// MarkComponent is one graded column of a subject within a single exam
// (e.g. "Written" out of 60). A subject's max marks for that exam is the sum
// of its template's components.
type MarkComponent struct {
	Key      string `json:"key"`
	Label    string `json:"label"`
	MaxMarks int    `json:"max_marks"`
}

var kgComponents = []MarkComponent{
	{"written", "WRITTEN", 25},
	{"notebook", "NOTE BOOK", 5},
	{"activity", "ACTIVITY", 5},
	{"oral", "ORAL", 5},
	{"test", "TEST", 10},
}

var primaryComponents = []MarkComponent{
	{"written", "WRITTEN", 60},
	{"notebook", "NOTE BOOK", 5},
	{"activity", "ACTIVITY", 5},
	{"oral_pro", "ORAL / PRO", 10},
	{"test", "TEST", 20},
}

var middleComponents = []MarkComponent{
	{"test", "TEST", 20},
	{"project", "PROJECT", 20},
	{"theory", "THEORY", 60},
}

// MarkComponentsForTemplate returns the component scheme for a report-card
// template key, or nil for a grade with no template (which keeps using a
// single plain marks_obtained/max_marks, same as before this feature).
func MarkComponentsForTemplate(template *string) []MarkComponent {
	if template == nil {
		return nil
	}
	switch *template {
	case "kg":
		return kgComponents
	case "primary":
		return primaryComponents
	case "middle":
		return middleComponents
	default:
		return nil
	}
}

// componentsMaxTotal sums a scheme's max marks -- a subject's max marks for
// one exam under that template.
func componentsMaxTotal(components []MarkComponent) int {
	total := 0
	for _, c := range components {
		total += c.MaxMarks
	}
	return total
}

// disciplineCriteria are the "middle" template's fixed Co-Scholastic/
// Discipline criteria. Always typed by a teacher/admin (A+/A/B+/B/...),
// never computed from marks -- the source workbook has no formula for these.
var disciplineCriteria = []string{
	"WORK EDUCATION",
	"ARTS",
	"HEALTH / PHYSICAL EDUCATION",
	"SCIENTIFIC SKILLS",
	"SOCIAL SKILLS",
	"REGULARITY & PUNCTUALITY",
	"BEHAVIOUR & VALUES",
	"ATTITUDE TOWARDS TEACHER",
	"ATTITUDE TOWARDS SCHOOL MATES",
}

// gradeKG implements the KG template's exact formula:
// =IF(pct>=90,"A+",IF(pct>=80,"A",IF(pct>=70,"B+",IF(pct>=60,"B",
//    IF(pct>=50,"C+",IF(pct>=40,"C",IF(pct>=30,"D","F")))))))
func gradeKG(pct float64) string {
	switch {
	case pct >= 90:
		return "A+"
	case pct >= 80:
		return "A"
	case pct >= 70:
		return "B+"
	case pct >= 60:
		return "B"
	case pct >= 50:
		return "C+"
	case pct >= 40:
		return "C"
	case pct >= 30:
		return "D"
	default:
		return "F"
	}
}

// gradePrimary implements the 1st-4th template's exact formula:
// =IF(pct>=90,"A",IF(pct>=80,"B",IF(pct>=70,"C",IF(pct>=60,"D","F"))))
func gradePrimary(pct float64) string {
	switch {
	case pct >= 90:
		return "A"
	case pct >= 80:
		return "B"
	case pct >= 70:
		return "C"
	case pct >= 60:
		return "D"
	default:
		return "F"
	}
}

// gradeForTemplate returns the overall letter grade for a template, or ""
// for "middle" (that template prints percentage only -- no letter grade
// formula exists in the source workbook for the academic total).
func gradeForTemplate(template string, pct float64) string {
	switch template {
	case "kg":
		return gradeKG(pct)
	case "primary":
		return gradePrimary(pct)
	default:
		return ""
	}
}
