package parsing

import "errors"

const (
	AssetTypeResumePDF   = "resume_pdf"
	AssetTypeResumeDOCX  = "resume_docx"
	AssetTypeResumeImage = "resume_image"
	AssetTypeGitRepo     = "git_repo"
	AssetTypeNote        = "note"
)

var (
	ErrDatabaseNotConfigured      = errors.New("database is not configured")
	ErrAssetURIMissing            = errors.New("asset uri is required")
	ErrAssetContentMissing        = errors.New("asset content is required")
	ErrAssetTypeSkipped           = errors.New("asset type is skipped in v1")
	ErrUnsupportedAssetType       = errors.New("unsupported asset type")
	ErrProjectNotFound            = errors.New("project not found")
	ErrNoUsableAssets             = errors.New("project has no usable assets")
	ErrPDFParserNotConfigured     = errors.New("pdf parser is not configured")
	ErrDOCXParserNotConfigured    = errors.New("docx parser is not configured")
	ErrGitExtractorNotConfigured  = errors.New("git extractor is not configured")
)

type ParsedImage struct {
	Description string `json:"description"`
	DataBase64  string `json:"data_base64"`
}

type ParsedContent struct {
	AssetID uint          `json:"asset_id"`
	Type    string        `json:"type"`
	Text    string        `json:"text"`
	Images  []ParsedImage `json:"images,omitempty"`
}

type PdfParser interface {
	Parse(path string) (*ParsedContent, error)
}

type DocxParser interface {
	Parse(path string) (*ParsedContent, error)
}

type GitExtractor interface {
	Extract(repoURL string) (*ParsedContent, error)
}
