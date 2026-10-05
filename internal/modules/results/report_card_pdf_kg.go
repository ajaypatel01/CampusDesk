package results

import (
	"bytes"

	"github.com/go-pdf/fpdf"
)

// generateKGReportCardPDF renders the Nursery/LKG/UKG report card: same
// structure as generatePrimaryReportCardPDF, but with the "kg" component
// scheme/grading scale and the source workbook's "* SUBJECT - NOT ADDED IN
// TOTAL" footnote (no MORAL/G.K remark row -- that's primary-only). theme
// only changes colors -- see report_card_theme.go.
func generateKGReportCardPDF(rc ReportCard, theme pdfTheme) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(true, 15)
	pdf.SetMargins(12, 12, 12)
	pdf.AddPage()
	w := 186.0

	writeReportCardHeader(pdf, w, rc, theme)

	for i := range rc.Exams {
		writeExamComponentTable(pdf, w, rc, i, theme)
	}
	writeOverallTable(pdf, w, rc, theme)
	writeReportCardFooterDetails(pdf, w, rc)

	pdf.SetFont("Arial", "I", 7)
	pdf.CellFormat(w, 4, "* SUBJECT - NOT ADDED IN TOTAL (if any subject on this report is marked co-scholastic)", "", 1, "L", false, 0, "")

	writeReportCardSignatureBlock(pdf, w, theme)

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
