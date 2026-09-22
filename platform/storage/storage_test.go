package storage

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLocalRoundTrip covers put/get/delete plus traversal rejection.
func TestLocalRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s, err := openLocal(dir)
	require.NoError(t, err)

	payload := []byte("hello storage")
	require.NoError(t, s.Put(context.Background(), "logos/u1.png", bytes.NewReader(payload), int64(len(payload)), "image/png"))

	rc, ct, err := s.Get(context.Background(), "logos/u1.png")
	require.NoError(t, err)
	got, err := io.ReadAll(rc)
	require.NoError(t, err)
	require.NoError(t, rc.Close())
	assert.Equal(t, payload, got)
	assert.Equal(t, "image/png", ct)

	assert.Equal(t, "/uploads/logos/u1.png", s.URLFor("logos/u1.png", Public))
	assert.Equal(t, "receipts/u1/e1.pdf", s.URLFor("receipts/u1/e1.pdf", Private))

	require.NoError(t, s.Delete(context.Background(), "logos/u1.png"))
	_, _, err = s.Get(context.Background(), "logos/u1.png")
	require.Error(t, err)
	// Deleting a missing key is not an error.
	require.NoError(t, s.Delete(context.Background(), "logos/u1.png"))

	// Parent segments cannot escape the root: Clean anchors at "/".
	require.NoError(t, s.Put(context.Background(), "../escape.png", bytes.NewReader(payload), 1, "image/png"))
	assert.FileExists(t, filepath.Join(dir, "escape.png"))
}

// TestLocalCreatesDir ensures the upload root is auto-created.
func TestLocalCreatesDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "uploads")
	_, err := openLocal(dir)
	require.NoError(t, err)
	assert.DirExists(t, dir)
}

// TestOpenLocalDefault resolves the local backend from env.
func TestOpenLocalDefault(t *testing.T) {
	t.Setenv("STORAGE_BACKEND", "local")
	t.Setenv("STORAGE_DIR", t.TempDir())
	s, err := Open()
	require.NoError(t, err)
	assert.NotNil(t, s)
}

// TestLogoURLForPassThrough keeps legacy values rendering.
func TestLogoURLForPassThrough(t *testing.T) {
	s, err := openLocal(t.TempDir())
	require.NoError(t, err)
	assert.Equal(t, "", LogoURLFor(s, ""))
	assert.Equal(t, "data:image/png;base64,AAA", LogoURLFor(s, "data:image/png;base64,AAA"))
	assert.Equal(t, "https://cdn.example.com/l.png", LogoURLFor(s, "https://cdn.example.com/l.png"))
	assert.Equal(t, "/uploads/logos/u1.png", LogoURLFor(s, "logos/u1.png"))
}

// TestMain isolates storage tests from the repo .env.
func TestMain(m *testing.M) {
	os.Setenv("STORAGE_BACKEND", "local")
	os.Exit(m.Run())
}
