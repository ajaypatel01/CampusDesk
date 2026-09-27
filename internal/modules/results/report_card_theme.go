package results

import "github.com/go-pdf/fpdf"

// pdfTheme is a visual skin for the report-card PDF -- colors only, never
// content. The three content templates (kg/primary/middle) encode the
// school's actual grading rubric and stay exactly as they are regardless of
// theme; a theme only changes how a section band, a header row, a totals
// row, or a grade cell is painted.
type pdfTheme struct {
	Name string
	// ShadeBands: true paints section/header/total rows with a solid fill
	// (classic, modern); false (minimal) paints no fill and colors the text
	// instead, for a lighter, less "boxy" look.
	ShadeBands               bool
	SectionFill, SectionText [3]int
	HeaderFill, HeaderText   [3]int
	TotalFill, TotalText     [3]int
	RuleColor                [3]int
	// ColorBadges: true renders a grade cell with a color-coded background
	// (green/amber/red by first letter of the grade) instead of plain text.
	ColorBadges bool
}

var classicTheme = pdfTheme{
	Name:        "classic",
	ShadeBands:  true,
	SectionFill: [3]int{220, 230, 245},
	SectionText: [3]int{0, 0, 0},
	HeaderFill:  [3]int{235, 235, 235},
	HeaderText:  [3]int{0, 0, 0},
	TotalFill:   [3]int{230, 240, 230},
	TotalText:   [3]int{0, 0, 0},
	RuleColor:   [3]int{120, 120, 120},
	ColorBadges: false,
}

var modernTheme = pdfTheme{
	Name:        "modern",
	ShadeBands:  true,
	SectionFill: [3]int{37, 99, 235},
	SectionText: [3]int{255, 255, 255},
	HeaderFill:  [3]int{219, 234, 254},
	HeaderText:  [3]int{30, 58, 138},
	TotalFill:   [3]int{16, 129, 87},
	TotalText:   [3]int{255, 255, 255},
	RuleColor:   [3]int{37, 99, 235},
	ColorBadges: true,
}

var minimalTheme = pdfTheme{
	Name:        "minimal",
	ShadeBands:  false,
	SectionFill: [3]int{255, 255, 255},
	SectionText: [3]int{55, 65, 81},
	HeaderFill:  [3]int{255, 255, 255},
	HeaderText:  [3]int{55, 65, 81},
	TotalFill:   [3]int{255, 255, 255},
	TotalText:   [3]int{17, 24, 39},
	RuleColor:   [3]int{180, 180, 180},
	ColorBadges: false,
}

// themeByName defaults to classic for anything unrecognized (including no
// design param at all), so an old link or a typo renders exactly what it
// always has.
func themeByName(name string) pdfTheme {
	switch name {
	case "modern":
		return modernTheme
	case "minimal":
		return minimalTheme
	default:
		return classicTheme
	}
}

// setSectionStyle/setHeaderStyle/setTotalStyle set the fill+text color for
// the next cell(s) and return the fill flag to pass into CellFormat --
// callers must SetTextColor(0,0,0) back afterward.
func (t pdfTheme) setSectionStyle(pdf *fpdf.Fpdf) bool {
	pdf.SetFillColor(t.SectionFill[0], t.SectionFill[1], t.SectionFill[2])
	pdf.SetTextColor(t.SectionText[0], t.SectionText[1], t.SectionText[2])
	return t.ShadeBands
}

func (t pdfTheme) setHeaderStyle(pdf *fpdf.Fpdf) bool {
	pdf.SetFillColor(t.HeaderFill[0], t.HeaderFill[1], t.HeaderFill[2])
	pdf.SetTextColor(t.HeaderText[0], t.HeaderText[1], t.HeaderText[2])
	return t.ShadeBands
}

func (t pdfTheme) setTotalStyle(pdf *fpdf.Fpdf) bool {
	pdf.SetFillColor(t.TotalFill[0], t.TotalFill[1], t.TotalFill[2])
	pdf.SetTextColor(t.TotalText[0], t.TotalText[1], t.TotalText[2])
	return t.ShadeBands
}

// gradeBadge returns the fill+text color for a grade cell. With
// ColorBadges off it's just white-on-black -- i.e. CellFormat should be
// called with fill=false, plain text, same as before this feature.
func (t pdfTheme) gradeBadge(grade string) (fill [3]int, text [3]int, shaded bool) {
	if !t.ColorBadges || grade == "" {
		return [3]int{255, 255, 255}, [3]int{0, 0, 0}, false
	}
	switch grade[0] {
	case 'A':
		return [3]int{209, 250, 229}, [3]int{5, 150, 105}, true
	case 'B':
		return [3]int{254, 243, 199}, [3]int{180, 83, 9}, true
	default: // C, D, F, or anything else
		return [3]int{254, 226, 226}, [3]int{190, 18, 18}, true
	}
}
