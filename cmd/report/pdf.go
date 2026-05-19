package main

import (
	"fmt"
	"strings"

	"github.com/jung-kurt/gofpdf"
)



func SavePDF(s *ReportStats, periodLabel string, showCharts, showPersonnelDetails bool) (string, error) {
	pdf := gofpdf.New("P", "mm", "Letter", "")
	pdf.SetMargins(15, 15, 15)
	pdf.SetAutoPageBreak(true, 15)

	// ---- Title page ----
	pdf.AddPage()
	pageW, _ := pdf.GetPageSize()

	// Red header bar
	pdf.SetFillColor(196, 30, 58)
	pdf.Rect(0, 0, pageW, 28, "F")
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Helvetica", "B", 18)
	pdf.SetY(8)
	pdf.CellFormat(0, 12, safe(s.DeptName), "", 1, "C", false, 0, "")
	pdf.SetFont("Helvetica", "", 10)
	pdf.CellFormat(0, 6, safe(fmt.Sprintf("%s  -  Generated %s", periodLabel, s.GeneratedAt.Format("January 2, 2006"))), "", 1, "C", false, 0, "")

	pdf.SetTextColor(26, 26, 46)
	pdf.SetY(35)

	// Summary cards
	cardW := (pageW - 45) / 4
	cardH := 22.0
	cardY := pdf.GetY()
	drawSummaryCard(pdf, 15, cardY, cardW, cardH, "Total Calls", fmt.Sprintf("%d", s.TotalCalls))
	drawSummaryCard(pdf, 15+cardW+5, cardY, cardW, cardH, "EMS Calls", fmt.Sprintf("%d (%.1f%%)", s.EMSCount, s.EMSPct))
	drawSummaryCard(pdf, 15+(cardW+5)*2, cardY, cardW, cardH, "Fire Calls", fmt.Sprintf("%d (%.1f%%)", s.FireCount, s.FirePct))
	drawSummaryCard(pdf, 15+(cardW+5)*3, cardY, cardW, cardH, "Avg Personnel", fmt.Sprintf("%.1f", s.AvgPersonnel))
	pdf.SetY(cardY + cardH + 10)

	// ---- Response Times ----
	sectionHeader(pdf, "Response Times")
	pdf.SetFont("Helvetica", "", 10)
	addTableRow(pdf, []string{"Metric", "Average (trimmed)"}, true)
	addTableRow(pdf, []string{"Dispatch to En Route", formatDurationStr(s.DispatchToEnRouteAvg)}, false)
	addTableRow(pdf, []string{"Dispatch to Arrival", formatDurationStr(s.DispatchToArrivalAvg)}, false)
	addTableRow(pdf, []string{"Time on Scene", formatDurationStr(s.TimeOnSceneAvg)}, false)

	// ---- Concurrent ----
	sectionHeader(pdf, "Concurrent Calls")
	pdf.SetFont("Helvetica", "", 10)
	pdf.CellFormat(0, 6, safe(fmt.Sprintf("%d of %d calls overlapped (%.1f%%)", s.ConcurrentCount, s.TotalCalls, s.ConcurrentPct)), "", 1, "L", false, 0, "")

	// ---- Mutual Aid ----
	sectionHeader(pdf, "Mutual Aid")
	addTableRow(pdf, []string{"Direction", "Count", "%"}, true)
	addTableRow(pdf, []string{"Given", fmt.Sprintf("%d", s.MutualAidGiven), fmt.Sprintf("%.1f%%", s.MutualAidGivenPct)}, false)
	addTableRow(pdf, []string{"Received", fmt.Sprintf("%d", s.MutualAidReceived), fmt.Sprintf("%.1f%%", s.MutualAidReceivedPct)}, false)

	if len(s.MutualAidGivenDistricts) > 0 {
		ensureSpace(pdf, 40)
		pdf.SetFont("Helvetica", "B", 10)
		pdf.SetTextColor(196, 30, 58)
		pdf.CellFormat(0, 6, "Given -- by District", "", 1, "L", false, 0, "")
		pdf.SetTextColor(26, 26, 46)
		pdf.SetFont("Helvetica", "", 10)
		addTableRow(pdf, []string{"District", "Count"}, true)
		for d, c := range s.MutualAidGivenDistricts {
			addTableRow(pdf, []string{safe(d), fmt.Sprintf("%d", c)}, false)
		}
	}

	// ---- EMS Mutual Aid ----
	sectionHeader(pdf, "EMS Mutual Aid by District")
	pdf.CellFormat(0, 6, safe(fmt.Sprintf("Total: %d", s.EMSMutualAidTotal)), "", 1, "L", false, 0, "")
	if len(s.EMSMutualAidDistricts) > 0 {
		addTableRow(pdf, []string{"District", "Count"}, true)
		for d, c := range s.EMSMutualAidDistricts {
			addTableRow(pdf, []string{safe(d), fmt.Sprintf("%d", c)}, false)
		}
	}

	// ---- Shifts ----
	sectionHeader(pdf, "Calls by Shift")
	addTableRow(pdf, []string{"Shift", "Count", "%"}, true)
	for shift, count := range s.ShiftCounts {
		pct := 0.0
		if s.TotalCalls > 0 { pct = float64(count) / float64(s.TotalCalls) * 100 }
		addTableRow(pdf, []string{safe(shift), fmt.Sprintf("%d", count), fmt.Sprintf("%.1f%%", pct)}, false)
	}

	// ---- Special Types ----
	sectionHeader(pdf, "Special Incident Types")
	addTableRow(pdf, []string{"Type", "Count", "%"}, true)
	addTableRow(pdf, []string{"Hazardous Materials", fmt.Sprintf("%d", s.HazmatCount), fmt.Sprintf("%.1f%%", s.HazmatPct)}, false)
	addTableRow(pdf, []string{"Motor Vehicle Accidents", fmt.Sprintf("%d", s.MVACount), fmt.Sprintf("%.1f%%", s.MVAPct)}, false)
	addTableRow(pdf, []string{"Carbon Monoxide", fmt.Sprintf("%d", s.COCount), fmt.Sprintf("%.1f%%", s.COPct)}, false)

	// ---- Special Units ----
	sectionHeader(pdf, "Special Unit Responses")
	addTableRow(pdf, []string{"Unit", "Count", "%"}, true)
	addTableRow(pdf, []string{"Drone Team (UAV163)", fmt.Sprintf("%d", s.DroneTeamCount), fmt.Sprintf("%.1f%%", s.DroneTeamPct)}, false)
	addTableRow(pdf, []string{"Rehab Team (S263)", fmt.Sprintf("%d", s.RehabTeamCount), fmt.Sprintf("%.1f%%", s.RehabTeamPct)}, false)

	// ---- Personnel Response ----
	ensureSpace(pdf, 50)
	pdf.SetFont("Helvetica", "B", 13)
	pdf.SetTextColor(196, 30, 58)
	pdf.CellFormat(0, 8, "Personnel Response", "", 1, "L", false, 0, "")
	pdf.SetDrawColor(200, 200, 200)
	pageW, _ = pdf.GetPageSize()
	pdf.Line(15, pdf.GetY(), pageW-15, pdf.GetY())
	pdf.Ln(3)
	pdf.SetFont("Helvetica", "", 10)
	pdf.SetTextColor(26, 26, 46)
	pdf.CellFormat(0, 6, safe(fmt.Sprintf("%d total responding members", s.PersonnelTotalResponding)), "", 1, "L", false, 0, "")
	pdf.Ln(2)

	addPersonnelSection(pdf, safe(fmt.Sprintf("High Responders (>=30%%) -- %d members", s.PersonnelHighCount)), s.PersonnelHigh, showPersonnelDetails)
	addPersonnelSection(pdf, safe(fmt.Sprintf("Active Members (>=20%%) -- %d cumulative", s.PersonnelActiveCount)), s.PersonnelActive, showPersonnelDetails)
	addPersonnelSection(pdf, safe(fmt.Sprintf("Members in Good Standing (>=10%%) -- %d cumulative", s.PersonnelGoodStandingCount)), s.PersonnelGoodStanding, showPersonnelDetails)

	// Footer
	pdf.SetY(-25)
	pdf.SetFont("Helvetica", "I", 8)
	pdf.SetTextColor(102, 102, 102)
	pdf.CellFormat(0, 5, "Generated by Emergency Networking Reporting Tool", "", 1, "C", false, 0, "")

	filename := sanitizeFilename(s.DeptName) + "_" + sanitizeFilename(periodLabel) + ".pdf"
	if err := pdf.OutputFileAndClose(filename); err != nil {
		return "", fmt.Errorf("write pdf: %w", err)
	}
	return filename, nil
}

func drawSummaryCard(pdf *gofpdf.Fpdf, x, y, w, h float64, label, value string) {
	// Card background
	pdf.SetFillColor(255, 255, 255)
	pdf.SetDrawColor(238, 238, 238)
	pdf.RoundedRect(x, y, w, h, 3, "1234", "DF")
	// Label
	pdf.SetFont("Helvetica", "", 8)
	pdf.SetTextColor(102, 102, 102)
	pdf.SetXY(x+3, y+2)
	pdf.CellFormat(w-6, 5, safe(label), "", 0, "L", false, 0, "")
	// Value
	pdf.SetFont("Helvetica", "B", 14)
	pdf.SetTextColor(196, 30, 58)
	pdf.SetXY(x+3, y+7)
	pdf.CellFormat(w-6, 10, safe(value), "", 0, "L", false, 0, "")
	pdf.SetTextColor(26, 26, 46)
}

func safe(s string) string {
	r := strings.NewReplacer(
		"—", "--", "–", "--", "‘", "'", "’", "'",
		"“", `"`, "”", `"`, "…", "...", " ", " ",
		"≥", ">=", "≤", "<=", "·", "-", "•", "-",
		"é", "e", "è", "e", "ñ", "n", "ü", "u", "ä", "a", "ö", "o",
	)
	return r.Replace(s)
}

func ensureSpace(pdf *gofpdf.Fpdf, neededMM float64) {
	_, pageH := pdf.GetPageSize()
	_, _, _, bottom := pdf.GetMargins()
	if pdf.GetY()+neededMM > pageH-bottom {
		pdf.AddPage()
	}
}

func sectionHeader(pdf *gofpdf.Fpdf, title string) {
	ensureSpace(pdf, 30)
	pdf.Ln(4)
	pdf.SetFont("Helvetica", "B", 13)
	pdf.SetTextColor(196, 30, 58)
	pdf.CellFormat(0, 8, safe(title), "", 1, "L", false, 0, "")
	pdf.SetDrawColor(238, 238, 238)
	pageW, _ := pdf.GetPageSize()
	pdf.Line(15, pdf.GetY(), pageW-15, pdf.GetY())
	pdf.Ln(3)
	pdf.SetTextColor(26, 26, 46)
}

func addTableRow(pdf *gofpdf.Fpdf, cells []string, header bool) {
	var widths []float64
	switch len(cells) {
	case 2: widths = []float64{130, 50}
	case 5: widths = []float64{50, 25, 20, 25, 30}
	default: widths = []float64{80, 50, 50}
	}
	if header {
		pdf.SetFont("Helvetica", "B", 10)
		pdf.SetTextColor(102, 102, 102)
		pdf.SetFillColor(248, 249, 250)
	} else {
		pdf.SetFont("Helvetica", "", 10)
		pdf.SetTextColor(26, 26, 46)
		pdf.SetFillColor(255, 255, 255)
	}
	for i, cell := range cells {
		w := 30.0
		if i < len(widths) { w = widths[i] }
		pdf.CellFormat(w, 6, safe(cell), "1", 0, "L", true, 0, "")
	}
	pdf.Ln(-1)
	pdf.SetTextColor(26, 26, 46)
}

func addPersonnelSection(pdf *gofpdf.Fpdf, title string, records []PersonnelRecord, showDetails bool) {
	ensureSpace(pdf, 35)
	pdf.SetFont("Helvetica", "B", 10)
	pdf.SetTextColor(196, 30, 58)
	pdf.CellFormat(0, 7, safe(title), "", 1, "L", false, 0, "")
	pdf.SetTextColor(26, 26, 46)
	if showDetails && len(records) > 0 {
		pdf.SetFont("Helvetica", "", 9)
		addTableRow(pdf, []string{"Name", "Calls", "%", "On Scene", "Not On Scene"}, true)
		for i, r := range records {
			if i > 0 && i%30 == 0 {
				ensureSpace(pdf, 40)
			}
			addTableRow(pdf, []string{
				safe(r.Name), fmt.Sprintf("%d", r.TotalCalls),
				fmt.Sprintf("%.1f%%", r.Pct),
				fmt.Sprintf("%d", r.OnScene),
				fmt.Sprintf("%d", r.NotOnScene),
			}, false)
		}
	}
	pdf.Ln(3)
}

func formatDurationStr(seconds float64) string {
	if seconds <= 0 { return "N/A" }
	d := int(seconds)
	m := d / 60
	s := d % 60
	if m > 0 { return fmt.Sprintf("%dm %ds", m, s) }
	return fmt.Sprintf("%ds", s)
}
