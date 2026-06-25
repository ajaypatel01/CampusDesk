package results

import (
	"bytes"
	"fmt"

	"github.com/go-pdf/fpdf"
)

func generateMarksheetPDF(ms StudentMarksheet) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(true, 20)
	pdf.SetMargins(15, 15, 15)
	pdf.AddPage()

	w := 180.0

	// Header
	pdf.SetFont("Arial", "B", 16)
	pdf.CellFormat(w, 10, ms.SchoolName, "", 1, "C", false, 0, "")
	pdf.SetFont("Arial", "B", 12)
	pdf.CellFormat(w, 7, "REPORT CARD / MARKSHEET", "B", 1, "C", false, 0, "")
	pdf.Ln(3)

	// Exam info
	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(w/2, 6, "Exam: "+ms.ExamName, "", 0, "L", false, 0, "")
	pdf.CellFormat(w/2, 6, "Academic Year: "+ms.AcademicYear, "", 1, "R", false, 0, "")

	// Student info box
	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(w, 7, "Student Details", "", 1, "L", false, 0, "")
	pdf.SetDrawColor(180, 180, 180)
	pdf.Line(15, pdf.GetY(), 195, pdf.GetY())
	pdf.Ln(2)

	row := func(label, value string) {
		pdf.SetFont("Arial", "", 10)
		pdf.CellFormat(45, 6, label, "", 0, "L", false, 0, "")
		pdf.SetFont("Arial", "B", 10)
		pdf.CellFormat(w-45, 6, value, "", 1, "L", false, 0, "")
	}
	row("Student Name:", ms.StudentName)
	row("Scholar No:", ms.StudentCode)
	row("Class:", ms.GradeLevelName)
	pdf.Ln(5)

	// Marks table
	pdf.SetFont("Arial", "B", 10)
	pdf.SetFillColor(200, 220, 240)
	colW := []float64{60, 20, 20, 28, 20, 20, 12}
	headers := []string{"Subject", "Max", "Pass", "Obtained", "%", "Grade", "Status"}
	for i, h := range headers {
		pdf.CellFormat(colW[i], 8, h, "1", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)

	for _, r := range ms.Rows {
		pdf.SetFont("Arial", "", 9)
		pdf.CellFormat(colW[0], 7, r.SubjectName, "1", 0, "L", false, 0, "")
		pdf.CellFormat(colW[1], 7, fmt.Sprintf("%d", r.MaxMarks), "1", 0, "C", false, 0, "")
		pdf.CellFormat(colW[2], 7, fmt.Sprintf("%d", r.PassingMarks), "1", 0, "C", false, 0, "")
		if r.IsAbsent {
			pdf.CellFormat(colW[3], 7, "Absent", "1", 0, "C", false, 0, "")
			pdf.CellFormat(colW[4], 7, "-", "1", 0, "C", false, 0, "")
			pdf.CellFormat(colW[5], 7, "AB", "1", 0, "C", false, 0, "")
			pdf.CellFormat(colW[6], 7, "AB", "1", 0, "C", false, 0, "")
		} else {
			pdf.CellFormat(colW[3], 7, fmt.Sprintf("%.1f", r.MarksObtained), "1", 0, "C", false, 0, "")
			pdf.CellFormat(colW[4], 7, fmt.Sprintf("%.1f", r.Percentage), "1", 0, "C", false, 0, "")
			pdf.CellFormat(colW[5], 7, r.Grade, "1", 0, "C", false, 0, "")
			pdf.SetFont("Arial", "B", 9)
			if r.Status == "Fail" {
				pdf.SetTextColor(200, 0, 0)
			}
			pdf.CellFormat(colW[6], 7, r.Status[:1], "1", 0, "C", false, 0, "")
			pdf.SetTextColor(0, 0, 0)
		}
		pdf.Ln(-1)
	}

	// Totals
	pdf.SetFont("Arial", "B", 10)
	pdf.SetFillColor(230, 240, 230)
	pdf.CellFormat(colW[0]+colW[1]+colW[2], 8, "Total / Result", "1", 0, "R", true, 0, "")
	pdf.CellFormat(colW[3], 8, fmt.Sprintf("%.1f / %d", ms.TotalObtained, ms.TotalMax), "1", 0, "C", true, 0, "")
	pdf.CellFormat(colW[4], 8, fmt.Sprintf("%.1f%%", ms.Percentage), "1", 0, "C", true, 0, "")
	pdf.CellFormat(colW[5], 8, ms.OverallGrade, "1", 0, "C", true, 0, "")
	if ms.Result == "Fail" {
		pdf.SetTextColor(200, 0, 0)
	}
	pdf.CellFormat(colW[6], 8, ms.Result[:1], "1", 0, "C", true, 0, "")
	pdf.SetTextColor(0, 0, 0)
	pdf.Ln(-1)
	pdf.Ln(5)

	// CGPA
	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(w/2, 6, fmt.Sprintf("CGPA: %.2f", ms.CGPA), "", 0, "L", false, 0, "")
	pdf.CellFormat(w/2, 6, "Overall Result: "+ms.Result, "", 1, "R", false, 0, "")
	pdf.Ln(12)

	// Grading scale
	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(w, 5, "Grading Scale:", "", 1, "L", false, 0, "")
	pdf.SetFont("Arial", "", 8)
	scale := "A1(91-100,10) | A2(81-90,9) | B1(71-80,8) | B2(61-70,7) | C1(51-60,6) | C2(41-50,5) | D(33-40,4) | F(<33,0)"
	pdf.CellFormat(w, 5, scale, "", 1, "L", false, 0, "")
	pdf.Ln(10)

	// Signatures
	pdf.SetDrawColor(0, 0, 0)
	pdf.Line(15, pdf.GetY(), 195, pdf.GetY())
	pdf.Ln(3)
	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(w/3, 5, "________________", "", 0, "C", false, 0, "")
	pdf.CellFormat(w/3, 5, "________________", "", 0, "C", false, 0, "")
	pdf.CellFormat(w/3, 5, "________________", "", 1, "C", false, 0, "")
	pdf.CellFormat(w/3, 5, "Class Teacher", "", 0, "C", false, 0, "")
	pdf.CellFormat(w/3, 5, "Parent / Guardian", "", 0, "C", false, 0, "")
	pdf.CellFormat(w/3, 5, "Principal", "", 1, "C", false, 0, "")

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
