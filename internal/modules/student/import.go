package student

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

// This file is the downloadable/fillable Excel template + bulk-upload path
// for a new school entering its student roster, instead of one-by-one
// through the "Add Student" form. It targets exactly the same fields and
// validation as Service.Create (CreateInput) -- a bulk-imported row is
// subject to the same rules a manually-added one is, nothing looser.

const importSheetName = "Students"

// importColumns is both the template's header row and the parser's column
// order -- keeping them in one place means the two can never drift apart.
var importColumns = []string{
	"Student Code *", "First Name *", "Last Name", "Gender (male/female)", "Date of Birth (YYYY-MM-DD)",
	"Email", "Phone", "Address", "Admission Date (YYYY-MM-DD)", "Caste", "Category",
	"Aadhar Number", "Samagra ID", "PEN Number", "APAAR ID", "Enrollment Number",
	"Admission Class", "Admission Year", "Previous School",
	"Bank Name", "Bank IFSC", "Bank Account Number", "Bank Holder Name", "Bank Branch",
	"Status (active/inactive/graduated/transferred)",
}

const importTemplateDataRows = 500 // how many blank rows get the dropdown validation applied

// GenerateImportTemplate builds the downloadable .xlsx: a "Students" sheet
// with the header row, one example row, and dropdown validation on Gender/
// Status; and an "Instructions" sheet explaining the format.
func GenerateImportTemplate() ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	f.SetSheetName("Sheet1", importSheetName)

	headerStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"4F46E5"}, Pattern: 1},
	})
	if err != nil {
		return nil, err
	}
	for i, col := range importColumns {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		if err := f.SetCellValue(importSheetName, cell, col); err != nil {
			return nil, err
		}
		if err := f.SetCellStyle(importSheetName, cell, cell, headerStyle); err != nil {
			return nil, err
		}
	}
	if err := f.SetColWidth(importSheetName, "A", string(rune('A'+len(importColumns)-1)), 20); err != nil {
		return nil, err
	}

	// One example row so a school sees the expected shape before typing
	// over it -- not real data, deleted before or along with their upload.
	example := []interface{}{
		"S-1001", "Aisha", "Khan", "female", "2015-06-12",
		"aisha.parent@example.com", "9876543210", "123 MG Road, City", "2025-06-01", "General", "OBC",
		"", "", "", "", "",
		"", "", "",
		"", "", "", "", "",
		"active",
	}
	for i, v := range example {
		cell, _ := excelize.CoordinatesToCellName(i+1, 2)
		if err := f.SetCellValue(importSheetName, cell, v); err != nil {
			return nil, err
		}
	}

	genderCol, _ := excelize.ColumnNumberToName(4)
	statusCol, _ := excelize.ColumnNumberToName(len(importColumns))
	genderDV := excelize.NewDataValidation(true)
	genderDV.SetSqref(fmt.Sprintf("%s2:%s%d", genderCol, genderCol, importTemplateDataRows))
	if err := genderDV.SetDropList([]string{"male", "female"}); err != nil {
		return nil, err
	}
	if err := f.AddDataValidation(importSheetName, genderDV); err != nil {
		return nil, err
	}
	statusDV := excelize.NewDataValidation(true)
	statusDV.SetSqref(fmt.Sprintf("%s2:%s%d", statusCol, statusCol, importTemplateDataRows))
	if err := statusDV.SetDropList([]string{"active", "inactive", "graduated", "transferred"}); err != nil {
		return nil, err
	}
	if err := f.AddDataValidation(importSheetName, statusDV); err != nil {
		return nil, err
	}

	f.NewSheet("Instructions")
	instructions := []string{
		"How to use this template",
		"",
		"1. Fill in one row per student on the \"Students\" sheet. Delete the example row (row 2) first.",
		"2. Student Code and First Name are required for every row; everything else is optional.",
		"3. Student Code must be unique within your school -- it's how each student is identified going forward.",
		"4. Dates must be in YYYY-MM-DD format, e.g. 2015-06-12.",
		"5. Gender must be exactly \"male\" or \"female\" (use the dropdown).",
		"6. Status defaults to \"active\" if left blank.",
		"7. Save the file and upload it from the Students page. You'll see a row-by-row result --",
		"   rows that fail (e.g. a duplicate Student Code) are reported individually; everything else still imports.",
	}
	for i, line := range instructions {
		cell, _ := excelize.CoordinatesToCellName(1, i+1)
		if err := f.SetCellValue("Instructions", cell, line); err != nil {
			return nil, err
		}
	}
	if err := f.SetColWidth("Instructions", "A", "A", 100); err != nil {
		return nil, err
	}
	f.SetActiveSheet(f.GetSheetIndex(importSheetName))

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ImportRow is one parsed (but not yet saved) row from an uploaded file,
// alongside its original row number for error reporting.
type ImportRow struct {
	RowNumber int
	Input     CreateInput
	ParseErr  string // set when the row itself is malformed (bad date, bad gender, ...)
}

// ParseImportFile reads the uploaded .xlsx's "Students" sheet (falling back
// to the first sheet if that name isn't found, in case someone renames it)
// into rows, in file order. A row that's entirely blank is silently
// skipped -- not every row after the last real one needs to be deleted.
// SchoolID is deliberately not part of CreateInput here -- the caller sets
// it on every row before saving, same as the single "Add Student" form does.
func ParseImportFile(data []byte) ([]ImportRow, error) {
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("could not read this file as an Excel workbook: %w", err)
	}
	defer f.Close()

	sheet := importSheetName
	found := false
	for _, name := range f.GetSheetList() {
		if name == sheet {
			found = true
			break
		}
	}
	if !found {
		sheet = f.GetSheetList()[0]
	}

	rows, err := f.GetRows(sheet)
	if err != nil {
		return nil, err
	}

	var out []ImportRow
	for i, row := range rows {
		rowNum := i + 1
		if rowNum == 1 {
			continue // header
		}
		if isBlankRow(row) {
			continue
		}
		out = append(out, parseImportRow(rowNum, row))
	}
	return out, nil
}

func isBlankRow(row []string) bool {
	for _, v := range row {
		if strings.TrimSpace(v) != "" {
			return false
		}
	}
	return true
}

func cellAt(row []string, i int) string {
	if i >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[i])
}

func parseDate(s string) (*time.Time, string) {
	if s == "" {
		return nil, ""
	}
	// Accept both a plain "2006-01-02" and Excel sometimes round-tripping a
	// date cell as "2006-01-02 00:00:00" or "1/2/2006".
	for _, layout := range []string{"2006-01-02", "2006-01-02 15:04:05", "1/2/2006", "01/02/2006"} {
		if t, err := time.Parse(layout, s); err == nil {
			return &t, ""
		}
	}
	return nil, fmt.Sprintf("could not read %q as a date (use YYYY-MM-DD)", s)
}

func parseImportRow(rowNum int, row []string) ImportRow {
	studentCode := cellAt(row, 0)
	firstName := cellAt(row, 1)
	if studentCode == "" || firstName == "" {
		return ImportRow{RowNumber: rowNum, ParseErr: "Student Code and First Name are required"}
	}

	gender := strings.ToLower(cellAt(row, 3))
	if gender != "" && gender != "male" && gender != "female" {
		return ImportRow{RowNumber: rowNum, ParseErr: fmt.Sprintf("gender must be \"male\" or \"female\", got %q", gender)}
	}

	dob, dobErr := parseDate(cellAt(row, 4))
	if dobErr != "" {
		return ImportRow{RowNumber: rowNum, ParseErr: "Date of Birth: " + dobErr}
	}
	admissionDate, admErr := parseDate(cellAt(row, 8))
	if admErr != "" {
		return ImportRow{RowNumber: rowNum, ParseErr: "Admission Date: " + admErr}
	}

	status := strings.ToLower(cellAt(row, 24))
	switch status {
	case "", "active", "inactive", "graduated", "transferred":
		// ok
	default:
		return ImportRow{RowNumber: rowNum, ParseErr: fmt.Sprintf("Status must be active/inactive/graduated/transferred, got %q", status)}
	}

	return ImportRow{
		RowNumber: rowNum,
		Input: CreateInput{
			StudentCode:       studentCode,
			FirstName:         firstName,
			LastName:          cellAt(row, 2),
			Gender:            gender,
			DateOfBirth:       dob,
			Email:             cellAt(row, 5),
			Phone:             cellAt(row, 6),
			Address:           cellAt(row, 7),
			AdmissionDate:     admissionDate,
			Caste:             cellAt(row, 9),
			Category:          cellAt(row, 10),
			AadharNumber:      cellAt(row, 11),
			SamagraID:         cellAt(row, 12),
			PenNumber:         cellAt(row, 13),
			AparID:            cellAt(row, 14),
			EnrollmentNumber:  cellAt(row, 15),
			AdmissionClass:    cellAt(row, 16),
			AdmissionYear:     cellAt(row, 17),
			PreviousSchool:    cellAt(row, 18),
			BankName:          cellAt(row, 19),
			BankIFSC:          cellAt(row, 20),
			BankAccountNumber: cellAt(row, 21),
			BankHolderName:    cellAt(row, 22),
			BankBranch:        cellAt(row, 23),
			Status:            status,
		},
	}
}
