package lmtp

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"sync/atomic"
	"testing"

	"raven/internal/delivery/config"
	"raven/internal/socketmap/protocol"
)

// mockSocketmapServer starts a TCP listener and handles connections using the provided handler.
// It returns the listener's address and a cleanup function.
func mockSocketmapServer(t *testing.T, handler func(net.Conn)) (string, func()) {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			conn, err := l.Accept()
			if err != nil {
				return // Listener closed
			}
			go func(c net.Conn) {
				defer c.Close()
				handler(c)
			}(conn)
		}
	}()

	cleanup := func() {
		_ = l.Close()
		<-done
	}

	return l.Addr().String(), cleanup
}

func defaultMockHandler(t *testing.T, responses map[string]string) func(net.Conn) {
	return func(conn net.Conn) {
		reader := bufio.NewReader(conn)
		for {
			req, err := protocol.ReadNetstring(reader)
			if err != nil {
				return // Client disconnected or error
			}

			if !strings.HasPrefix(req, "user-exists ") {
				_ = protocol.WriteNetstring(conn, "PERM invalid command")
				continue
			}

			recipient := strings.TrimPrefix(req, "user-exists ")
			resp, ok := responses[recipient]
			if !ok {
				resp = "NOTFOUND"
			}

			err = protocol.WriteNetstring(conn, resp)
			if err != nil {
				return
			}
		}
	}
}

func TestSocketmapResolver_HappyPath(t *testing.T) {
	responses := map[string]string{
		"alias@example.com": "OK mapped@example.com",
	}

	addr, cleanup := mockSocketmapServer(t, defaultMockHandler(t, responses))
	defer cleanup()

	cfg := &config.Config{
		Socketmap: config.SocketmapConfig{
			Enabled:        true,
			Network:        "tcp",
			Address:        addr,
			TimeoutSeconds: 2,
		},
	}

	resolver := newIdentityResolver(cfg)
	if resolver == nil {
		t.Fatal("expected resolver, got nil")
	}
	defer resolver.Close()

	mapped, err := resolver.Resolve("alias@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mapped != "mapped@example.com" {
		t.Errorf("expected mapped@example.com, got %s", mapped)
	}
}

func TestSocketmapResolver_OKWithoutMapping(t *testing.T) {
	responses := map[string]string{
		"user@example.com": "OK ", // Or just "OK" depending on exactly how Fields splits
	}

	addr, cleanup := mockSocketmapServer(t, defaultMockHandler(t, responses))
	defer cleanup()

	cfg := &config.Config{
		Socketmap: config.SocketmapConfig{
			Enabled:        true,
			Network:        "tcp",
			Address:        addr,
			TimeoutSeconds: 2,
		},
	}

	resolver := newSocketmapIdentityResolver(cfg)
	defer resolver.Close()

	mapped, err := resolver.Resolve("user@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mapped != "user@example.com" {
		t.Errorf("expected original user@example.com, got %s", mapped)
	}
}

func TestSocketmapResolver_NotFound(t *testing.T) {
	responses := map[string]string{}

	addr, cleanup := mockSocketmapServer(t, defaultMockHandler(t, responses))
	defer cleanup()

	cfg := &config.Config{
		Socketmap: config.SocketmapConfig{
			Enabled:        true,
			Network:        "tcp",
			Address:        addr,
			TimeoutSeconds: 2,
		},
	}

	resolver := newSocketmapIdentityResolver(cfg)
	defer resolver.Close()

	mapped, err := resolver.Resolve("unknown@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mapped != "unknown@example.com" {
		t.Errorf("expected original unknown@example.com, got %s", mapped)
	}
}

func TestSocketmapResolver_ErrorResponses(t *testing.T) {
	responses := map[string]string{
		"perm@example.com": "PERM something bad",
		"temp@example.com": "TEMP try later",
	}

	addr, cleanup := mockSocketmapServer(t, defaultMockHandler(t, responses))
	defer cleanup()

	cfg := &config.Config{
		Socketmap: config.SocketmapConfig{
			Enabled:        true,
			Network:        "tcp",
			Address:        addr,
			TimeoutSeconds: 2,
		},
	}

	resolver := newSocketmapIdentityResolver(cfg)
	defer resolver.Close()

	_, err := resolver.Resolve("perm@example.com")
	if err == nil || !strings.Contains(err.Error(), "socketmap returned \"PERM something bad\"") {
		t.Errorf("expected PERM error, got %v", err)
	}

	_, err = resolver.Resolve("temp@example.com")
	if err == nil || !strings.Contains(err.Error(), "socketmap returned \"TEMP try later\"") {
		t.Errorf("expected TEMP error, got %v", err)
	}
}

func TestSocketmapResolver_Reconnect(t *testing.T) {
	var requests atomic.Int32
	handler := func(conn net.Conn) {
		reader := bufio.NewReader(conn)
		for {
			req, err := protocol.ReadNetstring(reader)
			if err != nil {
				return
			}

			reqNum := requests.Add(1)

			if reqNum == 1 {
				// Drop connection on first request
				conn.Close()
				return
			}

			if strings.HasPrefix(req, "user-exists ") {
				_ = protocol.WriteNetstring(conn, "OK mapped@example.com")
			}
		}
	}

	addr, cleanup := mockSocketmapServer(t, handler)
	defer cleanup()

	cfg := &config.Config{
		Socketmap: config.SocketmapConfig{
			Enabled:        true,
			Network:        "tcp",
			Address:        addr,
			TimeoutSeconds: 2,
		},
	}

	resolver := newSocketmapIdentityResolver(cfg)
	defer resolver.Close()

	mapped, err := resolver.Resolve("user@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mapped != "mapped@example.com" {
		t.Errorf("expected mapped@example.com, got %s", mapped)
	}
}

func TestSocketmapResolver_DialError(t *testing.T) {
	cfg := &config.Config{
		Socketmap: config.SocketmapConfig{
			Enabled:        true,
			Network:        "tcp",
			Address:        "127.0.0.1:0", // Connect to unused port
			TimeoutSeconds: 1,
		},
	}

	resolver := newSocketmapIdentityResolver(cfg)
	defer resolver.Close()

	_, err := resolver.Resolve("user@example.com")
	if err == nil {
		t.Fatal("expected dial error, got nil")
	}
}

func TestSocketmapResolver_ReadError(t *testing.T) {
	handler := func(conn net.Conn) {
		reader := bufio.NewReader(conn)
		req, err := protocol.ReadNetstring(reader)
		if err != nil {
			return
		}
		if strings.HasPrefix(req, "user-exists ") {
			// Write invalid netstring
			_, _ = conn.Write([]byte("invalid netstring"))
		}
	}

	addr, cleanup := mockSocketmapServer(t, handler)
	defer cleanup()

	cfg := &config.Config{
		Socketmap: config.SocketmapConfig{
			Enabled:        true,
			Network:        "tcp",
			Address:        addr,
			TimeoutSeconds: 2,
		},
	}

	resolver := newSocketmapIdentityResolver(cfg)
	defer resolver.Close()

	_, err := resolver.Resolve("user@example.com")
	if err == nil {
		t.Fatal("expected read error, got nil")
	}
}

func TestNewIdentityResolver_Disabled(t *testing.T) {
	resolver := newIdentityResolver(nil)
	if resolver != nil {
		t.Errorf("expected nil resolver when cfg is nil, got %v", resolver)
	}

	cfg := &config.Config{
		Socketmap: config.SocketmapConfig{
			Enabled: false,
		},
	}
	resolver = newIdentityResolver(cfg)
	if resolver != nil {
		t.Errorf("expected nil resolver when disabled, got %v", resolver)
	}
}

func TestIdentityResolverFunc(t *testing.T) {
	f := identityResolverFunc(func(recipient string) (string, error) {
		if recipient == "foo" {
			return "bar", nil
		}
		return "", fmt.Errorf("error")
	})

	res, err := f.Resolve("foo")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if res != "bar" {
		t.Errorf("expected bar, got %s", res)
	}

	err = f.Close()
	if err != nil {
		t.Errorf("unexpected error from Close: %v", err)
	}
}
