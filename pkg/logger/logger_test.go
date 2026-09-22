package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLazyDefault ensures L works without Init (tests, CLIs).
func TestLazyDefault(t *testing.T) {
	require.NotNil(t, L())
}

// TestInitStages ensures both formats initialize without panic.
func TestInitStages(t *testing.T) {
	Init("dev", "debug")
	require.NotNil(t, L())
	Init("prod", "warn")
	require.NotNil(t, L())
	Init("dev", "bogus-level-falls-back-to-info")
	require.NotNil(t, L())
}

// TestWithRequestID carries the correlation attribute.
func TestWithRequestID(t *testing.T) {
	assert.NotNil(t, WithRequestID("abc-123"))
	assert.Equal(t, L(), WithRequestID(""))
	assert.Equal(t, L(), WithRequestID("   "))
}
