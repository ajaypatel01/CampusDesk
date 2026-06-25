package documents

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/go-pdf/fpdf"
)

func generateBonafidePDF(d BonafideData) ([]byte, error) {
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

	// Certificate Title
	pdf.SetFont("Arial", "B", 15)
	pdf.CellFormat(w, 10, "BONAFIDE CERTIFICATE", "B", 1, "C", false, 0, "")
	pdf.Ln(6)

	// Issue date (top right)
	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(w, 6, "Date: "+d.IssueDate.Format("02/01/2006"), "", 1, "R", false, 0, "")
	pdf.Ln(4)

	// Certificate body
	fatherRef := ""
	if d.GuardianName != "" {
		rel := strings.Title(strings.ToLower(d.GuardianRelation))
		if rel == "" {
			rel = "Parent"
		}
		fatherRef = fmt.Sprintf(" %s/o %s,", rel, d.GuardianName)
	}

	classRef := ""
	if d.ClassName != "" {
		classRef = fmt.Sprintf(" studying in Class <b>%s</b>", d.ClassName)
	}
	yearRef := ""
	if d.AcademicYear != "" {
		yearRef = fmt.Sprintf(" for the Academic Year <b>%s</b>", d.AcademicYear)
	}

	pdf.SetFont("Arial", "", 11)
	bodyText := fmt.Sprintf(
		"This is to certify that <b>%s</b> (Scholar No: %s),%s a bona fide student of this school%s%s.",
		d.StudentName, d.StudentCode, fatherRef, classRef, yearRef,
	)
	writeMixedLine(pdf, w, bodyText)
	pdf.Ln(4)

	// Details table
	row := func(label, value string) {
		pdf.SetFont("Arial", "", 10)
		pdf.CellFormat(55, 7, label, "", 0, "L", false, 0, "")
		pdf.SetFont("Arial", "B", 10)
		pdf.CellFormat(w-55, 7, value, "", 1, "L", false, 0, "")
	}

	pdf.SetFont("Arial", "B", 11)
	pdf.CellFormat(w, 7, "Student Details", "", 1, "L", false, 0, "")
	pdf.SetDrawColor(180, 180, 180)
	pdf.Line(20, pdf.GetY(), 190, pdf.GetY())
	pdf.Ln(2)

	row("Name", d.StudentName)
	row("Scholar No", d.StudentCode)
	if d.DOB != nil {
		row("Date of Birth", d.DOB.Format("02/01/2006"))
	}
	if d.Gender != "" {
		row("Gender", strings.Title(strings.ToLower(d.Gender)))
	}
	if d.Category != "" {
		row("Category", d.Category)
	}
	if d.Caste != "" {
		row("Caste", d.Caste)
	}
	if d.AdmissionDate != nil {
		row("Date of Admission", d.AdmissionDate.Format("02/01/2006"))
	}
	if d.ClassName != "" {
		row("Class", d.ClassName)
	}
	if d.AcademicYear != "" {
		row("Academic Year", d.AcademicYear)
	}
	if d.GuardianName != "" {
		rel := strings.Title(strings.ToLower(d.GuardianRelation))
		if rel == "" {
			rel = "Parent"
		}
		row(rel+"'s Name", d.GuardianName)
	}

	pdf.Ln(10)

	// Purpose note
	pdf.SetFont("Arial", "I", 10)
	pdf.CellFormat(w, 6, "This certificate is issued on the request of the student for official purposes.", "", 1, "C", false, 0, "")
	pdf.Ln(16)

	// Signature
	pdf.SetDrawColor(0, 0, 0)
	pdf.Line(20, pdf.GetY(), 190, pdf.GetY())
	pdf.Ln(3)
	pdf.SetFont("Arial", "I", 8)
	pdf.CellFormat(w/2, 5, "This is a computer-generated certificate.", "", 0, "L", false, 0, "")
	pdf.Ln(14)
	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(w, 5, "________________________", "", 1, "R", false, 0, "")
	pdf.CellFormat(w, 5, "Principal / Authorized Signatory", "", 1, "R", false, 0, "")

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	if pdf.Error() != nil {
		return nil, pdf.Error()
	}
	return buf.Bytes(), nil
}

// writeMixedLine renders a line with basic <b>...</b> bold segments.
func writeMixedLine(pdf *fpdf.Fpdf, w float64, text string) {
	pdf.SetX(20)
	parts := strings.Split(text, "<b>")
	for i, part := range parts {
		if i == 0 {
			pdf.SetFont("Arial", "", 11)
			if part != "" {
				pdf.Write(7, part)
			}
			continue
		}
		idx := strings.Index(part, "</b>")
		if idx == -1 {
			pdf.SetFont("Arial", "", 11)
			pdf.Write(7, part)
			continue
		}
		pdf.SetFont("Arial", "B", 11)
		pdf.Write(7, part[:idx])
		pdf.SetFont("Arial", "", 11)
		rest := part[idx+4:]
		if rest != "" {
			pdf.Write(7, rest)
		}
	}
	pdf.Ln(7)
}

func buildContact(phone, email string) string {
	var parts []string
	if phone != "" {
		parts = append(parts, "Phone: "+phone)
	}
	if email != "" {
		parts = append(parts, "Email: "+email)
	}
	return strings.Join(parts, "  |  ")
}
