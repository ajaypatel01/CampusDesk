package results

import (
	"bytes"

	"github.com/go-pdf/fpdf"
)

// generateMiddleReportCardPDF renders the 6th-7th report card: Test/Project/
// Theory components (via the shared writeExamComponentTable, same as kg/
// primary), plus the Co-Scholastic/Discipline grade grid. Deliberately no
// combined "OVER ALL" block -- the source workbook has none for this
// template, only a percentage per exam.
func generateMiddleReportCardPDF(rc ReportCard) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(true, 15)
	pdf.SetMargins(12, 12, 12)
	pdf.AddPage()
	w := 186.0

	writeReportCardHeader(pdf, w, rc)

	components := MarkComponentsForTemplate(&rc.Template)
	for i := range rc.Exams {
		writeExamComponentTable(pdf, w, rc, i, components)
	}
	writeDisciplineTable(pdf, w, rc)
	writeReportCardFooterDetails(pdf, w, rc)
	writeReportCardSignatureBlock(pdf, w)

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func writeDisciplineTable(pdf *fpdf.Fpdf, w float64, rc ReportCard) {
	pdf.SetFont("Arial", "B", 10)
	pdf.SetFillColor(220, 230, 245)
	pdf.CellFormat(w, 7, "CO-SCHOLASTIC / DISCIPLINE", "1", 1, "C", true, 0, "")

	gradeByKey := make(map[string]string, len(rc.DisciplineGrades))
	for _, g := range rc.DisciplineGrades {
		gradeByKey[g.CriterionKey] = g.Grade
	}
	labelW := w / 2 * 0.75
	gradeW := w / 2 * 0.25
	pdf.SetFont("Arial", "", 8)
	for i := 0; i < len(disciplineCriteria); i += 2 {
		pdf.CellFormat(labelW, 6, disciplineCriteria[i], "1", 0, "L", false, 0, "")
		pdf.CellFormat(gradeW, 6, gradeByKey[disciplineCriteria[i]], "1", 0, "C", false, 0, "")
		if i+1 < len(disciplineCriteria) {
			pdf.CellFormat(labelW, 6, disciplineCriteria[i+1], "1", 0, "L", false, 0, "")
			pdf.CellFormat(gradeW, 6, gradeByKey[disciplineCriteria[i+1]], "1", 1, "C", false, 0, "")
		} else {
			pdf.CellFormat(labelW+gradeW, 6, "", "0", 1, "L", false, 0, "")
		}
	}
	pdf.Ln(3)
}
