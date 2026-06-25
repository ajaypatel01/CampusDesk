package idcard

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/go-pdf/fpdf"
)

const (
	cardW = 86.0 // mm — standard credit-card width
	cardH = 54.0 // mm
	pad   = 3.0
)

// generateStudentCards produces an A4 PDF with up to 6 cards (3 rows × 2 cols).
func generateStudentCards(cards []StudentCardData, photoData map[string][]byte) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(false, 0)
	pdf.SetMargins(10, 10, 10)

	cols := 2
	xOffsets := []float64{10.0, 10.0 + cardW + 8.0}
	yOffsets := []float64{10.0, 10.0 + cardH + 6.0, 10.0 + (cardH+6.0)*2}

	pdf.AddPage()
	for i, data := range cards {
		col := i % cols
		row := (i / cols) % 3
		if i > 0 && i%6 == 0 {
			pdf.AddPage()
		}
		x := xOffsets[col]
		y := yOffsets[row%3]
		drawStudentCard(pdf, data, photoData[data.PhotoKey], x, y)
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func drawStudentCard(pdf *fpdf.Fpdf, d StudentCardData, photo []byte, x, y float64) {
	// Card border
	pdf.SetDrawColor(30, 90, 160)
	pdf.SetLineWidth(0.5)
	pdf.Rect(x, y, cardW, cardH, "D")

	// Header bar
	pdf.SetFillColor(30, 90, 160)
	pdf.Rect(x, y, cardW, 9, "F")
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Arial", "B", 7)
	pdf.SetXY(x+1, y+1)
	pdf.CellFormat(cardW-2, 7, d.SchoolName, "", 0, "C", false, 0, "")

	pdf.SetTextColor(0, 0, 0)

	// Photo area (left side)
	photoX := x + pad
	photoY := y + 10
	photoW := 20.0
	photoH := 24.0
	pdf.SetFillColor(220, 220, 220)
	pdf.Rect(photoX, photoY, photoW, photoH, "FD")

	if len(photo) > 0 {
		imgType := "JPG"
		if len(photo) > 4 && photo[0] == 0x89 {
			imgType = "PNG"
		}
		imgName := fmt.Sprintf("student_%s", d.StudentCode)
		pdf.RegisterImageOptionsReader(imgName, fpdf.ImageOptions{ImageType: imgType, AllowNegativePosition: false}, bytes.NewReader(photo))
		pdf.ImageOptions(imgName, photoX, photoY, photoW, photoH, false, fpdf.ImageOptions{ImageType: imgType}, 0, "")
	} else {
		pdf.SetFont("Arial", "", 5)
		pdf.SetXY(photoX, photoY+photoH/2-2)
		pdf.CellFormat(photoW, 4, "PHOTO", "", 0, "C", false, 0, "")
	}

	// Student details (right of photo)
	detX := x + pad + photoW + 2
	detW := cardW - pad - photoW - 3
	detY := y + 10.5

	cardLine := func(label, value string) {
		pdf.SetFont("Arial", "", 5)
		pdf.SetXY(detX, detY)
		pdf.CellFormat(detW*0.38, 4, label+":", "", 0, "L", false, 0, "")
		pdf.SetFont("Arial", "B", 5)
		pdf.CellFormat(detW*0.62, 4, truncate(value, 20), "", 1, "L", false, 0, "")
		detY += 4
	}

	cardLine("Name", d.StudentName)
	cardLine("Scholar No", d.StudentCode)
	if d.ClassName != "" {
		cardLine("Class", d.ClassName)
	}
	if d.AcYear != "" {
		cardLine("Year", d.AcYear)
	}
	if d.BloodGroup != "" {
		cardLine("Blood", d.BloodGroup)
	}
	if d.DOB != nil {
		cardLine("DOB", d.DOB.Format("02/01/2006"))
	}
	if d.Phone != "" {
		cardLine("Phone", d.Phone)
	}

	// Footer
	footerY := y + cardH - 7
	pdf.SetFillColor(245, 245, 245)
	pdf.Rect(x, footerY, cardW, 7, "FD")
	pdf.SetFont("Arial", "I", 5)
	pdf.SetXY(x+pad, footerY+1)
	pdf.CellFormat(cardW/2-pad, 5, d.SchoolPhone, "", 0, "L", false, 0, "")
	pdf.SetFont("Arial", "", 5)
	pdf.SetXY(x+cardW/2, footerY+2)
	pdf.CellFormat(cardW/2-pad, 4, "_______________", "", 0, "R", false, 0, "")
	pdf.SetXY(x+cardW/2, footerY+4.5)
	pdf.CellFormat(cardW/2-pad, 3, "Principal", "", 0, "R", false, 0, "")
}

func generateTeacherCards(cards []TeacherCardData, photoData map[string][]byte) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(false, 0)
	pdf.SetMargins(10, 10, 10)

	cols := 2
	xOffsets := []float64{10.0, 10.0 + cardW + 8.0}
	yOffsets := []float64{10.0, 10.0 + cardH + 6.0, 10.0 + (cardH+6.0)*2}

	pdf.AddPage()
	for i, data := range cards {
		col := i % cols
		row := (i / cols) % 3
		if i > 0 && i%6 == 0 {
			pdf.AddPage()
		}
		x := xOffsets[col]
		y := yOffsets[row%3]
		drawTeacherCard(pdf, data, photoData[data.PhotoKey], x, y)
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func drawTeacherCard(pdf *fpdf.Fpdf, d TeacherCardData, photo []byte, x, y float64) {
	pdf.SetDrawColor(20, 120, 80)
	pdf.SetLineWidth(0.5)
	pdf.Rect(x, y, cardW, cardH, "D")

	// Header
	pdf.SetFillColor(20, 120, 80)
	pdf.Rect(x, y, cardW, 9, "F")
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Arial", "B", 7)
	pdf.SetXY(x+1, y+1)
	pdf.CellFormat(cardW-2, 7, d.SchoolName, "", 0, "C", false, 0, "")
	pdf.SetTextColor(0, 0, 0)

	// Staff badge label
	pdf.SetFillColor(240, 255, 240)
	pdf.Rect(x, y+9, cardW, 5, "F")
	pdf.SetFont("Arial", "B", 6)
	pdf.SetXY(x, y+10)
	pdf.CellFormat(cardW, 4, "STAFF IDENTITY CARD", "", 0, "C", false, 0, "")

	// Photo
	photoX := x + pad
	photoY := y + 15
	photoW := 20.0
	photoH := 24.0
	pdf.SetFillColor(220, 220, 220)
	pdf.Rect(photoX, photoY, photoW, photoH, "FD")

	if len(photo) > 0 {
		imgType := "JPG"
		if len(photo) > 4 && photo[0] == 0x89 {
			imgType = "PNG"
		}
		imgName := fmt.Sprintf("teacher_%s", d.EmployeeID)
		if imgName == "teacher_" {
			imgName = fmt.Sprintf("teacher_%s", d.UserID.String()[:8])
		}
		pdf.RegisterImageOptionsReader(imgName, fpdf.ImageOptions{ImageType: imgType}, bytes.NewReader(photo))
		pdf.ImageOptions(imgName, photoX, photoY, photoW, photoH, false, fpdf.ImageOptions{ImageType: imgType}, 0, "")
	} else {
		pdf.SetFont("Arial", "", 5)
		pdf.SetXY(photoX, photoY+photoH/2-2)
		pdf.CellFormat(photoW, 4, "PHOTO", "", 0, "C", false, 0, "")
	}

	detX := x + pad + photoW + 2
	detW := cardW - pad - photoW - 3
	detY := y + 15.5

	cardLine := func(label, value string) {
		pdf.SetFont("Arial", "", 5)
		pdf.SetXY(detX, detY)
		pdf.CellFormat(detW*0.38, 4, label+":", "", 0, "L", false, 0, "")
		pdf.SetFont("Arial", "B", 5)
		pdf.CellFormat(detW*0.62, 4, truncate(value, 20), "", 1, "L", false, 0, "")
		detY += 4
	}

	cardLine("Name", d.TeacherName)
	if d.EmployeeID != "" {
		cardLine("Emp ID", d.EmployeeID)
	}
	cardLine("Role", strings.Title(strings.ReplaceAll(strings.ToLower(d.Designation), "_", " ")))
	if d.Department != "" {
		cardLine("Dept", d.Department)
	}
	if d.Phone != "" {
		cardLine("Phone", d.Phone)
	}

	footerY := y + cardH - 7
	pdf.SetFillColor(245, 255, 245)
	pdf.Rect(x, footerY, cardW, 7, "FD")
	pdf.SetFont("Arial", "I", 5)
	pdf.SetXY(x+pad, footerY+1)
	pdf.CellFormat(cardW/2-pad, 5, d.SchoolPhone, "", 0, "L", false, 0, "")
	pdf.SetFont("Arial", "", 5)
	pdf.SetXY(x+cardW/2, footerY+2)
	pdf.CellFormat(cardW/2-pad, 4, "_______________", "", 0, "R", false, 0, "")
	pdf.SetXY(x+cardW/2, footerY+4.5)
	pdf.CellFormat(cardW/2-pad, 3, "Principal", "", 0, "R", false, 0, "")
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}
