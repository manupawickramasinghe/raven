package thunder

import (
	"crypto/tls"
	"net/http"
	"os"
	"testing"
)

func TestGetHTTPClient(t *testing.T) {
	client := GetHTTPClient()

	if client == nil {
		t.Fatalf("GetHTTPClient returned nil")
	}

	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("Transport is not *http.Transport")
	}

	tlsConfig := transport.TLSClientConfig
	if tlsConfig == nil {
		t.Fatalf("TLSClientConfig is nil")
	}

	if tlsConfig.InsecureSkipVerify {
		t.Errorf("Expected InsecureSkipVerify to be false, got true")
	}

	if tlsConfig.MinVersion != tls.VersionTLS12 {
		t.Errorf("Expected MinVersion to be TLS 1.2 (%x), got %x", tls.VersionTLS12, tlsConfig.MinVersion)
	}
}

func TestGetHTTPClient_WithInternalCA(t *testing.T) {
	// Create a dummy CA cert file
	tmpFile, err := os.CreateTemp("", "dummy-ca-*.pem")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write some dummy content (doesn't need to be a valid cert, just testing file existence check)
	// Actually, AppendCertsFromPEM will just fail to append if it's invalid, which is fine for this test
	// We just want to ensure it doesn't panic and reads the env var
	_, _ = tmpFile.WriteString("-----BEGIN CERTIFICATE-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA...\n-----END CERTIFICATE-----")
	tmpFile.Close()

	os.Setenv("INTERNAL_CA_CERT_PATH", tmpFile.Name())
	defer os.Unsetenv("INTERNAL_CA_CERT_PATH")

	client := GetHTTPClient()

	if client == nil {
		t.Fatalf("GetHTTPClient returned nil")
	}
}
