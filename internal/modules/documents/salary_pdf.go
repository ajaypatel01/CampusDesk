package documents

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/go-pdf/fpdf"
)

func generateSalarySlipPDF(d SalarySlipData) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(true, 20)
	pdf.SetMargins(20, 20, 20)
	pdf.AddPage()

	w := 170.0

	// School Header
	if d.SchoolName != "" {
		pdf.SetFont("Arial", "B", 18)
		pdf.CellFormat(w, 10, d.SchoolName, "", 1, "C", false, 0, "")
		pdf.SetFont("Arial", "", 10)
		if d.SchoolAddress != "" {
			pdf.CellFormat(w, 5, d.SchoolAddress, "", 1, "C", false, 0, "")
		}
		contact := buildContact(d.SchoolPhone, d.SchoolEmail)
		if contact != "" {
			pdf.CellFormat(w, 5, contact, "", 1, "C", false, 0, "")
		}
		pdf.Ln(3)
		pdf.SetDrawColor(0, 0, 0)
		pdf.Line(20, pdf.GetY(), 190, pdf.GetY())
		pdf.Ln(4)
	}

	// Title
	pdf.SetFont("Arial", "B", 15)
	pdf.CellFormat(w, 10, "SALARY SLIP", "B", 1, "C", false, 0, "")
	pdf.Ln(2)

	// Month / Year banner
	pdf.SetFont("Arial", "B", 11)
	pdf.SetFillColor(230, 230, 230)
	pdf.CellFormat(w, 8, fmt.Sprintf("Month: %s %d", d.Month, d.Year), "1", 1, "C", true, 0, "")
	pdf.Ln(4)

	// Employee Details
	detailRow := func(label, value string) {
		pdf.SetFont("Arial", "", 10)
		pdf.CellFormat(50, 7, label, "", 0, "L", false, 0, "")
		pdf.SetFont("Arial", "B", 10)
		pdf.CellFormat(w-50, 7, value, "", 1, "L", false, 0, "")
	}

	pdf.SetFont("Arial", "B", 11)
	pdf.CellFormat(w, 7, "Employee Details", "", 1, "L", false, 0, "")
	pdf.SetDrawColor(180, 180, 180)
	pdf.Line(20, pdf.GetY(), 190, pdf.GetY())
	pdf.Ln(2)

	detailRow("Employee Name:", d.EmployeeName)
	if d.EmployeeID != "" {
		detailRow("Employee ID:", d.EmployeeID)
	}
	if d.Designation != "" {
		detailRow("Designation:", d.Designation)
	}
	if d.Department != "" {
		detailRow("Department:", d.Department)
	}
	detailRow("Issue Date:", d.IssueDate.Format("02/01/2006"))
	pdf.Ln(6)

	// Earnings & Deductions side by side
	col := w / 2
	tableY := pdf.GetY()

	// ---- Earnings ----
	pdf.SetFont("Arial", "B", 11)
	pdf.SetFillColor(220, 240, 220)
	pdf.CellFormat(col-5, 8, "Earnings", "1", 0, "C", true, 0, "")
	pdf.SetFont("Arial", "B", 11)
	pdf.SetFillColor(240, 220, 220)
	pdf.CellFormat(col-5, 8, "Deductions", "1", 1, "C", true, 0, "")

	salaryRow := func(label string, amt int, colIdx int) {
		pdf.SetFont("Arial", "", 10)
		cellW := col - 5
		if colIdx == 0 {
			pdf.CellFormat(cellW*0.6, 7, label, "LR", 0, "L", false, 0, "")
			pdf.CellFormat(cellW*0.4, 7, rupeeStr(amt), "R", 0, "R", false, 0, "")
		} else {
			pdf.CellFormat(cellW*0.6, 7, label, "LR", 0, "L", false, 0, "")
			pdf.CellFormat(cellW*0.4, 7, rupeeStr(amt), "R", 1, "R", false, 0, "")
		}
	}

	type earningRow struct{ label string; amt int }
	earnings := []earningRow{
		{"Basic Salary", d.BasicSalary},
		{"House Rent Allowance (HRA)", d.HRA},
		{"Dearness Allowance (DA)", d.DA},
		{"Transport Allowance (TA)", d.TA},
		{"Medical Allowance", d.MedicalAllowance},
		{"Other Allowance", d.OtherAllowance},
	}
	type deductRow struct{ label string; amt int }
	deductions := []deductRow{
		{"Provident Fund (PF)", d.PF},
		{"Tax Deducted (TDS)", d.TDS},
		{"ESI", d.ESI},
		{"Other Deduction", d.OtherDeduction},
	}

	// Pad deductions to match earnings length
	maxRows := len(earnings)
	if len(deductions) > maxRows {
		maxRows = len(deductions)
	}

	for i := 0; i < maxRows; i++ {
		pdf.SetX(20)
		if i < len(earnings) {
			salaryRow(earnings[i].label, earnings[i].amt, 0)
		} else {
			pdf.CellFormat((col-5)*0.6, 7, "", "LR", 0, "L", false, 0, "")
			pdf.CellFormat((col-5)*0.4, 7, "", "R", 0, "R", false, 0, "")
		}
		if i < len(deductions) {
			salaryRow(deductions[i].label, deductions[i].amt, 1)
		} else {
			pdf.CellFormat((col-5)*0.6, 7, "", "LR", 0, "L", false, 0, "")
			pdf.CellFormat((col-5)*0.4, 7, "", "R", 1, "R", false, 0, "")
		}
	}

	_ = tableY // alignment reference stored for future use

	// Totals row
	pdf.SetFont("Arial", "B", 10)
	pdf.SetFillColor(240, 240, 240)
	totalColW := col - 5
	pdf.CellFormat(totalColW*0.6, 7, "Gross Salary", "1", 0, "L", true, 0, "")
	pdf.CellFormat(totalColW*0.4, 7, rupeeStr(d.GrossSalary), "1", 0, "R", true, 0, "")
	pdf.CellFormat(totalColW*0.6, 7, "Total Deductions", "1", 0, "L", true, 0, "")
	pdf.CellFormat(totalColW*0.4, 7, rupeeStr(d.TotalDeduction), "1", 1, "R", true, 0, "")
	pdf.Ln(4)

	// Net Salary
	pdf.SetFont("Arial", "B", 12)
	pdf.SetFillColor(200, 230, 200)
	pdf.CellFormat(w*0.6, 9, "NET SALARY (Take Home)", "1", 0, "L", true, 0, "")
	pdf.CellFormat(w*0.4, 9, fmt.Sprintf("Rs. %d/-", d.NetSalary), "1", 1, "R", true, 0, "")
	pdf.Ln(3)

	// Amount in words
	pdf.SetFont("Arial", "I", 10)
	pdf.CellFormat(w, 6, "Amount in Words: "+salaryAmountToWords(d.NetSalary), "", 1, "L", false, 0, "")
	pdf.Ln(14)

	// Footer
	pdf.SetDrawColor(0, 0, 0)
	pdf.Line(20, pdf.GetY(), 190, pdf.GetY())
	pdf.Ln(3)
	pdf.SetFont("Arial", "I", 8)
	pdf.CellFormat(w/2, 5, "This is a computer-generated salary slip.", "", 0, "L", false, 0, "")
	pdf.Ln(14)
	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(w/2, 5, "________________________", "", 0, "L", false, 0, "")
	pdf.CellFormat(w/2, 5, "________________________", "", 1, "R", false, 0, "")
	pdf.CellFormat(w/2, 5, "Employee Signature", "", 0, "L", false, 0, "")
	pdf.CellFormat(w/2, 5, "Authorized Signatory", "", 1, "R", false, 0, "")

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	if pdf.Error() != nil {
		return nil, pdf.Error()
	}
	return buf.Bytes(), nil
}

func rupeeStr(n int) string {
	return fmt.Sprintf("Rs. %d/-", n)
}

var salaryOnes = []string{
	"", "One", "Two", "Three", "Four", "Five", "Six", "Seven", "Eight", "Nine",
	"Ten", "Eleven", "Twelve", "Thirteen", "Fourteen", "Fifteen", "Sixteen",
	"Seventeen", "Eighteen", "Nineteen",
}

var salaryTens = []string{
	"", "", "Twenty", "Thirty", "Forty", "Fifty", "Sixty", "Seventy", "Eighty", "Ninety",
}

func salaryTwoDigit(n int) string {
	if n < 20 {
		return salaryOnes[n]
	}
	s := salaryTens[n/10]
	if n%10 > 0 {
		s += " " + salaryOnes[n%10]
	}
	return s
}

func salaryAmountToWords(n int) string {
	if n == 0 {
		return "Zero Rupees Only"
	}
	var parts []string
	if lakh := n / 100000; lakh > 0 {
		parts = append(parts, salaryTwoDigit(lakh)+" Lakh")
		n %= 100000
	}
	if thousand := n / 1000; thousand > 0 {
		parts = append(parts, salaryTwoDigit(thousand)+" Thousand")
		n %= 1000
	}
	if hundred := n / 100; hundred > 0 {
		parts = append(parts, salaryOnes[hundred]+" Hundred")
		n %= 100
	}
	if n > 0 {
		parts = append(parts, salaryTwoDigit(n))
	}
	return strings.Join(parts, " ") + " Rupees Only"
}
