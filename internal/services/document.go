package services

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jung-kurt/gofpdf"

	"aist-bot/internal/i18n"
	"aist-bot/internal/models"
	"aist-bot/internal/utils"
)

// Page geometry, in millimetres on A4.
const (
	pageWidth    = 210.0
	pageHeight   = 297.0
	marginLeft   = 16.0
	marginRight  = 16.0
	marginTop    = 18.0
	marginBottom = 20.0
	contentWidth = pageWidth - marginLeft - marginRight
	labelWidth   = 46.0

	// Capped so two landscape photos still share a page, which keeps a
	// three-photo complaint to two pages instead of four.
	maxPhotoHeight = 108.0
)

type rgb struct{ r, g, b int }

// The palette is deliberately small: one accent per appeal type, and a
// neutral scale shared by every document.
var (
	colorInk       = rgb{31, 41, 55}
	colorMuted     = rgb{107, 114, 128}
	colorLine      = rgb{226, 232, 240}
	colorRowFill   = rgb{248, 250, 252}
	colorWhite     = rgb{255, 255, 255}
	colorComplaint = rgb{185, 60, 45}
	colorProposal  = rgb{29, 78, 216}
	colorGratitude = rgb{21, 128, 61}
)

func accentFor(t models.SubmissionType) rgb {
	switch t {
	case models.TypeProposal:
		return colorProposal
	case models.TypeGratitude:
		return colorGratitude
	default:
		return colorComplaint
	}
}

// DocumentService renders appeals into PDF documents.
type DocumentService struct {
	bot     *tgbotapi.BotAPI
	orgName string
	client  *http.Client

	// fetchPhoto is swapped out in tests so the layout can be exercised
	// without a live Telegram connection.
	fetchPhoto func(fileID string) (data []byte, imageType string, err error)
}

func NewDocumentService(bot *tgbotapi.BotAPI, orgName string) *DocumentService {
	s := &DocumentService{
		bot:     bot,
		orgName: orgName,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
	s.fetchPhoto = s.downloadPhoto

	return s
}

// AppealDocument is everything the PDF needs to show.
type AppealDocument struct {
	Type           models.SubmissionType
	Reference      string
	DepartmentName string
	ClassName      string
	ChildName      string
	ParentPhone    string
	ParentUsername string
	Text           string
	Photos         []models.PhotoData
	Language       i18n.Language
	CreatedAt      time.Time
}

// Generate renders the appeal and returns the PDF bytes. The whole document
// is built in memory: nothing is written to disk, so there is no temporary
// file to clean up and no personal data left behind on the server.
func (s *DocumentService) Generate(doc AppealDocument) ([]byte, error) {
	lang := doc.Language
	accent := accentFor(doc.Type)

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(marginLeft, marginTop, marginRight)
	pdf.SetAutoPageBreak(true, marginBottom)

	pdf.AddUTF8FontFromBytes("DejaVu", "", fontRegular)
	pdf.AddUTF8FontFromBytes("DejaVu", "B", fontBold)
	pdf.SetFont("DejaVu", "", 10)

	pdf.AliasNbPages("{nb}")
	s.setHeader(pdf, accent)
	s.setFooter(pdf, doc, lang)

	pdf.AddPage()

	s.drawTitleBlock(pdf, doc, accent, lang)
	s.drawDetails(pdf, doc, accent, lang)
	s.drawMessage(pdf, doc, accent, lang)

	if len(doc.Photos) > 0 {
		s.drawPhotos(pdf, doc, accent, lang)
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("failed to render pdf: %w", err)
	}

	if buf.Len() == 0 {
		return nil, fmt.Errorf("rendered pdf is empty")
	}

	return buf.Bytes(), nil
}

// setHeader draws the thin accent rule and organisation name repeated on
// every page.
func (s *DocumentService) setHeader(pdf *gofpdf.Fpdf, accent rgb) {
	pdf.SetHeaderFunc(func() {
		pdf.SetFillColor(accent.r, accent.g, accent.b)
		pdf.Rect(0, 0, pageWidth, 3.5, "F")

		pdf.SetY(8)
		pdf.SetX(marginLeft)
		pdf.SetFont("DejaVu", "B", 9)
		pdf.SetTextColor(colorMuted.r, colorMuted.g, colorMuted.b)
		pdf.CellFormat(contentWidth, 5, utils.PDFSafe(s.orgName), "", 0, "L", false, 0, "")

		pdf.SetY(marginTop)
	})
}

// setFooter draws the generation note and page numbering.
func (s *DocumentService) setFooter(pdf *gofpdf.Fpdf, doc AppealDocument, lang i18n.Language) {
	pdf.SetFooterFunc(func() {
		pdf.SetY(-16)

		pdf.SetDrawColor(colorLine.r, colorLine.g, colorLine.b)
		pdf.SetLineWidth(0.2)
		pdf.Line(marginLeft, pdf.GetY(), pageWidth-marginRight, pdf.GetY())

		pdf.Ln(2)
		pdf.SetFont("DejaVu", "", 8)
		pdf.SetTextColor(colorMuted.r, colorMuted.g, colorMuted.b)

		pdf.CellFormat(contentWidth*0.75, 5, utils.PDFSafe(i18n.Get(i18n.PDFFooter, lang)), "", 0, "L", false, 0, "")
		pdf.CellFormat(contentWidth*0.25, 5,
			utils.PDFSafe(fmt.Sprintf(i18n.Get(i18n.PDFPage, lang), pdf.PageNo()))+" / {nb}",
			"", 0, "R", false, 0, "")
	})
}

// drawTitleBlock draws the appeal type as a filled badge, with the document
// number and timestamp beside it.
func (s *DocumentService) drawTitleBlock(pdf *gofpdf.Fpdf, doc AppealDocument, accent rgb, lang i18n.Language) {
	title := utils.PDFSafe(i18n.Get(i18n.PDFTitleKey(string(doc.Type)), lang))

	pdf.SetFont("DejaVu", "B", 20)
	badgeWidth := pdf.GetStringWidth(title) + 16
	badgeHeight := 13.0
	y := pdf.GetY()

	pdf.SetFillColor(accent.r, accent.g, accent.b)
	pdf.RoundedRect(marginLeft, y, badgeWidth, badgeHeight, 2, "1234", "F")

	pdf.SetTextColor(colorWhite.r, colorWhite.g, colorWhite.b)
	pdf.SetXY(marginLeft, y)
	pdf.CellFormat(badgeWidth, badgeHeight, title, "", 0, "C", false, 0, "")

	// Document number and timestamp, right aligned against the badge.
	pdf.SetFont("DejaVu", "B", 10)
	pdf.SetTextColor(colorInk.r, colorInk.g, colorInk.b)
	pdf.SetXY(marginLeft+badgeWidth, y)
	pdf.CellFormat(contentWidth-badgeWidth, 7, utils.PDFSafe(doc.Reference), "", 0, "R", false, 0, "")

	pdf.SetFont("DejaVu", "", 9)
	pdf.SetTextColor(colorMuted.r, colorMuted.g, colorMuted.b)
	pdf.SetXY(marginLeft+badgeWidth, y+6.5)
	pdf.CellFormat(contentWidth-badgeWidth, 6, utils.FormatDateTime(doc.CreatedAt), "", 0, "R", false, 0, "")

	pdf.SetY(y + badgeHeight + 9)
}

// sectionHeading draws an uppercase label with a short accent underline.
func (s *DocumentService) sectionHeading(pdf *gofpdf.Fpdf, text string, accent rgb) {
	pdf.SetFont("DejaVu", "B", 10)
	pdf.SetTextColor(colorInk.r, colorInk.g, colorInk.b)
	pdf.SetX(marginLeft)
	pdf.CellFormat(contentWidth, 6, utils.PDFSafe(text), "", 1, "L", false, 0, "")

	y := pdf.GetY() + 0.5
	pdf.SetFillColor(accent.r, accent.g, accent.b)
	pdf.Rect(marginLeft, y, 18, 0.9, "F")

	pdf.SetY(y + 4)
}

// drawDetails renders the label/value table: who wrote in, about which
// department and class, and how to reach them.
func (s *DocumentService) drawDetails(pdf *gofpdf.Fpdf, doc AppealDocument, accent rgb, lang i18n.Language) {
	s.sectionHeading(pdf, i18n.Get(i18n.PDFSectionDetails, lang), accent)

	username := doc.ParentUsername
	if username == "" {
		username = i18n.Get(i18n.LblNone, lang)
	} else if !strings.HasPrefix(username, "@") {
		username = "@" + username
	}

	rows := [][2]string{
		{i18n.Get(i18n.LblType, lang), i18n.Get(i18n.TypeKey(string(doc.Type)), lang)},
		{i18n.Get(i18n.LblDepartment, lang), doc.DepartmentName},
		{i18n.Get(i18n.LblClass, lang), doc.ClassName},
		{i18n.Get(i18n.LblChild, lang), doc.ChildName},
		{i18n.Get(i18n.LblPhone, lang), utils.FormatPhone(doc.ParentPhone)},
		{i18n.Get(i18n.LblUsername, lang), username},
		{i18n.Get(i18n.LblDate, lang), utils.FormatDateTime(doc.CreatedAt)},
	}

	pdf.SetDrawColor(colorLine.r, colorLine.g, colorLine.b)
	pdf.SetLineWidth(0.15)

	for i, row := range rows {
		label := utils.PDFSafe(row[0])
		value := utils.PDFSafe(row[1])

		// A long department name may need two lines; measure before drawing
		// so the label cell and the value cell stay the same height.
		pdf.SetFont("DejaVu", "", 10)
		lines := pdf.SplitLines([]byte(value), contentWidth-labelWidth-6)
		if len(lines) == 0 {
			lines = [][]byte{[]byte("")}
		}
		rowHeight := float64(len(lines))*5.2 + 3.4

		y := pdf.GetY()
		if y+rowHeight > pageHeight-marginBottom {
			pdf.AddPage()
			y = pdf.GetY()
		}

		if i%2 == 0 {
			pdf.SetFillColor(colorRowFill.r, colorRowFill.g, colorRowFill.b)
			pdf.Rect(marginLeft, y, contentWidth, rowHeight, "F")
		}

		pdf.SetFont("DejaVu", "", 9)
		pdf.SetTextColor(colorMuted.r, colorMuted.g, colorMuted.b)
		pdf.SetXY(marginLeft+3, y+1.7)
		pdf.CellFormat(labelWidth, 5.2, label, "", 0, "L", false, 0, "")

		pdf.SetFont("DejaVu", "B", 10)
		pdf.SetTextColor(colorInk.r, colorInk.g, colorInk.b)
		pdf.SetXY(marginLeft+labelWidth+3, y+1.7)
		pdf.MultiCell(contentWidth-labelWidth-6, 5.2, value, "", "L", false)

		pdf.SetY(y + rowHeight)
		pdf.Line(marginLeft, pdf.GetY(), pageWidth-marginRight, pdf.GetY())
	}

	pdf.SetY(pdf.GetY() + 8)
}

// drawMessage renders the parent's own words inside a tinted panel with a
// coloured edge, so it reads as a quotation rather than as form data.
func (s *DocumentService) drawMessage(pdf *gofpdf.Fpdf, doc AppealDocument, accent rgb, lang i18n.Language) {
	s.sectionHeading(pdf, i18n.Get(i18n.PDFSectionText, lang), accent)

	text := utils.PDFSafe(doc.Text)
	if text == "" {
		text = "—"
	}

	const (
		padding    = 5.0
		lineHeight = 5.6
		edgeWidth  = 1.6
	)

	pdf.SetFont("DejaVu", "", 10.5)
	innerWidth := contentWidth - edgeWidth - padding*2

	// Paragraphs are measured individually so blank lines survive.
	paragraphs := strings.Split(text, "\n")
	totalLines := 0
	for _, paragraph := range paragraphs {
		if strings.TrimSpace(paragraph) == "" {
			totalLines++
			continue
		}
		lines := pdf.SplitLines([]byte(paragraph), innerWidth)
		if len(lines) == 0 {
			totalLines++
			continue
		}
		totalLines += len(lines)
	}

	boxHeight := float64(totalLines)*lineHeight + padding*2
	y := pdf.GetY()

	// A panel taller than the remaining space starts on a fresh page, but a
	// very long message simply flows: the tint is then drawn per page.
	available := pageHeight - marginBottom - y
	if boxHeight > available && boxHeight <= pageHeight-marginTop-marginBottom {
		pdf.AddPage()
		y = pdf.GetY()
		available = pageHeight - marginBottom - y
	}

	drawnHeight := boxHeight
	if drawnHeight > available {
		drawnHeight = available
	}

	pdf.SetFillColor(colorRowFill.r, colorRowFill.g, colorRowFill.b)
	pdf.Rect(marginLeft, y, contentWidth, drawnHeight, "F")
	pdf.SetFillColor(accent.r, accent.g, accent.b)
	pdf.Rect(marginLeft, y, edgeWidth, drawnHeight, "F")

	pdf.SetTextColor(colorInk.r, colorInk.g, colorInk.b)
	pdf.SetXY(marginLeft+edgeWidth+padding, y+padding)
	pdf.MultiCell(innerWidth, lineHeight, text, "", "L", false)

	pdf.SetY(y + drawnHeight + 9)
}

// drawPhotos appends the attached photos, one block each, framed and
// captioned. Every photo is fetched and measured before anything is drawn,
// so the section heading is never stranded at the foot of a page with its
// photos overleaf. A photo that cannot be downloaded is skipped rather than
// failing the whole document.
func (s *DocumentService) drawPhotos(pdf *gofpdf.Fpdf, doc AppealDocument, accent rgb, lang i18n.Language) {
	type preparedPhoto struct {
		name          string
		imageType     string
		width, height float64
	}

	prepared := make([]preparedPhoto, 0, len(doc.Photos))

	for i, photo := range doc.Photos {
		data, imageType, err := s.fetchPhoto(photo.FileID)
		if err != nil {
			continue
		}

		name := fmt.Sprintf("photo_%d", i)
		info := pdf.RegisterImageOptionsReader(name, gofpdf.ImageOptions{ImageType: imageType}, bytes.NewReader(data))
		if info == nil || pdf.Err() {
			// Clear the error so the remaining photos still render.
			pdf.ClearError()
			continue
		}

		width, height := scaleToFit(info.Width(), info.Height(), contentWidth, maxPhotoHeight)
		prepared = append(prepared, preparedPhoto{name: name, imageType: imageType, width: width, height: height})
	}

	if len(prepared) == 0 {
		return
	}

	const headingHeight = 12.0

	if pdf.GetY()+headingHeight+photoBlockHeight(prepared[0].height) > pageHeight-marginBottom {
		pdf.AddPage()
	}

	s.sectionHeading(pdf, i18n.Get(i18n.PDFSectionPhotos, lang), accent)

	total := len(prepared)

	for i, photo := range prepared {
		if pdf.GetY()+photoBlockHeight(photo.height) > pageHeight-marginBottom {
			pdf.AddPage()
		}

		pdf.SetFont("DejaVu", "", 9)
		pdf.SetTextColor(colorMuted.r, colorMuted.g, colorMuted.b)
		pdf.SetX(marginLeft)
		pdf.CellFormat(contentWidth, 5, utils.PDFSafe(fmt.Sprintf(i18n.Get(i18n.PDFPhotoNo, lang), i+1, total)), "", 1, "L", false, 0, "")

		x := marginLeft + (contentWidth-photo.width)/2
		y := pdf.GetY() + 1

		pdf.ImageOptions(photo.name, x, y, photo.width, photo.height, false,
			gofpdf.ImageOptions{ImageType: photo.imageType}, 0, "")

		pdf.SetDrawColor(colorLine.r, colorLine.g, colorLine.b)
		pdf.SetLineWidth(0.3)
		pdf.Rect(x, y, photo.width, photo.height, "D")

		pdf.SetY(y + photo.height + 7)
	}
}

// photoBlockHeight is the caption, the photo and the gap that follows it.
func photoBlockHeight(imageHeight float64) float64 {
	return imageHeight + 13
}

// scaleToFit shrinks an image to fit the given box while keeping its ratio.
// Images smaller than the box are never enlarged past it.
func scaleToFit(srcWidth, srcHeight, maxWidth, maxHeight float64) (width, height float64) {
	if srcWidth <= 0 || srcHeight <= 0 {
		return maxWidth, maxWidth * 0.75
	}

	width = maxWidth
	height = srcHeight / srcWidth * maxWidth

	if height > maxHeight {
		height = maxHeight
		width = srcWidth / srcHeight * maxHeight
	}

	return width, height
}

// downloadPhoto fetches a Telegram photo into memory and reports its format.
func (s *DocumentService) downloadPhoto(fileID string) ([]byte, string, error) {
	file, err := s.bot.GetFile(tgbotapi.FileConfig{FileID: fileID})
	if err != nil {
		return nil, "", fmt.Errorf("failed to locate photo: %w", err)
	}

	resp, err := s.client.Get(file.Link(s.bot.Token))
	if err != nil {
		return nil, "", fmt.Errorf("failed to download photo: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("photo download returned %s", resp.Status)
	}

	// 20 MB is far above Telegram's photo limit and bounds a bad response.
	data, err := io.ReadAll(io.LimitReader(resp.Body, 20<<20))
	if err != nil {
		return nil, "", fmt.Errorf("failed to read photo: %w", err)
	}

	return data, detectImageType(data), nil
}

// detectImageType maps the magic bytes to the format names gofpdf expects.
func detectImageType(data []byte) string {
	switch {
	case bytes.HasPrefix(data, []byte{0x89, 'P', 'N', 'G'}):
		return "PNG"
	case bytes.HasPrefix(data, []byte{'G', 'I', 'F'}):
		return "GIF"
	default:
		return "JPG"
	}
}

// SetPhotoFetcher replaces how attachments are downloaded. Production code
// never calls it; it exists so the bot can be driven end to end in tests
// without reaching out to Telegram for image data.
func (s *DocumentService) SetPhotoFetcher(fetch func(fileID string) ([]byte, string, error)) {
	s.fetchPhoto = fetch
}
