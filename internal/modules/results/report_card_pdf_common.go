package results

import (
	"fmt"
	"math"

	"github.com/go-pdf/fpdf"
)

// This file holds the layout pieces shared by the "kg" and "primary"
// report-card PDFs (they're structurally identical -- same header, same
// per-exam component table, same OVER ALL table -- differing only in the
// component scheme/labels, the grading scale, and whether the MORAL/G.K
// remark row is printed). Each exam is rendered as its own full-width block,
// stacked top to bottom, rather than the source spreadsheet's side-by-side
// pairing -- same fields, same numbers, same order, just simpler and more
// robust to lay out than replicating exact Excel cell coordinates.
//
// Every function here takes a pdfTheme (see report_card_theme.go): the
// theme only ever changes fill/text colors, never the content or layout
// itself -- the school's actual rubric and grading formulas stay identical
// across every theme.

func formatMarks(v float64) string {
	if v == math.Trunc(v) {
		return fmt.Sprintf("%d", int(v))
	}
	return fmt.Sprintf("%.1f", v)
}

func romanNumeral(n int) string {
	switch n {
	case 1:
		return "I"
	case 2:
		return "II"
	case 3:
		return "III"
	default:
		return fmt.Sprintf("%d", n)
	}
}

// reportCardDetailsFields pulls out the handwritten fields with nil-safety,
// since rc.Details is nil until a teacher/admin has filled them in once.
func reportCardDetailsFields(rc ReportCard) (rollNo, attendance, remark, promotedTo, moral, gk string) {
	if rc.Details == nil {
		return "", "", "", "", "", ""
	}
	d := rc.Details
	return d.RollNo, d.Attendance, d.Remark, d.PromotedTo, d.MoralRemark, d.GKRemark
}

func writeReportCardHeader(pdf *fpdf.Fpdf, w float64, rc ReportCard, theme pdfTheme) {
	pdf.SetFont("Arial", "B", 11)
	codeLine := fmt.Sprintf("SCHOOL CODE - %s", rc.SchoolCode)
	if rc.DiceCode != "" {
		codeLine += fmt.Sprintf("        DICE CODE - %s", rc.DiceCode)
	}
	pdf.CellFormat(w, 6, codeLine, "", 1, "C", false, 0, "")
	pdf.SetFont("Arial", "B", 13)
	pdf.CellFormat(w, 7, rc.SchoolName, "", 1, "C", false, 0, "")
	if rc.SchoolAddress != "" {
		pdf.SetFont("Arial", "", 9)
		pdf.CellFormat(w, 5, rc.SchoolAddress, "", 1, "C", false, 0, "")
	}
	pdf.SetFont("Arial", "BU", 11)
	pdf.CellFormat(w, 8, fmt.Sprintf("REPORT CARD - ACADEMIC SESSION %s", rc.AcademicYear), "", 1, "C", false, 0, "")
	pdf.Ln(1)

	rollNo, _, _, _, _, _ := reportCardDetailsFields(rc)
	dob := ""
	if rc.DateOfBirth != nil {
		dob = rc.DateOfBirth.Format("02/01/2006")
	}

	labelValue := func(label, value string, lw, vw float64) {
		pdf.SetFont("Arial", "B", 9)
		pdf.CellFormat(lw, 6, label, "1", 0, "L", false, 0, "")
		pdf.SetFont("Arial", "", 9)
		pdf.CellFormat(vw, 6, value, "1", 0, "L", false, 0, "")
	}
	third := w / 3
	labelValue("SCHOLAR NO.", rc.StudentCode, third*0.55, third*0.45)
	labelValue("PEN NUMBER", rc.PenNumber, third*0.55, third*0.45)
	labelValue("APAR ID", rc.AparID, third*0.55, third*0.45)
	pdf.Ln(-1)
	labelValue("STUDENT NAME", rc.StudentName, third*0.55, third*0.45)
	labelValue("D.O.B", dob, third*0.55, third*0.45)
	labelValue("CLASS", rc.GradeLevelName, third*0.55, third*0.45)
	pdf.Ln(-1)
	labelValue("FATHER'S NAME", rc.FatherName, third*0.55, third*0.45)
	labelValue("ROLL NO.", rollNo, third*0.55, third*0.45)
	labelValue("MOTHER'S NAME", rc.MotherName, third*0.55, third*0.45)
	pdf.Ln(-1)
	pdf.Ln(3)
}

// writeGradeCell renders one grade value, color-coded per theme.ColorBadges
// (a plain bordered cell with theme off, exactly as before this feature).
func writeGradeCell(pdf *fpdf.Fpdf, w, h float64, grade string, theme pdfTheme) {
	fill, text, shaded := theme.gradeBadge(grade)
	pdf.SetFillColor(fill[0], fill[1], fill[2])
	pdf.SetTextColor(text[0], text[1], text[2])
	pdf.CellFormat(w, h, grade, "1", 1, "C", shaded, 0, "")
	pdf.SetTextColor(0, 0, 0)
}

// examColumns is the union of the fields the exam's subjects use, in first-
// seen order, with each field's max marks when every subject that has it
// agrees (0 otherwise -- printed as "-"). Subjects can have different fields
// per exam, so this is built from the report card's cells rather than the
// grade template.
func examColumns(rc ReportCard, examIdx int) []MarkComponent {
	var cols []MarkComponent
	idx := map[string]int{}
	for _, sub := range rc.Subjects {
		for _, cv := range sub.ByExam[examIdx].Components {
			i, seen := idx[cv.Key]
			if !seen {
				idx[cv.Key] = len(cols)
				cols = append(cols, MarkComponent{Key: cv.Key, Label: cv.Label, MaxMarks: cv.MaxMarks})
				continue
			}
			if cols[i].MaxMarks != cv.MaxMarks {
				cols[i].MaxMarks = 0
			}
		}
	}
	return cols
}

// writeExamComponentTable renders one exam's SUBJECT/MARKING/subjects/
// G.TOTAL/PERCENTAGE block for the "kg"/"primary"/"middle" templates.
func writeExamComponentTable(pdf *fpdf.Fpdf, w float64, rc ReportCard, examIdx int, theme pdfTheme) {
	exam := rc.Exams[examIdx]
	components := examColumns(rc, examIdx)
	pdf.SetFont("Arial", "B", 10)
	fill := theme.setSectionStyle(pdf)
	pdf.CellFormat(w, 7, fmt.Sprintf("%s (%s)", exam.ExamName, romanNumeral(exam.Position)), "1", 1, "C", fill, 0, "")
	pdf.SetTextColor(0, 0, 0)

	totalW := 22.0
	subjectW := 45.0
	compW := 0.0
	if len(components) > 0 {
		compW = (w - subjectW - totalW) / float64(len(components))
	} else {
		subjectW = w - totalW
	}

	pdf.SetFont("Arial", "B", 8)
	fill = theme.setHeaderStyle(pdf)
	pdf.CellFormat(subjectW, 7, "SUBJECT", "1", 0, "C", fill, 0, "")
	for _, c := range components {
		pdf.CellFormat(compW, 7, c.Label, "1", 0, "C", fill, 0, "")
	}
	pdf.CellFormat(totalW, 7, "TOTAL", "1", 1, "C", fill, 0, "")
	pdf.SetTextColor(0, 0, 0)

	pdf.SetFont("Arial", "", 8)
	pdf.CellFormat(subjectW, 6, "MARKING", "1", 0, "L", false, 0, "")
	uniform := true
	markingTotal := 0
	for _, c := range components {
		if c.MaxMarks == 0 {
			uniform = false
			pdf.CellFormat(compW, 6, "-", "1", 0, "C", false, 0, "")
			continue
		}
		pdf.CellFormat(compW, 6, fmt.Sprintf("%d", c.MaxMarks), "1", 0, "C", false, 0, "")
		markingTotal += c.MaxMarks
	}
	if uniform && len(components) > 0 {
		pdf.CellFormat(totalW, 6, fmt.Sprintf("%d", markingTotal), "1", 1, "C", false, 0, "")
	} else {
		pdf.CellFormat(totalW, 6, "-", "1", 1, "C", false, 0, "")
	}

	colTotals := make([]float64, len(components))
	var gTotal float64
	var gMax int
	for _, sub := range rc.Subjects {
		cell := sub.ByExam[examIdx]
		gMax += cell.MaxMarks
		pdf.CellFormat(subjectW, 6, sub.SubjectName, "1", 0, "L", false, 0, "")
		if cell.IsAbsent {
			for range components {
				pdf.CellFormat(compW, 6, "AB", "1", 0, "C", false, 0, "")
			}
			pdf.CellFormat(totalW, 6, "AB", "1", 1, "C", false, 0, "")
			continue
		}
		byKey := make(map[string]float64, len(cell.Components))
		for _, cv := range cell.Components {
			byKey[cv.Key] = cv.Obtained
		}
		for i, c := range components {
			v, has := byKey[c.Key]
			if !has {
				pdf.CellFormat(compW, 6, "-", "1", 0, "C", false, 0, "")
				continue
			}
			colTotals[i] += v
			pdf.CellFormat(compW, 6, formatMarks(v), "1", 0, "C", false, 0, "")
		}
		total := formatMarks(cell.Obtained)
		if cell.GradeLetter != "" {
			total = cell.GradeLetter
		} else if !uniform || len(components) == 0 {
			total = fmt.Sprintf("%s/%d", total, cell.MaxMarks)
		}
		pdf.CellFormat(totalW, 6, total, "1", 1, "C", false, 0, "")
		gTotal += cell.Obtained
	}

	pdf.SetFont("Arial", "B", 8)
	pdf.CellFormat(subjectW, 6, "G.TOTAL", "1", 0, "L", false, 0, "")
	for _, ct := range colTotals {
		pdf.CellFormat(compW, 6, formatMarks(ct), "1", 0, "C", false, 0, "")
	}
	pdf.CellFormat(totalW, 6, formatMarks(gTotal), "1", 1, "C", false, 0, "")

	// Each subject's own max for this exam: identical to the old
	// MARKING-total x subjects when every subject shares one scheme.
	pct := 0.0
	if gMax > 0 {
		pct = gTotal / float64(gMax) * 100
	}
	pdf.CellFormat(w-totalW, 6, "PERCENTAGE", "1", 0, "L", false, 0, "")
	pdf.CellFormat(totalW, 6, fmt.Sprintf("%.1f%%", pct), "1", 1, "C", false, 0, "")
	pdf.Ln(2)
}

// writeOverallTable renders the "OVER ALL (I+II+...)" per-subject total/%/
// grade block, using rc's own combined totals rather than re-summing --
// GetReportCard already excludes co-scholastic subjects the same way the
// existing single-exam marksheet does.
func writeOverallTable(pdf *fpdf.Fpdf, w float64, rc ReportCard, theme pdfTheme) {
	romanSpan := ""
	for i, e := range rc.Exams {
		if i > 0 {
			romanSpan += "+"
		}
		romanSpan += romanNumeral(e.Position)
	}
	pdf.SetFont("Arial", "B", 10)
	fill := theme.setSectionStyle(pdf)
	pdf.CellFormat(w, 7, fmt.Sprintf("OVER ALL (%s)", romanSpan), "1", 1, "C", fill, 0, "")
	pdf.SetTextColor(0, 0, 0)

	subjectW := 60.0
	otherW := (w - subjectW) / 3
	pdf.SetFont("Arial", "B", 8)
	fill = theme.setHeaderStyle(pdf)
	pdf.CellFormat(subjectW, 7, "SUBJECT", "1", 0, "C", fill, 0, "")
	pdf.CellFormat(otherW, 7, "TOTAL", "1", 0, "C", fill, 0, "")
	pdf.CellFormat(otherW, 7, "%", "1", 0, "C", fill, 0, "")
	pdf.CellFormat(otherW, 7, "GRADE", "1", 1, "C", fill, 0, "")
	pdf.SetTextColor(0, 0, 0)

	pdf.SetFont("Arial", "", 8)
	for _, sub := range rc.Subjects {
		pdf.CellFormat(subjectW, 6, sub.SubjectName, "1", 0, "L", false, 0, "")
		if sub.IsGraded {
			pdf.CellFormat(otherW, 6, "-", "1", 0, "C", false, 0, "")
			pdf.CellFormat(otherW, 6, "-", "1", 0, "C", false, 0, "")
		} else {
			pdf.CellFormat(otherW, 6, fmt.Sprintf("%s / %d", formatMarks(sub.OverallObtained), sub.OverallMax), "1", 0, "C", false, 0, "")
			pdf.CellFormat(otherW, 6, fmt.Sprintf("%.1f%%", sub.OverallPercent), "1", 0, "C", false, 0, "")
		}
		writeGradeCell(pdf, otherW, 6, sub.Grade, theme)
	}

	pdf.SetFont("Arial", "B", 9)
	fill = theme.setTotalStyle(pdf)
	pdf.CellFormat(subjectW, 7, fmt.Sprintf("G.TOTAL (%d)", rc.OverallMax), "1", 0, "L", fill, 0, "")
	pdf.CellFormat(otherW, 7, formatMarks(rc.OverallObtained), "1", 0, "C", fill, 0, "")
	pdf.CellFormat(otherW, 7, fmt.Sprintf("%.1f%%", rc.OverallPercent), "1", 0, "C", fill, 0, "")
	pdf.CellFormat(otherW, 7, rc.OverallGrade, "1", 1, "C", fill, 0, "")
	pdf.SetTextColor(0, 0, 0)
	pdf.Ln(3)
}

func writeReportCardFooterDetails(pdf *fpdf.Fpdf, w float64, rc ReportCard) {
	_, attendance, remark, promotedTo, _, _ := reportCardDetailsFields(rc)
	pdf.SetFont("Arial", "B", 9)
	third := w / 3
	pdf.CellFormat(third, 6, "ATTENDANCE: "+attendance, "1", 0, "L", false, 0, "")
	pdf.CellFormat(third, 6, "REMARK: "+remark, "1", 0, "L", false, 0, "")
	pdf.CellFormat(third, 6, "PROMOTED TO: "+promotedTo, "1", 1, "L", false, 0, "")
	pdf.Ln(2)
}

func writeReportCardSignatureBlock(pdf *fpdf.Fpdf, w float64, theme pdfTheme) {
	pdf.Ln(8)
	pdf.SetDrawColor(theme.RuleColor[0], theme.RuleColor[1], theme.RuleColor[2])
	pdf.Line(12, pdf.GetY(), 12+w, pdf.GetY())
	pdf.Ln(2)
	pdf.SetFont("Arial", "", 9)
	half := w / 2
	pdf.CellFormat(half, 5, "CLASS TEACHER SIGNATURE", "", 0, "L", false, 0, "")
	pdf.CellFormat(half, 5, "PRINCIPAL", "", 1, "R", false, 0, "")
}
