package hmac

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

var ErrVerificationFailed = errors.New("verification failed")

type Verifier interface {
	Verify(req *http.Request, body []byte) error
}

func NewVerifier(secret string) Verifier {
	return &verifier{
		secret: secret,
	}
}

type verifier struct {
	secret string
}

func (v *verifier) Verify(req *http.Request, body []byte) error {
	requestId := req.Header.Get(HeaderRequestId)
	if requestId == "" {
		return ErrVerificationFailed
	}

	timestamp := req.Header.Get(HeaderRequestTimestamp)
	if timestamp == "" {
		return ErrVerificationFailed
	}

	signatureHeader := req.Header.Get(HeaderSignature)
	if signatureHeader == "" || !strings.HasPrefix(signatureHeader, "sha256=") {
		return ErrVerificationFailed
	}
	expectedHash := []byte(strings.TrimPrefix(req.Header.Get(HeaderSignature), "sha256="))

	hash := hmac.New(sha256.New, []byte(v.secret))
	if _, err := hash.Write([]byte(requestId)); err != nil {
		return fmt.Errorf("failed to write request ID to hash: %w", err)
	}
	if _, err := hash.Write([]byte(timestamp)); err != nil {
		return fmt.Errorf("failed to write timestamp string to hash: %w", err)
	}
	if _, err := hash.Write(body); err != nil {
		return fmt.Errorf("failed to write request body to hash: %w", err)
	}
	computedHash := hex.EncodeToString(hash.Sum(nil))

	if !hmac.Equal(expectedHash, []byte(computedHash)) {
		return ErrVerificationFailed
	}
	return nil
}

var _ Verifier = (*verifier)(nil)
