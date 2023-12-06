package hmac

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Verify(t *testing.T) {
	v := NewVerifier("my-secret")

	t.Run("request with missing signature is not verified", func(t *testing.T) {
		body := []byte("hello world")
		req, err := http.NewRequest(http.MethodPost, "/somewhere", bytes.NewReader(body))
		assert.NoError(t, err)
		err = v.Verify(req, body)
		assert.ErrorIs(t, err, ErrVerificationFailed)
	})

	t.Run("request with incorrect signature is not verified", func(t *testing.T) {
		body := []byte("hello world")
		req, err := http.NewRequest(http.MethodPost, "/somewhere", bytes.NewReader(body))
		assert.NoError(t, err)
		req.Header.Set(HeaderRequestId, "d6c6a6d0-bb4e-4ff2-8188-4dda238f9223")
		req.Header.Set(HeaderRequestTimestamp, "2023-12-06T21:06:04+00:00")
		req.Header.Set(HeaderSignature, "sha256=deadbeef")
		err = v.Verify(req, body)
		assert.ErrorIs(t, err, ErrVerificationFailed)
	})

	t.Run("request with correct signature is verified", func(t *testing.T) {
		body := []byte("hello world")
		req, err := http.NewRequest(http.MethodPost, "/somewhere", bytes.NewReader(body))
		assert.NoError(t, err)
		req.Header.Set(HeaderRequestId, "d6c6a6d0-bb4e-4ff2-8188-4dda238f9223")
		req.Header.Set(HeaderRequestTimestamp, "2023-12-06T21:06:04+00:00")
		req.Header.Set(HeaderSignature, "sha256=d1550fb3eea5eb856f5d0297f45568dfb19cfa4f4df3bb8a02e57487a6a8951b")
		err = v.Verify(req, body)
		assert.NoError(t, err)
	})
}
