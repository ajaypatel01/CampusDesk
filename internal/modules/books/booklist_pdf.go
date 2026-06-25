package books

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/go-pdf/fpdf"
)

func generateBookListPDF(d BookListPDFData) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(true, 20)
	pdf.SetMargins(15, 15, 15)
	pdf.AddPage()

	w := 180.0

	// School header
	pdf.SetFont("Arial", "B", 16)
	pdf.CellFormat(w, 10, d.SchoolName, "", 1, "C", false, 0, "")
	pdf.Ln(2)
	pdf.SetDrawColor(0, 0, 0)
	pdf.Line(15, pdf.GetY(), 195, pdf.GetY())
	pdf.Ln(3)

	// Title
	pdf.SetFont("Arial", "B", 13)
	pdf.CellFormat(w, 8, "BOOK LIST", "B", 1, "C", false, 0, "")
	pdf.Ln(3)

	// Meta
	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(w/2, 6, "Class: "+d.GradeLevelName, "", 0, "L", false, 0, "")
	pdf.CellFormat(w/2, 6, "Academic Year: "+d.AcademicYearName, "", 1, "R", false, 0, "")
	if d.ListName != "Book List" && d.ListName != "" {
		pdf.CellFormat(w, 6, d.ListName, "", 1, "C", false, 0, "")
	}
	pdf.Ln(4)

	// Table header
	sNoW := 12.0
	titleW := 75.0
	authorW := 40.0
	pubW := 30.0
	priceW := w - sNoW - titleW - authorW - pubW

	pdf.SetFont("Arial", "B", 10)
	pdf.SetFillColor(220, 220, 220)
	pdf.CellFormat(sNoW, 8, "S.No", "1", 0, "C", true, 0, "")
	pdf.CellFormat(titleW, 8, "Book Title", "1", 0, "C", true, 0, "")
	pdf.CellFormat(authorW, 8, "Author", "1", 0, "C", true, 0, "")
	pdf.CellFormat(pubW, 8, "Publisher", "1", 0, "C", true, 0, "")
	pdf.CellFormat(priceW, 8, "Price (Rs.)", "1", 1, "C", true, 0, "")

	// Group by subject
	grouped := groupBySubject(d.Items)
	rowNum := 0

	for _, grp := range grouped {
		if grp.Subject != "" {
			pdf.SetFont("Arial", "B", 10)
			pdf.SetFillColor(240, 240, 240)
			pdf.CellFormat(w, 7, strings.ToUpper(grp.Subject), "1", 1, "L", true, 0, "")
		}

		for _, item := range grp.Items {
			rowNum++
			mandatory := ""
			if !item.IsMandatory {
				mandatory = "*"
			}
			pdf.SetFont("Arial", "", 9)
			pdf.CellFormat(sNoW, 7, fmt.Sprintf("%d", rowNum), "1", 0, "C", false, 0, "")
			pdf.CellFormat(titleW, 7, item.BookTitle+mandatory, "1", 0, "L", false, 0, "")
			pdf.CellFormat(authorW, 7, item.Author, "1", 0, "L", false, 0, "")
			pdf.CellFormat(pubW, 7, item.Publisher, "1", 0, "L", false, 0, "")
			priceStr := fmt.Sprintf("%d", item.UnitPrice)
			if item.Quantity > 1 {
				priceStr = fmt.Sprintf("%d x%d = %d", item.UnitPrice, item.Quantity, item.TotalPrice)
			}
			pdf.CellFormat(priceW, 7, priceStr, "1", 1, "R", false, 0, "")
		}
	}

	// Total
	pdf.SetFont("Arial", "B", 10)
	pdf.SetFillColor(230, 230, 230)
	pdf.CellFormat(w-priceW, 8, "Total Estimated Cost", "1", 0, "R", true, 0, "")
	pdf.CellFormat(priceW, 8, fmt.Sprintf("Rs. %d/-", d.TotalPrice), "1", 1, "R", true, 0, "")
	pdf.Ln(5)

	// Footer note
	pdf.SetFont("Arial", "I", 8)
	pdf.CellFormat(w, 5, "* Optional books   |   Prices are approximate and subject to change.", "", 1, "L", false, 0, "")

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	if pdf.Error() != nil {
		return nil, pdf.Error()
	}
	return buf.Bytes(), nil
}

type subjectGroup struct {
	Subject string
	Items   []BookListItemDetail
}

func groupBySubject(items []BookListItemDetail) []subjectGroup {
	seen := map[string]int{}
	var groups []subjectGroup
	for _, item := range items {
		subj := item.Subject
		if idx, ok := seen[subj]; ok {
			groups[idx].Items = append(groups[idx].Items, item)
		} else {
			seen[subj] = len(groups)
			groups = append(groups, subjectGroup{Subject: subj, Items: []BookListItemDetail{item}})
		}
	}
	return groups
}
