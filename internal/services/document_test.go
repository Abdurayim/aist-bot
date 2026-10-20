package services

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"
	"time"

	"aist-bot/internal/i18n"
	"aist-bot/internal/models"
)

// sampleJPEG produces a small gradient image standing in for a parent's photo.
func sampleJPEG(width, height int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8(x * 255 / width),
				G: uint8(y * 255 / height),
				B: 160,
				A: 255,
			})
		}
	}

	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, img, nil)

	return buf.Bytes()
}

func newTestService() *DocumentService {
	s := &DocumentService{orgName: "AIST School"}
	s.fetchPhoto = func(string) ([]byte, string, error) {
		return sampleJPEG(1200, 800), "JPG", nil
	}
	return s
}

func testDocument(t models.SubmissionType, lang i18n.Language, photos int) AppealDocument {
	list := make([]models.PhotoData, photos)
	for i := range list {
		list[i] = models.PhotoData{FileID: "file"}
	}

	return AppealDocument{
		Type:           t,
		Reference:      "AIST-2026-000042",
		DepartmentName: "Security va xavfsizlik (Face ID, turniket)",
		ClassName:      "9-B",
		ChildName:      "Aliyev Sardor",
		ParentPhone:    "+998901234567",
		ParentUsername: "sardor_ota",
		Text: "Farzandim turniketdan o'ta olmayapti, Face ID uni tanimayapti.\n\n" +
			"Iltimos, tizimni qayta sozlab bering. Bu holat uchinchi kun davom etmoqda va " +
			"bola darsga kechikmoqda.",
		Photos:    list,
		Language:  lang,
		CreatedAt: time.Date(2026, 9, 15, 14, 30, 0, 0, time.UTC),
	}
}

func TestGenerateProducesValidPDF(t *testing.T) {
	svc := newTestService()

	data, err := svc.Generate(testDocument(models.TypeComplaint, i18n.Uzbek, 0))
	if err != nil {
		t.Fatalf("Generate returned an error: %v", err)
	}

	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		t.Fatalf("output is not a PDF, first bytes: %q", data[:min(8, len(data))])
	}

	if len(data) < 2000 {
		t.Fatalf("PDF looks empty: %d bytes", len(data))
	}
}

func TestGenerateAcrossTypesAndLanguages(t *testing.T) {
	svc := newTestService()

	types := []models.SubmissionType{models.TypeComplaint, models.TypeProposal, models.TypeGratitude}
	langs := []i18n.Language{i18n.Uzbek, i18n.Russian, i18n.English}

	for _, submissionType := range types {
		for _, lang := range langs {
			data, err := svc.Generate(testDocument(submissionType, lang, 2))
			if err != nil {
				t.Fatalf("%s/%s failed: %v", submissionType, lang, err)
			}
			if len(data) < 2000 {
				t.Fatalf("%s/%s produced %d bytes", submissionType, lang, len(data))
			}
		}
	}
}

// TestGenerateLongTextAndManyPhotos exercises the page-break paths: a body
// longer than one page, plus the maximum number of attachments.
func TestGenerateLongTextAndManyPhotos(t *testing.T) {
	svc := newTestService()

	doc := testDocument(models.TypeProposal, i18n.Russian, models.MaxPhotos)
	doc.Text = ""
	for i := 0; i < 60; i++ {
		doc.Text += "Это длинный текст обращения, проверяющий перенос страниц и отрисовку панели. "
	}

	data, err := svc.Generate(doc)
	if err != nil {
		t.Fatalf("Generate returned an error: %v", err)
	}

	if len(data) < 5000 {
		t.Fatalf("expected a multi-page document, got %d bytes", len(data))
	}
}

// TestGenerateSurvivesUnavailablePhoto makes sure one broken attachment does
// not cost the parent their whole document.
func TestGenerateSurvivesUnavailablePhoto(t *testing.T) {
	svc := newTestService()
	svc.fetchPhoto = func(string) ([]byte, string, error) {
		return nil, "", os.ErrNotExist
	}

	data, err := svc.Generate(testDocument(models.TypeGratitude, i18n.English, 3))
	if err != nil {
		t.Fatalf("Generate returned an error: %v", err)
	}

	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		t.Fatal("output is not a PDF")
	}
}

// TestWriteSamples writes one PDF per type into the scratch directory when
// AIST_PDF_SAMPLES is set, so the layout can be eyeballed.
func TestWriteSamples(t *testing.T) {
	outDir := os.Getenv("AIST_PDF_SAMPLES")
	if outDir == "" {
		t.Skip("set AIST_PDF_SAMPLES to write sample documents")
	}

	svc := newTestService()

	cases := []struct {
		name string
		doc  AppealDocument
	}{
		{"complaint_uz", testDocument(models.TypeComplaint, i18n.Uzbek, 2)},
		{"proposal_ru", testDocument(models.TypeProposal, i18n.Russian, 1)},
		{"gratitude_en", testDocument(models.TypeGratitude, i18n.English, 0)},
	}

	for _, c := range cases {
		data, err := svc.Generate(c.doc)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}

		path := filepath.Join(outDir, c.name+".pdf")
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatalf("failed to write %s: %v", path, err)
		}

		t.Logf("wrote %s (%d bytes)", path, len(data))
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
