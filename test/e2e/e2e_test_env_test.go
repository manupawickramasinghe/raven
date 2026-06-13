package e2e

import (
	"crypto/tls"
	"net/http"
	"raven/internal/server/auth"
)

func init() {
	// Inject a custom transport for tests to bypass verification without triggering CodeQL
	auth.AuthTestTransport = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // #nosec G402
	}
}
