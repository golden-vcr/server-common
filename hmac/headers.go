package hmac

const (
	// HeaderRequestId is the name of the header that carries a unique ID generated for
	// an HMAC-signed request
	HeaderRequestId = "x-hmac-request-id"

	// HeaderRequestTimestamp is the name of the header that carries an RFC3339
	// timestamp indicating when the request was made
	HeaderRequestTimestamp = "x-hmac-request-timestamp"

	// HeaderSignature is the name of the header that carries the HMAC signature
	// computed from the concatenation of the request ID, timestamp string, and request
	// payload body
	HeaderSignature = "x-hmac-signature"
)
