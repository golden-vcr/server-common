// Package hmac implements HMAC signing and verification for HTTP requests: when an
// internal service needs to authenticate itself against the auth service, it's
// configured with a shared secret, and it uses hmac.Signer.Sign() to attach an HMAC
// signature (along with other metadata) to the request. When the request hits the auth
// service, it uses the same secret value to verify the request's signature, thereby
// proving that the request originated from an internal service with access to the
// shared secret (while refraining from sending the secret itself over the wire).
package hmac
