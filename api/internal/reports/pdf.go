package reports

import (
	"bytes"
	"fmt"
	"time"

	"github.com/jung-kurt/gofpdf"
)

type ReportCardPDFData struct {
	ReportCard   *ReportCard
	GeneratedAt  string
}

func GenerateReportCardPDF(data ReportCardPDFData) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)

	pdf.Cell(0, 10, "CBC Report Card")
	pdf.Ln(12)
	pdf.SetFont("Arial", "", 11)
	pdf.Cell(0, 8, fmt.Sprintf("Learner: %s", data.ReportCard.LearnerName))
	pdf.Ln(6)
	pdf.Cell(0, 8, fmt.Sprintf("UPI: %s", data.ReportCard.UPI))
	pdf.Ln(6)
	pdf.Cell(0, 8, fmt.Sprintf("Grade: %s | Stream: %s", data.ReportCard.Grade, data.ReportCard.Stream))
	pdf.Ln(6)
	pdf.Cell(0, 8, fmt.Sprintf("Term: %d | Year: %d", data.ReportCard.Term, data.ReportCard.Year))
	pdf.Ln(6)
	pdf.Cell(0, 8, fmt.Sprintf("Generated: %s", data.GeneratedAt))
	pdf.Ln(12)

	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(0, 8, "Learning Area Performance")
	pdf.Ln(10)

	pdf.SetFont("Arial", "B", 10)
	pdf.Cell(60, 8, "Learning Area")
	pdf.Cell(50, 8, "Strand")
	pdf.Cell(30, 8, "Level")
	pdf.Cell(40, 8, "Comment")
	pdf.Ln(8)

	pdf.SetFont("Arial", "", 10)
	for _, item := range data.ReportCard.Items {
		pdf.Cell(60, 8, item.LearningArea)
		pdf.Cell(50, 8, item.StrandName)
		pdf.Cell(30, 8, item.RubricLabel)
		pdf.Cell(40, 8, "")
		pdf.Ln(8)
	}

	pdf.Ln(12)
	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(0, 8, "Teacher Comments")
	pdf.Ln(10)
	pdf.SetFont("Arial", "", 10)
	for area, comment := range data.ReportCard.TeacherComments {
		pdf.MultiCell(0, 6, fmt.Sprintf("%s: %s", area, comment), "", "", false)
		pdf.Ln(4)
	}

	pdf.Ln(12)
	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(0, 8, "Signature")
	pdf.Ln(10)
	pdf.Cell(90, 10, "Class Teacher: ____________________")
	pdf.Cell(90, 10, "Principal: ____________________")
	pdf.Ln(10)
	pdf.Cell(90, 10, "Date: ____________________")
	pdf.Cell(90, 10, "Date: ____________________")

	buf := new(bytes.Buffer)
	if err := pdf.Output(buf); err != nil {
		return nil, fmt.Errorf("generate pdf: %w", err)
	}
	return buf.Bytes(), nil
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

func GenerateReportCardFileName(learnerName string, term, year int) string {
	clean := fmt.Sprintf("%s_T%d_%d", learnerName, term, year)
	return fmt.Sprintf("report_card_%d_%s.pdf", time.Now().Unix(), clean)
}
