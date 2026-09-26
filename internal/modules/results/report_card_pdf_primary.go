package results

import (
	"bytes"

	"github.com/go-pdf/fpdf"
)

// generatePrimaryReportCardPDF renders the 1st-4th report card: same
// structure as generateKGReportCardPDF (the "primary" component scheme/
// grading scale instead), plus the source workbook's blank MORAL / G.K
// remark line -- handwritten fields on paper, not scored subjects.
func generatePrimaryReportCardPDF(rc ReportCard) ([]byte, error) {
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
	writeOverallTable(pdf, w, rc)

	_, _, _, _, moral, gk := reportCardDetailsFields(rc)
	pdf.SetFont("Arial", "B", 9)
	half := w / 2
	pdf.CellFormat(half, 6, "MORAL: "+moral, "1", 0, "L", false, 0, "")
	pdf.CellFormat(half, 6, "G.K: "+gk, "1", 1, "L", false, 0, "")
	pdf.Ln(2)

	writeReportCardFooterDetails(pdf, w, rc)
	writeReportCardSignatureBlock(pdf, w)

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
