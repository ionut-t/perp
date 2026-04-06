package genai

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/genai"
)

// TestErrorsAsAPIError verifies that errors.As correctly extracts a genai.APIError
// even when it is wrapped. This is the pattern used in Ask() and SetModel() after
// fixing the original code which passed &genai.APIError{} (a temporary) and then
// did a direct type assertion that would panic on wrapped errors.
func TestErrorsAsAPIError(t *testing.T) {
	t.Parallel()

	original := genai.APIError{
		Code:    429,
		Message: "quota exceeded",
		Status:  "RESOURCE_EXHAUSTED",
	}

	t.Run("direct error", func(t *testing.T) {
		extracted, ok := errors.AsType[genai.APIError](original)
		require.True(t, ok, "errors.AsType should match a direct genai.APIError")
		assert.Equal(t, original.Message, extracted.Message)
	})

	t.Run("wrapped error", func(t *testing.T) {
		wrapped := fmt.Errorf("transport: %w", original)

		extracted, ok := errors.AsType[genai.APIError](wrapped)
		require.True(t, ok, "errors.AsType should match genai.APIError through wrapping")
		assert.Equal(t, original.Message, extracted.Message)
	})

	t.Run("doubly wrapped error", func(t *testing.T) {
		wrapped := fmt.Errorf("outer: %w", fmt.Errorf("inner: %w", original))

		extracted, ok := errors.AsType[genai.APIError](wrapped)
		require.True(t, ok, "errors.AsType should match genai.APIError through double wrapping")
		assert.Equal(t, original.Code, extracted.Code)
	})

	t.Run("unrelated error does not match", func(t *testing.T) {
		_, ok := errors.AsType[genai.APIError](errors.New("some other error"))
		assert.False(t, ok, "errors.AsType should not match an unrelated error")
	})
}
