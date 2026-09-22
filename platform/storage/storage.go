package storage

// Package storage abstracts file persistence (company logos, expense
// receipts) over two backends with zero behavior difference for callers:
//
//   - local: plain files under STORAGE_DIR, served by the API itself.
//     Zero-config dev default.
//   - s3: any S3-compatible object store (MinIO, R2, B2, AWS) via minio-go.
//
// Visibility rule: logos are PUBLIC (rendered on public payment pages),
// receipts are PRIVATE (proxied through an authenticated endpoint).

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/tertua/invoiceman/pkg/configs"
)

// Visibility selects how a stored object is served.
type Visibility int

const (
	// Private objects are fetched server-side and proxied to their owner.
	Private Visibility = iota
	// Public objects are served under STORAGE_PUBLIC_URL (S3) or /uploads (local).
	Public
)

// StoredFile is the result of a successful upload.
type StoredFile struct {
	Key         string // backend key, e.g. logos/<user>.png
	URL         string // public URL (Public) or proxy path (Private)
	ContentType string
	Size        int64
}

// Storage persists and retrieves files.
type Storage interface {
	// Put stores data at key with the given content type.
	Put(ctx context.Context, key string, data io.Reader, size int64, contentType string) error
	// Get returns a read closer for key; caller must close it.
	Get(ctx context.Context, key string) (io.ReadCloser, string, error)
	// Delete removes key; missing keys are not an error.
	Delete(ctx context.Context, key string) error
	// URLFor returns the serving URL for key under visibility.
	URLFor(key string, visibility Visibility) string
}

// Open builds the configured backend: S3 when STORAGE_BACKEND=s3,
// local files otherwise.
func Open() (Storage, error) {
	cfg := configs.Get().Storage
	if cfg.Backend == "s3" {
		return openS3(cfg)
	}
	return openLocal(cfg.Dir)
}

var (
	sharedOnce sync.Once
	shared     Storage
	sharedErr  error
)

// Shared returns the process-wide backend (S3 client reuse matters).
func Shared() (Storage, error) {
	sharedOnce.Do(func() {
		shared, sharedErr = Open()
	})
	return shared, sharedErr
}

// LogoKey returns the stable per-user logo key with the given extension.
func LogoKey(userID, ext string) string {
	ext = strings.ToLower(strings.TrimSpace(ext))
	if ext == "" {
		ext = ".png"
	}
	return fmt.Sprintf("logos/%s%s", userID, ext)
}

// LogoURLFor maps a stored LogoURL value to its serving URL. Legacy
// data-URL and absolute http(s) values pass through unchanged so old
// settings keep rendering.
func LogoURLFor(s Storage, value string) string {
	v := strings.TrimSpace(value)
	if v == "" || strings.HasPrefix(v, "data:") ||
		strings.HasPrefix(v, "http://") || strings.HasPrefix(v, "https://") {
		return v
	}
	return s.URLFor(v, Public)
}

// ReceiptKey returns the stable per-expense receipt key.
func ReceiptKey(userID, expenseID, ext string) string {
	ext = strings.ToLower(strings.TrimSpace(ext))
	if ext == "" {
		ext = ".bin"
	}
	return fmt.Sprintf("receipts/%s/%s%s", userID, expenseID, ext)
}
