package reports

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jung-kurt/gofpdf"
)

// rubricShort is the level as it is printed in the narrow column.
func rubricShort(level *int) string {
	if level == nil {
		return "-"
	}
	return fmt.Sprintf("%d  %s", *level, rubricLabel(*level))
}

// RenderReportCardPDF lays a report card out on A4.
//
// The built-in fonts cover Latin-1 only, so text goes through a translator:
// a name with a character outside it prints with a substitute instead of
// breaking the document.
func RenderReportCardPDF(school School, card *ReportCard) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	tr := pdf.UnicodeTranslatorFromDescriptor("")
	pdf.SetMargins(15, 15, 15)
	pdf.SetAutoPageBreak(true, 18)
	pdf.SetTitle(tr(fmt.Sprintf("Report card - %s - Term %d %d", card.LearnerName, card.Term, card.Year)), false)
	pdf.SetFooterFunc(func() {
		pdf.SetY(-12)
		pdf.SetFont("Arial", "", 8)
		pdf.SetTextColor(110, 110, 110)
		pdf.CellFormat(0, 5, tr(fmt.Sprintf("%s  |  %s  |  Term %d %d  |  Page %d", school.Name, card.LearnerName, card.Term, card.Year, pdf.PageNo())), "", 0, "C", false, 0, "")
		pdf.SetTextColor(0, 0, 0)
	})
	pdf.AddPage()
	const width = 180.0

	// School
	pdf.SetFont("Arial", "B", 16)
	pdf.CellFormat(width, 8, tr(school.Name), "", 1, "C", false, 0, "")
	pdf.SetFont("Arial", "", 9)
	contact := []string{}
	for _, part := range []string{school.Address, school.County, school.Phone, school.Email} {
		if strings.TrimSpace(part) != "" {
			contact = append(contact, strings.TrimSpace(part))
		}
	}
	if len(contact) > 0 {
		pdf.CellFormat(width, 5, tr(strings.Join(contact, "  |  ")), "", 1, "C", false, 0, "")
	}
	pdf.Ln(2)
	pdf.SetFont("Arial", "B", 12)
	title := fmt.Sprintf("Learner progress report - Term %d, %d", card.Term, card.Year)
	if card.Status != StatusFinal {
		title += "  (DRAFT - not yet published)"
	}
	pdf.CellFormat(width, 8, tr(title), "TB", 1, "C", false, 0, "")
	pdf.Ln(3)

	// Learner
	field := func(label, value string, w float64) {
		pdf.SetFont("Arial", "B", 10)
		pdf.CellFormat(28, 6, tr(label), "", 0, "L", false, 0, "")
		pdf.SetFont("Arial", "", 10)
		pdf.CellFormat(w-28, 6, tr(value), "", 0, "L", false, 0, "")
	}
	class := card.Grade
	if card.Stream != "" {
		class += " " + card.Stream
	}
	field("Learner", card.LearnerName, 110)
	field("Class", class, 70)
	pdf.Ln(6)
	field("UPI", card.UPI, 110)
	overall := "-"
	if card.OverallRating != nil {
		overall = fmt.Sprintf("%d - %s", *card.OverallRating, rubricLabel(*card.OverallRating))
	}
	field("Overall", overall, 70)
	pdf.Ln(9)

	// Attendance
	if days, ok := number(card.AttendanceSummary["total_days"]); ok && days > 0 {
		get := func(key string) int {
			v, _ := number(card.AttendanceSummary[key])
			return int(v)
		}
		rate, _ := number(card.AttendanceSummary["attendance_rate"])
		pdf.SetFont("Arial", "B", 10)
		pdf.CellFormat(28, 6, "Attendance", "", 0, "L", false, 0, "")
		pdf.SetFont("Arial", "", 10)
		pdf.CellFormat(width-28, 6, tr(fmt.Sprintf("%.0f%% - present %d, late %d, absent %d, excused %d of %d days marked",
			rate, get("present_days"), get("late_days"), get("absent_days"), get("excused_days"), int(days))), "", 1, "L", false, 0, "")
		pdf.Ln(3)
	}

	// Learning areas
	widths := []float64{62, 40, 78}
	header := func() {
		pdf.SetFont("Arial", "B", 9)
		pdf.SetFillColor(235, 235, 235)
		for i, h := range []string{"Sub-strand", "Level", "Teacher's note"} {
			pdf.CellFormat(widths[i], 7, h, "1", 0, "L", true, 0, "")
		}
		pdf.Ln(-1)
	}
	area, strand := "", ""
	for _, item := range card.Items {
		if item.LearningArea != area {
			area, strand = item.LearningArea, ""
			if pdf.GetY() > 250 {
				pdf.AddPage()
			}
			pdf.Ln(2)
			pdf.SetFont("Arial", "B", 11)
			pdf.CellFormat(width, 7, tr(area), "", 1, "L", false, 0, "")
			header()
		}
		if item.StrandName != strand {
			strand = item.StrandName
			pdf.SetFont("Arial", "I", 9)
			pdf.CellFormat(width, 6, tr(strand), "LR", 1, "L", false, 0, "")
		}
		pdf.SetFont("Arial", "", 9)
		cells := []string{tr(item.SubStrandName), tr(rubricShort(item.RubricLevel)), tr(item.Comment)}
		// The row is as tall as its tallest cell.
		lines := 1
		for i, text := range cells {
			if n := len(pdf.SplitLines([]byte(text), widths[i]-2)); n > lines {
				lines = n
			}
		}
		height := float64(lines) * 5
		if pdf.GetY()+height > 275 {
			pdf.AddPage()
			header()
			pdf.SetFont("Arial", "", 9)
		}
		x, y := pdf.GetXY()
		for i, text := range cells {
			pdf.Rect(x, y, widths[i], height, "D")
			pdf.SetXY(x+1, y)
			pdf.MultiCell(widths[i]-2, 5, text, "", "L", false)
			x += widths[i]
		}
		pdf.SetXY(15, y+height)
	}

	section := func(title string, remarks map[string]string) {
		if len(remarks) == 0 {
			return
		}
		keys := make([]string, 0, len(remarks))
		for k := range remarks {
			if strings.TrimSpace(remarks[k]) != "" {
				keys = append(keys, k)
			}
		}
		if len(keys) == 0 {
			return
		}
		sort.Strings(keys)
		pdf.Ln(5)
		pdf.SetFont("Arial", "B", 11)
		pdf.CellFormat(width, 7, title, "", 1, "L", false, 0, "")
		for _, k := range keys {
			pdf.SetFont("Arial", "B", 9)
			pdf.MultiCell(width, 5, tr(k), "", "L", false)
			pdf.SetFont("Arial", "", 9)
			pdf.MultiCell(width, 5, tr(remarks[k]), "", "L", false)
			pdf.Ln(1)
		}
	}
	section("Core competencies", card.CoreCompetencyRemarks)
	section("Teacher's comments", card.TeacherComments)

	// Key and signatures
	if pdf.GetY() > 240 {
		pdf.AddPage()
	}
	pdf.Ln(5)
	pdf.SetFont("Arial", "", 8)
	pdf.MultiCell(width, 4, "Levels: 4 Exceeding Expectation, 3 Meeting Expectation, 2 Approaching Expectation, 1 Below Expectation.", "", "L", false)
	pdf.Ln(10)
	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(90, 6, "Class teacher: ______________________", "", 0, "L", false, 0, "")
	pdf.CellFormat(90, 6, "Head of school: ______________________", "", 1, "L", false, 0, "")
	pdf.Ln(4)
	pdf.CellFormat(90, 6, "Parent/guardian: ____________________", "", 0, "L", false, 0, "")
	pdf.CellFormat(90, 6, "Date: ______________________", "", 1, "L", false, 0, "")
	if strings.TrimSpace(school.Footer) != "" {
		pdf.Ln(5)
		pdf.SetFont("Arial", "I", 9)
		pdf.MultiCell(width, 5, tr(school.Footer), "", "C", false)
	}

	buf := new(bytes.Buffer)
	if err := pdf.Output(buf); err != nil {
		return nil, fmt.Errorf("generate pdf: %w", err)
	}
	return buf.Bytes(), nil
}

// number reads a figure out of the attendance summary, which comes back from
// JSON as float64.
func number(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int64:
		return float64(n), true
	case int:
		return float64(n), true
	}
	return 0, false
}

// ReportCardFileName is the name the PDF is downloaded under: letters, digits
// and dashes only, so it is safe in a header and on any file system.
func ReportCardFileName(card *ReportCard) string {
	var b strings.Builder
	for _, r := range strings.ToLower(card.LearnerName) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case b.Len() > 0 && !strings.HasSuffix(b.String(), "-"):
			b.WriteByte('-')
		}
	}
	name := strings.Trim(b.String(), "-")
	if name == "" {
		name = "learner"
	}
	return fmt.Sprintf("report-card-%s-term-%d-%d.pdf", name, card.Term, card.Year)
}

func GenerateReceiptPDF(invoiceNumber, learnerName, amountKES string, paidAt time.Time) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(0, 10, "Payment Receipt")
	pdf.Ln(12)
	pdf.SetFont("Arial", "", 11)
	pdf.Cell(0, 8, fmt.Sprintf("Invoice: %s", invoiceNumber))
	pdf.Ln(6)
	pdf.Cell(0, 8, fmt.Sprintf("Learner: %s", learnerName))
	pdf.Ln(6)
	pdf.Cell(0, 8, fmt.Sprintf("Amount: KES %s", amountKES))
	pdf.Ln(6)
	pdf.Cell(0, 8, fmt.Sprintf("Paid: %s", paidAt.Format("2006-01-02 15:04")))
	pdf.Ln(12)
	pdf.SetFont("Arial", "", 10)
	pdf.Cell(0, 8, "Thank you for your payment.")
	pdf.Ln(10)
	pdf.Cell(90, 10, "Received by: ____________________")
	pdf.Cell(90, 10, "Date: ____________________")

	buf := new(bytes.Buffer)
	if err := pdf.Output(buf); err != nil {
		return nil, fmt.Errorf("generate receipt pdf: %w", err)
	}
	return buf.Bytes(), nil
}

func GenerateReceiptFileName(invoiceNumber string) string {
	return fmt.Sprintf("receipt_%s_%d.pdf", invoiceNumber, time.Now().Unix())
}
