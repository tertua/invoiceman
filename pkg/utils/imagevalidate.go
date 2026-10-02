package utils

import (
	"io"
	"net/http"
	"strings"
)

// AllowedTypes is a content-type → file-extension lookup. Callers supply the
// set of mime types they accept (e.g. raster images, or raster images + PDF).
type AllowedTypes map[string]string

// LogoAllowedTypes and ReceiptAllowedTypes are predefined sets used by the
// logo and receipt upload handlers respectively. Add new upload kinds by
// adding a new package-level variable here, not by re-implementing validation.
//
// Logo: raster only (PNG, JPEG, GIF, WEBP). SVG is intentionally excluded
// because inline SVG executes scripts in the viewer's origin (stored XSS).
//
// Receipt: raster + PDF.
var (
	LogoAllowedTypes = AllowedTypes{
		"image/png":  ".png",
		"image/jpeg": ".jpg",
		"image/gif":  ".gif",
		"image/webp": ".webp",
	}
	ReceiptAllowedTypes = AllowedTypes{
		"image/png":       ".png",
		"image/jpeg":      ".jpg",
		"image/gif":       ".gif",
		"image/webp":      ".webp",
		"application/pdf": ".pdf",
	}
)

// ValidateImage sniffs the content type by reading the first 512 bytes from r.
// Returns the normalized content type, the matching extension (from allowed),
// and ok=true if the sniffed type is in the allowed set. SVG is always
// rejected regardless of allowed because SVG content can carry scripts.
//
// IMPORTANT: r is consumed up to 512 bytes. Callers that need to upload the body
// after validation must Seek(0, io.SeekStart) before reading again.
func ValidateImage(r io.Reader, allowed AllowedTypes) (ct, ext string, ok bool) {
	head := make([]byte, 512)
	n, _ := io.ReadFull(r, head)
	ct = http.DetectContentType(head[:n])
	ct = strings.ToLower(strings.TrimSpace(ct))
	if ct == "image/svg+xml" {
		return ct, "", false
	}
	ext, ok = allowed[ct]
	return ct, ext, ok
}
