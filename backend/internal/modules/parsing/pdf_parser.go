package parsing

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"

	"github.com/ledongthuc/pdf"
)

type PDFParserImpl struct{}

func NewPDFParser() *PDFParserImpl {
	return &PDFParserImpl{}
}

func (p *PDFParserImpl) Parse(path string) (*ParsedContent, error) {
	text, err := ExtractTextFromPDF(path)
	if err != nil {
		return nil, err
	}

	images, err := ExtractImagesFromPDF(path)
	if err != nil {
		return nil, err
	}

	return &ParsedContent{
		Text:   text,
		Images: images,
	}, nil
}

func ExtractTextFromPDF(path string) (string, error) {
	file, reader, err := pdf.Open(path)
	if err != nil {
		return "", fmt.Errorf("open pdf: %w", err)
	}
	defer file.Close()

	plainText, err := reader.GetPlainText()
	if err != nil {
		return "", fmt.Errorf("extract pdf text: %w", err)
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, plainText); err != nil {
		return "", fmt.Errorf("read pdf text: %w", err)
	}

	return normalizePDFText(buf.String()), nil
}

func normalizePDFText(raw string) string {
	return cleanParsedText(raw)
}

func ExtractImagesFromPDF(path string) ([]ParsedImage, error) {
	file, reader, err := pdf.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open pdf: %w", err)
	}
	defer file.Close()

	return extractImagesFromPDFReader(reader)
}

func extractImagesFromPDFReader(reader *pdf.Reader) ([]ParsedImage, error) {
	images := make([]ParsedImage, 0)
	for pageNum := 1; pageNum <= reader.NumPage(); pageNum++ {
		page := reader.Page(pageNum)
		if page.V.IsNull() {
			continue
		}

		pageImages := extractPageImages(pageNum, page.Resources().Key("XObject"))
		images = append(images, pageImages...)
	}

	return images, nil
}

func extractPageImages(pageNum int, xObjects pdf.Value) []ParsedImage {
	if xObjects.IsNull() {
		return nil
	}

	images := make([]ParsedImage, 0, len(xObjects.Keys()))
	for _, objectName := range xObjects.Keys() {
		objectValue := xObjects.Key(objectName)
		if objectValue.Key("Subtype").Name() != "Image" {
			continue
		}

		parsedImage, ok := extractImageObject(pageNum, objectName, objectValue)
		if !ok {
			continue
		}
		images = append(images, parsedImage)
	}

	return images
}

func extractImageObject(pageNum int, objectName string, objectValue pdf.Value) (ParsedImage, bool) {
	if !supportsEmbeddedImageFilter(objectValue.Key("Filter")) {
		return ParsedImage{}, false
	}

	rawImageData, err := readPDFStreamSafely(objectValue)
	if err != nil {
		return ParsedImage{}, false
	}

	base64PNG, err := encodeEmbeddedImageAsPNGBase64(objectValue, rawImageData)
	if err != nil {
		return ParsedImage{}, false
	}

	return ParsedImage{
		Description: fmt.Sprintf("Embedded image from page %d (%s)", pageNum, objectName),
		DataBase64:  base64PNG,
	}, true
}

func supportsEmbeddedImageFilter(filter pdf.Value) bool {
	switch filter.Kind() {
	case pdf.Null:
		return true
	case pdf.Name:
		name := filter.Name()
		return name != "" && isSupportedEmbeddedImageFilterName(name)
	case pdf.Array:
		for index := 0; index < filter.Len(); index++ {
			filterName := filter.Index(index)
			if filterName.Kind() != pdf.Name || !isSupportedEmbeddedImageFilterName(filterName.Name()) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func isSupportedEmbeddedImageFilterName(name string) bool {
	switch name {
	case "FlateDecode", "ASCII85Decode":
		return true
	default:
		return false
	}
}

func readPDFStreamSafely(value pdf.Value) (data []byte, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("decode image stream: %v", recovered)
		}
	}()

	stream := value.Reader()
	defer func() {
		closeErr := stream.Close()
		if err == nil && closeErr != nil {
			err = fmt.Errorf("close image stream: %w", closeErr)
		}
	}()

	data, err = io.ReadAll(stream)
	if err != nil {
		return nil, fmt.Errorf("read image stream: %w", err)
	}

	return data, nil
}

func encodeEmbeddedImageAsPNGBase64(objectValue pdf.Value, rawImageData []byte) (string, error) {
	width := int(objectValue.Key("Width").Int64())
	height := int(objectValue.Key("Height").Int64())
	if width <= 0 || height <= 0 {
		return "", fmt.Errorf("invalid image dimensions %dx%d", width, height)
	}

	bitsPerComponent := int(objectValue.Key("BitsPerComponent").Int64())
	if bitsPerComponent != 8 {
		return "", fmt.Errorf("unsupported bits per component %d", bitsPerComponent)
	}

	colorSpace := resolveEmbeddedImageColorSpace(objectValue.Key("ColorSpace"))
	if colorSpace == "" {
		return "", fmt.Errorf("unsupported color space")
	}

	var img image.Image
	switch colorSpace {
	case "DeviceGray":
		img = buildGrayImage(width, height, rawImageData)
	case "DeviceRGB":
		img = buildRGBAImage(width, height, rawImageData)
	case "DeviceCMYK":
		img = buildCMYKImage(width, height, rawImageData)
	default:
		return "", fmt.Errorf("unsupported color space %s", colorSpace)
	}
	if img == nil {
		return "", fmt.Errorf("invalid %s image payload", colorSpace)
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", fmt.Errorf("encode png: %w", err)
	}

	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

func resolveEmbeddedImageColorSpace(value pdf.Value) string {
	switch value.Kind() {
	case pdf.Name:
		return value.Name()
	case pdf.Array:
		if value.Len() == 0 {
			return ""
		}

		switch value.Index(0).Name() {
		case "ICCBased":
			switch value.Index(1).Key("N").Int64() {
			case 1:
				return "DeviceGray"
			case 3:
				return "DeviceRGB"
			case 4:
				return "DeviceCMYK"
			default:
				return ""
			}
		default:
			return value.Index(0).Name()
		}
	default:
		return ""
	}
}

func buildGrayImage(width int, height int, rawImageData []byte) image.Image {
	expectedBytes := width * height
	if len(rawImageData) < expectedBytes {
		return nil
	}

	img := image.NewGray(image.Rect(0, 0, width, height))
	copy(img.Pix, rawImageData[:expectedBytes])
	return img
}

func buildRGBAImage(width int, height int, rawImageData []byte) image.Image {
	expectedBytes := width * height * 3
	if len(rawImageData) < expectedBytes {
		return nil
	}

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	dataIndex := 0
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			pixelIndex := img.PixOffset(x, y)
			img.Pix[pixelIndex+0] = rawImageData[dataIndex+0]
			img.Pix[pixelIndex+1] = rawImageData[dataIndex+1]
			img.Pix[pixelIndex+2] = rawImageData[dataIndex+2]
			img.Pix[pixelIndex+3] = 0xff
			dataIndex += 3
		}
	}

	return img
}

func buildCMYKImage(width int, height int, rawImageData []byte) image.Image {
	expectedBytes := width * height * 4
	if len(rawImageData) < expectedBytes {
		return nil
	}

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	dataIndex := 0
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			cmyk := color.CMYK{
				C: rawImageData[dataIndex+0],
				M: rawImageData[dataIndex+1],
				Y: rawImageData[dataIndex+2],
				K: rawImageData[dataIndex+3],
			}
			r, g, b, _ := cmyk.RGBA()
			pixelIndex := img.PixOffset(x, y)
			img.Pix[pixelIndex+0] = uint8(r >> 8)
			img.Pix[pixelIndex+1] = uint8(g >> 8)
			img.Pix[pixelIndex+2] = uint8(b >> 8)
			img.Pix[pixelIndex+3] = 0xff
			dataIndex += 4
		}
	}

	return img
}
