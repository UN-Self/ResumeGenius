package parsing

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewPDFParser(t *testing.T) {
	parser := NewPDFParser()
	if parser == nil {
		t.Fatal("expected parser instance")
	}
}

func TestExtractTextFromPDFFixture(t *testing.T) {
	path := fixturePath(t, "sample_resume.pdf")

	text, err := ExtractTextFromPDF(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text == "" {
		t.Fatal("expected non-empty text")
	}
	if !strings.Contains(text, "张三") {
		t.Fatalf("expected extracted text to contain 张三, got: %s", text)
	}
	if !strings.Contains(text, "工作经历") {
		t.Fatalf("expected extracted text to contain 工作经历, got: %s", text)
	}
}

func TestPDFParserParseReturnsParsedContent(t *testing.T) {
	path := fixturePath(t, "sample_resume.pdf")
	parser := NewPDFParser()

	parsed, err := parser.Parse(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if parsed == nil {
		t.Fatal("expected parsed content")
	}
	if parsed.Text == "" {
		t.Fatal("expected parsed text")
	}
}

func TestPDFParserParseExtractsEmbeddedImages(t *testing.T) {
	path := writePDFWithImageFixture(t, embeddedImagePDFOptions{
		FilterName: "FlateDecode",
		ImageData:  compressZlibBytes(t, []byte{255, 0, 0}),
	})
	parser := NewPDFParser()

	parsed, err := parser.Parse(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(parsed.Images) != 1 {
		t.Fatalf("expected 1 embedded image, got %d", len(parsed.Images))
	}
}

func TestExtractImagesFromPDFReturnsEmbeddedImages(t *testing.T) {
	path := writePDFWithImageFixture(t, embeddedImagePDFOptions{
		FilterName: "FlateDecode",
		ImageData:  compressZlibBytes(t, []byte{255, 0, 0}),
	})

	images, err := ExtractImagesFromPDF(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(images) != 1 {
		t.Fatalf("expected 1 extracted image, got %d", len(images))
	}
	if !strings.Contains(images[0].Description, "page 1") {
		t.Fatalf("expected description to mention page 1, got %q", images[0].Description)
	}
	if !strings.Contains(images[0].Description, "Im0") {
		t.Fatalf("expected description to mention xobject name, got %q", images[0].Description)
	}

	pngBytes, err := base64.StdEncoding.DecodeString(images[0].DataBase64)
	if err != nil {
		t.Fatalf("decode base64 png: %v", err)
	}
	img, err := png.Decode(bytes.NewReader(pngBytes))
	if err != nil {
		t.Fatalf("decode png: %v", err)
	}
	bounds := img.Bounds()
	if bounds.Dx() != 1 || bounds.Dy() != 1 {
		t.Fatalf("expected 1x1 image, got %dx%d", bounds.Dx(), bounds.Dy())
	}

	r, g, b, _ := img.At(bounds.Min.X, bounds.Min.Y).RGBA()
	if uint8(r>>8) != 255 || uint8(g>>8) != 0 || uint8(b>>8) != 0 {
		t.Fatalf("expected red pixel, got r=%d g=%d b=%d", uint8(r>>8), uint8(g>>8), uint8(b>>8))
	}
}

func TestExtractImagesFromPDFReturnsEmptySliceWhenPDFHasNoImages(t *testing.T) {
	path := writePDFWithImageFixture(t, embeddedImagePDFOptions{
		IncludeImage: false,
	})

	images, err := ExtractImagesFromPDF(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(images) != 0 {
		t.Fatalf("expected no extracted images, got %d", len(images))
	}
}

func TestExtractImagesFromPDFSkipsUnsupportedFilters(t *testing.T) {
	path := writePDFWithImageFixture(t, embeddedImagePDFOptions{
		FilterName: "DCTDecode",
		ImageData:  []byte("fake-jpeg-payload"),
	})

	images, err := ExtractImagesFromPDF(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(images) != 0 {
		t.Fatalf("expected unsupported filter to be skipped, got %d images", len(images))
	}
}

func TestExtractTextFromPDFMissingFile(t *testing.T) {
	_, err := ExtractTextFromPDF(filepath.Join(t.TempDir(), "missing.pdf"))
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestExtractImagesFromPDFMissingFile(t *testing.T) {
	_, err := ExtractImagesFromPDF(filepath.Join(t.TempDir(), "missing.pdf"))
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestExtractTextFromPDFBrokenFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "broken.pdf")
	if err := os.WriteFile(path, []byte("not a valid pdf"), 0644); err != nil {
		t.Fatalf("write broken pdf: %v", err)
	}

	_, err := ExtractTextFromPDF(path)
	if err == nil {
		t.Fatal("expected error for broken pdf")
	}
}

func TestExtractImagesFromPDFBrokenFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "broken.pdf")
	if err := os.WriteFile(path, []byte("not a valid pdf"), 0644); err != nil {
		t.Fatalf("write broken pdf: %v", err)
	}

	_, err := ExtractImagesFromPDF(path)
	if err == nil {
		t.Fatal("expected error for broken pdf")
	}
}

func TestNormalizePDFText(t *testing.T) {
	raw := "\n张三\r\n\r\n  工作经历  \n\n\nABC 科技\r\n"

	normalized := normalizePDFText(raw)
	expected := "张三\n\n工作经历\n\nABC 科技"
	if normalized != expected {
		t.Fatalf("expected %q, got %q", expected, normalized)
	}
}

func fixturePath(t *testing.T, name string) string {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}

	candidates := []string{
		filepath.Join(wd, "..", "..", "..", "..", "fixtures", name),
		filepath.Join(wd, "fixtures", name),
		filepath.Join("fixtures", name),
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	t.Fatalf("fixture %s not found from %s", name, wd)
	return ""
}

type embeddedImagePDFOptions struct {
	IncludeImage bool
	FilterName   string
	ImageData    []byte
}

func writePDFWithImageFixture(t *testing.T, options embeddedImagePDFOptions) string {
	t.Helper()

	includeImage := options.IncludeImage || options.FilterName != "" || len(options.ImageData) > 0
	if includeImage {
		if options.FilterName == "" {
			options.FilterName = "FlateDecode"
		}
		if options.ImageData == nil {
			options.ImageData = compressZlibBytes(t, []byte{255, 0, 0})
		}
	}

	contentStream := []byte("q\n1 0 0 1 0 0 cm\n")
	pageResource := "<< >>"
	objects := []pdfTestObject{
		{ID: 1, Body: []byte("<< /Type /Catalog /Pages 2 0 R >>")},
		{ID: 2, Body: []byte("<< /Type /Pages /Count 1 /Kids [3 0 R] >>")},
	}

	if includeImage {
		contentStream = append(contentStream, []byte("/Im0 Do\nQ\n")...)
		pageResource = "<< /XObject << /Im0 5 0 R >> >>"
	} else {
		contentStream = append(contentStream, []byte("Q\n")...)
	}

	objects = append(objects,
		pdfTestObject{
			ID:   3,
			Body: []byte(fmt.Sprintf("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 100 100] /Resources %s /Contents 4 0 R >>", pageResource)),
		},
		pdfTestObject{
			ID:   4,
			Body: pdfStreamObjectBody(contentStream, fmt.Sprintf("<< /Length %d >>", len(contentStream))),
		},
	)

	if includeImage {
		imageHeader := fmt.Sprintf("<< /Type /XObject /Subtype /Image /Width 1 /Height 1 /ColorSpace /DeviceRGB /BitsPerComponent 8 /Filter /%s /Length %d >>", options.FilterName, len(options.ImageData))
		objects = append(objects, pdfTestObject{
			ID:   5,
			Body: pdfStreamObjectBody(options.ImageData, imageHeader),
		})
	}

	path := filepath.Join(t.TempDir(), "embedded-image.pdf")
	if err := os.WriteFile(path, buildPDFDocument(objects), 0644); err != nil {
		t.Fatalf("write test pdf: %v", err)
	}

	return path
}

type pdfTestObject struct {
	ID   int
	Body []byte
}

func buildPDFDocument(objects []pdfTestObject) []byte {
	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n")

	maxObjectID := 0
	for _, object := range objects {
		if object.ID > maxObjectID {
			maxObjectID = object.ID
		}
	}

	offsets := make([]int, maxObjectID+1)
	for _, object := range objects {
		offsets[object.ID] = buf.Len()
		fmt.Fprintf(&buf, "%d 0 obj\n", object.ID)
		buf.Write(object.Body)
		if len(object.Body) == 0 || object.Body[len(object.Body)-1] != '\n' {
			buf.WriteByte('\n')
		}
		buf.WriteString("endobj\n")
	}

	xrefOffset := buf.Len()
	fmt.Fprintf(&buf, "xref\n0 %d\n", maxObjectID+1)
	buf.WriteString("0000000000 65535 f \n")
	for objectID := 1; objectID <= maxObjectID; objectID++ {
		fmt.Fprintf(&buf, "%010d 00000 n \n", offsets[objectID])
	}

	fmt.Fprintf(&buf, "trailer\n<< /Size %d /Root 1 0 R >>\n", maxObjectID+1)
	buf.WriteString("startxref\n")
	fmt.Fprintf(&buf, "%d\n%%%%EOF\n", xrefOffset)
	return buf.Bytes()
}

func pdfStreamObjectBody(streamData []byte, header string) []byte {
	body := make([]byte, 0, len(header)+len(streamData)+32)
	body = append(body, []byte(header)...)
	body = append(body, '\n')
	body = append(body, []byte("stream\n")...)
	body = append(body, streamData...)
	body = append(body, '\n')
	body = append(body, []byte("endstream")...)
	return body
}

func compressZlibBytes(t *testing.T, data []byte) []byte {
	t.Helper()

	var buf bytes.Buffer
	writer := zlib.NewWriter(&buf)
	if _, err := writer.Write(data); err != nil {
		t.Fatalf("compress test image bytes: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close zlib writer: %v", err)
	}

	return buf.Bytes()
}
