package contenttype

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsURLs(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		input := `https://www.google.com
https://www.google.com
https://www.google.com`
		require.True(t, IsURLs(input))
	})

	t.Run("fail", func(t *testing.T) {
		input := `https://www.google.com
https://www.google.com
https://www.google.com
not a url`
		require.False(t, IsURLs(input))
	})
}
