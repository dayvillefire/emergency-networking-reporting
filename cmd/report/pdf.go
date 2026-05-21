package main

import (
	"fmt"
	"strings"

	"github.com/jung-kurt/gofpdf"
)

// Chart dataset types for PDF chart rendering.
type pdfBarDataset struct {
	Label string
	Data  []float64
	Color [3]int // RGB
}

type pdfLineDataset struct {
	Label string
	Data  []float64
	Color [3]int // RGB
}

// Chart layout constants
const (
	chartLeftFull   = 30.0
	chartRightFull  = 170.0
	chartHeightFull = 38.0
	chartWidthFull  = chartRightFull - chartLeftFull // 140

	chartLeftHalfL  = 28.0
	chartWidthHalf  = 75.0
	chartLeftHalfR  = 113.0
	chartHeightHalf = 32.0
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
	pdf.Rect(0, 0, pageW, 18, "F")
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Helvetica", "B", 14)
	pdf.SetY(4)
	pdf.CellFormat(0, 9, safe(s.DeptName), "", 1, "C", false, 0, "")
	pdf.SetFont("Helvetica", "", 8)
	pdf.CellFormat(0, 5, safe(fmt.Sprintf("%s  -  Generated %s", periodLabel, s.GeneratedAt.Format("January 2, 2006"))), "", 1, "C", false, 0, "")

	pdf.SetTextColor(26, 26, 46)
	pdf.SetY(24)

	// Summary cards
	cardW := (pageW - 45) / 4
	cardH := 16.0
	cardY := pdf.GetY()
	drawSummaryCard(pdf, 15, cardY, cardW, cardH, "Total Calls", fmt.Sprintf("%d", s.TotalCalls))
	drawSummaryCard(pdf, 15+cardW+5, cardY, cardW, cardH, "EMS Calls", fmt.Sprintf("%d (%.1f%%)", s.EMSCount, s.EMSPct))
	drawSummaryCard(pdf, 15+(cardW+5)*2, cardY, cardW, cardH, "Fire Calls", fmt.Sprintf("%d (%.1f%%)", s.FireCount, s.FirePct))
	drawSummaryCard(pdf, 15+(cardW+5)*3, cardY, cardW, cardH, "Avg Personnel", fmt.Sprintf("%.1f", s.AvgPersonnel))
	pdf.SetY(cardY + cardH + 6)

	// ---- Charts (multi-month only) ----
	if showCharts && len(s.Monthly) > 0 {
		months := make([]string, len(s.Monthly))
		for i, m := range s.Monthly {
			months[i] = m.Month
		}

		// Pair 1: Call Volume (bar) + Average Personnel (line)
		emsData := make([]float64, len(s.Monthly))
		fireData := make([]float64, len(s.Monthly))
		for i, m := range s.Monthly {
			emsData[i] = float64(m.EMS)
			fireData[i] = float64(m.Fire)
		}
		avgPersData := make([]float64, len(s.Monthly))
		for i, m := range s.Monthly {
			avgPersData[i] = m.AvgPersonnel
		}
		drawSideBySide(pdf, 60,
			func() {
				drawPDFBarChart(pdf, "Call Volume", months, []pdfBarDataset{
					{Label: "EMS", Data: emsData, Color: [3]int{37, 99, 235}},
					{Label: "Fire", Data: fireData, Color: [3]int{196, 30, 58}},
				}, chartLeftHalfL, chartWidthHalf, chartHeightHalf, true)
			},
			func() {
				drawPDFLineChart(pdf, "Average Personnel", months, []pdfLineDataset{
					{Label: "Avg Personnel", Data: avgPersData, Color: [3]int{37, 99, 235}},
				}, chartLeftHalfR, chartWidthHalf, chartHeightHalf, true)
			},
		)

		// Chart 3: Response Times (full-width, multi-line)
		enRouteData := make([]float64, len(s.Monthly))
		arrivalData := make([]float64, len(s.Monthly))
		sceneData := make([]float64, len(s.Monthly))
		for i, m := range s.Monthly {
			enRouteData[i] = m.DispatchToEnRoute
			arrivalData[i] = m.DispatchToArrival
			sceneData[i] = m.TimeOnScene
		}
		ensureSpace(pdf, 60)
		drawPDFLineChart(pdf, "Response Times (seconds)", months, []pdfLineDataset{
			{Label: "Disp→EnRoute", Data: enRouteData, Color: [3]int{34, 197, 94}},
			{Label: "Disp→Arrival", Data: arrivalData, Color: [3]int{245, 158, 11}},
			{Label: "On Scene", Data: sceneData, Color: [3]int{139, 92, 246}},
		}, chartLeftFull, chartWidthFull, chartHeightFull, false)

		// Pair 2: Concurrent Calls (line) + Mutual Aid (bar)
		concurData := make([]float64, len(s.Monthly))
		for i, m := range s.Monthly {
			concurData[i] = float64(m.Concurrent)
		}
		givenData := make([]float64, len(s.Monthly))
		receivedData := make([]float64, len(s.Monthly))
		for i, m := range s.Monthly {
			givenData[i] = float64(m.MutualAidGiven)
			receivedData[i] = float64(m.MutualAidReceived)
		}
		drawSideBySide(pdf, 60,
			func() {
				drawPDFLineChart(pdf, "Concurrent Calls", months, []pdfLineDataset{
					{Label: "Concurrent Calls", Data: concurData, Color: [3]int{239, 68, 68}},
				}, chartLeftHalfL, chartWidthHalf, chartHeightHalf, true)
			},
			func() {
				drawPDFBarChart(pdf, "Mutual Aid", months, []pdfBarDataset{
					{Label: "Given", Data: givenData, Color: [3]int{34, 197, 94}},
					{Label: "Received", Data: receivedData, Color: [3]int{245, 158, 11}},
				}, chartLeftHalfR, chartWidthHalf, chartHeightHalf, true)
			},
		)

		// Pair 3: Special Incident Types (bar, 3 datasets) + Special Unit Responses (bar)
		hazmatData := make([]float64, len(s.Monthly))
		mvaData := make([]float64, len(s.Monthly))
		coData := make([]float64, len(s.Monthly))
		for i, m := range s.Monthly {
			hazmatData[i] = float64(m.Hazmat)
			mvaData[i] = float64(m.MVA)
			coData[i] = float64(m.CO)
		}
		droneData := make([]float64, len(s.Monthly))
		rehabData := make([]float64, len(s.Monthly))
		for i, m := range s.Monthly {
			droneData[i] = float64(m.DroneTeam)
			rehabData[i] = float64(m.RehabTeam)
		}
		drawSideBySide(pdf, 65,
			func() {
				drawPDFBarChart(pdf, "Special Incident Types", months, []pdfBarDataset{
					{Label: "Hazmat", Data: hazmatData, Color: [3]int{239, 68, 68}},
					{Label: "MVA", Data: mvaData, Color: [3]int{249, 115, 22}},
					{Label: "CO", Data: coData, Color: [3]int{99, 102, 241}},
				}, chartLeftHalfL, chartWidthHalf, chartHeightHalf, true)
			},
			func() {
				drawPDFBarChart(pdf, "Special Unit Responses", months, []pdfBarDataset{
					{Label: "Drone (UAV163)", Data: droneData, Color: [3]int{6, 182, 212}},
					{Label: "Rehab (S263)", Data: rehabData, Color: [3]int{168, 85, 247}},
				}, chartLeftHalfR, chartWidthHalf, chartHeightHalf, true)
			},
		)
	}

	// ---- Response Times ----
	sectionHeader(pdf, "Response Times")
	pdf.SetFont("Helvetica", "", 8)
	addTableRow(pdf, []string{"Metric", "Average (trimmed)"}, true)
	addTableRow(pdf, []string{"Dispatch to En Route", formatDurationStr(s.DispatchToEnRouteAvg)}, false)
	addTableRow(pdf, []string{"Dispatch to Arrival", formatDurationStr(s.DispatchToArrivalAvg)}, false)
	addTableRow(pdf, []string{"Time on Scene", formatDurationStr(s.TimeOnSceneAvg)}, false)

	// ---- Concurrent ----
	sectionHeader(pdf, "Concurrent Calls")
	pdf.SetFont("Helvetica", "", 8)
	pdf.CellFormat(0, 5, safe(fmt.Sprintf("%d of %d calls overlapped (%.1f%%)", s.ConcurrentCount, s.TotalCalls, s.ConcurrentPct)), "", 1, "L", false, 0, "")

	// ---- Mutual Aid ----
	sectionHeader(pdf, "Mutual Aid")
	addTableRow(pdf, []string{"Direction", "Count", "%"}, true)
	addTableRow(pdf, []string{"Given", fmt.Sprintf("%d", s.MutualAidGiven), fmt.Sprintf("%.1f%%", s.MutualAidGivenPct)}, false)
	addTableRow(pdf, []string{"Received", fmt.Sprintf("%d", s.MutualAidReceived), fmt.Sprintf("%.1f%%", s.MutualAidReceivedPct)}, false)

	if len(s.MutualAidGivenDistricts) > 0 {
		ensureSpace(pdf, 30)
		pdf.SetFont("Helvetica", "B", 8)
		pdf.SetTextColor(196, 30, 58)
		pdf.CellFormat(0, 5, "Given -- by District", "", 1, "L", false, 0, "")
		pdf.SetTextColor(26, 26, 46)
		pdf.SetFont("Helvetica", "", 8)
		addTableRow(pdf, []string{"District", "Count"}, true)
		for d, c := range s.MutualAidGivenDistricts {
			addTableRow(pdf, []string{safe(d), fmt.Sprintf("%d", c)}, false)
		}
	}

	// ---- EMS Mutual Aid ----
	sectionHeader(pdf, "EMS Mutual Aid by District")
	pdf.CellFormat(0, 5, safe(fmt.Sprintf("Total: %d", s.EMSMutualAidTotal)), "", 1, "L", false, 0, "")
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
		if s.TotalCalls > 0 {
			pct = float64(count) / float64(s.TotalCalls) * 100
		}
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
	ensureSpace(pdf, 35)
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetTextColor(196, 30, 58)
	pdf.CellFormat(0, 6, "Personnel Response", "", 1, "L", false, 0, "")
	pdf.SetDrawColor(200, 200, 200)
	pageW, _ = pdf.GetPageSize()
	pdf.Line(15, pdf.GetY(), pageW-15, pdf.GetY())
	pdf.Ln(2)
	pdf.SetFont("Helvetica", "", 8)
	pdf.SetTextColor(26, 26, 46)
	pdf.CellFormat(0, 5, safe(fmt.Sprintf("%d total responding members", s.PersonnelTotalResponding)), "", 1, "L", false, 0, "")
	pdf.Ln(1)

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
	pdf.SetFont("Helvetica", "", 6)
	pdf.SetTextColor(102, 102, 102)
	pdf.SetXY(x+3, y+2)
	pdf.CellFormat(w-6, 4, safe(label), "", 0, "L", false, 0, "")
	// Value
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetTextColor(196, 30, 58)
	pdf.SetXY(x+3, y+6)
	pdf.CellFormat(w-6, 8, safe(value), "", 0, "L", false, 0, "")
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
	ensureSpace(pdf, 20)
	pdf.Ln(2)
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetTextColor(196, 30, 58)
	pdf.CellFormat(0, 6, safe(title), "", 1, "L", false, 0, "")
	pdf.SetDrawColor(238, 238, 238)
	pageW, _ := pdf.GetPageSize()
	pdf.Line(15, pdf.GetY(), pageW-15, pdf.GetY())
	pdf.Ln(2)
	pdf.SetTextColor(26, 26, 46)
}

// drawSideBySide draws leftFn and rightFn at the same Y position.
func drawSideBySide(pdf *gofpdf.Fpdf, spaceNeeded float64, leftFn, rightFn func()) {
	ensureSpace(pdf, spaceNeeded)
	y0 := pdf.GetY()
	leftFn()
	yAfterLeft := pdf.GetY()
	pdf.SetY(y0)
	rightFn()
	yAfterRight := pdf.GetY()
	if yAfterLeft > yAfterRight {
		pdf.SetY(yAfterLeft)
	}
}

func addTableRow(pdf *gofpdf.Fpdf, cells []string, header bool) {
	var widths []float64
	switch len(cells) {
	case 2:
		widths = []float64{130, 50}
	case 5:
		widths = []float64{50, 25, 20, 25, 30}
	default:
		widths = []float64{80, 50, 50}
	}
	if header {
		pdf.SetFont("Helvetica", "B", 9)
		pdf.SetTextColor(102, 102, 102)
		pdf.SetFillColor(248, 249, 250)
	} else {
		pdf.SetFont("Helvetica", "", 8)
		pdf.SetTextColor(26, 26, 46)
		pdf.SetFillColor(255, 255, 255)
	}
	for i, cell := range cells {
		w := 30.0
		if i < len(widths) {
			w = widths[i]
		}
		pdf.CellFormat(w, 5, safe(cell), "1", 0, "L", true, 0, "")
	}
	pdf.Ln(-1)
	pdf.SetTextColor(26, 26, 46)
}

func addPersonnelSection(pdf *gofpdf.Fpdf, title string, records []PersonnelRecord, showDetails bool) {
	ensureSpace(pdf, 25)
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetTextColor(196, 30, 58)
	pdf.CellFormat(0, 5, safe(title), "", 1, "L", false, 0, "")
	pdf.SetTextColor(26, 26, 46)
	if showDetails && len(records) > 0 {
		pdf.SetFont("Helvetica", "", 7)
		addTableRow(pdf, []string{"Name", "Calls", "%", "On Scene", "Not On Scene"}, true)
		for i, r := range records {
			if i > 0 && i%30 == 0 {
				ensureSpace(pdf, 30)
			}
			addTableRow(pdf, []string{
				safe(r.Name), fmt.Sprintf("%d", r.TotalCalls),
				fmt.Sprintf("%.1f%%", r.Pct),
				fmt.Sprintf("%d", r.OnScene),
				fmt.Sprintf("%d", r.NotOnScene),
			}, false)
		}
	}
	pdf.Ln(2)
}

func formatDurationStr(seconds float64) string {
	if seconds <= 0 {
		return "N/A"
	}
	d := int(seconds)
	m := d / 60
	s := d % 60
	if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

func yAxisScale(maxVal float64) float64 {
	if maxVal <= 0 {
		return 10
	}
	// Round up to a nice ceiling.
	nice := []float64{5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000}
	for _, n := range nice {
		if maxVal <= n {
			return n
		}
	}
	// Fallback: round up to nearest multiple of 5000.
	return ((maxVal / 5000) + 1) * 5000
}

func drawPDFBarChart(pdf *gofpdf.Fpdf, title string, months []string, datasets []pdfBarDataset, xLeft, chartWidth, chartHeight float64, compact bool) {
	pdf.Ln(4)
	chartRight := xLeft + chartWidth
	chartLeft := xLeft
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetTextColor(196, 30, 58)
	if compact {
		pdf.SetXY(xLeft+2, pdf.GetY())
		pdf.CellFormat(chartWidth-4, 7, safe(title), "", 1, "L", false, 0, "")
	} else {
		pdf.CellFormat(0, 7, safe(title), "", 1, "L", false, 0, "")
	}
	pdf.SetTextColor(26, 26, 46)

	const barGroupGap = 3.0
	chartTop := pdf.GetY() + 2

	// Compute max Y (sum of all datasets per month, for stacked bars).
	var maxY float64
	for i := range months {
		var sum float64
		for _, ds := range datasets {
			if i < len(ds.Data) {
				sum += ds.Data[i]
			}
		}
		if sum > maxY {
			maxY = sum
		}
	}
	yMax := yAxisScale(maxY)

	// Draw axes.
	pdf.SetDrawColor(180, 180, 180)
	pdf.SetLineWidth(0.3)
	pdf.Line(chartLeft, chartTop, chartLeft, chartTop+chartHeight)     // Y axis
	pdf.Line(chartLeft, chartTop+chartHeight, chartRight, chartTop+chartHeight) // X axis

	// Y-axis ticks and labels.
	labelX := chartLeft - 16.0
	labelW := 14.0
	if compact {
		labelX = chartLeft - 10.0
		labelW = 8.0
		pdf.SetFont("Helvetica", "", 6)
	} else {
		pdf.SetFont("Helvetica", "", 7)
	}
	pdf.SetTextColor(120, 120, 120)
	for _, frac := range []float64{0, 0.25, 0.5, 0.75, 1.0} {
		y := chartTop + chartHeight - (frac * chartHeight)
		val := frac * yMax
		label := fmt.Sprintf("%.0f", val)
		pdf.SetXY(labelX, y-3)
		pdf.CellFormat(labelW, 6, label, "", 0, "R", false, 0, "")
		pdf.Line(chartLeft-2, y, chartLeft, y)
	}

	// Draw bars.
	nMonths := len(months)
	if nMonths == 0 {
		return
	}
	barAreaWidth := chartWidth - barGroupGap*float64(nMonths-1)
	barW := barAreaWidth / float64(nMonths)
	if barW > 18 {
		barW = 18
	}
	barGap := barGroupGap

	for i := 0; i < nMonths; i++ {
		x := chartLeft + float64(i)*(barW+barGap)
		var stackY float64
		for _, ds := range datasets {
			var val float64
			if i < len(ds.Data) {
				val = ds.Data[i]
			}
			h := (val / yMax) * chartHeight
			if h < 0.5 && val > 0 {
				h = 0.5 // minimum visible bar
			}
			if h > 0 {
				pdf.SetFillColor(ds.Color[0], ds.Color[1], ds.Color[2])
				pdf.SetDrawColor(255, 255, 255)
				pdf.Rect(x, chartTop+chartHeight-stackY-h, barW, h, "FD")
			}
			stackY += h
		}
	}

	// X-axis labels (abbreviated months).
	pdf.SetFont("Helvetica", "", 7)
	pdf.SetTextColor(100, 100, 100)
	for i, m := range months {
		x := chartLeft + float64(i)*(barW+barGap) + barW/2
		label := m
		if len(m) > 3 {
			label = m[:3]
		}
		pdf.SetXY(x-12, chartTop+chartHeight+2)
		pdf.CellFormat(24, 5, label, "", 0, "C", false, 0, "")
	}

	// Legend.
	if compact && len(datasets) >= 3 {
		// Vertical stacked legend for compact charts with 3+ datasets.
		pdf.SetY(chartTop + chartHeight + 8)
		pdf.SetFont("Helvetica", "", 6)
		for _, ds := range datasets {
			pdf.SetFillColor(ds.Color[0], ds.Color[1], ds.Color[2])
			pdf.Rect(xLeft+2, pdf.GetY()+1, 3, 3, "F")
			pdf.SetXY(xLeft+7, pdf.GetY())
			pdf.SetTextColor(80, 80, 80)
			pdf.CellFormat(chartWidth-10, 4.5, ds.Label, "", 0, "L", false, 0, "")
			pdf.SetY(pdf.GetY() + 4.5)
		}
		pdf.SetTextColor(26, 26, 46)
		pdf.SetY(chartTop + chartHeight + 8 + float64(len(datasets))*4.5 + 4)
	} else {
		pdf.SetY(chartTop + chartHeight + 8)
		legendSpacing := 42.0
		if compact {
			pdf.SetFont("Helvetica", "", 6)
			legendSpacing = 28.0
		} else {
			pdf.SetFont("Helvetica", "", 8)
		}
		legendX := chartLeft
		for _, ds := range datasets {
			swatch := 5.0
			if compact {
				swatch = 3.0
			}
			pdf.SetFillColor(ds.Color[0], ds.Color[1], ds.Color[2])
			pdf.Rect(legendX, pdf.GetY()+1, swatch, swatch, "F")
			pdf.SetXY(legendX+swatch+2, pdf.GetY())
			pdf.SetTextColor(80, 80, 80)
			pdf.CellFormat(legendSpacing, 6, ds.Label, "", 0, "L", false, 0, "")
			legendX += legendSpacing
		}
		pdf.SetTextColor(26, 26, 46)
		pdf.SetY(chartTop + chartHeight + 16)
	}
}

func drawPDFLineChart(pdf *gofpdf.Fpdf, title string, months []string, datasets []pdfLineDataset, xLeft, chartWidth, chartHeight float64, compact bool) {
	pdf.Ln(4)
	chartRight := xLeft + chartWidth
	chartLeft := xLeft
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetTextColor(196, 30, 58)
	if compact {
		pdf.SetXY(xLeft+2, pdf.GetY())
		pdf.CellFormat(chartWidth-4, 7, safe(title), "", 1, "L", false, 0, "")
	} else {
		pdf.CellFormat(0, 7, safe(title), "", 1, "L", false, 0, "")
	}
	pdf.SetTextColor(26, 26, 46)

	chartTop := pdf.GetY() + 2

	// Compute max Y across all datasets.
	var maxY float64
	for _, ds := range datasets {
		for _, v := range ds.Data {
			if v > maxY {
				maxY = v
			}
		}
	}
	yMax := yAxisScale(maxY)

	// Draw axes.
	pdf.SetDrawColor(180, 180, 180)
	pdf.SetLineWidth(0.3)
	pdf.Line(chartLeft, chartTop, chartLeft, chartTop+chartHeight)       // Y axis
	pdf.Line(chartLeft, chartTop+chartHeight, chartRight, chartTop+chartHeight) // X axis

	// Y-axis ticks and labels.
	labelX := chartLeft - 16.0
	labelW := 14.0
	if compact {
		labelX = chartLeft - 10.0
		labelW = 8.0
		pdf.SetFont("Helvetica", "", 6)
	} else {
		pdf.SetFont("Helvetica", "", 7)
	}
	pdf.SetTextColor(120, 120, 120)
	for _, frac := range []float64{0, 0.25, 0.5, 0.75, 1.0} {
		y := chartTop + chartHeight - (frac * chartHeight)
		val := frac * yMax
		label := fmt.Sprintf("%.0f", val)
		pdf.SetXY(labelX, y-3)
		pdf.CellFormat(labelW, 6, label, "", 0, "R", false, 0, "")
		pdf.Line(chartLeft-2, y, chartLeft, y)
	}

	// Draw light horizontal grid lines.
	pdf.SetDrawColor(230, 230, 230)
	pdf.SetLineWidth(0.1)
	for _, frac := range []float64{0.25, 0.5, 0.75, 1.0} {
		y := chartTop + chartHeight - (frac * chartHeight)
		pdf.Line(chartLeft, y, chartRight, y)
	}

	// Draw lines and data points.
	nMonths := len(months)
	if nMonths == 0 {
		return
	}
	// Compute x positions for each month (center of each interval).
	intervalW := chartWidth / float64(nMonths)
	xPos := make([]float64, nMonths)
	for i := 0; i < nMonths; i++ {
		xPos[i] = chartLeft + intervalW*float64(i) + intervalW/2
	}

	for _, ds := range datasets {
		pdf.SetDrawColor(ds.Color[0], ds.Color[1], ds.Color[2])
		pdf.SetFillColor(ds.Color[0], ds.Color[1], ds.Color[2])
		pdf.SetLineWidth(0.8)
		var prevX, prevY float64
		first := true
		for i, v := range ds.Data {
			x := xPos[i]
			y := chartTop + chartHeight - (v/yMax)*chartHeight
			if !first {
				pdf.Line(prevX, prevY, x, y)
			}
			prevX, prevY = x, y
			first = false
		}
		// Draw data point markers.
		for i, v := range ds.Data {
			x := xPos[i]
			y := chartTop + chartHeight - (v/yMax)*chartHeight
			pdf.Circle(x, y, 1.5, "FD")
		}
	}

	// X-axis labels.
	pdf.SetFont("Helvetica", "", 7)
	pdf.SetTextColor(100, 100, 100)
	for i, m := range months {
		x := xPos[i]
		label := m
		if len(m) > 3 {
			label = m[:3]
		}
		pdf.SetXY(x-12, chartTop+chartHeight+2)
		pdf.CellFormat(24, 5, label, "", 0, "C", false, 0, "")
	}

	// Legend.
	if compact && len(datasets) >= 3 {
		// Vertical stacked legend for compact charts with 3+ datasets.
		pdf.SetY(chartTop + chartHeight + 8)
		pdf.SetFont("Helvetica", "", 6)
		for _, ds := range datasets {
			ly := pdf.GetY() + 2.5
			pdf.SetDrawColor(ds.Color[0], ds.Color[1], ds.Color[2])
			pdf.SetLineWidth(0.8)
			pdf.Line(xLeft+2, ly, xLeft+8, ly)
			pdf.SetFillColor(ds.Color[0], ds.Color[1], ds.Color[2])
			pdf.Circle(xLeft+5, ly, 1.0, "F")
			pdf.SetXY(xLeft+11, pdf.GetY())
			pdf.SetTextColor(80, 80, 80)
			pdf.CellFormat(chartWidth-14, 4.5, ds.Label, "", 0, "L", false, 0, "")
			pdf.SetY(pdf.GetY() + 4.5)
		}
		pdf.SetTextColor(26, 26, 46)
		pdf.SetY(chartTop + chartHeight + 8 + float64(len(datasets))*4.5 + 4)
	} else {
		pdf.SetY(chartTop + chartHeight + 8)
		legendSpacing := 50.0
		if compact {
			pdf.SetFont("Helvetica", "", 6)
			legendSpacing = 32.0
		} else {
			pdf.SetFont("Helvetica", "", 8)
		}
		legendX := chartLeft
		for _, ds := range datasets {
			lineLen := 10.0
			circleR := 1.5
			if compact {
				lineLen = 6.0
				circleR = 1.0
			}
			ly := pdf.GetY() + 4
			pdf.SetDrawColor(ds.Color[0], ds.Color[1], ds.Color[2])
			pdf.SetLineWidth(1.2)
			pdf.Line(legendX, ly, legendX+lineLen, ly)
			pdf.SetFillColor(ds.Color[0], ds.Color[1], ds.Color[2])
			pdf.Circle(legendX+lineLen/2, ly, circleR, "F")
			pdf.SetXY(legendX+lineLen+3, pdf.GetY())
			pdf.SetTextColor(80, 80, 80)
			pdf.CellFormat(legendSpacing, 6, ds.Label, "", 0, "L", false, 0, "")
			legendX += legendSpacing
		}
		pdf.SetTextColor(26, 26, 46)
		pdf.SetY(chartTop + chartHeight + 16)
	}
}
