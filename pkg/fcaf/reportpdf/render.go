// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package reportpdf

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/go-pdf/fpdf"
)

const (
	pageWidth       = 210.0
	pageHeight      = 297.0
	pageMargin      = 18.0
	pageBottomLimit = 276.0
	bodyWidth       = pageWidth - 2*pageMargin
)

type renderer struct {
	ctx              context.Context
	pdf              *fpdf.Fpdf
	document         Document
	registeredImages map[string]string
	figureNumbers    map[string]int
	section          string
	sectionColor     [3]int
	testProgress     string
	testCount        int
	evidenceCount    int
	onBackCover      bool
}

func Render(ctx context.Context, document Document) ([]byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(pageMargin, 25, pageMargin)
	pdf.SetAutoPageBreak(true, 21)
	pdf.SetCompression(true)
	pdf.AliasNbPages("{nb}")
	pdf.AddUTF8FontFromBytes("Inter", "", interRegular)
	pdf.AddUTF8FontFromBytes("Inter", "M", interMedium)
	pdf.AddUTF8FontFromBytes("Inter", "B", interBold)
	pdf.AddUTF8FontFromBytes("SourceCodePro", "", sourceCodeProRegular)
	pdf.AddUTF8FontFromBytes("SourceCodePro", "B", sourceCodeProSemibold)
	pdf.RegisterImageOptionsReader(
		"credimi-wordmark",
		fpdf.ImageOptions{ImageType: "PNG"},
		bytes.NewReader(credimiWordmark),
	)
	pdf.SetTitle("FCAF conformance assessment report", true)
	pdf.SetAuthor("Credimi", true)
	pdf.SetCreator("Credimi", true)
	pdf.SetProducer("Credimi FCAF report generator", true)
	pdf.SetSubject("FCAF wallet-solution relying-party conformance evidence", true)
	pdf.SetKeywords("FCAF, EUDI, EUDIW, conformance, evidence", true)

	r := &renderer{
		ctx:              ctx,
		pdf:              pdf,
		document:         document,
		registeredImages: map[string]string{},
		figureNumbers:    map[string]int{},
	}
	r.installHeaderAndFooter()
	if err := r.render(); err != nil {
		return nil, err
	}
	if pdf.Error() != nil {
		return nil, fmt.Errorf("render PDF: %w", pdf.Error())
	}
	var output bytes.Buffer
	if err := pdf.Output(&output); err != nil {
		return nil, fmt.Errorf("write PDF: %w", err)
	}
	return output.Bytes(), nil
}

func (r *renderer) installHeaderAndFooter() {
	r.pdf.SetHeaderFuncMode(func() {
		if r.pdf.PageNo() == 1 {
			return
		}
		r.pdf.ImageOptions(
			"credimi-wordmark",
			pageMargin,
			8,
			26,
			0,
			false,
			fpdf.ImageOptions{ImageType: "PNG"},
			0,
			"",
		)
		r.pdf.SetFont("Inter", "M", 8)
		r.pdf.SetTextColor(70, 68, 86)
		r.pdf.SetXY(50, 9)
		r.pdf.CellFormat(bodyWidth-32, 5, "FCAF conformance assessment", "", 0, "R", false, 0, "")
		r.pdf.SetDrawColor(225, 222, 237)
		r.pdf.Line(pageMargin, 16, pageWidth-pageMargin, 16)
	}, true)
	r.pdf.SetFooterFunc(func() {
		if r.onBackCover {
			return
		}
		r.pdf.SetY(-14)
		r.pdf.SetDrawColor(225, 222, 237)
		r.pdf.Line(pageMargin, r.pdf.GetY(), pageWidth-pageMargin, r.pdf.GetY())
		r.pdf.SetY(-11)
		r.pdf.SetFont("SourceCodePro", "", 7)
		r.pdf.SetTextColor(100, 98, 112)

		left := r.section
		hasSection := left != ""
		if !hasSection {
			left = strings.TrimSpace(r.document.Metadata.WorkflowID)
			if left == "" {
				left = "FCAF assessment"
			}
		}

		leftWidth := bodyWidth / 2
		if hasSection {
			r.pdf.SetFillColor(r.sectionColor[0], r.sectionColor[1], r.sectionColor[2])
			r.pdf.Rect(pageMargin, r.pdf.GetY()+0.5, 1.5, 4.5, "F")
			r.pdf.SetXY(pageMargin+5, r.pdf.GetY())
			leftWidth -= 5
		}
		r.pdf.CellFormat(leftWidth, 5, left, "", 0, "L", false, 0, "")

		right := fmt.Sprintf("Page %d of {nb}", r.pdf.PageNo())
		if r.testProgress != "" {
			right = r.testProgress + " · " + right
		}
		r.pdf.CellFormat(bodyWidth/2, 5, right, "", 0, "R", false, 0, "")
	})
}

func (r *renderer) render() error {
	if err := r.checkContext(); err != nil {
		return err
	}
	r.renderCover()
	r.renderExecutiveSummary()
	r.renderGroupSummary()
	for _, category := range r.document.Categories {
		if err := r.checkContext(); err != nil {
			return err
		}
		r.renderCategory(category)
	}
	r.renderEvidenceIndex()
	r.renderUnassignedImages()
	r.renderWarnings()
	r.renderBackCover()
	return r.pdf.Error()
}

func (r *renderer) renderCover() {
	r.pdf.AddPage()
	r.pdf.ImageOptions(
		"credimi-wordmark",
		pageMargin,
		24,
		65,
		0,
		false,
		fpdf.ImageOptions{ImageType: "PNG"},
		0,
		"",
	)
	r.pdf.SetFillColor(239, 236, 252)
	r.pdf.Rect(0, 58, pageWidth, 103, "F")
	r.pdf.SetXY(pageMargin, 76)
	r.pdf.SetFont("Inter", "B", 25)
	r.pdf.SetTextColor(41, 18, 120)
	r.pdf.MultiCell(bodyWidth, 11, "FCAF conformance\nassessment report", "", "L", false)
	r.pdf.Ln(5)
	r.pdf.SetFont("Inter", "", 11)
	r.pdf.SetTextColor(70, 68, 86)
	suite := firstNonEmpty(r.document.Report.Suite, "wallet_solution/relying_party")
	r.pdf.MultiCell(bodyWidth, 6, suite, "", "L", false)

	r.pdf.SetXY(pageMargin, 177)
	r.renderMetadataRow("Result", statusLabel(r.document.Report.Status), false)
	r.renderMetadataRow(
		"Tests",
		fmt.Sprintf(
			"%d over %d tests passed",
			r.document.Report.Summary.Pass,
			len(r.document.Report.ExecutedTests),
		),
		false,
	)
	r.renderMetadataRow("Pipeline", firstNonEmpty(r.document.Metadata.PipelineName, "—"), false)
	r.renderMetadataRow(
		"Organization",
		firstNonEmpty(r.document.Metadata.OrganizationName, "—"),
		false,
	)
	if r.document.Metadata.Runner.Name != "" {
		r.renderMetadataRow("Runner", r.document.Metadata.Runner.Name, false)
	}
	if r.document.Metadata.Runner.Type != "" {
		r.renderMetadataRow("Runner type", r.document.Metadata.Runner.Type, false)
	}
	if r.document.Metadata.Runner.Serial != "" {
		r.renderMetadataRow("Serial", r.document.Metadata.Runner.Serial, true)
	}
	r.renderMetadataRow("Workflow ID", firstNonEmpty(r.document.Metadata.WorkflowID, "—"), true)
	r.renderMetadataRow("Run ID", firstNonEmpty(r.document.Metadata.RunID, "—"), true)
	date := "—"
	if !r.document.Metadata.GeneratedAt.IsZero() {
		date = r.document.Metadata.GeneratedAt.Format("02/01/2006")
	}
	r.renderMetadataRow("Report date", date, true)
	r.pdf.Ln(10)
	r.pdf.SetFont("Inter", "M", 9)
	r.pdf.SetTextColor(41, 18, 120)
	r.pdf.CellFormat(bodyWidth, 6, "By Forkbomb BV", "", 0, "L", false, 0, "")
}

func (r *renderer) renderBackCover() {
	r.pdf.AddPage()
	r.onBackCover = true
	r.pdf.Bookmark("About Credimi", 0, -1)

	r.pdf.SetFillColor(239, 236, 252)
	r.pdf.Rect(0, 0, pageWidth, pageHeight, "F")

	r.pdf.ImageOptions(
		"credimi-wordmark",
		(pageWidth-90)/2,
		34,
		90,
		0,
		false,
		fpdf.ImageOptions{ImageType: "PNG"},
		0,
		"",
	)

	r.pdf.SetXY(pageMargin, 92)
	r.pdf.SetFont("Inter", "B", 19)
	r.pdf.SetTextColor(41, 18, 120)
	r.pdf.MultiCell(bodyWidth, 10, "Your trustworthy compliance checker", "", "C", false)
	r.pdf.SetFont("Inter", "", 12)
	r.pdf.SetTextColor(70, 68, 86)
	r.pdf.MultiCell(bodyWidth, 7, "for decentralized identity solutions", "", "C", false)

	r.pdf.Ln(16)
	r.pdf.SetFont("Inter", "", 10)
	r.pdf.SetTextColor(70, 68, 86)
	r.pdf.MultiCell(
		bodyWidth,
		6,
		"Credimi automates FCAF conformance assessment for EUDI wallets and relying parties.",
		"",
		"C",
		false,
	)

	r.pdf.Ln(20)
	r.pdf.SetFont("Inter", "B", 11)
	r.pdf.SetTextColor(41, 18, 120)
	r.pdf.MultiCell(bodyWidth, 7, "Get in touch", "", "C", false)
	r.pdf.Ln(3)

	contacts := []string{
		"credimi.io",
		"docs.credimi.io",
		"info@forkbomb.com",
	}
	for _, contact := range contacts {
		r.pdf.SetFont("SourceCodePro", "", 9.5)
		r.pdf.SetTextColor(70, 68, 86)
		r.pdf.MultiCell(bodyWidth, 6.5, contact, "", "C", false)
	}

	r.pdf.Ln(24)
	r.pdf.SetFont("Inter", "", 8)
	r.pdf.SetTextColor(100, 98, 112)
	r.pdf.MultiCell(bodyWidth, 5, "© Forkbomb BV. All rights reserved.", "", "C", false)
}

func (r *renderer) renderMetadataRow(label, value string, monospace bool) {
	r.pdf.SetFont("Inter", "M", 8)
	r.pdf.SetTextColor(100, 98, 112)
	r.pdf.CellFormat(36, 6, label, "", 0, "L", false, 0, "")
	if monospace {
		r.pdf.SetFont("SourceCodePro", "", 8)
	} else {
		r.pdf.SetFont("Inter", "", 9)
	}
	r.pdf.SetTextColor(40, 38, 48)
	r.pdf.MultiCell(bodyWidth-36, 6, value, "", "L", false)
}

func (r *renderer) renderExecutiveSummary() {
	r.pdf.AddPage()
	r.heading(1, "Executive summary")
	r.paragraph(
		"This report records the automated FCAF wallet-solution relying-party assessment performed by Credimi. It presents test definitions, validation outcomes, exact evidence references, and visual artifacts captured during the pipeline execution.",
	)
	r.paragraph(
		"This document is supporting conformance evidence. It does not replace certification, conformity assessment body review, or external security evaluation where those are required.",
	)

	r.heading(2, "Test results:")
	counts := []struct {
		label string
		value int
	}{
		{"Passed", r.document.Report.Summary.Pass},
		{"Failed", r.document.Report.Summary.Fail},
		{"Blocked", r.document.Report.Summary.Blocked},
		{"Inconclusive", r.document.Report.Summary.Inconclusive},
		{"Skipped", r.document.Report.Summary.Skipped},
		{"Not applicable", r.document.Report.Summary.NotApplicable},
		{"Error", r.document.Report.Summary.Error},
	}
	for _, count := range counts {
		r.pdf.SetFont("Inter", "M", 9)
		r.pdf.SetTextColor(statusColor(count.label))
		r.pdf.CellFormat(42, 7, count.label, "1", 0, "L", false, 0, "")
		r.pdf.SetFont("SourceCodePro", "B", 9)
		r.pdf.SetTextColor(40, 38, 48)
		r.pdf.CellFormat(20, 7, fmt.Sprintf("%d", count.value), "1", 0, "R", false, 0, "")
		r.pdf.Ln(7)
	}

	r.heading(2, "Evidence integrity:")
	r.renderMetadataRow(
		"JSON artifact",
		firstNonEmpty(r.document.Metadata.JSONFilename, "fcaf-assessment.json"),
		true,
	)
	r.renderMetadataRow("SHA-256", firstNonEmpty(r.document.JSONSHA256, "—"), true)
	r.paragraph(
		"Raw protocol evidence remains in the JSON artifact. Test sections reference that evidence by stable evidence key; raw values are not duplicated in this PDF.",
	)

	r.heading(2, "Status definitions:")
	definitions := []string{
		"Passed — every required assertion passed.",
		"Failed — one or more required assertions failed.",
		"Blocked — required evidence or execution capability was unavailable.",
		"Inconclusive — available evidence did not support a definitive result.",
		"Skipped or not applicable — test was outside the executed scope.",
		"Error — validation could not complete because of an execution or validator error.",
	}
	for _, definition := range definitions {
		r.bullet(definition)
	}
}

func (r *renderer) renderGroupSummary() {
	r.pdf.AddPage()
	r.pdf.Bookmark("Test groups", 0, -1)
	r.heading(1, "Test groups")
	r.paragraph(
		"Conformance checks are organised by FCAF area and subsection. Each area is colour-coded throughout this report.",
	)

	r.ensureSpace(10)
	r.pdf.SetFont("Inter", "M", 8.5)
	r.pdf.SetTextColor(100, 98, 112)
	r.pdf.CellFormat(bodyWidth, 5, "Colour key", "", 0, "L", false, 0, "")
	r.pdf.Ln(6)
	for _, category := range r.document.Categories {
		r.colorSwatch(category.Color, category.Name)
	}

	r.pdf.Ln(4)
	r.heading(2, "Areas under test:")
	for _, category := range r.document.Categories {
		r.ensureSpace(18)
		r.coloredHeading(1, category.Name, category.Color)
		passed, total := 0, 0
		for _, group := range category.Groups {
			for _, test := range group.Tests {
				total++
				if strings.HasPrefix(strings.ToLower(test.Execution.Status), "pass") {
					passed++
				}
			}
		}
		r.paragraph(fmt.Sprintf("%d over %d tests passed", passed, total))
		for _, group := range category.Groups {
			groupPassed := 0
			for _, test := range group.Tests {
				if strings.HasPrefix(strings.ToLower(test.Execution.Status), "pass") {
					groupPassed++
				}
			}
			r.summaryRow(group.Name, groupPassed, len(group.Tests), category.Color)
		}
	}
}

func (r *renderer) renderCategory(category Category) {
	r.pdf.AddPage()
	r.pdf.Bookmark(category.Name, 0, -1)
	r.section = category.Name
	r.sectionColor = category.Color
	r.testProgress = ""
	r.coloredHeading(1, category.Name, category.Color)
	passed, total := 0, 0
	for _, group := range category.Groups {
		for _, test := range group.Tests {
			total++
			if strings.HasPrefix(strings.ToLower(test.Execution.Status), "pass") {
				passed++
			}
		}
	}
	r.paragraph(fmt.Sprintf("%d over %d tests passed", passed, total))
	for _, group := range category.Groups {
		r.renderSubgroup(group, category.Name, category.Color)
	}
}

func (r *renderer) renderSubgroup(group TestGroup, categoryName string, color [3]int) {
	r.pdf.AddPage()
	r.section = categoryName + " · " + group.Name
	r.sectionColor = color
	r.testProgress = ""
	r.coloredHeading(2, group.Name, color)
	passed := 0
	for _, test := range group.Tests {
		if strings.HasPrefix(strings.ToLower(test.Execution.Status), "pass") {
			passed++
		}
	}
	r.paragraph(fmt.Sprintf("%d over %d tests passed", passed, len(group.Tests)))
	for _, test := range group.Tests {
		r.renderTest(test)
	}
}

func (r *renderer) renderTest(test TestEntry) {
	r.pdf.AddPage()
	r.testCount++
	r.testProgress = fmt.Sprintf("Test %d of %d", r.testCount, r.document.SelectedCount)
	r.pdf.Bookmark(test.Execution.TestID, 1, -1)

	title := firstNonEmpty(test.Execution.Title, test.Execution.TestID)
	r.pdf.SetFont("Inter", "B", 12)
	r.pdf.SetTextColor(41, 18, 120)
	titleY := r.pdf.GetY()
	r.pdf.SetXY(pageMargin, titleY)
	r.pdf.MultiCell(bodyWidth-32, 6, r.wrapToken(title, bodyWidth-32), "", "L", false)
	titleBottom := r.pdf.GetY()
	r.pdf.SetXY(pageWidth-pageMargin-27, titleY+1)
	r.statusCell(test.Execution.Status, 23)

	r.pdf.SetXY(pageMargin, max(titleBottom, titleY+8)+1)
	r.pdf.SetFont("SourceCodePro", "", 7.5)
	r.pdf.SetTextColor(100, 98, 112)
	r.pdf.MultiCell(bodyWidth, 4.5, r.wrapToken(test.Execution.TestID, bodyWidth), "", "L", false)
	r.pdf.Ln(4)

	if test.Execution.Outcome.Reason != "" {
		r.labelledText("Outcome", test.Execution.Outcome.Reason)
	}
	r.sourceSection("Objective", test.Source.Objective)
	r.sourceSection("Profile applicability", test.Source.Applicability)
	r.sourceSection("EUDI-wallet relevancy", test.Source.WalletRelevancy)
	r.sourceSection("Preconditions", test.Source.Preconditions)
	r.sourceSection("Test scenario", test.Source.Scenario)
	r.sourceSection("Expected results", test.Source.ExpectedResults)

	if len(test.Execution.Assertions) > 0 {
		r.heading(3, "Assertions:")
		for _, assertion := range test.Execution.Assertions {
			r.ensureSpace(14)
			assertionY := r.pdf.GetY()
			r.pdf.SetXY(pageMargin, assertionY)
			r.pdf.SetFont("SourceCodePro", "B", 7.5)
			r.pdf.SetTextColor(40, 38, 48)
			r.pdf.MultiCell(
				bodyWidth-30,
				4.5,
				r.wrapToken(assertion.ID, bodyWidth-30),
				"",
				"L",
				false,
			)
			assertionTextBottom := r.pdf.GetY()
			r.pdf.SetXY(pageWidth-pageMargin-27, assertionY)
			r.statusCell(assertion.Status, 23)
			r.pdf.SetXY(pageMargin, max(assertionTextBottom, assertionY+6.5))
			if assertion.Message != "" {
				r.pdf.SetFont("Inter", "", 8.5)
				r.pdf.SetTextColor(75, 72, 90)
				r.pdf.MultiCell(bodyWidth, 4.5, cleanMarkdown(assertion.Message), "", "L", false)
			}
			if len(assertion.EvidenceKeys) > 0 {
				r.pdf.SetX(pageMargin)
				r.pdf.SetFont("SourceCodePro", "", 7)
				r.pdf.SetTextColor(83, 62, 160)
				r.pdf.MultiCell(
					bodyWidth,
					4,
					"Evidence: "+strings.Join(assertion.EvidenceKeys, ", "),
					"",
					"L",
					false,
				)
			}
			r.pdf.Ln(1)
		}
	}

	if len(test.Definition.NormativeReferences) > 0 || test.Source.References != "" {
		r.heading(3, "Normative references:")
		for _, reference := range test.Definition.NormativeReferences {
			if reference.Title == "" && reference.Section == "" && reference.URL == "" {
				continue
			}
			label := firstNonEmpty(reference.Title, "Reference")
			if reference.Section != "" {
				label += ", section " + reference.Section
			}
			if reference.URL != "" {
				r.linkBullet(label, reference.URL)
			} else {
				r.bullet(label)
			}
		}
		if test.Source.References != "" && len(test.Definition.NormativeReferences) == 0 {
			r.bullet(cleanMarkdown(test.Source.References))
		}
	}

	if sourceURL := fcafSourceURL(test.Execution.TestID); sourceURL != "" {
		r.linkBullet("Open FCAF source test", sourceURL)
	}

	if len(test.Evidence) > 0 {
		r.heading(3, "Referenced evidence:")
		for _, evidence := range test.Evidence {
			parts := []string{evidence.Key}
			if evidence.Record.Type != "" {
				parts = append(parts, "type="+evidence.Record.Type)
			}
			if evidence.Record.SourceNode != "" {
				parts = append(parts, "source="+evidence.Record.SourceNode)
			}
			if evidence.Record.Path != "" {
				parts = append(parts, "path="+evidence.Record.Path)
			}
			r.bullet(strings.Join(parts, " · "))
		}
	}
	if len(test.Images) > 0 {
		r.heading(3, "Visual evidence:")
		r.renderImageGrid(test.Images)
	}
	r.pdf.Ln(5)
}

func (r *renderer) renderEvidenceIndex() {
	if len(r.document.Evidence) == 0 {
		return
	}
	r.pdf.AddPage()
	r.pdf.Bookmark("Evidence index", 0, -1)
	r.heading(1, "Evidence index")
	r.paragraph(
		"Raw values remain in the canonical JSON artifact. This index records each evidence key and its provenance.",
	)
	for _, evidence := range r.document.Evidence {
		r.ensureSpace(14)
		r.pdf.SetFont("SourceCodePro", "B", 7.5)
		r.pdf.SetTextColor(40, 38, 48)
		r.pdf.MultiCell(bodyWidth, 4.5, evidence.Key, "", "L", false)
		metadata := []string{}
		if evidence.Record.Type != "" {
			metadata = append(metadata, "type="+evidence.Record.Type)
		}
		if evidence.Record.SourceNode != "" {
			metadata = append(metadata, "source="+evidence.Record.SourceNode)
		}
		if evidence.Record.Path != "" {
			metadata = append(metadata, "path="+evidence.Record.Path)
		}
		if evidence.Record.From != "" {
			metadata = append(metadata, "from="+evidence.Record.From)
		}
		if !evidence.Referenced {
			metadata = append(metadata, "not referenced by an executed assertion")
		}
		r.pdf.SetFont("Inter", "", 8)
		r.pdf.SetTextColor(75, 72, 90)
		r.pdf.MultiCell(bodyWidth, 4.5, strings.Join(metadata, " · "), "", "L", false)
		r.pdf.Ln(1.5)
	}
}

func (r *renderer) renderUnassignedImages() {
	if len(r.document.Unassigned) == 0 {
		return
	}
	r.pdf.AddPage()
	r.pdf.Bookmark("Unassigned visual artifacts", 0, -1)
	r.heading(1, "Unassigned visual artifacts")
	r.paragraph(
		"These images were stored with the pipeline result but were not exactly associated with an executed assertion. They are included for completeness and are not represented as proof for a specific test.",
	)
	for _, image := range r.document.Unassigned {
		r.renderImage(image)
	}
}

func (r *renderer) renderWarnings() {
	if len(r.document.Warnings) == 0 {
		return
	}
	warnings := append([]string(nil), r.document.Warnings...)
	sort.Strings(warnings)
	r.pdf.AddPage()
	r.pdf.Bookmark("Report warnings", 0, -1)
	r.heading(1, "Report warnings")
	for _, warning := range warnings {
		r.bullet(warning)
	}
}

func (r *renderer) registerImage(image ImageAsset) (string, float64, float64, bool) {
	if len(image.Data) == 0 {
		return "", 0, 0, false
	}
	registeredName, found := r.registeredImages[image.Filename]
	if !found {
		registeredName = fmt.Sprintf("evidence-image-%d", len(r.registeredImages)+1)
		r.pdf.RegisterImageOptionsReader(
			registeredName,
			fpdf.ImageOptions{ImageType: "JPEG"},
			bytes.NewReader(image.Data),
		)
		if r.pdf.Error() != nil {
			return "", 0, 0, false
		}
		r.registeredImages[image.Filename] = registeredName
	}
	info := r.pdf.GetImageInfo(registeredName)
	if info == nil {
		return "", 0, 0, false
	}
	width, height := info.Extent()
	if width <= 0 || height <= 0 {
		return "", 0, 0, false
	}
	return registeredName, width, height, true
}

func (r *renderer) renderImage(image ImageAsset) {
	name, width, height, ok := r.registerImage(image)
	if !ok {
		return
	}
	maxWidth, maxHeight := 120.0, 92.0
	if width > maxWidth {
		height *= maxWidth / width
		width = maxWidth
	}
	if height > maxHeight {
		width *= maxHeight / height
		height = maxHeight
	}
	r.ensureSpace(height + 11)
	x := pageMargin + (bodyWidth-width)/2
	r.pdf.ImageOptions(
		name,
		x,
		r.pdf.GetY(),
		width,
		height,
		false,
		fpdf.ImageOptions{ImageType: "JPEG"},
		0,
		"",
	)
	r.pdf.SetY(r.pdf.GetY() + height + 1.5)
	r.pdf.SetFont("SourceCodePro", "", 7)
	r.pdf.SetTextColor(100, 98, 112)
	caption := image.Filename
	if image.EvidenceKey != "" {
		caption += " · " + image.EvidenceKey
	}
	r.pdf.MultiCell(bodyWidth, 4, caption, "", "C", false)
	r.pdf.Ln(2)
}

func (r *renderer) renderImageGrid(images []ImageAsset) {
	const columns = 4
	const gap = 4.0
	const captionHeight = 8.0
	const maxImageHeight = 68.0

	cellWidth := (bodyWidth - float64(columns-1)*gap) / columns
	rowY := 0.0

	type figure struct {
		number      int
		filename    string
		evidenceKey string
	}
	figures := make([]figure, 0, len(images))
	for i, image := range images {
		col := i % columns
		if col == 0 {
			r.ensureSpace(maxImageHeight + captionHeight + 6)
			rowY = r.pdf.GetY()
		}

		name, width, height, ok := r.registerImage(image)
		if !ok {
			continue
		}
		scale := min(cellWidth/width, maxImageHeight/height)
		drawW := width * scale
		drawH := height * scale
		figureNumber, seen := r.figureNumbers[image.Filename]
		if !seen {
			r.evidenceCount++
			figureNumber = r.evidenceCount
			r.figureNumbers[image.Filename] = figureNumber
		}
		figures = append(figures, figure{
			number:      figureNumber,
			filename:    image.Filename,
			evidenceKey: image.EvidenceKey,
		})

		x := pageMargin + float64(col)*(cellWidth+gap) + (cellWidth-drawW)/2
		r.pdf.ImageOptions(
			name,
			x,
			rowY,
			drawW,
			drawH,
			false,
			fpdf.ImageOptions{ImageType: "JPEG"},
			0,
			"",
		)

		r.pdf.SetXY(pageMargin+float64(col)*(cellWidth+gap), rowY+maxImageHeight+1.0)
		r.pdf.SetFont("SourceCodePro", "", 6.5)
		r.pdf.SetTextColor(100, 98, 112)
		r.pdf.CellFormat(
			cellWidth,
			3,
			fmt.Sprintf("Evidence %d", figureNumber),
			"",
			0,
			"C",
			false,
			0,
			"",
		)

		if col == columns-1 || i == len(images)-1 {
			r.pdf.SetY(rowY + maxImageHeight + captionHeight + 2)
		}
	}

	// Contextual table of pictures: full filenames, untruncated, right after the grid.
	for _, fig := range figures {
		r.ensureSpace(8)
		r.pdf.SetFont("Inter", "M", 8)
		r.pdf.SetTextColor(40, 38, 48)
		r.pdf.CellFormat(24, 4.5, fmt.Sprintf("Evidence %d:", fig.number), "", 0, "L", false, 0, "")
		name := fig.filename
		if fig.evidenceKey != "" {
			name += " · " + fig.evidenceKey
		}
		r.pdf.SetFont("SourceCodePro", "", 7)
		r.pdf.SetTextColor(75, 72, 90)
		r.pdf.MultiCell(bodyWidth-24, 4.5, r.wrapToken(name, bodyWidth-24), "", "L", false)
	}
	r.pdf.Ln(2)
}

func (r *renderer) sourceSection(label, content string) {
	content = cleanMarkdown(content)
	if content == "" {
		return
	}
	r.heading(3, label+":")
	r.paragraph(content)
}

func (r *renderer) labelledText(label, content string) {
	r.ensureSpace(12)
	r.pdf.SetFont("Inter", "M", 8.5)
	r.pdf.SetTextColor(40, 38, 48)
	r.pdf.CellFormat(22, 5, label+":", "", 0, "L", false, 0, "")
	r.pdf.SetFont("Inter", "", 8.5)
	r.pdf.SetTextColor(75, 72, 90)
	r.pdf.MultiCell(bodyWidth-22, 5, cleanMarkdown(content), "", "L", false)
}

func (r *renderer) heading(level int, text string) {
	sizes := map[int]float64{1: 16, 2: 11, 3: 8.5}
	spaces := map[int]float64{1: 9, 2: 7, 3: 5}
	r.ensureSpace(spaces[level] + 8)
	r.pdf.Ln(spaces[level] / 2)
	r.pdf.SetFont("Inter", "B", sizes[level])
	r.pdf.SetTextColor(41, 18, 120)
	r.pdf.MultiCell(bodyWidth, spaces[level], r.wrapToken(text, bodyWidth), "", "L", false)
	r.pdf.Ln(1)
}

func (r *renderer) coloredHeading(level int, text string, color [3]int) {
	sizes := map[int]float64{1: 15, 2: 10.5}
	spaces := map[int]float64{1: 8, 2: 6}
	r.ensureSpace(spaces[level] + 8)
	r.pdf.Ln(spaces[level] / 2)
	y := r.pdf.GetY()
	r.pdf.SetFillColor(color[0], color[1], color[2])
	r.pdf.Rect(pageMargin, y, 3.5, sizes[level]+3, "F")
	r.pdf.SetXY(pageMargin+11, y)
	r.pdf.SetFont("Inter", "B", sizes[level])
	r.pdf.SetTextColor(color[0], color[1], color[2])
	r.pdf.MultiCell(bodyWidth-11, spaces[level], r.wrapToken(text, bodyWidth-11), "", "L", false)
	r.pdf.SetY(max(r.pdf.GetY(), y+sizes[level]+3))
	r.pdf.Ln(1)
}

func (r *renderer) colorSwatch(color [3]int, label string) {
	r.ensureSpace(8)
	y := r.pdf.GetY()
	r.pdf.SetFillColor(color[0], color[1], color[2])
	r.pdf.Rect(pageMargin, y+1, 6, 6, "F")
	r.pdf.SetXY(pageMargin+10, y)
	r.pdf.SetFont("Inter", "", 9)
	r.pdf.SetTextColor(40, 38, 48)
	r.pdf.CellFormat(bodyWidth-10, 7, label, "", 0, "L", false, 0, "")
	r.pdf.Ln(8)
}

func (r *renderer) summaryRow(label string, passed, total int, color [3]int) {
	r.ensureSpace(7)
	y := r.pdf.GetY()
	r.pdf.SetFillColor(color[0], color[1], color[2])
	r.pdf.Rect(pageMargin+18, y+1.5, 2.5, 4.5, "F")
	r.pdf.SetXY(pageMargin+26, y)
	r.pdf.SetFont("Inter", "", 9)
	r.pdf.SetTextColor(40, 38, 48)
	r.pdf.CellFormat(bodyWidth-78, 6, label, "", 0, "L", false, 0, "")
	r.pdf.SetFont("SourceCodePro", "", 8.5)
	r.pdf.SetTextColor(75, 72, 90)
	r.pdf.CellFormat(52, 6, fmt.Sprintf("%d/%d passed", passed, total), "", 0, "R", false, 0, "")
	r.pdf.Ln(7)
}

func (r *renderer) paragraph(text string) {
	text = cleanMarkdown(text)
	if text == "" {
		return
	}
	r.pdf.SetFont("Inter", "", 9)
	r.pdf.SetTextColor(55, 53, 64)
	r.pdf.MultiCell(bodyWidth, 5, text, "", "L", false)
	r.pdf.Ln(2)
}

func (r *renderer) bullet(text string) {
	text = cleanMarkdown(text)
	if text == "" {
		return
	}
	r.ensureSpace(8)
	r.pdf.SetFont("Inter", "M", 9)
	r.pdf.SetTextColor(41, 18, 120)
	r.pdf.CellFormat(5, 5, "•", "", 0, "L", false, 0, "")
	r.pdf.SetFont("Inter", "", 8.5)
	r.pdf.SetTextColor(55, 53, 64)
	r.pdf.MultiCell(bodyWidth-5, 5, text, "", "L", false)
}

func (r *renderer) linkBullet(label, target string) {
	if strings.TrimSpace(target) == "" {
		r.bullet(label)
		return
	}
	r.ensureSpace(8)
	r.pdf.SetFont("Inter", "M", 9)
	r.pdf.SetTextColor(41, 18, 120)
	r.pdf.CellFormat(5, 5, "•", "", 0, "L", false, 0, "")
	r.pdf.SetFont("Inter", "", 8.5)
	r.pdf.SetTextColor(49, 38, 145)
	r.pdf.WriteLinkString(5, cleanMarkdown(label), target)
	r.pdf.Ln(5)
}

func (r *renderer) statusCell(status string, width float64) {
	label := statusLabel(status)
	red, green, blue := statusColor(label)
	r.pdf.SetFillColor(red, green, blue)
	r.pdf.SetTextColor(255, 255, 255)
	r.pdf.SetFont("Inter", "M", 7.5)
	r.pdf.CellFormat(width, 5.5, label, "", 0, "C", true, 0, "")
}

func (r *renderer) ensureSpace(height float64) {
	if r.pdf.GetY()+height > pageBottomLimit {
		r.pdf.AddPage()
	}
}

// wrapToken breaks long underscore-joined identifiers so they fit the
// available width instead of overflowing as a single unbreakable word.
func (r *renderer) wrapToken(text string, width float64) string {
	text = strings.TrimSpace(text)
	if text == "" || r.pdf.GetStringWidth(text) <= width {
		return text
	}
	segments := strings.SplitAfter(text, "_")
	var output strings.Builder
	line := ""
	for _, segment := range segments {
		candidate := line + segment
		if line != "" && r.pdf.GetStringWidth(candidate) > width {
			output.WriteString(line)
			output.WriteByte('\n')
			line = segment
		} else {
			line = candidate
		}
	}
	if line != "" {
		output.WriteString(line)
	}
	return output.String()
}

func (r *renderer) checkContext() error {
	select {
	case <-r.ctx.Done():
		return fmt.Errorf("generate FCAF PDF: %w", r.ctx.Err())
	default:
		return nil
	}
}

func statusLabel(status string) string {
	status = strings.TrimSpace(strings.ToLower(status))
	switch {
	case strings.HasPrefix(status, "pass"):
		return "Passed"
	case strings.HasPrefix(status, "fail"):
		return "Failed"
	case status == "blocked":
		return "Blocked"
	case status == "inconclusive":
		return "Inconclusive"
	case status == "skipped":
		return "Skipped"
	case status == "not_applicable", status == "not applicable":
		return "Not applicable"
	case status == "error":
		return "Error"
	case status == "":
		return "Unknown"
	default:
		return strings.ToUpper(status[:1]) + status[1:]
	}
}

func statusColor(status string) (int, int, int) {
	switch strings.ToLower(status) {
	case "passed":
		return 35, 126, 74
	case "failed", "error":
		return 181, 48, 48
	case "blocked", "inconclusive":
		return 164, 103, 16
	case "skipped", "not applicable":
		return 99, 99, 109
	default:
		return 74, 55, 168
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func cleanMarkdown(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "`", "")
	linkPattern := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	value = linkPattern.ReplaceAllString(value, "$1 ($2)")
	lines := strings.Split(value, "\n")
	for index, line := range lines {
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "- ")
		line = strings.TrimPrefix(line, "* ")
		lines[index] = line
	}
	value = strings.Join(lines, "\n")
	value = strings.Map(func(character rune) rune {
		if character == '\t' {
			return ' '
		}
		if unicode.IsControl(character) && character != '\n' {
			return -1
		}
		return character
	}, value)
	return strings.TrimSpace(value)
}

func fcafSourceURL(testID string) string {
	if strings.TrimSpace(testID) == "" {
		return ""
	}
	anchor := strings.ToLower(testID)
	var builder strings.Builder
	underscore := false
	for _, character := range anchor {
		if unicode.IsLetter(character) || unicode.IsDigit(character) {
			builder.WriteRune(character)
			underscore = false
		} else if !underscore {
			builder.WriteByte('_')
			underscore = true
		}
	}
	anchor = strings.Trim(builder.String(), "_")
	return "https://conformance.eudi.dev/latest-draft/fcaf/suts/wallet_solution/relying_party/ws_rp/#" + anchor
}
