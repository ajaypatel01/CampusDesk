package documents

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"
)

func generateTCPDF(d TCData) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(true, 20)
	pdf.SetMargins(20, 20, 20)
	pdf.AddPage()

	w := 170.0

	// School Header
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

	// Title
	pdf.SetFont("Arial", "B", 15)
	pdf.CellFormat(w, 10, "TRANSFER CERTIFICATE", "B", 1, "C", false, 0, "")
	pdf.Ln(5)

	// TC No + Date
	tcNo := fmt.Sprintf("TC-%s", strings.ToUpper(d.StudentCode))
	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(w/2, 6, "TC No: "+tcNo, "", 0, "L", false, 0, "")
	pdf.CellFormat(w/2, 6, "Date of Issue: "+d.IssueDate.Format("02/01/2006"), "", 1, "R", false, 0, "")
	pdf.Ln(4)

	// Details table
	labelW := 75.0
	valueW := w - labelW

	tcRow := func(no, label, value string) {
		pdf.SetFont("Arial", "", 10)
		pdf.CellFormat(8, 7, no+".", "", 0, "R", false, 0, "")
		pdf.CellFormat(labelW-8, 7, label, "", 0, "L", false, 0, "")
		pdf.SetFont("Arial", "B", 10)
		pdf.CellFormat(valueW, 7, value, "", 1, "L", false, 0, "")
	}

	tcRow("1", "Name of Student", d.StudentName)
	tcRow("2", "Scholar No", d.StudentCode)

	guardianLabel := "Father's / Mother's Name"
	guardianValue := d.GuardianName
	if d.GuardianRelation != "" {
		guardianLabel = strings.Title(strings.ToLower(d.GuardianRelation)) + "'s Name"
	}
	tcRow("3", guardianLabel, guardianValue)

	dob := "—"
	if d.DOB != nil {
		dob = d.DOB.Format("02/01/2006")
	}
	tcRow("4", "Date of Birth", dob)
	tcRow("5", "Gender", strings.Title(strings.ToLower(d.Gender)))
	tcRow("6", "Category", d.Category)
	tcRow("7", "Caste", d.Caste)
	if d.AadharNumber != "" {
		tcRow("8", "Aadhar No", d.AadharNumber)
	}

	admDate := "—"
	if d.AdmissionDate != nil {
		admDate = d.AdmissionDate.Format("02/01/2006")
	}
	tcRow("9", "Date of First Admission", admDate)
	tcRow("10", "Class in which First Admitted", d.AdmittedClass)
	tcRow("11", "Last Class Attended", d.LastClass)
	tcRow("12", "Academic Year (Last)", d.LastAcademicYear)

	leavingDate := d.DateOfLeaving
	if leavingDate.IsZero() {
		leavingDate = time.Now()
	}
	tcRow("13", "Date of Leaving", leavingDate.Format("02/01/2006"))
	tcRow("14", "Reason for Leaving", d.ReasonForLeaving)
	tcRow("15", "Character & Conduct", d.Conduct)

	feeStatus := "Cleared"
	if !d.FeeCleared {
		feeStatus = fmt.Sprintf("Dues Pending (Rs. %d/-)", d.OutstandingFees)
	}
	tcRow("16", "School Fee Status", feeStatus)

	if d.PreviousSchool != "" {
		tcRow("17", "Previous School", d.PreviousSchool)
	}

	pdf.Ln(8)

	// Remark line
	pdf.SetFont("Arial", "I", 10)
	pdf.CellFormat(w, 6, "This certificate is issued on the request of the student/parent for transfer purposes.", "", 1, "C", false, 0, "")
	pdf.Ln(14)

	// Signature block
	pdf.SetDrawColor(0, 0, 0)
	pdf.Line(20, pdf.GetY(), 190, pdf.GetY())
	pdf.Ln(3)
	pdf.SetFont("Arial", "I", 8)
	pdf.CellFormat(w/2, 5, "This is a computer-generated certificate.", "", 0, "L", false, 0, "")
	pdf.Ln(14)
	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(w/2, 5, "________________________", "", 0, "L", false, 0, "")
	pdf.CellFormat(w/2, 5, "________________________", "", 1, "R", false, 0, "")
	pdf.CellFormat(w/2, 5, "Class Teacher", "", 0, "L", false, 0, "")
	pdf.CellFormat(w/2, 5, "Principal / Authorized Signatory", "", 1, "R", false, 0, "")

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	if pdf.Error() != nil {
		return nil, pdf.Error()
	}
	return buf.Bytes(), nil
}
