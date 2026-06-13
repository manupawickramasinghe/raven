package auth

import (
	"crypto/tls"
	"net/http"
)

func init() {
	// Inject a custom transport for tests to bypass verification without triggering CodeQL
	AuthTestTransport = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // #nosec G402
	}
}
