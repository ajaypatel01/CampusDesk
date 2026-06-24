package fee

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/go-pdf/fpdf"
)

var ones = []string{
	"", "One", "Two", "Three", "Four", "Five", "Six", "Seven", "Eight", "Nine",
	"Ten", "Eleven", "Twelve", "Thirteen", "Fourteen", "Fifteen", "Sixteen",
	"Seventeen", "Eighteen", "Nineteen",
}

var tens = []string{
	"", "", "Twenty", "Thirty", "Forty", "Fifty", "Sixty", "Seventy", "Eighty", "Ninety",
}

func twoDigitWords(n int) string {
	if n < 20 {
		return ones[n]
	}
	s := tens[n/10]
	if n%10 > 0 {
		s += " " + ones[n%10]
	}
	return s
}

func amountToWords(n int) string {
	if n == 0 {
		return "Zero Rupees Only"
	}
	if n < 0 {
		return "Minus " + amountToWords(-n)
	}

	var parts []string

	crore := n / 10000000
	n %= 10000000
	if crore > 0 {
		parts = append(parts, twoDigitWords(crore)+" Crore")
	}

	lakh := n / 100000
	n %= 100000
	if lakh > 0 {
		parts = append(parts, twoDigitWords(lakh)+" Lakh")
	}

	thousand := n / 1000
	n %= 1000
	if thousand > 0 {
		parts = append(parts, twoDigitWords(thousand)+" Thousand")
	}

	hundred := n / 100
	n %= 100
	if hundred > 0 {
		parts = append(parts, ones[hundred]+" Hundred")
	}

	if n > 0 {
		parts = append(parts, twoDigitWords(n))
	}

	return strings.Join(parts, " ") + " Rupees Only"
}

func feeTypeLabel(ft string, installment *int) string {
	label := strings.Title(strings.ReplaceAll(ft, "_", " "))
	if installment != nil {
		qLabels := map[int]string{1: "Q1", 2: "Q2", 3: "Q3", 4: "Q4"}
		if q, ok := qLabels[*installment]; ok {
			label += " - " + q
		}
	}
	return label
}

func generateReceiptPDF(data ReceiptData) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(true, 20)
	pdf.SetMargins(15, 15, 15)
	pdf.AddPage()

	w := 180.0

	// School Header
	pdf.SetFont("Arial", "B", 18)
	pdf.CellFormat(w, 10, data.SchoolName, "", 1, "C", false, 0, "")

	pdf.SetFont("Arial", "", 10)
	if data.SchoolAddress != "" {
		pdf.CellFormat(w, 5, data.SchoolAddress, "", 1, "C", false, 0, "")
	}
	contact := ""
	if data.SchoolPhone != "" {
		contact += "Phone: " + data.SchoolPhone
	}
	if data.SchoolEmail != "" {
		if contact != "" {
			contact += "  |  "
		}
		contact += "Email: " + data.SchoolEmail
	}
	if contact != "" {
		pdf.CellFormat(w, 5, contact, "", 1, "C", false, 0, "")
	}

	pdf.Ln(3)
	pdf.SetDrawColor(0, 0, 0)
	pdf.Line(15, pdf.GetY(), 195, pdf.GetY())
	pdf.Ln(3)

	// FEE RECEIPT title
	pdf.SetFont("Arial", "B", 16)
	pdf.CellFormat(w, 10, "FEE RECEIPT", "B", 1, "C", false, 0, "")
	pdf.Ln(5)

	// Receipt No + Date
	pdf.SetFont("Arial", "", 10)
	receiptNo := data.PaymentID.String()[:8]
	pdf.CellFormat(w/2, 7, "Receipt No: "+receiptNo, "", 0, "L", false, 0, "")
	pdf.CellFormat(w/2, 7, "Date: "+data.PaymentDate.Format("02/01/2006"), "", 1, "R", false, 0, "")
	pdf.Ln(3)

	// Student Details Box
	boxY := pdf.GetY()
	pdf.SetFont("Arial", "B", 11)
	pdf.CellFormat(w, 7, "Student Details", "", 1, "L", false, 0, "")

	pdf.SetFont("Arial", "", 10)
	detailRow := func(label, value string) {
		pdf.CellFormat(45, 6, label, "", 0, "L", false, 0, "")
		pdf.SetFont("Arial", "B", 10)
		pdf.CellFormat(w-45, 6, value, "", 1, "L", false, 0, "")
		pdf.SetFont("Arial", "", 10)
	}
	detailRow("Student Name:", data.StudentName)
	if data.FatherName != "" {
		detailRow("Father's Name:", data.FatherName)
	}
	detailRow("Scholar No:", data.StudentCode)
	detailRow("Class:", data.GradeLevelName)
	detailRow("Academic Year:", data.AcademicYearName)

	boxEndY := pdf.GetY()
	pdf.Rect(15, boxY-1, w, boxEndY-boxY+3, "D")
	pdf.Ln(6)

	// Payment Details Table
	pdf.SetFont("Arial", "B", 11)
	pdf.CellFormat(w, 7, "Payment Details", "", 1, "L", false, 0, "")

	// Table header
	pdf.SetFont("Arial", "B", 10)
	pdf.SetFillColor(230, 230, 230)
	pdf.CellFormat(100, 7, "Particulars", "1", 0, "C", true, 0, "")
	pdf.CellFormat(80, 7, "Amount (Rs.)", "1", 1, "C", true, 0, "")

	// Table row
	pdf.SetFont("Arial", "", 10)
	label := feeTypeLabel(data.FeeType, data.InstallmentNum)
	pdf.CellFormat(100, 7, label, "1", 0, "L", false, 0, "")
	pdf.CellFormat(80, 7, fmt.Sprintf("%d/-", data.Amount), "1", 1, "R", false, 0, "")

	// Total row
	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(100, 7, "Total Paid (This Receipt)", "1", 0, "R", false, 0, "")
	pdf.CellFormat(80, 7, fmt.Sprintf("Rs. %d/-", data.Amount), "1", 1, "R", false, 0, "")
	pdf.Ln(3)

	// Payment Mode
	pdf.SetFont("Arial", "", 10)
	modeStr := "Payment Mode: " + strings.Title(data.PaymentMode)
	if data.ReferenceNumber != "" {
		modeStr += "  |  Ref: " + data.ReferenceNumber
	}
	pdf.CellFormat(w, 6, modeStr, "", 1, "L", false, 0, "")
	pdf.Ln(4)

	// Fee Summary Box
	pdf.SetFont("Arial", "B", 11)
	pdf.CellFormat(w, 7, "Fee Summary", "", 1, "L", false, 0, "")

	summaryRow := func(label string, amount int, bold bool) {
		if bold {
			pdf.SetFont("Arial", "B", 10)
		} else {
			pdf.SetFont("Arial", "", 10)
		}
		pdf.CellFormat(110, 6, label, "", 0, "L", false, 0, "")
		pdf.CellFormat(70, 6, fmt.Sprintf("Rs. %d/-", amount), "", 1, "R", false, 0, "")
	}

	summaryRow("Total Annual Fee", data.TuitionFee, false)
	if data.DiscountAmount > 0 {
		summaryRow("Discount", -data.DiscountAmount, false)
	}
	if data.VanFee > 0 {
		summaryRow("Van/Bus Fee", data.VanFee, false)
	}
	if data.PreviousYearDues > 0 {
		summaryRow("Previous Year Dues", data.PreviousYearDues, false)
	}

	pdf.Line(15, pdf.GetY(), 195, pdf.GetY())
	summaryRow("Total Due", data.TotalDue, true)
	summaryRow("Previously Paid", data.TotalPaidOther, false)
	summaryRow("Paid (This Receipt)", data.Amount, true)
	pdf.Line(15, pdf.GetY(), 195, pdf.GetY())
	summaryRow("Balance Remaining", data.BalanceAfter, true)
	pdf.Ln(5)

	// Amount in Words
	pdf.SetFont("Arial", "I", 10)
	pdf.CellFormat(w, 6, "Amount in Words: "+amountToWords(data.Amount), "", 1, "L", false, 0, "")
	pdf.Ln(8)

	// Footer
	pdf.Line(15, pdf.GetY(), 195, pdf.GetY())
	pdf.Ln(3)
	pdf.SetFont("Arial", "I", 8)
	pdf.CellFormat(w/2, 5, "This is a computer-generated receipt.", "", 0, "L", false, 0, "")
	pdf.Ln(10)
	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(w, 5, "________________________", "", 1, "R", false, 0, "")
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
