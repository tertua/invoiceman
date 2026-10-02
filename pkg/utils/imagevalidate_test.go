package utils

import (
	"bytes"
	"testing"
)

func TestValidateImage(t *testing.T) {
	png := []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D}
	jpeg := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 'J', 'F', 'I', 'F', 0x00}
	gif := []byte("GIF89a\x01\x00\x01\x00")
	webp := []byte("RIFF\x00\x00\x00\x00WEBPVP8 ")
	pdf := []byte("%PDF-1.4\n%\xc3\xa4\xc3\xbc\n")
	svg := []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`)
	html := []byte(`<html><body><script>alert(1)</script></body></html>`)
	js := []byte(`function x(){alert(1)}`)

	tests := []struct {
		name    string
		body    []byte
		allowed AllowedTypes
		wantCT  string
		wantExt string
		wantOK  bool
	}{
		{"png receipt", png, ReceiptAllowedTypes, "image/png", ".png", true},
		{"png logo", png, LogoAllowedTypes, "image/png", ".png", true},
		{"jpeg receipt", jpeg, ReceiptAllowedTypes, "image/jpeg", ".jpg", true},
		{"jpeg logo", jpeg, LogoAllowedTypes, "image/jpeg", ".jpg", true},
		{"gif receipt", gif, ReceiptAllowedTypes, "image/gif", ".gif", true},
		{"webp receipt", webp, ReceiptAllowedTypes, "image/webp", ".webp", true},
		{"pdf allowed as receipt", pdf, ReceiptAllowedTypes, "application/pdf", ".pdf", true},
		{"pdf rejected as logo", pdf, LogoAllowedTypes, "application/pdf", "", false},
		// DetectContentType does not recognize SVG: it reports text/plain, which
		// is not in any allowed set, so SVG is rejected on that path. The
		// explicit image/svg+xml guard is defence-in-depth in case the sniffed
		// type ever matches an allowed map that (mistakenly) lists SVG.
		{"svg rejected by sniffing", svg, LogoAllowedTypes, "text/plain; charset=utf-8", "", false},
		{"svg rejected even if allowed", svg, AllowedTypes{"image/svg+xml": ".svg"}, "text/plain; charset=utf-8", "", false},
		{"html disguised as image", html, ReceiptAllowedTypes, "text/html; charset=utf-8", "", false},
		{"js disguised as image", js, ReceiptAllowedTypes, "text/plain; charset=utf-8", "", false},
		{"empty body", nil, ReceiptAllowedTypes, "text/plain; charset=utf-8", "", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ct, ext, ok := ValidateImage(bytes.NewReader(tc.body), tc.allowed)
			if ct != tc.wantCT {
				t.Errorf("ct = %q; want %q", ct, tc.wantCT)
			}
			if ext != tc.wantExt {
				t.Errorf("ext = %q; want %q", ext, tc.wantExt)
			}
			if ok != tc.wantOK {
				t.Errorf("ok = %v; want %v", ok, tc.wantOK)
			}
		})
	}
}
