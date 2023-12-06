package hmac

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type Signer interface {
	Sign(req *http.Request, body []byte) (*http.Request, error)
}

func NewSigner(secret string) Signer {
	return &signer{
		secret: secret,
	}
}

type signer struct {
	secret string
}

func (s *signer) Sign(req *http.Request, body []byte) (*http.Request, error) {
	requestId := req.Header.Get(HeaderRequestId)
	if requestId == "" {
		requestId = uuid.NewString()
		req.Header.Set(HeaderRequestId, requestId)
	}

	timestamp := req.Header.Get(HeaderRequestTimestamp)
	if timestamp == "" {
		timestamp = time.Now().Format(time.RFC3339)
		req.Header.Set(HeaderRequestTimestamp, timestamp)
	}

	hash := hmac.New(sha256.New, []byte(s.secret))
	if _, err := hash.Write([]byte(requestId)); err != nil {
		return nil, fmt.Errorf("failed to write request ID to hash: %w", err)
	}
	if _, err := hash.Write([]byte(timestamp)); err != nil {
		return nil, fmt.Errorf("failed to write timestamp string to hash: %w", err)
	}
	if _, err := hash.Write(body); err != nil {
		return nil, fmt.Errorf("failed to write request body to hash: %w", err)
	}
	signature := fmt.Sprintf("sha256=%s", hex.EncodeToString(hash.Sum(nil)))
	req.Header.Set(HeaderSignature, signature)
	return req, nil
}

var _ Signer = (*signer)(nil)
