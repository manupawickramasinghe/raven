package thunder

import (
	"crypto/tls"
	"crypto/x509"
	"log"
	"net/http"
	"os"
	"time"
)

// GetHTTPClient returns an HTTP client with secure TLS defaults.
func GetHTTPClient() *http.Client {
	caCertPool, err := x509.SystemCertPool()
	if err != nil {
		log.Printf("Failed to load system root CAs, falling back to empty pool: %v", err)
		caCertPool = x509.NewCertPool()
	}

	caCertPath := os.Getenv("INTERNAL_CA_CERT_PATH")
	if caCertPath == "" {
		caCertPath = "/certs/ca.pem"
	}

	if caCert, err := os.ReadFile(caCertPath); err == nil {
		if ok := caCertPool.AppendCertsFromPEM(caCert); !ok {
			log.Printf("Failed to append internal CA cert from %s", caCertPath)
		} else {
			log.Printf("Successfully loaded internal CA cert from %s", caCertPath)
		}
	} else if !os.IsNotExist(err) {
		log.Printf("Failed to read internal CA cert from %s: %v", caCertPath, err)
	}

	return &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
				RootCAs:    caCertPool,
			},
		},
	}
}
