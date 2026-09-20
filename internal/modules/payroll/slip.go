package payroll

import (
	"bytes"
	"fmt"
	"time"

	"github.com/go-pdf/fpdf"
)

// SlipData is everything the salary-slip PDF needs, computed from real
// attendance/leave records rather than manually typed in.
type SlipData struct {
	SchoolName    string
	SchoolAddress string
	SchoolPhone   string
	SchoolEmail   string
	Logo          []byte // optional; school letterhead logo, nil if none on file
	Signature     []byte // optional; authorized signatory's signature image, nil if none on file

	Month time.Month
	Year  int

	Row MonthRow

	BankName          string
	BankAccountNumber string
	BankIFSC          string
}

// fmtDays renders a day count without a trailing ".0" for whole numbers,
// while still showing fractional days like "3.5" for half-day leave.
func fmtDays(v float64) string {
	if v == float64(int(v)) {
		return fmt.Sprintf("%d", int(v))
	}
	return fmt.Sprintf("%.1f", v)
}

func generateSalarySlipPDF(d SlipData) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(true, 20)
	pdf.SetMargins(15, 15, 15)
	pdf.AddPage()

	w := 180.0

	if len(d.Logo) > 0 {
		opt := fpdf.ImageOptions{ImageType: "JPEG", ReadDpi: true}
		pdf.RegisterImageOptionsReader("school-logo", opt, bytes.NewReader(d.Logo))
		pdf.ImageOptions("school-logo", 15, 10, 20, 0, false, opt, 0, "")
	}

	pdf.SetFont("Arial", "B", 18)
	pdf.CellFormat(w, 10, d.SchoolName, "", 1, "C", false, 0, "")

	pdf.SetFont("Arial", "", 10)
	if d.SchoolAddress != "" {
		pdf.CellFormat(w, 5, d.SchoolAddress, "", 1, "C", false, 0, "")
	}
	contact := ""
	if d.SchoolPhone != "" {
		contact += "Phone: " + d.SchoolPhone
	}
	if d.SchoolEmail != "" {
		if contact != "" {
			contact += "  |  "
		}
		contact += "Email: " + d.SchoolEmail
	}
	if contact != "" {
		pdf.CellFormat(w, 5, contact, "", 1, "C", false, 0, "")
	}

	pdf.Ln(3)
	pdf.Line(15, pdf.GetY(), 195, pdf.GetY())
	pdf.Ln(3)

	pdf.SetFont("Arial", "B", 16)
	pdf.CellFormat(w, 10, "SALARY SLIP", "B", 1, "C", false, 0, "")
	pdf.Ln(2)
	pdf.SetFont("Arial", "", 11)
	pdf.CellFormat(w, 7, fmt.Sprintf("For the month of %s %d", d.Month.String(), d.Year), "", 1, "C", false, 0, "")
	pdf.Ln(3)

	// Employee details box
	boxY := pdf.GetY()
	pdf.SetFont("Arial", "B", 11)
	pdf.CellFormat(w, 7, "Employee Details", "", 1, "L", false, 0, "")

	pdf.SetFont("Arial", "", 10)
	detailRow := func(label, value string) {
		pdf.CellFormat(50, 6, label, "", 0, "L", false, 0, "")
		pdf.SetFont("Arial", "B", 10)
		pdf.CellFormat(w-50, 6, value, "", 1, "L", false, 0, "")
		pdf.SetFont("Arial", "", 10)
	}
	detailRow("Employee Name:", d.Row.FirstName+" "+d.Row.LastName)
	if d.Row.Designation != "" {
		detailRow("Designation:", d.Row.Designation)
	}
	if d.BankName != "" {
		detailRow("Bank:", d.BankName)
	}
	if d.BankAccountNumber != "" {
		detailRow("Account Number:", d.BankAccountNumber)
	}
	if d.BankIFSC != "" {
		detailRow("IFSC Code:", d.BankIFSC)
	}

	boxEndY := pdf.GetY()
	pdf.Rect(15, boxY-1, w, boxEndY-boxY+3, "D")
	pdf.Ln(6)

	// Attendance table
	pdf.SetFont("Arial", "B", 11)
	pdf.CellFormat(w, 7, "Attendance", "", 1, "L", false, 0, "")

	presentDays := float64(d.Row.WorkingDays) - d.Row.DeductedDays
	attRow := func(label string, value string, shade bool) {
		if shade {
			pdf.SetFillColor(245, 245, 245)
		}
		pdf.SetFont("Arial", "", 10)
		pdf.CellFormat(120, 7, label, "1", 0, "L", shade, 0, "")
		pdf.CellFormat(60, 7, value, "1", 1, "R", shade, 0, "")
	}
	attRow("Working Days in Month", fmt.Sprintf("%d", d.Row.WorkingDays), true)
	attRow("Present Days", fmtDays(presentDays), false)
	attRow("CL Availed (Paid)", fmtDays(d.Row.CLPaidDays), true)
	attRow("CL Availed (Beyond Quota)", fmtDays(d.Row.CLExcessDays), false)
	attRow("Unpaid Leave Days", fmtDays(d.Row.UnpaidDays), true)
	attRow("Total Deducted Days", fmtDays(d.Row.DeductedDays), false)
	pdf.Ln(4)

	// Earnings / Deductions table
	pdf.SetFont("Arial", "B", 11)
	pdf.CellFormat(w, 7, "Salary Calculation", "", 1, "L", false, 0, "")

	pdf.SetFont("Arial", "B", 10)
	pdf.SetFillColor(230, 230, 230)
	pdf.CellFormat(120, 7, "Particulars", "1", 0, "C", true, 0, "")
	pdf.CellFormat(60, 7, "Amount (Rs.)", "1", 1, "C", true, 0, "")

	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(120, 7, "Monthly Salary (Gross)", "1", 0, "L", false, 0, "")
	pdf.CellFormat(60, 7, fmt.Sprintf("%d/-", d.Row.MonthlySalary), "1", 1, "R", false, 0, "")

	pdf.CellFormat(120, 7, fmt.Sprintf("Per Day Rate (Salary / %d days)", d.Row.WorkingDays), "1", 0, "L", false, 0, "")
	pdf.CellFormat(60, 7, fmt.Sprintf("%.2f", d.Row.PerDayRate), "1", 1, "R", false, 0, "")

	pdf.CellFormat(120, 7, fmt.Sprintf("Deduction (%s day(s) x Rs. %.2f)", fmtDays(d.Row.DeductedDays), d.Row.PerDayRate), "1", 0, "L", false, 0, "")
	pdf.CellFormat(60, 7, fmt.Sprintf("- %d/-", d.Row.Deduction), "1", 1, "R", false, 0, "")

	pdf.SetFont("Arial", "B", 10)
	pdf.SetFillColor(230, 230, 230)
	pdf.CellFormat(120, 8, "Net Salary Payable", "1", 0, "L", true, 0, "")
	pdf.CellFormat(60, 8, fmt.Sprintf("Rs. %d/-", d.Row.NetSalary), "1", 1, "R", true, 0, "")
	pdf.Ln(6)

	// CL balance
	pdf.SetFont("Arial", "B", 11)
	pdf.CellFormat(w, 7, "Casual Leave Balance (Year to Date)", "", 1, "L", false, 0, "")
	pdf.SetFont("Arial", "", 10)
	clRow := func(label string, value float64, shade bool) {
		if shade {
			pdf.SetFillColor(245, 245, 245)
		}
		pdf.CellFormat(120, 7, label, "1", 0, "L", shade, 0, "")
		pdf.CellFormat(60, 7, fmtDays(value), "1", 1, "R", shade, 0, "")
	}
	clRow("Annual CL Quota", float64(d.Row.CLQuota), true)
	clRow("CL Used So Far This Year", d.Row.CLUsedYTD, false)
	clRow("CL Balance Remaining", d.Row.CLBalance, true)
	pdf.Ln(8)

	pdf.Line(15, pdf.GetY(), 195, pdf.GetY())
	pdf.Ln(3)
	noteY := pdf.GetY()
	pdf.SetFont("Arial", "I", 8)
	pdf.CellFormat(w/2, 5, "This is a computer-generated salary slip.", "", 0, "L", false, 0, "")

	if len(d.Signature) > 0 {
		opt := fpdf.ImageOptions{ImageType: "PNG", ReadDpi: true}
		pdf.RegisterImageOptionsReader("auth-signature", opt, bytes.NewReader(d.Signature))
		sigW := 32.0
		sigH := sigW * (198.0 / 818.0)
		pdf.ImageOptions("auth-signature", 15+w-sigW, noteY, sigW, sigH, false, opt, 0, "")
		pdf.SetY(noteY + sigH + 1)
	} else {
		pdf.Ln(10)
		pdf.SetFont("Arial", "", 10)
		pdf.CellFormat(w, 5, "________________________", "", 1, "R", false, 0, "")
	}
	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(w, 5, "Authorized Signatory", "", 1, "R", false, 0, "")

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("pdf output: %w", err)
	}
	if pdf.Error() != nil {
		return nil, pdf.Error()
	}
	return buf.Bytes(), nil
}
