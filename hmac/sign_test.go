package hmac

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func Test_Sign(t *testing.T) {
	s := NewSigner("my-secret")

	t.Run("headers are populated as expected", func(t *testing.T) {
		// Verify that we can successfully sign a simple request
		body := []byte("hello world")
		req, err := http.NewRequest(http.MethodPost, "/somewhere", bytes.NewReader(body))
		assert.NoError(t, err)
		req, err = s.Sign(req, body)
		assert.NoError(t, err)

		// Verify that all expected headers are set on the resulting request
		assert.NotEmpty(t, req.Header.Get(HeaderRequestId))
		assert.NotEmpty(t, req.Header.Get(HeaderRequestTimestamp))
		assert.NotEmpty(t, req.Header.Get(HeaderSignature))

		// Verify that header values match expected types/formats
		_, err = uuid.Parse(req.Header.Get(HeaderRequestId))
		assert.NoError(t, err)
		_, err = time.Parse(time.RFC3339, req.Header.Get(HeaderRequestTimestamp))
		assert.NoError(t, err)
		assert.True(t, strings.HasPrefix(req.Header.Get(HeaderSignature), "sha256="))

		// Verify that the new request's body is still opened for read
		bodyCopy, err := io.ReadAll(req.Body)
		assert.NoError(t, err)
		assert.Equal(t, body, bodyCopy)
	})

	t.Run("signature is computed as expected", func(t *testing.T) {
		// Sign another request, this time with pre-filled ID and timestamp so the
		// signature is deterministic
		body := []byte("hello world")
		req, err := http.NewRequest(http.MethodPost, "/somewhere", bytes.NewReader(body))
		assert.NoError(t, err)
		req.Header.Set(HeaderRequestId, "d6c6a6d0-bb4e-4ff2-8188-4dda238f9223")
		req.Header.Set(HeaderRequestTimestamp, "2023-12-06T21:06:04+00:00")
		req, err = s.Sign(req, body)
		assert.NoError(t, err)
		assert.Equal(t, "sha256=d1550fb3eea5eb856f5d0297f45568dfb19cfa4f4df3bb8a02e57487a6a8951b", req.Header.Get(HeaderSignature))
	})
}
